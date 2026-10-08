package invoice

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/0xboris/invox/internal/archive"
	"github.com/0xboris/invox/internal/fsutil"
)

const (
	internalMetadataKey       = "_invox"
	internalArchivePathKey    = "archive_path"
	internalArchiveReplaceKey = "archive_replace_path"
)

func (h Host) GlobalInvoiceDefaultsPath() string {
	return filepath.Join(h.ConfigDir(), "invoice_defaults.yaml")
}

// LoadCustomer decodes the entry of customerID in customers.yaml.
func LoadCustomer(customersPath, customerID string) (Customer, error) {
	customers, err := loadCustomerTable(customersPath)
	if err != nil {
		return Customer{}, err
	}
	customer, ok, err := customers.customer(customerID, true)
	if !ok {
		return Customer{}, &UnknownCustomerError{Path: customersPath, CustomerID: customerID}
	}
	return customer, err
}

// LoadIssuerPayment decodes issuer.yaml and returns its payment details.
func LoadIssuerPayment(issuerPath string) (Payment, error) {
	var issuer IssuerFile
	if err := decodeYAMLFile(issuerPath, &issuer, true); err != nil {
		return Payment{}, err
	}
	if issuer.Payment == nil {
		return Payment{}, fmt.Errorf("%s: missing `payment` mapping", issuerPath)
	}
	return *issuer.Payment, nil
}

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
	if strings.TrimSpace(p.OutputPath) != "" && !p.Overwrite && fileExists(p.OutputPath) {
		return NewInvoice{}, &OutputExistsError{Path: p.OutputPath}
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

type IncrementedInvoice struct {
	CustomerID string
	OldNumber  string
	NewNumber  string
	// SkippedArchiveFiles are archived invoices of the customer whose numbers
	// do not match numbering.pattern, so they did not count towards NewNumber.
	SkippedArchiveFiles []string
}

// IncrementInvoiceNumber writes the next invoice number into the invoice at
// invoicePath. With dryRun it returns the same result and writes nothing.
func (h Host) IncrementInvoiceNumber(invoicePath, customersPath string, dryRun bool) (IncrementedInvoice, error) {
	customerID, issueDate, oldInvoiceNumber, err := readInvoiceIdentity(invoicePath)
	if err != nil {
		return IncrementedInvoice{}, err
	}

	customer, err := LoadCustomer(customersPath, customerID)
	if err != nil {
		return IncrementedInvoice{}, err
	}

	currentCounter, err := h.CounterFromInvoiceNumber(oldInvoiceNumber, customerID, issueDate, customer)
	if err != nil {
		return IncrementedInvoice{}, err
	}

	newInvoiceNumber, skipped, err := h.NextInvoiceNumber(customerID, issueDate, customer, currentCounter)
	if err != nil {
		return IncrementedInvoice{}, err
	}
	incremented := IncrementedInvoice{
		CustomerID:          customerID,
		OldNumber:           oldInvoiceNumber,
		NewNumber:           newInvoiceNumber,
		SkippedArchiveFiles: skipped,
	}
	if dryRun {
		return incremented, nil
	}
	if err := writeInvoiceNumber(invoicePath, newInvoiceNumber); err != nil {
		return IncrementedInvoice{}, err
	}
	return incremented, nil
}

func SetInvoiceStatus(invoicePath, status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("invoice status must not be empty")
	}
	return writeInvoiceStringField(invoicePath, "status", status)
}

// EditArchiveOptions controls EditArchivedInvoice.
type EditArchiveOptions struct {
	// Overwrite replaces an existing working copy, unless it is in the
	// archive directory.
	Overwrite bool
	// DryRun runs every check and returns the paths without writing.
	DryRun bool
}

