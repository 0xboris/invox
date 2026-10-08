package billing

import (
	"errors"

	"github.com/0xboris/invox/internal/invoice"
)

// ValidateResult is a valid invoice.
type ValidateResult struct {
	Context *invoice.Context
	// Duplicate is a *invoice.DuplicateInvoiceNumberError when an archived
	// invoice in another file has the invoice's number, or the error that
	// kept the archive from being checked. It never makes the invoice
	// invalid.
	Duplicate error
}

// Validate loads the invoice at path with its customer and issuer and
// checks them together.
func (s *Service) Validate(path string) (ValidateResult, error) {
	ctx, err := s.load(path)
	if err != nil {
		return ValidateResult{}, err
	}
	return ValidateResult{Context: ctx, Duplicate: s.CheckNumberUnique(path)}, nil
}

// load locates customers.yaml and issuer.yaml and loads the invoice at path
// with them.
func (s *Service) load(path string) (*invoice.Context, error) {
	customersPath, issuerPath, err := s.locateParties()
	if err != nil {
		return nil, err
	}
	return s.loadContext(customersPath, issuerPath, path)
}

func (s *Service) locateParties() (string, string, error) {
	customersPath, err := s.Directory.Locate(CustomersFile)
	if err != nil {
		return "", "", err
	}
	issuerPath, err := s.Directory.Locate(IssuerFile)
	if err != nil {
		return "", "", err
	}
	return customersPath, issuerPath, nil
}

// loadContext decodes the three files, validates them together and computes
// the invoice totals. It reports every problem at once: values that do not
// decode (*DecodeError, with file and line), then an unknown customer_id
// and the fields that are missing or out of range. Validation skips the
// fields that did not decode.
func (s *Service) loadContext(customersPath, issuerPath, invoicePath string) (*invoice.Context, error) {
	customers, err := s.Directory.Customers()
	if err != nil {
		return nil, err
	}
	issuer, issuerErr := s.Directory.Issuer()
	if issuerErr != nil && !isDecodeError(issuerErr) {
		return nil, issuerErr
	}
	inv, invoiceErr := s.Invoices.Load(invoicePath)
	if invoiceErr != nil && !isDecodeError(invoiceErr) {
		return nil, invoiceErr
	}

	var unknownCustomer error
	customerID := inv.CustomerID.Trim()
	// customer stays nil when the invoice names no usable customer, so its
	// fields are not reported missing one by one.
	var customer *invoice.Customer
	var customerErr error
	if customerID != "" {
		if found, ok, err := customers.Lookup(customerID, true); !ok {
			unknownCustomer = &invoice.UnknownCustomerError{Path: invoicePath, CustomerID: customerID}
		} else {
			customer, customerErr = &found, err
		}
	}
	decodeErr := errors.Join(customerErr, issuerErr, invoiceErr)
	failed := failedFields(decodeErr)

	problems := invoice.Validate(invoice.Bundle{
		Invoice:     inv,
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
	return invoice.NewContext(invoice.Bundle{Invoice: inv, Customer: customer, Issuer: issuer})
}
