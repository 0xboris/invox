package invoice

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLatexEscapeShieldsLeadingBracketAndStar(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "[Q1] 2026", want: "{}[Q1] 2026"},
		{input: "*Hauptstraße 1", want: "{}*Hauptstraße 1"},
		{input: "  [x]", want: "{}  [x]"},
		{input: "Q1 [2026]", want: "Q1 [2026]"},
		{input: "Main St. 1*", want: "Main St. 1*"},
		{input: "A & B_1 50%", want: `A \& B\_1 50\%`},
		{input: "", want: ""},
	}
	for _, tt := range tests {
		if got := latexEscape(tt.input); got != tt.want {
			t.Errorf("latexEscape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRenderInvoiceKeepsLeadingStarAndBracketAfterLineBreak(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	ctx.Customer["address"] = map[string]any{"street": "*Hinterhof", "city": "Vienna"}
	ctx.Invoice["period"] = "[Q1] 2026"

	templatePath := filepath.Join(t.TempDir(), "template.tex")
	template := "@@CUSTOMER_CITY@@\\\\\n@@CUSTOMER_STREET@@\\\\\nPeriod:\\\\ @@PERIOD_LABEL@@\n"
	if err := os.WriteFile(templatePath, []byte(template), 0o644); err != nil {
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
	want := "Vienna\\\\\n{}*Hinterhof\\\\\nPeriod:\\\\ {}[Q1] 2026\n"
	if string(rendered) != want {
		t.Fatalf("rendered = %q, want %q", rendered, want)
	}
}
