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

// draft is the invoice identity names, with only its customer and the
// header fields numbering reads.
func (identity invoiceIdentity) draft() invoice.Invoice {
	draft := invoice.Invoice{CustomerID: identity.CustomerID, Header: &invoice.Header{}}
	if identity.Invoice != nil {
		draft.Header.Number = identity.Invoice.Number
		draft.Header.Status = identity.Invoice.Status
	}
	return draft
}

// maxDraftScanSize skips large YAML files in the draft scan; invoices are
// far smaller.
const maxDraftScanSize = 1 << 20

// Drafts returns the invoices directly inside workDir and the directory of
// output. The scan is best effort: the archive check still guarantees
// unique numbers, so an unreadable directory or file is skipped.
func (s *Store) Drafts(workDir, output string) []invoice.Invoice {
	dirs := []string{workDir}
	if output != "" {
		dirs = append(dirs, filepath.Dir(output))
	}
	seen := make(map[string]bool, len(dirs))
	var drafts []invoice.Invoice
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
			drafts = append(drafts, identity.draft())
		}
	}
	return drafts
}

// Create writes a new invoice to path, or to <number>.yaml in opts.Dir,
// from the document at from, and returns the file.
func (s *Store) Create(path, from string, inv invoice.Invoice, opts billing.CreateOptions) (string, error) {
	if path == "" {
		path = filepath.Join(opts.Dir, inv.Header.Number.Trim()+".yaml")
	}
	if err := refuseDirOutput(path); err != nil {
		return "", err
	}
	if !opts.Overwrite && fileExists(path) {
		return "", &billing.OutputExistsError{Path: path}
	}
	if opts.Overwrite && s.Protected != nil {
		protected, err := s.Protected(path)
		if err != nil {
			return "", err
		}
		if protected {
			return "", &billing.ArchivedOutputError{Path: path}
		}
	}
	document, err := loadYAMLDocument(from)
	if err != nil {
		return "", err
	}
	root, err := documentRootMapping(document, from)
	if err != nil {
		return "", err
	}
	// The header is written in place, so it must be a mapping, not an
	// alias to one.
	if node := findMappingValue(root, "invoice"); node != nil && node.Kind != yaml.MappingNode && node.Tag != "!!null" {
		return "", fmt.Errorf("%s: `invoice` must be a mapping", from)
	}
	setArchiveLink(root, inv.Archive)
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
			return "", err
		}
	}
	data, err := encodeYAMLDocument(document)
	if err != nil {
		return "", err
	}
	if opts.DryRun {
		return path, nil
	}
	write := fsutil.WriteNewFile
	if opts.Overwrite {
		write = fsutil.WriteFile
	}
	if err := write(path, data, fsutil.Public); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", &billing.OutputExistsError{Path: path}
		}
		return "", err
	}
	return path, nil
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
	data, err := s.Rewrite(path, change)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(path, data, fsutil.Public)
}

// Rewrite returns the invoice at path with change applied, as Update
// writes it.
func (s *Store) Rewrite(path string, change func(*invoice.Invoice) error) ([]byte, error) {
	document, err := loadYAMLDocument(path)
	if err != nil {
		return nil, err
	}
	root, err := documentRootMapping(document, path)
	if err != nil {
		return nil, err
	}
	invoiceNode, err := invoiceMapping(root, path)
	if err != nil {
		return nil, err
	}
	// Values that do not decode read as unset; change leaves them alone.
	var before invoice.Invoice
	_ = decodeYAMLDocument(document, path, &before, false)
	if before.Header == nil {
		before.Header = &invoice.Header{}
	}
	// A null `_invox` decodes as no link, but it is a key that clearing
	// the link removes.
	if before.Archive == nil && findMappingValue(root, internalMetadataKey) != nil {
		before.Archive = &invoice.ArchiveLink{}
	}
	after := before
	header := *before.Header
	after.Header = &header
	if before.Archive != nil {
		link := *before.Archive
		after.Archive = &link
	}
	if err := change(&after); err != nil {
		return nil, err
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
	if !sameLink(after.Archive, before.Archive) {
		setArchiveLink(root, after.Archive)
	}
	return encodeYAMLDocument(document)
}

func sameLink(a, b *invoice.ArchiveLink) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// setArchiveLink writes link as the `_invox` mapping of root, or removes the
// key when link is nil.
func setArchiveLink(root *yaml.Node, link *invoice.ArchiveLink) {
	if link == nil {
		deleteMappingKey(root, internalMetadataKey)
		return
	}
	setArchiveMetadata(root, string(link.ArchivePath))
}

// ReadArchived reads what the archive lists of the invoice at path. ok is
// false for a document whose invoice fields cannot be read. An archived
// invoice without a status is listed as `archived`.
func ReadArchived(path string) (billing.ArchiveEntry, bool, error) {
	identity, ok, err := archivedInvoiceIdentity(path)
	if err != nil || !ok {
		return billing.ArchiveEntry{}, ok, err
	}
	status := identity.Invoice.Status
	if status == "" {
		status = invoice.Archived
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
