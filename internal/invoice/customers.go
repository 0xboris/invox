package invoice

import (
	"fmt"
	"sort"

	yaml "gopkg.in/yaml.v3"
)

type CustomerSummary struct {
	ID               string
	LegalCompanyName string
	Status           string
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
func (t customerTable) customer(customerID string, strict bool) (customer Customer, ok bool, err error) {
	entry, ok := t.entries[customerID]
	if !ok {
		return Customer{}, false, nil
	}
	entry = resolveYAMLAlias(entry)
	if entry.Kind != yaml.MappingNode {
		return Customer{}, true, &DecodeError{File: t.path, Line: entry.Line, Problem: fmt.Sprintf("customer `%s` must be a mapping", customerID), Field: "customer"}
	}
	err = decodeYAMLNode(entry, t.path, &customer, strict)
	return customer, true, err
}