// EditArchivedInvoice copies the archived invoice archiveName into workDir as
// a working copy and returns its path and the archived invoice's path.
func (h Host) EditArchivedInvoice(archiveName, workDir string, opts EditArchiveOptions) (string, string, error) {
	target, err := h.resolveArchiveInputPath(archiveName)
	if err != nil {
		return "", "", err
	}
	archivePath := target.Path

	document, ok, err := loadArchivedInvoiceDocument(archivePath)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", fmt.Errorf("%s: archived invoice could not be loaded", archivePath)
	}
	// The working copy keeps keys invox does not know, so opening an
	// archived invoice never fails over them; validate reports them.
	if err := decodeYAMLDocument(document, archivePath, &InvoiceFile{}, false); err != nil {
		return "", "", err
	}

	root, err := documentRootMapping(document, archivePath)
	if err != nil {
		return "", "", err
	}
	invoiceNode, err := invoiceMapping(root, archivePath)
	if err != nil {
		return "", "", err
	}

	edit := target.Edit()
	outputPath := filepath.Join(workDir, edit.Filename)
	if !opts.Overwrite && fileExists(outputPath) {
		return "", "", &OutputExistsError{Path: outputPath}
	}

	setMappingString(invoiceNode, "status", "editing")
	setArchiveMetadata(root, edit.Target, edit.Replace)

	data, err := encodeYAMLDocument(document)
	if err != nil {
		return "", "", err
	}
	if err := h.refuseArchivedOverwrite(outputPath, opts.Overwrite); err != nil {
		return "", "", err
	}
	if opts.DryRun {
		return outputPath, archivePath, nil
	}
	write := fsutil.WriteNewFile
	if opts.Overwrite {
		write = fsutil.WriteFile
	}
	if err := write(outputPath, data, fsutil.Public); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", "", &OutputExistsError{Path: outputPath}
		}
		return "", "", err
	}
	return outputPath, archivePath, nil
}

