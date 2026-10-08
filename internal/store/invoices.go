package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/fsutil"
	"github.com/0xboris/invox/internal/invoice"
)

// Load decodes the invoice at path strictly.
func (s *Store) Load(path string) (invoice.Invoice, error) {
	var inv invoice.Invoice
	err := decodeYAMLFile(path, &inv, true)
	return inv, err
}

// LoadArchived decodes the archived invoice at path strictly: a YAML file,
// or the front matter of a Markdown file.
func (s *Store) LoadArchived(path string) (invoice.Invoice, error) {
	document, ok, err := loadArchivedInvoiceDocument(path)
	if err != nil {
		return invoice.Invoice{}, err
	}
	if !ok {
		return invoice.Invoice{}, fmt.Errorf("%s: archived invoice could not be loaded", path)
	}
	var inv invoice.Invoice
	err = decodeYAMLDocument(document, path, &inv, true)
	return inv, err
}

// Head reads the identity and the archive link of the invoice at path.
func (s *Store) Head(path string) (billing.Head, error) {
	document, err := loadYAMLDocument(path)
	if err != nil {
		return billing.Head{}, err
	}
	var identity invoiceIdentity
	err = decodeYAMLDocument(document, path, &identity, false)
	if err != nil && !isDecodeError(err) {
		return billing.Head{}, err
	}
	head := identity.head()
	head.ArchivePath, head.ReplacePath = archiveMetadata(document.Content[0])
	return head, err
}

func (identity invoiceIdentity) head() billing.Head {
	head := billing.Head{CustomerID: identity.CustomerID.Trim()}
	if identity.Invoice != nil {
		head.HasHeader = true
		head.Number = identity.Invoice.Number.Trim()
		head.IssueDate = identity.Invoice.IssueDate.Trim()
		head.Status = invoice.Status(identity.Invoice.Status.Trim())
	}
	return head
}

// maxDraftScanSize skips large YAML files in the draft scan; invoices are
// far smaller.
const maxDraftScanSize = 1 << 20

// Drafts returns the invoices directly inside dirs. The scan is best
// effort: the archive check still guarantees unique numbers, so an
// unreadable directory or file is skipped.
func (s *Store) Drafts(dirs []string) []billing.Head {
	seen := make(map[string]bool, len(dirs))
	var heads []billing.Head
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" || seen[dir] {
			continue
		}
		seen[dir] = true

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil || info.Size() > maxDraftScanSize {
				continue
			}
			switch strings.ToLower(filepath.Ext(entry.Name())) {
			case ".yaml", ".yml":
			default:
				continue
			}
			var identity invoiceIdentity
			if err := decodeYAMLFile(filepath.Join(dir, entry.Name()), &identity, false); err != nil || identity.Invoice == nil {
				continue
			}
			heads = append(heads, identity.head())
		}
	}
	return heads
}

// CheckOutput refuses a directory at path, and an existing file unless
// overwrite is set.
func (s *Store) CheckOutput(path string, overwrite bool) error {
	if err := refuseDirOutput(path); err != nil {
		return err
	}
	if !overwrite && fileExists(path) {
		return &billing.OutputExistsError{Path: path}
	}
	return nil
}

// Exists reports whether path is a file.
func (s *Store) Exists(path string) bool { return fileExists(path) }

// Stat returns why path cannot be read, or nil.
func (s *Store) Stat(path string) error {
	_, err := os.Stat(path)
	return err
}

// Create writes a new invoice to path from the document at from.
func (s *Store) Create(path, from string, inv invoice.Invoice, opts billing.CreateOptions) error {
	document, err := loadSourceDocument(from)
	if err != nil {
		return err
	}
	root, err := documentRootMapping(document, from)
	if err != nil {
		return err
	}
	if link := inv.Archive; link == (invoice.ArchiveLink{}) {
		deleteMappingKey(root, internalMetadataKey)
	} else {
		setArchiveMetadata(root, string(link.ArchivePath), string(link.ArchiveReplacePath))
	}
	if inv.CustomerID.IsSet() {
		setMappingString(root, "customer_id", string(inv.CustomerID))
	}
	if h := inv.Header; h != nil {
		invoiceNode := getOrCreateMappingNode(root, "invoice")
		for _, field := range headerFields(h) {
			if field.value != "" && field.key != "vat_percent" {
				setMappingString(invoiceNode, field.key, field.value)
			}
		}
		// A rate fills in only where the source has none.
		if rate := h.VATPercent.String(); rate != "" && strings.TrimSpace(nodeText(findMappingValue(invoiceNode, "vat_percent"))) == "" {
			setMappingString(invoiceNode, "vat_percent", rate)
		}
	}
	if inv.Positions != nil && findMappingValue(root, "positions") == nil {
		setMappingSequence(root, "positions", []*yaml.Node{})
	}
	if opts.Check != billing.CheckNone {
		if err := decodeYAMLDocument(document, from, &invoice.Invoice{}, opts.Check == billing.CheckStrict); err != nil {
			return err
		}
	}
	data, err := encodeYAMLDocument(document)
	if err != nil {
		return err
	}
	if opts.DryRun {
		return nil
	}
	write := fsutil.WriteNewFile
	if opts.Overwrite {
		write = fsutil.WriteFile
	}
	if err := write(path, data, fsutil.Public); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return &billing.OutputExistsError{Path: path}
		}
		return err
	}
	return nil
}

