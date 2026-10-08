package invoice

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

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

func (h Host) CreateNewInvoice(now time.Time, workDir, defaultsPath, outputPath, customersPath, issuerPath, customerID string, fromLast bool) (NewInvoice, error) {
	if strings.TrimSpace(outputPath) != "" && fileExists(outputPath) {
		return NewInvoice{}, &OutputExistsError{Path: outputPath}
	}

	customer, err := LoadCustomer(customersPath, customerID)
	if err != nil {
		return NewInvoice{}, err
	}
	issuerPayment, err := LoadIssuerPayment(issuerPath)
	if err != nil {
		return NewInvoice{}, err
	}

	document, sourceLabel, err := h.loadNewInvoiceDocument(defaultsPath, customerID, fromLast)
	if err != nil {
		return NewInvoice{}, err
	}

	now = now.In(time.Local)
	issueDate := now.Format("2006-01-02")
	draftDirs := draftSearchDirs(workDir, outputPath)
	draftCounter, err := h.highestDraftCounter(draftDirs, customerID, issueDate, customer)
	if err != nil {
		return NewInvoice{}, err
	}
	invoiceNumber, skipped, err := h.NextInvoiceNumber(customerID, issueDate, customer, draftCounter)
	if err != nil {
		return NewInvoice{}, err
	}
	if strings.TrimSpace(outputPath) == "" {
		outputPath = filepath.Join(workDir, invoiceNumber+".yaml")
	}
	if fileExists(outputPath) {
		return NewInvoice{}, &OutputExistsError{Path: outputPath}
	}
	root, err := documentRootMapping(document, sourceLabel)
	if err != nil {
		return NewInvoice{}, err
	}
	deleteMappingKey(root, internalMetadataKey)

	setMappingString(root, "customer_id", customerID)

	invoiceNode := getOrCreateMappingNode(root, "invoice")
	dueDays, err := issuerDueDays(issuerPath, issuerPayment)
	if err != nil {
		return NewInvoice{}, err
	}

	setMappingString(invoiceNode, "number", invoiceNumber)
	setMappingString(invoiceNode, "issue_date", issueDate)
	setMappingString(invoiceNode, "due_date", now.AddDate(0, 0, dueDays).Format("2006-01-02"))
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
	if err := decodeYAMLDocument(document, sourceLabel, &InvoiceFile{}, true); err != nil {
		return NewInvoice{}, err
	}
	data, err := encodeYAMLDocument(document)
	if err != nil {
		return NewInvoice{}, err
	}
	if err := fsutil.WriteNewFile(outputPath, data, fsutil.Public); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return NewInvoice{}, &OutputExistsError{Path: outputPath}
		}
		return NewInvoice{}, err
	}

	return NewInvoice{Number: invoiceNumber, Path: outputPath, SkippedArchiveFiles: skipped}, nil
}

