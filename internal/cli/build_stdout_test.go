package cli

import (
	"strings"
	"testing"
)

func TestBuildKeepsTectonicOutputOffStdout(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicChatter)

	exitCode, stdout, stderr := captureRun(t, []string{
		"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	pdfPath := strings.TrimSuffix(invoicePath, ".yaml") + ".pdf"
	if want := pdfPath + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	want := "note: running TeX ...\nnote: writing `invoice.pdf`\nBuilt " + pdfPath + " for CUST-001 (CUST-001-001)\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}
