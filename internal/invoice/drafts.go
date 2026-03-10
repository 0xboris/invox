package invoice

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
)

var currentDate = func() time.Time {
	return time.Now()
}

const (
	internalMetadataKey       = "_invox"
	internalArchivePathKey    = "archive_path"
	internalArchiveReplaceKey = "archive_replace_path"
)

func GlobalInvoiceDefaultsPath() string {
	return filepath.Join(ConfigDir(), "invoice_defaults.yaml")
}

func ResolveDefaultInvoiceDefaultsPath(start string) (string, error) {
	return resolveDefaultPath(start, "defaults", []string{"invoice_defaults.yaml"}, []string{"invoice_defaults.yaml"})
}

func LoadCustomer(customersPath, customerID string) (map[string]any, error) {
	customersValue, err := loadYAML(customersPath)
	if err != nil {
		return nil, err
	}

	customers, ok := customersValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: root value must be a mapping", customersPath)
	}

	rawCustomer, ok := customers[customerID]
	if !ok {
		return nil, fmt.Errorf("%s: unknown customer_id `%s`", customersPath, customerID)
	}

	customer, ok := rawCustomer.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: customer `%s` must be a mapping", customersPath, customerID)
	}
	return customer, nil
}

func LoadIssuerPayment(issuerPath string) (map[string]any, error) {
	issuerValue, err := loadYAML(issuerPath)
	if err != nil {
		return nil, err
	}

	issuer, ok := issuerValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: root value must be a mapping", issuerPath)
	}

	payment, ok := issuer["payment"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: missing `payment` mapping", issuerPath)
	}
	return payment, nil
}

func CreateNewInvoice(defaultsPath, outputPath, customersPath, issuerPath, customerID string, fromLast bool) (string, string, error) {
	if strings.TrimSpace(outputPath) != "" && fileExists(outputPath) {
		return "", "", fmt.Errorf("%s already exists; choose a different -o/--output path", outputPath)
	}

	customer, err := LoadCustomer(customersPath, customerID)
	if err != nil {
		return "", "", err
	}
	issuerPayment, err := LoadIssuerPayment(issuerPath)
	if err != nil {
		return "", "", err
	}

	document, sourceLabel, err := loadNewInvoiceDocument(defaultsPath, customerID, fromLast)
	if err != nil {
		return "", "", err
	}

	now := currentDate().In(time.Local)
	issueDate := now.Format("2006-01-02")
	invoiceNumber, _, err := NextInvoiceNumber(customerID, issueDate, customer, 0)
	if err != nil {
		return "", "", err
	}
	if strings.TrimSpace(outputPath) == "" {
		outputPath, err = filepath.Abs(invoiceNumber + ".yaml")
		if err != nil {
			return "", "", err
		}
	}
	if fileExists(outputPath) {
		return "", "", fmt.Errorf("%s already exists; choose a different -o/--output path", outputPath)
	}
	root, err := documentRootMapping(document, sourceLabel)
	if err != nil {
		return "", "", err
	}
	deleteMappingKey(root, internalMetadataKey)

	setMappingString(root, "customer_id", customerID)

	invoiceNode := getOrCreateMappingNode(root, "invoice")
	dueDays, err := issuerDueDays(issuerPath, issuerPayment)
	if err != nil {
		return "", "", err
	}

	setMappingString(invoiceNode, "number", invoiceNumber)
	setMappingString(invoiceNode, "issue_date", issueDate)
	setMappingString(invoiceNode, "due_date", now.AddDate(0, 0, dueDays).Format("2006-01-02"))
	setMappingString(invoiceNode, "status", "draft")
	setMappingString(invoiceNode, "paid_amount", "0")

	if strings.TrimSpace(asString(nodeScalarValue(findMappingValue(invoiceNode, "vat_percent")))) == "" {
		if vatRate := strings.TrimSuffix(strings.TrimSpace(asString(getPath(customer, "tax.default_vat_rate"))), "%"); vatRate != "" {
			setMappingString(invoiceNode, "vat_percent", vatRate)
		}
	}

	if findMappingValue(root, "positions") == nil {
		setMappingSequence(root, "positions", []*yaml.Node{})
	}
	if err := writeYAMLDocument(outputPath, document); err != nil {
		return "", "", err
	}

	return invoiceNumber, outputPath, nil
}