// draftSearchDirs returns the directories whose drafts `new` takes into
// account: the current directory and the directory the new invoice is
// written to.
func draftSearchDirs(workDir, outputPath string) []string {
	dirs := []string{workDir}
	if strings.TrimSpace(outputPath) != "" {
		dirs = append(dirs, filepath.Dir(absPath(workDir, outputPath)))
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

func (h Host) IncrementInvoiceNumber(invoicePath, customersPath string) (IncrementedInvoice, error) {
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
	if err := writeInvoiceNumber(invoicePath, newInvoiceNumber); err != nil {
		return IncrementedInvoice{}, err
	}
	return IncrementedInvoice{
		CustomerID:          customerID,
		OldNumber:           oldInvoiceNumber,
		NewNumber:           newInvoiceNumber,
		SkippedArchiveFiles: skipped,
	}, nil
}

func SetInvoiceStatus(invoicePath, status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("invoice status must not be empty")
	}
	return writeInvoiceStringField(invoicePath, "status", status)
}

func (h Host) EditArchivedInvoice(archiveName, workDir string) (string, string, error) {
	archivePath, relativeArchivePath, err := h.resolveArchiveInputPath(archiveName)
	if err != nil {
		return "", "", err
	}

	document, ok, err := loadArchivedInvoiceDocument(archivePath)
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", fmt.Errorf("%s: archived invoice could not be loaded", archivePath)
	}
	if err := decodeYAMLDocument(document, archivePath, &InvoiceFile{}, true); err != nil {
		return "", "", err
	}

	root, err := documentRootMapping(document, archivePath)
	if err != nil {
		return "", "", err
	}
	invoiceNode := findMappingValue(root, "invoice")
	if invoiceNode == nil {
		return "", "", fmt.Errorf("%s: missing `invoice` mapping", archivePath)
	}
	if invoiceNode.Kind != yaml.MappingNode {
		return "", "", fmt.Errorf("%s: `invoice` must be a mapping", archivePath)
	}

	outputFilename, archiveTargetPath, archiveReplacePath := editableArchivePaths(relativeArchivePath)
	outputPath := filepath.Join(workDir, outputFilename)
	if fileExists(outputPath) {
		return "", "", fmt.Errorf("%s already exists; choose a different working directory", outputPath)
	}

	setMappingString(invoiceNode, "status", "editing")
	setArchiveMetadata(root, archiveTargetPath, archiveReplacePath)

	data, err := encodeYAMLDocument(document)
	if err != nil {
		return "", "", err
	}
	if err := fsutil.WriteNewFile(outputPath, data, fsutil.Public); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return "", "", fmt.Errorf("%s already exists; choose a different working directory", outputPath)
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

	invoiceNode := findMappingValue(root, "invoice")
	if invoiceNode == nil {
		return ArchiveResult{}, fmt.Errorf("%s: missing `invoice` mapping", invoicePath)
	}
	if invoiceNode.Kind != yaml.MappingNode {
		return ArchiveResult{}, fmt.Errorf("%s: `invoice` must be a mapping", invoicePath)
	}

	status := strings.TrimSpace(nodeText(findMappingValue(invoiceNode, "status")))
	archiveDir, err := h.ResolveArchiveDir()
	if err != nil {
		return ArchiveResult{}, err
	}
	if strings.TrimSpace(archiveDir) == "" {
		return ArchiveResult{}, fmt.Errorf("archive directory is unavailable")
	}

	archivePath := filepath.Join(archiveDir, filepath.Base(invoicePath))
	sourcePath := filepath.Clean(invoicePath)

	archiveTargetPath, archiveReplacePath := archiveMetadata(root)
	editingArchive := strings.TrimSpace(archiveTargetPath) != ""
	if editingArchive {
		switch status {
		case "editing", "built":
		default:
			return ArchiveResult{}, fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", invoicePath, status)
		}
		archivePath, err = resolveArchiveTargetPath(archiveDir, archiveTargetPath)
		if err != nil {
			return ArchiveResult{}, err
		}
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
	if err := h.checkArchivedNumberUnique(invoicePath, invoiceNumber, archiveDir, root); err != nil {
		return ArchiveResult{}, err
	}

	var replacePath string
	if editingArchive && strings.TrimSpace(archiveReplacePath) != "" && archiveReplacePath != archiveTargetPath {
		replacePath, err = resolveArchiveTargetPath(archiveDir, archiveReplacePath)
		if err != nil {
			return ArchiveResult{}, err
		}
		if replacePath == archivePath {
			replacePath = ""
		}
	}
	var replaced []string
	if editingArchive {
		replaced, err = existingArchivePaths(archivePath, replacePath)
		if err != nil {
			return ArchiveResult{}, err
		}
	}
	if len(replaced) > 0 && !opts.Replace {
		return ArchiveResult{}, &ArchiveReplaceError{
			InvoicePath: invoicePath,
			Paths:       replaced,
			HistoryDir:  filepath.Join(archiveDir, archiveHistoryDirName),
		}
	}
	if err := fsutil.MkdirAll(archiveDir, fsutil.Private); err != nil {
		return ArchiveResult{}, err
	}
	backups, err := backupArchivedFiles(archiveDir, replaced, now)
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
	return ArchiveResult{Path: archivePath, Replaced: backups}, nil
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

func loadYAMLDocument(path string) (*yaml.Node, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parseYAMLDocumentSource(source, path)
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

func parseYAMLDocumentSource(source []byte, label string) (*yaml.Node, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(source, &document); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if err := checkYAMLMappings(&document, label); err != nil {
		return nil, err
	}
	return &document, nil
}

// writeYAMLDocument replaces path with document, keeping the file's mode.
func writeYAMLDocument(path string, document *yaml.Node) error {
	data, err := encodeYAMLDocument(document)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(path, data, fsutil.Public)
}

func encodeYAMLDocument(document *yaml.Node) ([]byte, error) {
	clearYAMLMergeTags(document)
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
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

	invoiceNode := findMappingValue(root, "invoice")
	if invoiceNode == nil {
		return fmt.Errorf("%s: missing `invoice` mapping", path)
	}
	if invoiceNode.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: `invoice` must be a mapping", path)
	}

	setMappingString(invoiceNode, key, value)
	return writeYAMLDocument(path, document)
}

// nodeText is the text of a scalar node, "" for a missing node, a mapping
// or a list.
func nodeText(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	node = resolveYAMLAlias(node)
	if node.Kind != yaml.ScalarNode {
		return ""
	}
	return scalarText(node)
}

func documentRootMapping(document *yaml.Node, label string) (*yaml.Node, error) {
	if document == nil || len(document.Content) == 0 {
		return nil, fmt.Errorf("%s: root value must be a mapping", label)
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: root value must be a mapping", label)
	}
	return root, nil
}

func getOrCreateMappingNode(parent *yaml.Node, key string) *yaml.Node {
	if existing := findMappingValue(parent, key); existing != nil {
		if existing.Kind == yaml.MappingNode {
			return existing
		}
		existing.Kind = yaml.MappingNode
		existing.Tag = "!!map"
		existing.Value = ""
		existing.Content = nil
		return existing
	}

	child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	appendMappingNode(parent, key, child)
	return child
}

func setMappingString(parent *yaml.Node, key, value string) {
	if existing := findMappingValue(parent, key); existing != nil {
		existing.Kind = yaml.ScalarNode
		existing.Tag = "!!str"
		existing.Value = value
		existing.Content = nil
		return
	}
	appendMappingNode(parent, key, scalarNode(value))
}

func setMappingSequence(parent *yaml.Node, key string, content []*yaml.Node) {
	if existing := findMappingValue(parent, key); existing != nil {
		existing.Kind = yaml.SequenceNode
		existing.Tag = "!!seq"
		existing.Value = ""
		existing.Content = content
		return
	}
	appendMappingNode(parent, key, &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: content,
	})
}

func renameMappingKey(parent *yaml.Node, oldKey, newKey string) {
	if parent == nil || parent.Kind != yaml.MappingNode || oldKey == newKey {
		return
	}

	oldIndex := -1
	newIndex := -1
	for index := 0; index+1 < len(parent.Content); index += 2 {
		switch parent.Content[index].Value {
		case oldKey:
			oldIndex = index
		case newKey:
			newIndex = index
		}
	}
	if oldIndex == -1 {
		return
	}
	if newIndex == -1 {
		parent.Content[oldIndex].Value = newKey
		return
	}

	oldValue := parent.Content[oldIndex+1]
	newValue := parent.Content[newIndex+1]
	if nodeIsEmpty(newValue) && !nodeIsEmpty(oldValue) {
		parent.Content[newIndex+1] = oldValue
	}

	parent.Content = append(parent.Content[:oldIndex], parent.Content[oldIndex+2:]...)
}

func deleteMappingKey(parent *yaml.Node, key string) {
	if parent == nil || parent.Kind != yaml.MappingNode {
		return
	}
	for index := 0; index+1 < len(parent.Content); index += 2 {
		if parent.Content[index].Value == key {
			parent.Content = append(parent.Content[:index], parent.Content[index+2:]...)
			return
		}
	}
}

func editableArchivePaths(relativeArchivePath string) (string, string, string) {
	relativeArchivePath = filepath.Clean(relativeArchivePath)
	ext := strings.ToLower(filepath.Ext(relativeArchivePath))
	switch ext {
	case ".md", ".markdown":
		yamlRelativePath := replaceFileExtension(relativeArchivePath, ".yaml")
		return filepath.Base(yamlRelativePath), yamlRelativePath, relativeArchivePath
	default:
		return filepath.Base(relativeArchivePath), relativeArchivePath, ""
	}
}

func replaceFileExtension(path, ext string) string {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(ext) == "" {
		return path
	}
	currentExt := filepath.Ext(path)
	if currentExt == "" {
		return path + ext
	}
	return strings.TrimSuffix(path, currentExt) + ext
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

func nodeIsEmpty(node *yaml.Node) bool {
	if node == nil {
		return true
	}

	switch node.Kind {
	case yaml.MappingNode, yaml.SequenceNode:
		return len(node.Content) == 0
	default:
		return strings.TrimSpace(node.Value) == ""
	}
}