// ArchiveInvoice moves a built invoice, or a working copy from `archive edit`,
// into the archive. Re-archiving a working copy over archived files returns
// an *ArchiveReplaceError, after all other checks passed and before anything
// is written, unless opts.Replace is set. With opts.Replace the replaced
// files are first copied to the archive's history directory.
func (h Host) ArchiveInvoice(now time.Time, invoicePath string, opts ArchiveOptions) (ArchiveResult, error) {
	document, err := loadYAMLDocument(invoicePath)
	if err != nil {
		return ArchiveResult{}, err
	}

	root, err := documentRootMapping(document, invoicePath)
	if err != nil {
		return ArchiveResult{}, err
	}

	invoiceNode, err := invoiceMapping(root, invoicePath)
	if err != nil {
		return ArchiveResult{}, err
	}

	status := strings.TrimSpace(nodeText(findMappingValue(invoiceNode, "status")))
	if opts.AssumeBuilt {
		status = StatusAfterBuild(status)
	}
	store, err := h.archiveStore()
	if err != nil {
		return ArchiveResult{}, err
	}
	if strings.TrimSpace(store.Dir) == "" {
		return ArchiveResult{}, fmt.Errorf("archive directory is unavailable")
	}

	archivePath := filepath.Join(store.Dir, filepath.Base(invoicePath))
	sourcePath := filepath.Clean(invoicePath)

	archiveTargetPath, archiveReplacePath := archiveMetadata(root)
	editingArchive := strings.TrimSpace(archiveTargetPath) != ""
	if editingArchive {
		switch status {
		case "editing", "built":
		default:
			return ArchiveResult{}, fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", invoicePath, status)
		}
		target, err := store.Resolve(archiveTargetPath)
		if err != nil {
			return ArchiveResult{}, err
		}
		archivePath = target.Path
	} else {
		switch status {
		case "":
			return ArchiveResult{}, fmt.Errorf("%s: invoice.status: missing value", invoicePath)
		case "built":
		default:
			return ArchiveResult{}, fmt.Errorf("%s: invoice.status must be `built` before archiving, got `%s`", invoicePath, status)
		}
		if fileExists(archivePath) {
			return ArchiveResult{}, fmt.Errorf("%s already exists", archivePath)
		}
	}
	if sourcePath == archivePath {
		return ArchiveResult{}, fmt.Errorf("%s is already in the archive directory", invoicePath)
	}

	invoiceNumber := strings.TrimSpace(nodeText(findMappingValue(invoiceNode, "number")))
	if err := h.checkArchivedNumberUnique(invoicePath, invoiceNumber, store, root); err != nil {
		return ArchiveResult{}, err
	}

	var replacePath string
	if editingArchive && strings.TrimSpace(archiveReplacePath) != "" && archiveReplacePath != archiveTargetPath {
		target, err := store.Resolve(archiveReplacePath)
		if err != nil {
			return ArchiveResult{}, err
		}
		replacePath = target.Path
		if replacePath == archivePath {
			replacePath = ""
		}
	}
	var replaced []string
	if editingArchive {
		replaced, err = archive.ExistingFiles(archivePath, replacePath)
		if err != nil {
			return ArchiveResult{}, err
		}
	}
	if len(replaced) > 0 && !opts.Replace {
		return ArchiveResult{}, &ArchiveReplaceError{
			InvoicePath: invoicePath,
			Paths:       replaced,
			HistoryDir:  store.HistoryDir(),
		}
	}
	if opts.DryRun {
		result := ArchiveResult{Path: archivePath, HistoryDir: store.HistoryDir()}
		for _, path := range replaced {
			result.Replaced = append(result.Replaced, archive.Backup{Path: path})
		}
		return result, nil
	}
	if err := fsutil.MkdirAll(store.Dir, fsutil.Private); err != nil {
		return ArchiveResult{}, err
	}
	backups, err := store.Backup(replaced, now)
	if err != nil {
		return ArchiveResult{}, err
	}

	setMappingString(invoiceNode, "status", "archived")
	clearArchiveMetadata(root)
	data, err := encodeYAMLDocument(document)
	if err != nil {
		return ArchiveResult{}, err
	}
	if editingArchive {
		err = fsutil.WriteFile(archivePath, data, fsutil.Private)
	} else if err = fsutil.WriteNewFile(archivePath, data, fsutil.Private); errors.Is(err, fs.ErrExist) {
		return ArchiveResult{}, fmt.Errorf("%s already exists", archivePath)
	}
	if err != nil {
		return ArchiveResult{}, err
	}
	if replacePath != "" {
		if err := os.Remove(replacePath); err != nil && !os.IsNotExist(err) {
			return ArchiveResult{}, fmt.Errorf("remove %s: %w", replacePath, err)
		}
	}
	if err := os.Remove(sourcePath); err != nil {
		return ArchiveResult{}, fmt.Errorf("remove %s: %w", sourcePath, err)
	}
	return ArchiveResult{Path: archivePath, Replaced: backups, HistoryDir: store.HistoryDir()}, nil
}

// refuseArchivedOverwrite returns an *ArchivedOutputError when overwrite
// would replace an existing file inside the archive directory: an archived
// invoice is replaced only by re-archiving, which keeps a backup.
func (h Host) refuseArchivedOverwrite(outputPath string, overwrite bool) error {
	if !overwrite || !fileExists(outputPath) {
		return nil
	}
	store, err := h.archiveStore()
	if err != nil {
		return err
	}
	if strings.TrimSpace(store.Dir) == "" {
		return nil
	}
	archiveInfo, err := os.Stat(store.Dir)
	if err != nil {
		return nil //nolint:nilerr // no readable archive directory means no archived file to protect
	}
	if inDir(outputPath, archiveInfo) {
		return &ArchivedOutputError{Path: outputPath}
	}
	if resolved, err := filepath.EvalSymlinks(outputPath); err == nil && inDir(resolved, archiveInfo) {
		return &ArchivedOutputError{Path: outputPath}
	}
	return nil
}

