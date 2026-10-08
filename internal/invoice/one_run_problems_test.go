package invoice

import (
	"strings"
	"testing"
)

// One LoadContext reports values that do not decode together with missing
// fields, and does not also report a field that did not decode as missing.
func TestLoadContextReportsDecodeAndValidationProblemsTogether(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, customersPath, "  name: Appsters GmbH\n", "  name: [Appsters]\n")
	replaceInFixture(t, customersPath, "    vat_tax_id: ATU12345678\n", "    other: x\n")
	replaceInFixture(t, issuerPath, "  website: https://example.com\n", "")
	replaceInFixture(t, issuerPath, "  address:\n    street: Ring 1\n    postal_code: 1010\n    city: Vienna\n    country: Austria\n", "  address: Ring 1\n")
	replaceInFixture(t, invoicePath, "  period: Leistungszeitraum\n", "")
	replaceInFixture(t, invoicePath, "  vat_percent: 20\n", "  vat_percent: twenty\n")
	replaceInFixture(t, invoicePath, "  - name: Support\n    description: QA\n    unit_price: 10\n    quantity: 1\n", "  - just text\n  - name: Support\n    unit_price: 10\n    quantity: 0\n")

	_, err := LoadContext(customersPath, issuerPath, invoicePath)
	want := strings.Join([]string{
		customersPath + ":2: customer.name: expected a string, got a list",
		customersPath + `:13: unknown key "other" in customer.tax`,
		issuerPath + ":6: issuer.company.address must be a mapping, got a string",
		invoicePath + ":6: invoice.vat_percent: expected a number or percent string, got `twenty`",
		invoicePath + ":13: positions[2] must be a mapping, got a string",
		"customer.tax.vat_tax_id: missing value",
		"issuer.company.website: missing value",
		"invoice.period: missing value",
		"positions[3].description: missing value",
		"positions[3].quantity: must be > 0",
	}, "\n")
	if err == nil || err.Error() != want {
		t.Fatalf("error =\n%v\nwant\n%s", err, want)
	}
}

// Only a key at the top of a file may hold anchored definitions; deeper, an
// anchor does not make an unknown key acceptable.
func TestUnknownKeyWithAnchorIsRejectedBelowTheTopLevel(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, invoicePath, "  period: Leistungszeitraum\n", "  period: Leistungszeitraum\n  notes: &n hello\n")

	_, err := LoadContext(customersPath, issuerPath, invoicePath)
	if want := invoicePath + `:7: unknown key "notes" in invoice`; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}
