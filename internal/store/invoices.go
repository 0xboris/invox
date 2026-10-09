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

// ArchivedHead reads the head of the archived invoice at path, a YAML file
// or the front matter of a Markdown file, and decodes it strictly.
func (s *Store) ArchivedHead(path string) (billing.Head, error) {
	document, ok, err := loadArchivedInvoiceDocument(path)
	if err != nil {
		return billing.Head{}, err
	}
	if !ok {
		return billing.Head{}, fmt.Errorf("%s: archived invoice could not be loaded", path)
	}
	err = decodeYAMLDocument(document, path, &invoice.Invoice{}, true)
	if err != nil && !isDecodeError(err) {
		return billing.Head{}, err
	}
	// The lenient decode's errors are among the strict decode's.
	head, _ := documentHead(document, path)
	return head, err
}

// Head reads the identity and the archive link of the invoice at path.
func (s *Store) Head(path string) (billing.Head, error) {
	document, err := loadYAMLDocument(path)
	if err != nil {
		return billing.Head{}, err
	}
	return documentHead(document, path)
}

func documentHead(document *yaml.Node, path string) (billing.Head, error) {
	var identity invoiceIdentity
	err := decodeYAMLDocument(document, path, &identity, false)
	if err != nil && !isDecodeError(err) {
		return billing.Head{}, err
	}
	head := identity.head()
	root := document.Content[0]
	head.Header = headerShape(root)
	head.ArchivePath, head.ReplacePath = archiveMetadata(root)
	return head, err
}

// headerShape is what the `invoice` key of root holds, the node itself
// rather than what an alias points to, as invoiceMapping reads it.
func headerShape(root *yaml.Node) billing.HeaderShape {
	switch node := findMappingValue(root, "invoice"); {
	case node == nil:
		return billing.HeaderMissing
	case node.Kind == yaml.MappingNode:
		return billing.HeaderMapping
	}
	return billing.HeaderOther
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

// Drafts returns the invoices directly inside workDir and the directory of
// output. The scan is best effort: the archive check still guarantees
// unique numbers, so an unreadable directory or file is skipped.
func (s *Store) Drafts(workDir, output string) []billing.Head {
	dirs := []string{workDir}
	if output != "" {
		dirs = append(dirs, filepath.Dir(output))
	}
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

// Destination returns path, or <number>.yaml in workDir when path is "",
// after refusing a directory there, and an existing file unless overwrite
// is set.
func (s *Store) Destination(path, workDir, number string, overwrite bool) (string, error) {
	if path == "" {
		path = filepath.Join(workDir, number+".yaml")
	}
	if err := refuseDirOutput(path); err != nil {
		return "", err
	}
	if !overwrite && fileExists(path) {
		return "", &billing.OutputExistsError{Path: path}
	}
	return path, nil
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
	if !sameLink(after.Archive, before.Archive) {
		setArchiveLink(root, after.Archive)
	}
	return writeYAMLDocument(path, document)
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
	setArchiveMetadata(root, string(link.ArchivePath), string(link.ArchiveReplacePath))
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
