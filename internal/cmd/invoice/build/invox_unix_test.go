//go:build !windows

package build_test

import (
	"os"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestBuildKeepsPrivateInvoiceModeAndWritesPublicPDF(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	x.ExpectTectonic()
	if err := os.Chmod(fx.Invoice, 0o600); err != nil {
		t.Fatal(err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"build", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer, "-t", fx.Template,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	pdfPath := strings.TrimSuffix(fx.Invoice, ".yaml") + ".pdf"
	if want := "Built " + pdfPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != pdfPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, pdfPath+"\n")
	}
	updated, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "status: built") {
		t.Fatalf("invoice was not marked built:\n%s", updated)
	}
	testfixture.AssertFileMode(t, fx.Invoice, 0o600)
	testfixture.AssertFileMode(t, pdfPath, 0o644&^testfixture.ProcessUmask(t))
}