func loadNewInvoiceDocument(defaultsPath, customerID string, fromLast bool) (*yaml.Node, string, error) {
	if !fromLast {
		document, err := loadYAMLDocument(defaultsPath)
		if err != nil {
			return nil, "", err
		}
		if err := validateCanonicalInvoiceDocument(document, defaultsPath); err != nil {
			return nil, "", err
		}
		return document, defaultsPath, nil
	}

	archivePath, ok, err := latestArchivedInvoicePath(customerID)
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
	if err := validateCanonicalInvoiceDocument(document, archivePath); err != nil {
		return nil, "", err
	}
	return document, archivePath, nil
}

func IncrementInvoiceNumber(invoicePath, customersPath string) (string, string, string, error) {
	customerID, issueDate, oldInvoiceNumber, err := invoiceIdentity(invoicePath)
	if err != nil {
		return "", "", "", err
	}

	customer, err := LoadCustomer(customersPath, customerID)
	if err != nil {
		return "", "", "", err
	}

	currentCounter, err := CounterFromInvoiceNumber(oldInvoiceNumber, customerID, issueDate, customer)
	if err != nil {
		return "", "", "", err
	}

	newInvoiceNumber, _, err := NextInvoiceNumber(customerID, issueDate, customer, currentCounter)
	if err != nil {
		return "", "", "", err
	}
	if err := writeInvoiceNumber(invoicePath, newInvoiceNumber); err != nil {
		return "", "", "", err
	}
	return customerID, oldInvoiceNumber, newInvoiceNumber, nil
}

func SetInvoiceStatus(invoicePath, status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("invoice status must not be empty")
	}
	return writeInvoiceStringField(invoicePath, "status", status)
}

func EditArchivedInvoice(archiveName, workDir string) (string, string, error) {
	archivePath, relativeArchivePath, err := resolveArchiveInputPath(archiveName)
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
	if err := validateCanonicalInvoiceDocument(document, archivePath); err != nil {
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
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return "", "", err
	}
	if fileExists(outputPath) {
		return "", "", fmt.Errorf("%s already exists; choose a different working directory", outputPath)
	}

	setMappingString(invoiceNode, "status", "editing")
	setArchiveMetadata(root, archiveTargetPath, archiveReplacePath)

	if err := writeYAMLDocument(outputPath, document); err != nil {
		return "", "", err
	}
	return outputPath, archivePath, nil
}

func ArchiveInvoice(invoicePath string) (string, error) {
	document, err := loadYAMLDocument(invoicePath)
	if err != nil {
		return "", err
	}

	root, err := documentRootMapping(document, invoicePath)
	if err != nil {
		return "", err
	}

	invoiceNode := findMappingValue(root, "invoice")
	if invoiceNode == nil {
		return "", fmt.Errorf("%s: missing `invoice` mapping", invoicePath)
	}
	if invoiceNode.Kind != yaml.MappingNode {
		return "", fmt.Errorf("%s: `invoice` must be a mapping", invoicePath)
	}

	status := strings.TrimSpace(asString(nodeScalarValue(findMappingValue(invoiceNode, "status"))))
	archiveDir, err := ResolveArchiveDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(archiveDir) == "" {
		return "", fmt.Errorf("archive directory is unavailable")
	}

	archivePath := filepath.Join(archiveDir, filepath.Base(invoicePath))
	sourcePath, err := filepath.Abs(invoicePath)
	if err != nil {
		return "", err
	}
	archivePath, err = filepath.Abs(archivePath)
	if err != nil {
		return "", err
	}

	archiveTargetPath, archiveReplacePath := archiveMetadata(root)
	editingArchive := strings.TrimSpace(archiveTargetPath) != ""
	if editingArchive {
		switch status {
		case "editing", "built":
		default:
			return "", fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", invoicePath, status)
		}
		archivePath, err = resolveArchiveTargetPath(archiveDir, archiveTargetPath)
		if err != nil {
			return "", err
		}
	} else {
		switch status {
		case "":
			return "", fmt.Errorf("%s: invoice.status: missing value", invoicePath)
		case "built":
		default:
			return "", fmt.Errorf("%s: invoice.status must be `built` before archiving, got `%s`", invoicePath, status)
		}
		archivePath, err = filepath.Abs(filepath.Join(archiveDir, filepath.Base(invoicePath)))
		if err != nil {
			return "", err
		}
		if fileExists(archivePath) {
			return "", fmt.Errorf("%s already exists", archivePath)
		}
	}
	if sourcePath == archivePath {
		return "", fmt.Errorf("%s is already in the archive directory", invoicePath)
	}

	setMappingString(invoiceNode, "status", "archived")
	clearArchiveMetadata(root)
	if err := writeYAMLDocument(archivePath, document); err != nil {
		return "", err
	}
	if editingArchive && strings.TrimSpace(archiveReplacePath) != "" && archiveReplacePath != archiveTargetPath {
		replacePath, err := resolveArchiveTargetPath(archiveDir, archiveReplacePath)
		if err != nil {
			return "", err
		}
		if replacePath != archivePath {
			if err := os.Remove(replacePath); err != nil && !os.IsNotExist(err) {
				return "", fmt.Errorf("remove %s: %w", replacePath, err)
			}
		}
	}
	if err := os.Remove(sourcePath); err != nil {
		return "", fmt.Errorf("remove %s: %w", sourcePath, err)
	}
	return archivePath, nil
}

