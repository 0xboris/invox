package cli

import (
	"os"
	"path/filepath"
	"testing"
)

// Markdown archived invoices are no longer read. A command that walks the
// archive names them once on stderr and otherwise behaves as if they were
// not there: same exit code, same stdout, same --json.
func TestMarkdownArchiveWarning(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	workDir := t.TempDir()
	chdirForTest(t, workDir)
	archiveDir := filepath.Join(workDir, "archive")
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeTestFile(t, filepath.Join(archiveDir, "2026-03-05.yaml"), "customer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n  issue_date: 2026-03-05\n")

	type result struct {
		exitCode       int
		stdout, stderr string
	}
	runs := []struct {
		args []string
		want result
	}{
		{
			args: []string{"archive", "list"},
			want: result{stdout: "2026-03-05.yaml\tCUST-001\t2026-03-05\tarchived\n"},
		},
		{
			args: []string{"archive", "list", "--json", "file,number"},
			want: result{stdout: `[{"file":"2026-03-05.yaml","number":"CUST-001-001"}]` + "\n"},
		},
		{
			args: []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "--defaults", defaultsPath, "--dry-run"},
			want: result{stdout: "CUST-001-002.yaml\n", stderr: "Would create CUST-001-002.yaml for CUST-001 (CUST-001-002)\n"},
		},
	}
	check := func(warning string) {
		t.Helper()
		for _, run := range runs {
			exitCode, stdout, stderr := captureRun(t, run.args)
			want := run.want
			want.stderr = warning + want.stderr
			if got := (result{exitCode, stdout, stderr}); got != want {
				t.Errorf("invox %q =\n%+v\nwant\n%+v", run.args, got, want)
			}
		}
	}
	check("")

	// The Markdown invoice with the higher number no longer counts: new
	// still drafts CUST-001-002.
	frontMatter := "---\ncustomer_id: CUST-001\ninvoice:\n  number: CUST-001-015\n  issue_date: 2026-03-06\n---\n# Invoice\n"
	writeTestFile(t, filepath.Join(archiveDir, "old.md"), frontMatter)
	check("warning: 1 Markdown invoice in archive is no longer read; convert it to .yaml to include it:\n" +
		"  " + filepath.Join("archive", "old.md") + "\n")

	writeTestFile(t, filepath.Join(archiveDir, "sub", "older.markdown"), frontMatter)
	// A Markdown file without front matter was never an invoice.
	writeTestFile(t, filepath.Join(archiveDir, "README.md"), "# Archive\n")
	check("warning: 2 Markdown invoices in archive are no longer read; convert them to .yaml to include them:\n" +
		"  " + filepath.Join("archive", "old.md") + "\n" +
		"  " + filepath.Join("archive", "sub", "older.markdown") + "\n")

	if _, err := os.Stat(filepath.Join(workDir, "CUST-001-002.yaml")); !os.IsNotExist(err) {
		t.Fatalf("new --dry-run wrote CUST-001-002.yaml, Stat err = %v", err)
	}
}
