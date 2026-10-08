package latex

import "testing"

func TestValidateTemplateReportsUnsupportedPlaceholdersInTemplateOrder(t *testing.T) {
	template := "@@CUSTOMER_CITY_AND_POSTAL_CODE@@\n" +
		"VAT @@VAT_RATE@@\n" +
		"@@ISSUER_CITY_AND_POSTAL_CODE@@\n" +
		"@@VAT_AMOUNT@@ @@VAT_RATE@@\n"
	want := "@@CUSTOMER_CITY_AND_POSTAL_CODE@@: unsupported placeholder; use @@CUSTOMER_POSTAL_CODE@@ @@CUSTOMER_CITY@@\n" +
		"@@VAT_RATE@@: unsupported placeholder; use @@VAT_SUMMARY_ROWS@@\n" +
		"@@ISSUER_CITY_AND_POSTAL_CODE@@: unsupported placeholder; use @@ISSUER_POSTAL_CODE@@ @@ISSUER_CITY@@\n" +
		"@@VAT_AMOUNT@@: unsupported placeholder; use @@VAT_SUMMARY_ROWS@@"

	err := ValidateTemplate(template)
	if err == nil {
		t.Fatal("ValidateTemplate returned nil error for unsupported placeholders")
	}
	if err.Error() != want {
		t.Fatalf("ValidateTemplate error =\n%s\nwant\n%s", err.Error(), want)
	}
}