// loadSourceDocument reads the document a new invoice starts from: the
// front matter of a Markdown file, else YAML.
func loadSourceDocument(path string) (*yaml.Node, error) {
	if !isMarkdown(path) {
		return loadYAMLDocument(path)
	}
	document, ok, err := loadArchivedInvoiceDocument(path)
	if err == nil && !ok {
		err = fmt.Errorf("%s: archived invoice could not be loaded", path)
	}
	return document, err
}

func isMarkdown(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return true
	}
	return false
}

type headerField struct {
	key, value string
}

// headerFields lists the header's fields in the order a new invoice writes
// them, each as text, "" when unset.
func headerFields(h *invoice.Header) []headerField {
	return []headerField{
		{"number", string(h.Number)},
		{"issue_date", h.IssueDate.String()},
		{"due_date", h.DueDate.String()},
		{"status", string(h.Status)},
		{"paid_amount", h.PaidAmount.String()},
		{"vat_percent", h.VATPercent.String()},
		{"period", string(h.Period)},
	}
}

// Update rewrites the invoice at path with change applied. Only the fields
// change set to a different value are written, and an archive link it
// clears is removed; every other key, comment and anchor stays as it is.
func (s *Store) Update(path string, change func(*invoice.Invoice) error) error {
	document, err := loadYAMLDocument(path)
	if err != nil {
		return err
	}
	root, err := documentRootMapping(document, path)
	if err != nil {
		return err
	}
	invoiceNode, err := invoiceMapping(root, path)
	if err != nil {
		return err
	}
	// Values that do not decode read as unset; change leaves them alone.
	var before invoice.Invoice
	_ = decodeYAMLDocument(document, path, &before, false)
	if before.Header == nil {
		before.Header = &invoice.Header{}
	}
	after := before
	header := *before.Header
	after.Header = &header
	if err := change(&after); err != nil {
		return err
	}

	if after.CustomerID != before.CustomerID {
		setMappingString(root, "customer_id", string(after.CustomerID))
	}
	old := headerFields(before.Header)
	for index, field := range headerFields(after.Header) {
		if field.value != old[index].value {
			setMappingString(invoiceNode, field.key, field.value)
		}
	}
	if after.Archive != before.Archive {
		if after.Archive == (invoice.ArchiveLink{}) {
			deleteMappingKey(root, internalMetadataKey)
		} else {
			setArchiveMetadata(root, string(after.Archive.ArchivePath), string(after.Archive.ArchiveReplacePath))
		}
	}
	return writeYAMLDocument(path, document)
}

// ReadArchived reads what the archive lists of the invoice at path. ok is
// false for a file that is not an invoice: Markdown without front matter,
// or a document whose invoice fields cannot be read. An archived invoice
// without a status is listed as `archived`.
func ReadArchived(path string) (billing.ArchiveEntry, bool, error) {
	identity, ok, err := archivedInvoiceIdentity(path)
	if err != nil || !ok {
		return billing.ArchiveEntry{}, ok, err
	}
	status := identity.Invoice.Status.Trim()
	if status == "" {
		status = string(invoice.Archived)
	}
	return billing.ArchiveEntry{
		Path:       path,
		CustomerID: identity.CustomerID.Trim(),
		IssueDate:  identity.Invoice.IssueDate.Trim(),
		Status:     status,
		Number:     identity.Invoice.Number.Trim(),
	}, true, nil
}

// refuseDirOutput returns an OutputIsDirError when path is a directory or a
// symlink to one.
func refuseDirOutput(path string) error {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return &billing.OutputIsDirError{Path: path}
	}
	return nil
}

func isDecodeError(err error) bool {
	var decodeErr *billing.DecodeError
	return errors.As(err, &decodeErr)
}
