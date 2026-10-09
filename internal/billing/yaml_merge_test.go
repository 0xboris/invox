package billing_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestLoadContextAppliesVATFromMergeKey(t *testing.T) {
	fx := testfixture.WriteContext(t)
	replaceInFixture(t, fx.Invoice, "customer_id: CUST-001\n", "customer_id: CUST-001\nreduced: &reduced {vat_percent: 10}\n")
	replaceInFixture(t, fx.Invoice, "  - name: Support\n", "  - <<: *reduced\n    name: Support\n")

	ctx, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	// 200.00 at the invoice's 20% plus 10.00 at the merged 10%.
	if got := ctx.TotalCents; got != 25100 {
		t.Fatalf("total = %d cents, want 25100", got)
	}
}

func TestLoadContextRejectsDuplicateKeyInCustomers(t *testing.T) {
	fx := testfixture.WriteContext(t)
	replaceInFixture(t, fx.Customers, "  status: active\n", "  status: active\n  status: inactive\n")

	_, err := factorytest.LoadContext(t, fx.Customers, fx.Issuer, fx.Invoice)
	if err == nil {
		t.Fatal("LoadContext returned nil error, want duplicate key error")
	}
	if want := fx.Customers + `:4: duplicate key "status"`; !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}
