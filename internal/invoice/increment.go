package invoice

import (
	"bytes"
	"fmt"
	"time"

	"github.com/0xboris/invox/internal/fsutil"
	yaml "gopkg.in/yaml.v3"
)

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

func writeInvoiceNumber(path, invoiceNumber string) error {
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

	numberNode := findMappingValue(invoiceNode, "number")
	if numberNode == nil {
		appendMappingNode(invoiceNode, "number", scalarNode(invoiceNumber))
	} else {
		numberNode.Kind = yaml.ScalarNode
		numberNode.Tag = "!!str"
		numberNode.Value = invoiceNumber
	}

	clearYAMLMergeTags(document)
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	return fsutil.WriteFile(path, buffer.Bytes(), fsutil.Public)
}
