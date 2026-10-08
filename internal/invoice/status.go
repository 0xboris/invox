package invoice

import (
	"fmt"
	"strings"
)

func SetInvoiceStatus(invoicePath, status string) error {
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("invoice status must not be empty")
	}
	return writeInvoiceStringField(invoicePath, "status", status)
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

// MarkInvoiceBuilt sets invoice.status to `built` after a successful PDF
// build. An archived invoice keeps `archived`: rebuilding its PDF does not
// take it out of the archive.
func MarkInvoiceBuilt(invoicePath string) error {
	var identity invoiceIdentity
	if err := decodeYAMLFile(invoicePath, &identity, false); err != nil && !isDecodeError(err) {
		return err
	}
	status := ""
	if identity.Invoice != nil {
		status = identity.Invoice.Status.Trim()
	}
	if StatusAfterBuild(status) == "archived" {
		return nil
	}
	return SetInvoiceStatus(invoicePath, "built")
}

// StatusAfterBuild is the invoice.status a successful build leaves: `built`,
// except that an archived invoice stays `archived`.
func StatusAfterBuild(status string) string {
	if status == "archived" {
		return status
	}
	return "built"
}
