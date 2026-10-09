package cli_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestHelpTemplateDocumentsUnitPriceDecimals(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"help", "template"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	want := "@@LINE_ITEM_UNIT_PRICE@@         Formatted unit price with 2 to 4 decimals, as many as it needs (rounded half up beyond 4)"
	if !strings.Contains(stdout, want) {
		t.Fatalf("help template does not contain %q:\n%s", want, stdout)
	}
}
