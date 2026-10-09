package store

import (
	"fmt"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/invoice"
	yaml "gopkg.in/yaml.v3"
)

// loadCustomerEntries reads customers.yaml: each customer ID with its
// entry, not yet decoded.
func loadCustomerEntries(path string) (map[string]*yaml.Node, error) {
	document, err := loadYAMLDocument(path)
	if err != nil {
		return nil, err
	}
	root, err := documentRootMapping(document, path)
	if err != nil {
		return nil, err
	}
	entries := map[string]*yaml.Node{}
	for _, pair := range mappingPairs(root) {
		entries[pair.key.Value] = pair.value
	}
	return entries, nil
}

// decodeCustomer decodes entry, the entry of id in the customers.yaml at
// path, rejecting keys a customer does not have.
func decodeCustomer(path, id string, entry *yaml.Node) (invoice.Customer, error) {
	entry = resolveYAMLAlias(entry)
	if entry.Kind != yaml.MappingNode {
		return invoice.Customer{}, &billing.DecodeError{File: path, Line: entry.Line, Problem: fmt.Sprintf("customer `%s` must be a mapping", id), Field: "customer"}
	}
	var customer invoice.Customer
	err := decodeYAMLNode(entry, path, &customer, true)
	return customer, err
}
