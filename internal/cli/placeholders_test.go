package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// A placeholder invox does not know fails the command instead of reaching
// the output as written.
func TestUnknownPlaceholderFails(t *testing.T) {
	tests := []struct {
		name       string
		template   string
		config     string
		args       []string
		wantStderr string
	}{
		{
			name:       "render with a misspelt template placeholder",
			template:   "Customer @@CUSTOMR_NAME@@\n",
			args:       []string{"render", "invoice.yaml", "-o", "out.tex"},
			wantStderr: "error: invoice_template.tex: @@CUSTOMR_NAME@@: unknown placeholder\n",
		},
		{
			name:     "render with removed and unknown template placeholders",
			template: "@@VAT_AMOUNT@@ @@ISSUER_CITY_AND_POSTAL_CODE@@ @@VAT_AMOUNT@@\nVAT (@@VAT_RATE@@\\%): & @@VAT_AMOUNT@@\\\\\n",
			args:     []string{"render", "invoice.yaml", "-o", "out.tex"},
			wantStderr: "error: invoice_template.tex: @@VAT_AMOUNT@@: unknown placeholder\n" +
				"@@ISSUER_CITY_AND_POSTAL_CODE@@: unknown placeholder\n" +
				"@@VAT_RATE@@: unknown placeholder\n",
		},
		{
			name:       "email with a misspelt subject placeholder",
			args:       []string{"email", "invoice.yaml", "-o", "out.eml", "-n", "--subject", "Invoice {invoice_numbr}"},
			wantStderr: "error: email.subject: unknown placeholder {invoice_numbr}\n",
		},
		{
			name:   "email with unknown placeholders in the configured body and subject",
			config: "email:\n  subject: '{nope} {invoice_number}'\n  body: |\n    Dear {customr_name},\n    {customer_name} {customr_name} {due}\n",
			args:   []string{"email", "invoice.yaml", "-o", "out.eml", "-n"},
			wantStderr: "error: email.subject: unknown placeholder {nope}\n" +
				"email.body: unknown placeholder {customr_name}\n" +
				"email.body: unknown placeholder {due}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			if tt.template != "" {
				if err := os.WriteFile(templatePath, []byte(tt.template), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.config != "" {
				writeConfigFile(t, tt.config)
			}
			built := strings.Replace(readFileForTest(t, invoicePath), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
			if err := os.WriteFile(invoicePath, []byte(built), 0o644); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Dir(invoicePath)
			if err := os.WriteFile(filepath.Join(dir, "invoice.pdf"), []byte("%PDF-1.4\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			chdirForTest(t, dir)
			args := slices.Concat(tt.args, []string{"-c", customersPath, "-u", issuerPath})
			if tt.args[0] == "render" {
				args = append(args, "-t", templatePath)
			}

			exitCode, stdout, stderr := captureRun(t, args)
			if exitCode != 1 || stdout != "" || stderr != tt.wantStderr {
				t.Fatalf("exit = %d, stdout = %q, stderr = %q; want 1, \"\", %q", exitCode, stdout, stderr, tt.wantStderr)
			}
			for _, name := range []string{"out.tex", "out.eml"} {
				if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
					t.Fatalf("%s was written (stat error %v)", name, err)
				}
			}
		})
	}
}
