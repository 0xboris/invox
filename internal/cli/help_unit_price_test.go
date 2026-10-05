package cli

import (
	"strings"
	"testing"
)

func TestHelpTemplateDocumentsUnitPriceDecimals(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"help", "template"})
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
