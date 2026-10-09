package template_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestTemplateHelpShowsTemplateSubcommands(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"template", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Description:",
		"Author and discover the LaTeX templates used by render and build.",
		"invox template <subcommand> [flags]",
		"  list  List available invoice templates\n",
		"Important rules:",
		"Placeholder names are case-sensitive and must match exactly.",
		"Structured placeholders:",
		"Template workflow:",
		"Available .tex placeholders:",
		"@@LINE_ITEMS_BEGIN@@ ... @@LINE_ITEMS_END@@ repeats a custom snippet once per position.",
		"Custom line-item block example:",
		"@@LINE_ITEM_NAME@@ & @@LINE_ITEM_UNIT_PRICE@@ & @@LINE_ITEM_VAT_RATE@@ & @@LINE_ITEM_LINE_TOTAL@@\\\\",
		"Use @@VAT_LABEL@@ anywhere you want the same VAT label text in the template.",
		"render -i invoice.yaml -t multi_vat.tex",
		"invox template list --names",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
	for _, placeholder := range []string{
		"@@ISSUER_NAME@@",
		"@@ISSUER_COMPANY_REG_NO@@",
		"@@ISSUER_VAT_TAX_ID@@",
		"@@ISSUER_WEBSITE@@",
		"@@ISSUER_EMAIL@@",
		"@@ISSUER_STREET@@",
		"@@ISSUER_CITY@@",
		"@@ISSUER_POSTAL_CODE@@",
		"@@ISSUER_COUNTRY@@",
		"@@CUSTOMER_NAME@@",
		"@@CUSTOMER_STREET@@",
		"@@CUSTOMER_CITY@@",
		"@@CUSTOMER_POSTAL_CODE@@",
		"@@CUSTOMER_COUNTRY@@",
		"@@CUSTOMER_VAT_TAX_ID@@",
		"@@CUSTOMER_EMAIL@@",
		"@@INVOICE_NUMBER@@",
		"@@ISSUE_DATE@@",
		"@@DUE_DATE@@",
		"@@PERIOD_LABEL@@",
		"@@LINE_ITEMS_ROWS@@",
		"@@LINE_ITEMS_ROWS_WITH_VAT@@",
		"@@LINE_ITEMS_BEGIN@@",
		"@@LINE_ITEMS_END@@",
		"@@LINE_ITEM_NAME@@",
		"@@LINE_ITEM_DESCRIPTION@@",
		"@@LINE_ITEM_UNIT_PRICE@@",
		"@@LINE_ITEM_QUANTITY@@",
		"@@LINE_ITEM_VAT_RATE@@",
		"@@LINE_ITEM_LINE_TOTAL@@",
		"@@LINE_ITEM_RULE@@",
		"@@SUBTOTAL@@",
		"@@VAT_SUMMARY_ROWS@@",
		"@@TOTAL@@",
		"@@PAID_AMOUNT@@",
		"@@OUTSTANDING_AMOUNT@@",
		"@@INVOICE_TOTAL@@",
		"@@OUTSTANDING_TOTAL@@",
		"@@PAYMENT_TERMS_TEXT@@",
		"@@VAT_LABEL@@",
		"@@BANK_NAME@@",
		"@@IBAN@@",
		"@@BIC@@",
		"@@EPC_QR_AVAILABLE@@",
		"@@EPC_QR_LABEL@@",
		"@@EPC_QR_CODE@@",
	} {
		if !strings.Contains(stdout, placeholder) {
			t.Fatalf("stdout %q does not contain placeholder %q", stdout, placeholder)
		}
	}
}
