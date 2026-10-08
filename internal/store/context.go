package store

import (
	"errors"

	"github.com/0xboris/invox/internal/invoice"
)

// LoadContext decodes the three files, validates them together and computes
// the invoice totals. It reports every problem at once: values that do not
// decode (*DecodeError, with file and line), then an unknown customer_id
// and the fields that are missing or out of range. Validation skips the
// fields that did not decode.
func LoadContext(customersPath, issuerPath, invoicePath string) (*invoice.Context, error) {
	customers, err := loadCustomerTable(customersPath)
	if err != nil {
		return nil, err
	}
	var issuer invoice.Issuer
	issuerErr := decodeYAMLFile(issuerPath, &issuer, true)
	if issuerErr != nil && !isDecodeError(issuerErr) {
		return nil, issuerErr
	}
	var invoiceFile invoice.Invoice
	invoiceErr := decodeYAMLFile(invoicePath, &invoiceFile, true)
	if invoiceErr != nil && !isDecodeError(invoiceErr) {
		return nil, invoiceErr
	}

	var unknownCustomer error
	customerID := invoiceFile.CustomerID.Trim()
	// customer stays nil when the invoice names no usable customer, so its
	// fields are not reported missing one by one.
	var customer *invoice.Customer
	var customerErr error
	if customerID != "" {
		if found, ok, err := customers.customer(customerID, true); !ok {
			unknownCustomer = &invoice.UnknownCustomerError{Path: invoicePath, CustomerID: customerID}
		} else {
			customer, customerErr = &found, err
		}
	}
	decodeErr := errors.Join(customerErr, issuerErr, invoiceErr)
	failed := failedFields(decodeErr)

	problems := invoice.Validate(invoice.Bundle{
		Invoice:     invoiceFile,
		InvoicePath: invoicePath,
		Customer:    customer,
		Issuer:      issuer,
		IssuerPath:  issuerPath,
		Undecoded:   func(field string) bool { return within(field, failed) },
	})
	var validationErr error
	if len(problems) > 0 {
		validationErr = &invoice.ValidationError{Problems: problems}
	}
	if err := errors.Join(unknownCustomer, decodeErr, validationErr); err != nil {
		return nil, err
	}

	return invoice.NewContext(invoice.Bundle{Invoice: invoiceFile, Customer: customer, Issuer: issuer})
}

func isDecodeError(err error) bool {
	var decodeErr *DecodeError
	return errors.As(err, &decodeErr)
}
