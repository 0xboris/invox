package invoice

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/fsutil"
	yaml "gopkg.in/yaml.v3"
)

type NewInvoice struct {
	Number string
	Path   string
	// SkippedArchiveFiles are archived invoices of the customer whose numbers
	// do not match numbering.pattern, so they did not count towards Number.
	SkippedArchiveFiles []string
}

// NewInvoiceParams says which invoice CreateNewInvoice drafts and where.
type NewInvoiceParams struct {
	// Now dates the invoice.
	Now time.Time
	// WorkDir is searched for unarchived drafts when numbering.
	WorkDir string
	// DefaultsPath is invoice_defaults.yaml.
	DefaultsPath  string
	OutputPath    string
	CustomersPath string
	IssuerPath    string
	CustomerID    string
	// FromLast starts from the customer's last archived invoice instead of
	// the defaults.
	FromLast bool
	// Overwrite replaces an existing file at OutputPath, unless it is in
	// the archive directory.
	Overwrite bool
	// DryRun runs every check and returns the result without writing.
	DryRun bool
}

func (h Host) CreateNewInvoice(p NewInvoiceParams) (NewInvoice, error) {
	if strings.TrimSpace(p.OutputPath) != "" {
		if err := refuseDirOutput(p.OutputPath); err != nil {
			return NewInvoice{}, err
		}
		if !p.Overwrite && fileExists(p.OutputPath) {
			return NewInvoice{}, &OutputExistsError{Path: p.OutputPath}
		}
	}

	customer, err := LoadCustomer(p.CustomersPath, p.CustomerID)
	if err != nil {
		return NewInvoice{}, err
	}
	issuerPayment, err := LoadIssuerPayment(p.IssuerPath)
	if err != nil {
		return NewInvoice{}, err
	}

	document, sourceLabel, err := h.loadNewInvoiceDocument(p.DefaultsPath, p.CustomerID, p.FromLast)
	if err != nil {
		return NewInvoice{}, err
	}

	p.Now = p.Now.In(time.Local)
	issueDate := p.Now.Format("2006-01-02")
	draftDirs := draftSearchDirs(p.WorkDir, p.OutputPath)
	draftCounter, err := h.highestDraftCounter(draftDirs, p.CustomerID, issueDate, customer)
	if err != nil {
		return NewInvoice{}, err
	}
	invoiceNumber, skipped, err := h.NextInvoiceNumber(p.CustomerID, issueDate, customer, draftCounter)
	if err != nil {
		return NewInvoice{}, err
	}
	if strings.TrimSpace(p.OutputPath) == "" {
		p.OutputPath = filepath.Join(p.WorkDir, invoiceNumber+".yaml")
	}
	if err := refuseDirOutput(p.OutputPath); err != nil {
		return NewInvoice{}, err
	}
	if !p.Overwrite && fileExists(p.OutputPath) {
		return NewInvoice{}, &OutputExistsError{Path: p.OutputPath}
	}
	root, err := documentRootMapping(document, sourceLabel)
	if err != nil {
		return NewInvoice{}, err
	}
	deleteMappingKey(root, internalMetadataKey)

	setMappingString(root, "customer_id", p.CustomerID)

	invoiceNode := getOrCreateMappingNode(root, "invoice")
	dueDays, err := issuerDueDays(p.IssuerPath, issuerPayment)
	if err != nil {
		return NewInvoice{}, err
	}

	setMappingString(invoiceNode, "number", invoiceNumber)
	setMappingString(invoiceNode, "issue_date", issueDate)
	setMappingString(invoiceNode, "due_date", p.Now.AddDate(0, 0, dueDays).Format("2006-01-02"))
	setMappingString(invoiceNode, "status", "draft")
	setMappingString(invoiceNode, "paid_amount", "0")

	if strings.TrimSpace(nodeText(findMappingValue(invoiceNode, "vat_percent"))) == "" {
		if vatRate := strings.TrimSuffix(customer.Tax.DefaultVATRate.text, "%"); vatRate != "" {
			setMappingString(invoiceNode, "vat_percent", vatRate)
		}
	}

	if findMappingValue(root, "positions") == nil {
		setMappingSequence(root, "positions", []*yaml.Node{})
	}
	// An archived invoice is a record: keys it has that invox does not
	// know are copied over as they are, for validate to report.
	if err := decodeYAMLDocument(document, sourceLabel, &InvoiceFile{}, !p.FromLast); err != nil {
		return NewInvoice{}, err
	}
	data, err := encodeYAMLDocument(document)
	if err != nil {
		return NewInvoice{}, err
	}
	if err := h.refuseArchivedOverwrite(p.OutputPath, p.Overwrite); err != nil {
		return NewInvoice{}, err
	}
	created := NewInvoice{Number: invoiceNumber, Path: p.OutputPath, SkippedArchiveFiles: skipped}
	if p.DryRun {
		return created, nil
	}
	write := fsutil.WriteNewFile
	if p.Overwrite {
		write = fsutil.WriteFile
	}
	if err := write(p.OutputPath, data, fsutil.Public); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return NewInvoice{}, &OutputExistsError{Path: p.OutputPath}
		}
		return NewInvoice{}, err
	}
	return created, nil
}

// draftSearchDirs returns the directories whose drafts `new` takes into
// account: the current directory and the directory the new invoice is
// written to.
func draftSearchDirs(workDir, outputPath string) []string {
	dirs := []string{workDir}
	if strings.TrimSpace(outputPath) != "" {
		dirs = append(dirs, filepath.Dir(AbsPath(workDir, outputPath)))
	}
	return dirs
}

func (h Host) loadNewInvoiceDocument(defaultsPath, customerID string, fromLast bool) (*yaml.Node, string, error) {
	if !fromLast {
		document, err := loadYAMLDocument(defaultsPath)
		if err != nil {
			return nil, "", err
		}
		return document, defaultsPath, nil
	}

	archivePath, ok, err := h.latestArchivedInvoicePath(customerID)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", fmt.Errorf("no archived invoice found for customer_id `%s`", customerID)
	}

	document, ok, err := loadArchivedInvoiceDocument(archivePath)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", fmt.Errorf("%s: archived invoice could not be loaded", archivePath)
	}
	return document, archivePath, nil
}

func issuerDueDays(issuerPath string, payment Payment) (int, error) {
	if !payment.DueDays.isSet() {
		return 0, fmt.Errorf("%s: payment.due_days: missing value", issuerPath)
	}
	if payment.DueDays.Int() < 0 {
		return 0, fmt.Errorf("%s: payment.due_days: must be >= 0", issuerPath)
	}
	return int(payment.DueDays.Int()), nil
}
