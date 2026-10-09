package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// A placeholder invox does not know fails the command instead of reaching
// the output as written.
func TestUnknownPlaceholderFails(t *testing.T) {
	tests := []struct {
		name       string
		template   string
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customersPath, issuerPath, _, templatePath := writeContextFixtures(t)
			if err := os.WriteFile(templatePath, []byte(tt.template), 0o644); err != nil {
				t.Fatal(err)
			}
			dir := filepath.Dir(customersPath)
			chdirForTest(t, dir)
			args := slices.Concat(tt.args, []string{"-c", customersPath, "-u", issuerPath, "-t", templatePath})

			exitCode, stdout, stderr := captureRun(t, args)
			if exitCode != 1 || stdout != "" || stderr != tt.wantStderr {
				t.Fatalf("exit = %d, stdout = %q, stderr = %q; want 1, \"\", %q", exitCode, stdout, stderr, tt.wantStderr)
			}
			if _, err := os.Stat(filepath.Join(dir, "out.tex")); !os.IsNotExist(err) {
				t.Fatalf("out.tex was written (stat error %v)", err)
			}
		})
	}
}
