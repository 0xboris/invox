//go:build !windows

package email_test

import (
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestEmailWritesPublicDraft(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)
	x.ExpectOpener(nil)
	outputPath := filepath.Join(t.TempDir(), "draft.eml")

	exitCode, stdout, stderr := x.Run([]string{
		"email", fx.Invoice, "-o", outputPath, "-c", fx.Customers, "-u", fx.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	testfixture.AssertFileMode(t, outputPath, 0o644&^testfixture.ProcessUmask(t))
}
