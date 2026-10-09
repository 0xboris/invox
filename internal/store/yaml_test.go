package store

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/invoice"
)

func TestLoadYAMLPreservesNumericLookingMappingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "customers.yaml")
	source := "0021:\n  name: Appsters GmbH\n  nested:\n    0007: yes\nissued_on: 2026-03-06\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	customers, err := loadCustomerEntries(path)
	if err != nil {
		t.Fatalf("loadCustomerEntries returned error: %v", err)
	}
	if _, exists := customers["17"]; exists {
		t.Fatalf("customers unexpectedly contains coerced key %q", "17")
	}
	entry, exists := customers["0021"]
	if !exists {
		t.Fatalf("customers does not contain key %q", "0021")
	}
	customer, err := decodeCustomer(path, "0021", entry)
	if want := path + `:3: unknown key "nested" in customer`; err == nil || err.Error() != want {
		t.Fatalf("customer(0021) error = %v, want %s", err, want)
	}
	if got := customer.DisplayName(); got != "Appsters GmbH" {
		t.Fatalf("customer name = %q, want %q", got, "Appsters GmbH")
	}

	root := decodeForTest[struct {
		Customer struct {
			Name   invoice.Text `yaml:"name"`
			Nested struct {
				Key invoice.Text `yaml:"0007"`
			} `yaml:"nested"`
		} `yaml:"0021"`
		IssuedOn invoice.Text `yaml:"issued_on"`
	}](t, source)
	if got := root.Customer.Nested.Key; got != "yes" {
		t.Fatalf("nested[0007] = %q, want %q", got, "yes")
	}
	if got := root.IssuedOn; got != "2026-03-06" {
		t.Fatalf("issued_on = %q, want %q", got, "2026-03-06")
	}
}
