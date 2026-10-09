package billing

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/invoice"
)

// ArchiveOptions control Archive.
type ArchiveOptions struct {
	// Replace allows re-archiving a working copy from `archive edit` over
	// the archived file it was opened from. The previous version is kept in
	// the archive's history directory.
	Replace bool
	// Confirm is asked, when Replace is not set and archiving would
	// replace archived files, whether to replace them. It returns nil to
	// go ahead. Without Confirm, Archive returns the *ArchiveReplaceError.
	Confirm func(*ArchiveReplaceError) error
	// DryRun runs every check and returns the result without writing. The
	// result's backups have no BackupPath.
	DryRun bool
	// AssumeBuilt checks the invoice as if `build` had already marked it
	// built, for `build --archive --dry-run`, which does not mark it.
	AssumeBuilt bool
}

// Archive moves a built invoice, or a working copy from `archive edit`,
// into the archive. Re-archiving a working copy over archived files
// returns an *ArchiveReplaceError, after all other checks passed and before
// anything is written, unless opts.Replace is set or opts.Confirm agrees.
// The replaced files are first copied to the archive's history directory.
func (s *Service) Archive(path string, opts ArchiveOptions) (ArchiveResult, error) {
	result, err := s.archive(path, opts)
	var replaceErr *ArchiveReplaceError
	if opts.Replace || opts.Confirm == nil || !errors.As(err, &replaceErr) {
		return result, err
	}
	if err := opts.Confirm(replaceErr); err != nil {
		return ArchiveResult{}, err
	}
	opts.Replace = true
	return s.archive(path, opts)
}

func (s *Service) archive(path string, opts ArchiveOptions) (ArchiveResult, error) {
	head, err := s.headWithInvoice(path)
	if err != nil {
		return ArchiveResult{}, err
	}
	status := head.Status
	if opts.AssumeBuilt {
		status, _ = status.Apply(invoice.Building)
	}
	dir, err := s.Archives.Dir()
	if err != nil {
		return ArchiveResult{}, err
	}
	if strings.TrimSpace(dir) == "" {
		return ArchiveResult{}, errors.New("archive directory is unavailable")
	}
	if err := archivable(path, head, status); err != nil {
		return ArchiveResult{}, err
	}
	place, err := s.Archives.Place(path, head)
	if err != nil {
		return ArchiveResult{}, err
	}
	if err := s.numberUnique(path, head); err != nil {
		return ArchiveResult{}, err
	}
	return s.Archives.Add(path, place, AddOptions{
		Replace: opts.Replace,
		DryRun:  opts.DryRun,
		Now:     s.Now(),
		Change: func(inv *invoice.Invoice) error {
			inv.Header.Status = invoice.Text(invoice.Archived)
			inv.Archive = nil
			return nil
		},
	})
}

// archivable returns why the invoice at path, whose head is head, cannot be
// archived with status: a working copy is re-archived when editing or
// built, any other invoice archived when built.
func archivable(path string, head Head, status invoice.Status) error {
	switch {
	case head.WorkingCopy() && !status.Allows(invoice.Rearchiving):
		return fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", path, status)
	case head.WorkingCopy():
		return nil
	case status == "":
		return fmt.Errorf("%s: invoice.status: missing value", path)
	case !status.Allows(invoice.Archiving):
		return fmt.Errorf("%s: invoice.status must be `built` before archiving, got `%s`", path, status)
	}
	return nil
}

// headWithInvoice reads the head of the invoice at path, which must have an
// `invoice` mapping. Values that do not decode read as unset.
func (s *Service) headWithInvoice(path string) (Head, error) {
	head, err := s.Invoices.Head(path)
	if err != nil && !isDecodeError(err) {
		return Head{}, err
	}
	if err := requireHeader(path, head); err != nil {
		return Head{}, err
	}
	return head, nil
}

// requireHeader returns why the `invoice` key of the invoice at path is not
// a mapping, or nil.
func requireHeader(path string, head Head) error {
	switch head.Header {
	case HeaderMissing:
		return fmt.Errorf("%s: missing `invoice` mapping", path)
	case HeaderOther:
		return fmt.Errorf("%s: `invoice` must be a mapping", path)
	}
	return nil
}

// CheckNumberUnique returns a *invoice.DuplicateInvoiceNumberError when
// the invoice at path uses a number that an archived invoice already has.
// The archived file the invoice was opened from (`archive edit`) does not
// count as a duplicate.
func (s *Service) CheckNumberUnique(path string) error {
	head, err := s.Invoices.Head(path)
	if err != nil && !isDecodeError(err) {
		return err
	}
	if !head.HasHeader {
		return nil
	}
	return s.numberUnique(path, head)
}

