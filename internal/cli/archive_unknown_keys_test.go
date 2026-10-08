package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// An archived invoice with a key invox does not know opens with `archive
// edit`; validate then reports the key with its line and what to do, and
// passes once the key is removed.
func TestArchiveEditThenValidateReportsUnknownKey(t *testing.T) {
	customersPath, issuerPath, fixturePath, _ := writeContextFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := filepath.Join(archiveDir, "first.yaml")
	writeBuildableInvoice(t, fixturePath, archivedPath, "archived")
	source := readFileForTest(t, archivedPath)
	if err := os.WriteFile(archivedPath, []byte("notes: edited\n"+source), 0o644); err != nil {
		t.Fatal(err)
	}

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	exitCode, _, stderr := captureRun(t, []string{"archive", "edit", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("archive edit exit code = %d, want 0, stderr=%q", exitCode, stderr)
	}

	workingCopy := filepath.Join(workDir, "first.yaml")
	edited := readFileForTest(t, workingCopy)
	line := strings.Count(edited[:strings.Index(edited, "notes: edited")], "\n") + 1
	validate := []string{"validate", "-i", "first.yaml", "-c", customersPath, "-u", issuerPath}

	exitCode, stdout, stderr := captureRun(t, validate)
	want := fmt.Sprintf("error: first.yaml:%d: unknown key \"notes\"\n"+
		"Remove the unknown keys or fix their spelling; 'invox help defaults' lists the supported fields.\n", line)
	if exitCode != 1 || stdout != "" || stderr != want {
		t.Fatalf("validate = exit %d, stdout %q, stderr %q; want exit 1, no stdout, stderr %q", exitCode, stdout, stderr, want)
	}

	if err := os.WriteFile(workingCopy, []byte(strings.Replace(edited, "notes: edited\n", "", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	exitCode, _, stderr = captureRun(t, validate)
	if exitCode != 0 || !strings.HasPrefix(stderr, "Validation OK: CUST-001-001 for CUST-001") {
		t.Fatalf("validate without the key = exit %d, stderr %q; want exit 0 and Validation OK", exitCode, stderr)
	}
}