func invoiceIdentity(invoicePath string) (string, string, string, error) {
	invoiceValue, err := loadYAML(invoicePath)
	if err != nil {
		return "", "", "", err
	}

	root, ok := invoiceValue.(map[string]any)
	if !ok {
		return "", "", "", fmt.Errorf("%s: root value must be a mapping", invoicePath)
	}

	customerID := strings.TrimSpace(asString(root["customer_id"]))
	if customerID == "" {
		return "", "", "", fmt.Errorf("%s: missing `customer_id`", invoicePath)
	}

	invoiceNode, ok := root["invoice"].(map[string]any)
	if !ok {
		return "", "", "", fmt.Errorf("%s: missing `invoice` mapping", invoicePath)
	}

	issueDate := strings.TrimSpace(asString(invoiceNode["issue_date"]))
	if issueDate == "" {
		return "", "", "", fmt.Errorf("%s: invoice.issue_date: missing value", invoicePath)
	}
	if _, err := time.Parse("2006-01-02", issueDate); err != nil {
		return "", "", "", fmt.Errorf("%s: invoice.issue_date: expected YYYY-MM-DD, got `%s`", invoicePath, issueDate)
	}

	invoiceNumber := strings.TrimSpace(asString(invoiceNode["number"]))
	if invoiceNumber == "" {
		return "", "", "", fmt.Errorf("%s: invoice.number: missing value", invoicePath)
	}

	return customerID, issueDate, invoiceNumber, nil
}

func issuerDueDays(issuerPath string, payment map[string]any) (int, error) {
	raw := strings.TrimSpace(asString(payment["due_days"]))
	if raw == "" {
		return 0, fmt.Errorf("%s: payment.due_days: missing value", issuerPath)
	}
	days, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%s: payment.due_days: expected a non-negative integer, got `%s`", issuerPath, raw)
	}
	if days < 0 {
		return 0, fmt.Errorf("%s: payment.due_days: must be >= 0", issuerPath)
	}
	return days, nil
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
	return &document, nil
}

func validateCanonicalInvoiceDocument(document *yaml.Node, sourceLabel string) error {
	root, ok := normalizeYAMLNode(document).(map[string]any)
	if !ok {
		return nil
	}

	var validationErrors []string
	appendUnsupportedInvoiceKeyErrors(root, &validationErrors)
	if len(validationErrors) == 0 {
		return nil
	}

	for index, validationError := range validationErrors {
		validationErrors[index] = fmt.Sprintf("%s: %s", sourceLabel, validationError)
	}
	return errors.New(strings.Join(validationErrors, "\n"))
}

func writeYAMLDocument(path string, document *yaml.Node) error {
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	return writeFileAtomic(path, buffer.Bytes(), 0o644)
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

func nodeScalarValue(node *yaml.Node) any {
	if node == nil {
		return nil
	}
	return normalizeYAMLNode(node)
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
	return strings.TrimSpace(asString(nodeScalarValue(findMappingValue(internalNode, internalArchivePathKey)))),
		strings.TrimSpace(asString(nodeScalarValue(findMappingValue(internalNode, internalArchiveReplaceKey))))
}

func setArchiveMetadata(root *yaml.Node, archivePath, archiveReplacePath string) {
	internalNode := getOrCreateMappingNode(root, internalMetadataKey)
	setMappingString(internalNode, internalArchivePathKey, archivePath)
	if strings.TrimSpace(archiveReplacePath) == "" || archiveReplacePath == archivePath {
		deleteMappingKey(internalNode, internalArchiveReplaceKey)
	} else {
		setMappingString(internalNode, internalArchiveReplaceKey, archiveReplacePath)
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
