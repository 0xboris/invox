package list_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestArchiveListHelpShowsOutputFormat(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"archive", "list", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"invox archive list",
		"FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestArchiveListPrintsArchivedInvoices(t *testing.T) {
	x := clitest.New(t)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	yamlArchivePath := filepath.Join(archiveDir, "2026-03-06.yaml")
	if err := os.WriteFile(yamlArchivePath, []byte(strings.TrimSpace(`
customer_id: CUST-YAML
invoice:
  number: CUST-YAML-001
  issue_date: 2026-03-06
  status: archived
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(yamlArchivePath) returned error: %v", err)
	}

	// No status lists as archived.
	unstatedArchivePath := filepath.Join(archiveDir, "2026-03-05.yaml")
	if err := os.WriteFile(unstatedArchivePath, []byte("customer_id: CUST-NS\ninvoice:\n  number: CUST-NS-001\n  issue_date: 2026-03-05\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(unstatedArchivePath) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{"archive", "list"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}

	want := strings.Join([]string{
		"2026-03-05.yaml\tCUST-NS\t2026-03-05\tarchived",
		"2026-03-06.yaml\tCUST-YAML\t2026-03-06\tarchived",
	}, "\n") + "\n"
	if stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}
