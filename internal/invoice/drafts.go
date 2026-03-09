package invoice

import (
	"bytes"
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

func CreateNewInvoice(defaultsPath, outputPath, customersPath, issuerPath, customerID string) (string, error) {
	if fileExists(outputPath) {
		return "", fmt.Errorf("%s already exists; choose a different -o/--output path", outputPath)
	}

	customer, err := LoadCustomer(customersPath, customerID)
	if err != nil {
		return "", err
	}
	issuerPayment, err := LoadIssuerPayment(issuerPath)
	if err != nil {
		return "", err
	}

	document, err := loadYAMLDocument(defaultsPath)
	if err != nil {
		return "", err
	}

	now := currentDate().In(time.Local)
	issueDate := now.Format("2006-01-02")
	invoiceNumber, _, err := NextInvoiceNumber(customerID, issueDate, customer, 0)
	if err != nil {
		return "", err
	}
	root, err := documentRootMapping(document, defaultsPath)
	if err != nil {
		return "", err
	}

	setMappingString(root, "customer_id", customerID)

	invoiceNode := getOrCreateMappingNode(root, "invoice")
	dueDays, err := issuerDueDays(issuerPath, issuerPayment)
	if err != nil {
		return "", err
	}

	setMappingString(invoiceNode, "number", invoiceNumber)
	setMappingString(invoiceNode, "issue_date", issueDate)
	setMappingString(invoiceNode, "due_date", now.AddDate(0, 0, dueDays).Format("2006-01-02"))
	setMappingString(invoiceNode, "paid_amount", "0")

	if vatRate := strings.TrimSuffix(strings.TrimSpace(asString(getPath(customer, "tax.default_vat_rate"))), "%"); vatRate != "" {
		setMappingString(invoiceNode, "vat_rate_percent", vatRate)
	}

	if findMappingValue(root, "line_items") == nil {
		setMappingSequence(root, "line_items", []*yaml.Node{})
	}
	if err := writeYAMLDocument(outputPath, document); err != nil {
		return "", err
	}

	return invoiceNumber, nil
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

	var document yaml.Node
	if err := yaml.Unmarshal(source, &document); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &document, nil
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
