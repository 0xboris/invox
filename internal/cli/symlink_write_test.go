package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIncrementWritesThroughSymlinkedInvoice(t *testing.T) {
	customersPath, _, _ := writeDraftFixtures(t)
	targetPath := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(targetPath, []byte("customer_id: CUST-001\ninvoice:\n  number: CUST-001-009\n  issue_date: 2026-03-06\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(t.TempDir(), "linked.yaml")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(t.TempDir())+"\n")

	exitCode, stdout, stderr := captureRun(t, []string{"increment", "-i", linkPath, "-c", customersPath})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Incremented " + linkPath + " for CUST-001: CUST-001-009 -> CUST-001-010\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != linkPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, linkPath+"\n")
	}

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("%s was replaced by a regular file", linkPath)
	}
	if got, err := os.Readlink(linkPath); err != nil || got != targetPath {
		t.Fatalf("Readlink(%s) = %q, %v, want %q", linkPath, got, err, targetPath)
	}
	updated, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "number: CUST-001-010") {
		t.Fatalf("link target was not updated:\n%s", updated)
	}
}
