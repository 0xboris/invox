package invoice

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderInvoiceDoesNotResubstitutePlaceholdersInValues(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	ctx.Customer["name"] = "Evil @@TOTAL@@ & @@IBAN@@ Ltd"
	ctx.Invoice["period"] = "@@SUBTOTAL@@"
	ctx.LineItems[0].Description = "Item @@TOTAL@@ 100%"

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.Join([]string{
		"Customer: @@CUSTOMER_NAME@@",
		"Period: @@PERIOD_LABEL@@",
		"Total: @@TOTAL@@",
		"IBAN: @@IBAN@@",
		"Subtotal: @@SUBTOTAL@@",
		"@@LINE_ITEMS_ROWS@@",
		"@@LINE_ITEMS_BEGIN@@",
		"Block: @@LINE_ITEM_DESCRIPTION@@ (@@CUSTOMER_NAME@@)",
		"@@LINE_ITEMS_END@@",
	}, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}

	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	var first string
	for i := 0; i < 100; i++ {
		if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
			t.Fatalf("RenderInvoice returned error: %v", err)
		}
		rendered, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("ReadFile(outputPath) returned error: %v", err)
		}
		if i == 0 {
			first = string(rendered)
			continue
		}
		if string(rendered) != first {
			t.Fatalf("render %d differs from first render:\nfirst:\n%s\ngot:\n%s", i, first, rendered)
		}
	}

	for _, want := range []string{
		`Customer: Evil @@TOTAL@@ \& @@IBAN@@ Ltd`,
		"Period: @@SUBTOTAL@@",
		`Item @@TOTAL@@ 100\%`,
		`Block: Item @@TOTAL@@ 100\% (Evil @@TOTAL@@ \& @@IBAN@@ Ltd)`,
	} {
		if !strings.Contains(first, want) {
			t.Fatalf("rendered output does not contain %q:\n%s", want, first)
		}
	}
	for _, unwanted := range []string{"Total: @@TOTAL@@", "IBAN: @@IBAN@@", "Subtotal: @@SUBTOTAL@@"} {
		if strings.Contains(first, unwanted) {
			t.Fatalf("rendered output still contains template placeholder %q:\n%s", unwanted, first)
		}
	}
}
