package store

import (
	"fmt"
	"sort"

	"github.com/0xboris/invox/internal/invoice"
	yaml "gopkg.in/yaml.v3"
)

type CustomerSummary struct {
	ID               string
	LegalCompanyName string
	Status           string
	// Email is where invoices go, as Customer.InvoiceEmail.
	Email    string
	Currency string
}

func ListCustomers(customersPath string) ([]CustomerSummary, error) {
	customers, err := loadCustomerTable(customersPath)
	if err != nil {
		return nil, err
	}

	customerIDs := make([]string, 0, len(customers.entries))
	for customerID := range customers.entries {
		customerIDs = append(customerIDs, customerID)
	}
	sort.Strings(customerIDs)

	summaries := make([]CustomerSummary, 0, len(customerIDs))
	for _, customerID := range customerIDs {
		// The list reads only names and statuses, so keys it does not know
		// are no reason to fail.
		customer, _, err := customers.customer(customerID, false)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, CustomerSummary{
			ID:               customerID,
			LegalCompanyName: customer.DisplayName(),
			Status:           customer.Status.Trim(),
			Email:            customer.InvoiceEmail(),
			Currency:         customer.BillingCurrency(),
		})
	}

	return summaries, nil
}

// customerTable is customers.yaml: each customer ID with its entry. An entry
// is decoded only when it is used, so a problem in one customer's entry
// never stops another customer's invoice.
type customerTable struct {
	path    string
	entries map[string]*yaml.Node
}

func loadCustomerTable(path string) (customerTable, error) {
	document, err := loadYAMLDocument(path)
	if err != nil {
		return customerTable{}, err
	}
	root, err := documentRootMapping(document, path)
	if err != nil {
		return customerTable{}, err
	}
	table := customerTable{path: path, entries: map[string]*yaml.Node{}}
	for _, pair := range mappingPairs(root) {
		table.entries[pair.key.Value] = pair.value
	}
	return table, nil
}

// customer decodes the entry of customerID. ok is false when customers.yaml
// has no such ID. A strict decode rejects keys a customer does not have.
func (t customerTable) customer(customerID string, strict bool) (customer invoice.Customer, ok bool, err error) {
	entry, ok := t.entries[customerID]
	if !ok {
		return invoice.Customer{}, false, nil
	}
	entry = resolveYAMLAlias(entry)
	if entry.Kind != yaml.MappingNode {
		return invoice.Customer{}, true, &DecodeError{File: t.path, Line: entry.Line, Problem: fmt.Sprintf("customer `%s` must be a mapping", customerID), Field: "customer"}
	}
	err = decodeYAMLNode(entry, t.path, &customer, strict)
	return customer, true, err
}

// LoadCustomer decodes the entry of customerID in customers.yaml.
func LoadCustomer(customersPath, customerID string) (invoice.Customer, error) {
	customers, err := loadCustomerTable(customersPath)
	if err != nil {
		return invoice.Customer{}, err
	}
	customer, ok, err := customers.customer(customerID, true)
	if !ok {
		return invoice.Customer{}, &invoice.UnknownCustomerError{Path: customersPath, CustomerID: customerID}
	}
	return customer, err
}

// LoadIssuerPayment decodes issuer.yaml and returns its payment details.
func LoadIssuerPayment(issuerPath string) (invoice.Payment, error) {
	var issuer invoice.Issuer
	if err := decodeYAMLFile(issuerPath, &issuer, true); err != nil {
		return invoice.Payment{}, err
	}
	if issuer.Payment == nil {
		return invoice.Payment{}, fmt.Errorf("%s: missing `payment` mapping", issuerPath)
	}
	return *issuer.Payment, nil
}
