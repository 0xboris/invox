package billing_test

import (
	"strings"
	"testing"
)

func TestLoadContextAppliesVATFromMergeKey(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, invoicePath, "customer_id: CUST-001\n", "customer_id: CUST-001\nreduced: &reduced {vat_percent: 10}\n")
	replaceInFixture(t, invoicePath, "  - name: Support\n", "  - <<: *reduced\n    name: Support\n")

	ctx, err := loadContext(t, customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	// 200.00 at the invoice's 20% plus 10.00 at the merged 10%.
	if got := ctx.TotalCents; got != 25100 {
		t.Fatalf("total = %d cents, want 25100", got)
	}
}

func TestLoadContextRejectsDuplicateKeyInCustomers(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, customersPath, "  status: active\n", "  status: active\n  status: inactive\n")

	_, err := loadContext(t, customersPath, issuerPath, invoicePath)
	if err == nil {
		t.Fatal("LoadContext returned nil error, want duplicate key error")
	}
	if want := customersPath + `:4: duplicate key "status"`; !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}