// inDir reports whether one of path's parent directories is dir. It compares
// files rather than names, so a differently cased path on a case-insensitive
// file system, or a symlinked parent, still matches.
func inDir(path string, dir os.FileInfo) bool {
	for parent := filepath.Dir(filepath.Clean(path)); ; parent = filepath.Dir(parent) {
		if info, err := os.Stat(parent); err == nil && os.SameFile(info, dir) {
			return true
		}
		if filepath.Dir(parent) == parent {
			return false
		}
	}
}

// readInvoiceIdentity reads the customer, issue date and number that
// numbering needs from the invoice at invoicePath.
func readInvoiceIdentity(invoicePath string) (string, string, string, error) {
	var identity invoiceIdentity
	if err := decodeYAMLFile(invoicePath, &identity, false); err != nil {
		return "", "", "", err
	}

	customerID := identity.CustomerID.Trim()
	if customerID == "" {
		return "", "", "", fmt.Errorf("%s: missing `customer_id`", invoicePath)
	}
	if identity.Invoice == nil {
		return "", "", "", fmt.Errorf("%s: missing `invoice` mapping", invoicePath)
	}

	issueDate := identity.Invoice.IssueDate.Trim()
	if issueDate == "" {
		return "", "", "", fmt.Errorf("%s: invoice.issue_date: missing value", invoicePath)
	}
	if _, err := time.Parse("2006-01-02", issueDate); err != nil {
		return "", "", "", fmt.Errorf("%s: invoice.issue_date: expected YYYY-MM-DD, got `%s`", invoicePath, issueDate)
	}

	invoiceNumber := identity.Invoice.Number.Trim()
	if invoiceNumber == "" {
		return "", "", "", fmt.Errorf("%s: invoice.number: missing value", invoicePath)
	}

	return customerID, issueDate, invoiceNumber, nil
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

func loadArchivedInvoiceDocument(path string) (*yaml.Node, bool, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		document, err := loadYAMLDocument(path)
		if err != nil {
			return nil, true, err
		}
		return document, true, nil
	case ".md", ".markdown":
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, false, err
		}
		frontMatter, ok := markdownFrontMatter(source)
		if !ok {
			return nil, false, nil
		}
		document, err := parseYAMLDocumentSource(frontMatter, "front matter in "+path)
		if err != nil {
			return nil, true, err
		}
		return document, true, nil
	default:
		return nil, false, nil
	}
}

func writeInvoiceStringField(path, key, value string) error {
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

	setMappingString(invoiceNode, key, value)
	return writeYAMLDocument(path, document)
}

func archiveMetadata(root *yaml.Node) (string, string) {
	internalNode := findMappingValue(root, internalMetadataKey)
	if internalNode == nil || internalNode.Kind != yaml.MappingNode {
		return "", ""
	}
	return filepath.FromSlash(strings.TrimSpace(nodeText(findMappingValue(internalNode, internalArchivePathKey)))),
		filepath.FromSlash(strings.TrimSpace(nodeText(findMappingValue(internalNode, internalArchiveReplaceKey))))
}

// setArchiveMetadata records archive-relative paths with forward slashes so
// a working copy stays portable between operating systems.
func setArchiveMetadata(root *yaml.Node, archivePath, archiveReplacePath string) {
	internalNode := getOrCreateMappingNode(root, internalMetadataKey)
	setMappingString(internalNode, internalArchivePathKey, filepath.ToSlash(archivePath))
	if strings.TrimSpace(archiveReplacePath) == "" || archiveReplacePath == archivePath {
		deleteMappingKey(internalNode, internalArchiveReplaceKey)
	} else {
		setMappingString(internalNode, internalArchiveReplaceKey, filepath.ToSlash(archiveReplacePath))
	}
}

func clearArchiveMetadata(root *yaml.Node) {
	deleteMappingKey(root, internalMetadataKey)
}
