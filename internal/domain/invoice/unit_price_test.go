package invoice

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderInvoiceShowsSubCentUnitPrice(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	ctx.LineItems = ctx.LineItems[:1]
	ctx.LineItems[0].UnitPrice = big.NewRat(1, 8)
	ctx.LineItems[0].Quantity = big.NewRat(8, 1)
	ctx.LineItems[0].LineTotalCents = 100

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	if err := os.WriteFile(templatePath, []byte(strings.Join([]string{
		"@@LINE_ITEMS_ROWS@@",
		"@@LINE_ITEMS_BEGIN@@",
		"Block: @@LINE_ITEM_UNIT_PRICE@@ x @@LINE_ITEM_QUANTITY@@ = @@LINE_ITEM_LINE_TOTAL@@",
		"@@LINE_ITEMS_END@@",
	}, "\n")+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(templatePath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.tex")
	if err := h.RenderInvoice(templatePath, outputPath, ctx); err != nil {
		t.Fatalf("RenderInvoice returned error: %v", err)
	}
	rendered, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	for _, want := range []string{
		`& 0,125 \euro & 8 & 1,00 \euro\\`,
		`Block: 0,125 \euro x 8 = 1,00 \euro`,
	} {
		if !strings.Contains(string(rendered), want) {
			t.Fatalf("rendered output does not contain %q:\n%s", want, rendered)
		}
	}
}
