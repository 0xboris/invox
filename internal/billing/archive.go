package billing

import (
	"errors"
	"fmt"
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
	inv, err := s.Invoices.Load(path)
	// Values that do not decode read as unset, except the header itself.
	if err := keepDecodeErrors(err, func(e *DecodeError) bool { return e.Field == "invoice" }); err != nil {
		return ArchiveResult{}, err
	}
	if inv.Header == nil {
		return ArchiveResult{}, fmt.Errorf("%s: missing `invoice` mapping", path)
	}
	status := inv.Header.Status
	if opts.AssumeBuilt {
		status, _ = status.Apply(invoice.Building)
	}
	if err := archivable(path, inv, status); err != nil {
		return ArchiveResult{}, err
	}
	unread, err := s.numberUnique(path, inv)
	if err != nil {
		return ArchiveResult{}, err
	}
	result, err := s.Archives.Add(path, inv, AddOptions{
		Replace: opts.Replace,
		DryRun:  opts.DryRun,
		Change: func(inv *invoice.Invoice) error {
			inv.Header.Status = invoice.Archived
			inv.Archive = nil
			return nil
		},
	})
	result.Unread = unread
	return result, err
}

// archivable returns why inv, the invoice at path, cannot be archived with
// status: a working copy is re-archived when editing or built, any other
// invoice archived when built.
func archivable(path string, inv invoice.Invoice, status invoice.Status) error {
	workingCopy := inv.Archive != nil && inv.Archive.ArchivePath.IsSet()
	switch {
	case workingCopy && !status.Allows(invoice.Rearchiving):
		return fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", path, status)
	case workingCopy:
		return nil
	case status == "":
		return fmt.Errorf("%s: invoice.status: missing value", path)
	case !status.Allows(invoice.Archiving):
		return fmt.Errorf("%s: invoice.status must be `built` before archiving, got `%s`", path, status)
	}
	return nil
}

// numberUnique returns a *invoice.DuplicateInvoiceNumberError when an
// archived invoice other than the one inv, the invoice at path, replaces
// has its number. The archived file a working copy from `archive edit` was
// opened from does not count as a duplicate.
func (s *Service) numberUnique(path string, inv invoice.Invoice) (Unread, error) {
	archived, unread, err := s.Archives.Duplicate(path, inv)
	if err != nil || archived == "" {
		return unread, err
	}
	return unread, &invoice.DuplicateInvoiceNumberError{InvoicePath: path, InvoiceNumber: inv.Header.Number.Trim(), ArchivedPath: archived}
}

// latestArchived returns the archived invoice of customerID issued last.
// New reports what the archive holds unread from its numbering walk.
func (s *Service) latestArchived(customerID string) (string, bool, error) {
	entries, _, err := s.Archives.Entries()
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
	leftDate, leftErr := time.Parse(time.DateOnly, strings.TrimSpace(e.IssueDate))
	rightDate, rightErr := time.Parse(time.DateOnly, strings.TrimSpace(other.IssueDate))
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
	// Unread is what the archive holds that invox no longer reads.
	Unread Unread
}

// ListArchive lists the archived invoices sorted by file name.
func (s *Service) ListArchive() (ArchiveList, error) {
	entries, unread, err := s.Archives.Entries()
	if err != nil {
		return ArchiveList{}, err
	}
	return ArchiveList{Dir: unread.Dir, Entries: entries, Unread: unread}, nil
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
	archived, err := s.Invoices.Load(checkout.Archived)
	if err := lenient(err); err != nil {
		return Edited{}, err
	}
	if archived.Header == nil {
		return Edited{}, fmt.Errorf("%s: missing `invoice` mapping", checkout.Archived)
	}
	_, err = s.Invoices.Create(checkout.Path, checkout.Archived, invoice.Invoice{
		Header:  &invoice.Header{Status: invoice.Editing},
		Archive: &checkout.Link,
	}, CreateOptions{Overwrite: opts.Overwrite, DryRun: opts.DryRun, Check: CheckNone})
	if err != nil {
		return Edited{}, err
	}
	return Edited{Path: checkout.Path, Archived: checkout.Archived}, nil
}