// numberUnique returns a *invoice.DuplicateInvoiceNumberError when an
// archived invoice other than the one the invoice at path replaces has its
// number.
func (s *Service) numberUnique(path string, head Head) error {
	archived, err := s.Archives.Duplicate(path, head)
	if err != nil || archived == "" {
		return err
	}
	return &invoice.DuplicateInvoiceNumberError{InvoicePath: path, InvoiceNumber: head.Number, ArchivedPath: archived}
}

// archiveEntries returns the archived invoices sorted by Filename.
func (s *Service) archiveEntries() ([]ArchiveEntry, error) {
	entries, err := s.Archives.Entries()
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Filename < entries[j].Filename })
	return entries, nil
}

// latestArchived returns the archived invoice of customerID issued last.
func (s *Service) latestArchived(customerID string) (string, bool, error) {
	entries, err := s.archiveEntries()
	if err != nil {
		return "", false, err
	}
	var latest ArchiveEntry
	found := false
	for _, entry := range entries {
		if entry.CustomerID != customerID {
			continue
		}
		if !found || entry.Newer(latest) {
			latest, found = entry, true
		}
	}
	return latest.Path, found, nil
}

// Newer reports whether e was issued after other: by issue date, where a
// date that parses beats one that does not, then by invoice number,
// Filename and Path.
func (e ArchiveEntry) Newer(other ArchiveEntry) bool {
	leftDate, leftErr := time.Parse("2006-01-02", strings.TrimSpace(e.IssueDate))
	rightDate, rightErr := time.Parse("2006-01-02", strings.TrimSpace(other.IssueDate))
	leftOK, rightOK := leftErr == nil, rightErr == nil

	switch {
	case leftOK && !rightOK:
		return true
	case !leftOK && rightOK:
		return false
	case leftOK && rightOK && !leftDate.Equal(rightDate):
		return leftDate.After(rightDate)
	}

	switch {
	case e.Number != other.Number:
		return e.Number > other.Number
	case e.Filename != other.Filename:
		return e.Filename > other.Filename
	default:
		return e.Path > other.Path
	}
}

// ArchiveList is the archive's content.
type ArchiveList struct {
	// Dir is the archive directory, "" when there is none.
	Dir     string
	Entries []ArchiveEntry
}

// ListArchive lists the archived invoices sorted by file name.
func (s *Service) ListArchive() (ArchiveList, error) {
	entries, err := s.archiveEntries()
	if err != nil {
		return ArchiveList{}, err
	}
	dir, err := s.Archives.Dir()
	if err != nil {
		return ArchiveList{}, err
	}
	return ArchiveList{Dir: dir, Entries: entries}, nil
}

// EditOptions control EditArchived.
type EditOptions struct {
	// Overwrite replaces an existing working copy, unless it is in the
	// archive directory.
	Overwrite bool
	// DryRun runs every check and writes nothing.
	DryRun bool
}

// Edited is the working copy EditArchived made.
type Edited struct {
	Path     string
	Archived string
}

// EditArchived copies the archived invoice ref into workDir as a working
// copy with status editing and a link back to where it is re-archived.
func (s *Service) EditArchived(ref, workDir string, opts EditOptions) (Edited, error) {
	checkout, err := s.Archives.Checkout(ref, workDir)
	if err != nil {
		return Edited{}, err
	}
	// The working copy keeps keys invox does not know, so opening an
	// archived invoice never fails over them; validate reports them.
	archived, err := s.Invoices.ArchivedHead(checkout.Archived)
	if err != nil && !isDecodeError(err) {
		return Edited{}, err
	}
	if err := lenient(err); err != nil {
		return Edited{}, err
	}
	if err := requireHeader(checkout.Archived, archived); err != nil {
		return Edited{}, err
	}
	if _, err := s.Invoices.Destination(checkout.Path, workDir, "", opts.Overwrite); err != nil {
		return Edited{}, err
	}
	if err := s.refuseArchivedOverwrite(checkout.Path, opts.Overwrite); err != nil {
		return Edited{}, err
	}
	err = s.Invoices.Create(checkout.Path, checkout.Archived, invoice.Invoice{
		Header:  &invoice.Header{Status: invoice.Text(invoice.Editing)},
		Archive: &checkout.Link,
	}, CreateOptions{Overwrite: opts.Overwrite, DryRun: opts.DryRun, Check: CheckNone})
	if err != nil {
		return Edited{}, err
	}
	return Edited{Path: checkout.Path, Archived: checkout.Archived}, nil
}

// refuseArchivedOverwrite returns an *ArchivedOutputError when overwrite
// would replace an existing file inside the archive directory: an archived
// invoice is replaced only by re-archiving, which keeps a backup.
func (s *Service) refuseArchivedOverwrite(path string, overwrite bool) error {
	if !overwrite {
		return nil
	}
	protected, err := s.Archives.Protects(path)
	if err != nil {
		return err
	}
	if protected {
		return &ArchivedOutputError{Path: path}
	}
	return nil
}
