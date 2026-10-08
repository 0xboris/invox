package store

import (
	"fmt"
	"sort"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/invoice"
	yaml "gopkg.in/yaml.v3"
)

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

// IDs returns the customer IDs, sorted.
func (t customerTable) IDs() []string {
	ids := make([]string, 0, len(t.entries))
	for id := range t.entries {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Lookup decodes the entry of customerID. ok is false when customers.yaml
// has no such ID. A strict decode rejects keys a customer does not have.
func (t customerTable) Lookup(customerID string, strict bool) (customer invoice.Customer, ok bool, err error) {
	entry, ok := t.entries[customerID]
	if !ok {
		return invoice.Customer{}, false, nil
	}
	entry = resolveYAMLAlias(entry)
	if entry.Kind != yaml.MappingNode {
		return invoice.Customer{}, true, &billing.DecodeError{File: t.path, Line: entry.Line, Problem: fmt.Sprintf("customer `%s` must be a mapping", customerID), Field: "customer"}
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
	customer, ok, err := customers.Lookup(customerID, true)
	if !ok {
		return invoice.Customer{}, &invoice.UnknownCustomerError{Path: customersPath, CustomerID: customerID}
	}
	return customer, err
}
