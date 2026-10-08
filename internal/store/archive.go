package store

import (
	"fmt"
	"path/filepath"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/0xboris/invox/internal/archive"
	"github.com/0xboris/invox/internal/invoice"
)

type ArchivedInvoiceSummary struct {
	Path       string
	Filename   string
	CustomerID string
	Number     string
	IssueDate  string
	Status     string
}

func (h Host) ListArchivedInvoices() ([]ArchivedInvoiceSummary, error) {
	records, err := h.collectArchivedInvoiceRecords()
	if err != nil {
		return nil, err
	}

	summaries := make([]ArchivedInvoiceSummary, 0, len(records))
	for _, record := range records {
		summaries = append(summaries, ArchivedInvoiceSummary{
			Path:       record.Path,
			Filename:   record.Filename,
			CustomerID: record.CustomerID,
			Number:     record.InvoiceNumber,
			IssueDate:  record.IssueDate,
			Status:     record.Status,
		})
	}
	return summaries, nil
}

func (h Host) latestArchivedInvoicePath(customerID string) (string, bool, error) {
	records, err := h.collectArchivedInvoiceRecords()
	if err != nil {
		return "", false, err
	}

	var latest archive.Entry
	found := false
	for _, record := range records {
		if record.CustomerID != customerID {
			continue
		}
		if !found || record.Newer(latest) {
			latest = record
			found = true
		}
	}
	if !found {
		return "", false, nil
	}
	return latest.Path, true, nil
}

// archiveStore is the configured archive. Its Dir is "" when no archive
// directory is configured.
func (h Host) archiveStore() (archive.Store, error) {
	archiveDir, err := h.ResolveArchiveDir()
	if err != nil {
		return archive.Store{}, err
	}
	return archive.Store{Dir: archiveDir}, nil
}

func (h Host) collectArchivedInvoiceRecords() ([]archive.Entry, error) {
	store, err := h.archiveStore()
	if err != nil {
		return nil, err
	}
	return store.List(readArchivedIdentity)
}

// archivedInvoiceIdentity reads what the archive lists of the invoice at
// path. ok is false for a file that is not an invoice: Markdown without
// front matter, or a document whose invoice fields cannot be read.
func archivedInvoiceIdentity(path string) (invoiceIdentity, bool, error) {
	document, ok, err := loadArchivedInvoiceDocument(path)
	if err != nil || !ok {
		return invoiceIdentity{}, false, err
	}
	root, err := documentRootMapping(document, path)
	if err != nil {
		return invoiceIdentity{}, false, nil //nolint:nilerr // not an invoice: the caller skips it
	}
	var identity invoiceIdentity
	if err := decodeYAMLNode(root, path, &identity, false); err != nil || identity.Invoice == nil {
		return invoiceIdentity{}, false, nil //nolint:nilerr // not an invoice: the caller skips it
	}
	return identity, true, nil
}

// readArchivedIdentity is the archive.Reader for invoices. An archived
// invoice without a status is listed as `archived`.
func readArchivedIdentity(path string) (archive.Identity, bool, error) {
	identity, ok, err := archivedInvoiceIdentity(path)
	if err != nil || !ok {
		return archive.Identity{}, ok, err
	}

	status := identity.Invoice.Status.Trim()
	if status == "" {
		status = string(invoice.Archived)
	}
	return archive.Identity{
		CustomerID:    identity.CustomerID.Trim(),
		IssueDate:     identity.Invoice.IssueDate.Trim(),
		Status:        status,
		InvoiceNumber: identity.Invoice.Number.Trim(),
	}, true, nil
}

func (h Host) resolveArchiveInputPath(name string) (archive.Target, error) {
	store, err := h.archiveStore()
	if err != nil {
		return archive.Target{}, err
	}
	if strings.TrimSpace(store.Dir) == "" {
		return archive.Target{}, fmt.Errorf("archive directory is unavailable")
	}
	return store.Find(name)
}

// CheckArchivedNumberUnique returns a *DuplicateInvoiceNumberError when the
// invoice at invoicePath uses a number that an archived invoice already has.
// The archived file the invoice was opened from (`archive edit`) does not
// count as a duplicate.
func (h Host) CheckArchivedNumberUnique(invoicePath string) error {
	document, err := loadYAMLDocument(invoicePath)
	if err != nil {
		return err
	}
	root, err := documentRootMapping(document, invoicePath)
	if err != nil {
		return err
	}
	invoiceNode := findMappingValue(root, "invoice")
	if invoiceNode == nil || invoiceNode.Kind != yaml.MappingNode {
		return nil
	}
	store, err := h.archiveStore()
	if err != nil {
		return err
	}
	if strings.TrimSpace(store.Dir) == "" {
		return nil
	}
	invoiceNumber := strings.TrimSpace(nodeText(findMappingValue(invoiceNode, "number")))
	return h.checkArchivedNumberUnique(invoicePath, invoiceNumber, store, root)
}

func (h Host) checkArchivedNumberUnique(invoicePath, invoiceNumber string, store archive.Store, root *yaml.Node) error {
	if invoiceNumber == "" {
		return nil
	}

	excluded := make(map[string]bool)
	excluded[filepath.Clean(invoicePath)] = true
	// Only a working copy from `archive edit` (archive_path set) may reuse the
	// number of the archived file it replaces, matching ArchiveInvoice.
	archiveTargetPath, archiveReplacePath := archiveMetadata(root)
	var editedPaths []string
	if strings.TrimSpace(archiveTargetPath) != "" {
		editedPaths = []string{archiveTargetPath, archiveReplacePath}
	}
	for _, relativePath := range editedPaths {
		if strings.TrimSpace(relativePath) == "" {
			continue
		}
		target, err := store.Resolve(relativePath)
		if err != nil {
			return err
		}
		excluded[target.Path] = true
	}

	records, err := store.List(readArchivedIdentity)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.InvoiceNumber != invoiceNumber {
			continue
		}
		recordPath := filepath.Clean(record.Path)
		if excluded[recordPath] {
			continue
		}
		return &invoice.DuplicateInvoiceNumberError{
			InvoicePath:   invoicePath,
			InvoiceNumber: invoiceNumber,
			ArchivedPath:  recordPath,
		}
	}
	return nil
}
