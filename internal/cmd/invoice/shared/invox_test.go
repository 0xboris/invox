package shared_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

// Markdown archived invoices are no longer read. A command that walks the
// archive names them once on stderr and otherwise behaves as if they were
// not there: same exit code, same stdout, same --json.
func TestMarkdownArchiveWarning(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	workDir := t.TempDir()
	x.Chdir(workDir)
	archiveDir := filepath.Join(workDir, "archive")
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	testfixture.WriteFile(t, filepath.Join(archiveDir, "2026-03-05.yaml"), "customer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n  issue_date: 2026-03-05\n")

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
			args: []string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults, "--dry-run"},
			want: result{stdout: "CUST-001-002.yaml\n", stderr: "Would create CUST-001-002.yaml for CUST-001 (CUST-001-002)\n"},
		},
	}
	check := func(warning string) {
		t.Helper()
		for _, run := range runs {
			exitCode, stdout, stderr := x.Run(run.args)
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
	testfixture.WriteFile(t, filepath.Join(archiveDir, "old.md"), frontMatter)
	check("warning: 1 Markdown invoice in archive is no longer read; convert it to .yaml to include it:\n" +
		"  " + filepath.Join("archive", "old.md") + "\n")

	testfixture.WriteFile(t, filepath.Join(archiveDir, "sub", "older.markdown"), frontMatter)
	// A Markdown file without front matter was never an invoice.
	testfixture.WriteFile(t, filepath.Join(archiveDir, "README.md"), "# Archive\n")
	check("warning: 2 Markdown invoices in archive are no longer read; convert them to .yaml to include them:\n" +
		"  " + filepath.Join("archive", "old.md") + "\n" +
		"  " + filepath.Join("archive", "sub", "older.markdown") + "\n")

	if _, err := os.Stat(filepath.Join(workDir, "CUST-001-002.yaml")); !os.IsNotExist(err) {
		t.Fatalf("new --dry-run wrote CUST-001-002.yaml, Stat err = %v", err)
	}
}

const skippedArchiveHint = "If they are obsolete, move them out of the archive or rename them to another extension. To continue their sequence, set numbering.start (or customers.CUST-001.numbering.start) to the next number.\n"

func TestNewWarnsAboutArchivedInvoicesThatDoNotMatchThePattern(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	testfixture.WriteNumberedInvoice(t, archiveDir, "a.yaml", "CUST-001-004", "archived")
	skippedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "old-format.yaml", "CUST-001/0009", "archived")
	x.Chdir(t.TempDir())

	exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "CUST-001-005.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-005.yaml\n")
	}
	want := "warning: numbering ignored 1 archived invoice(s) for CUST-001 that do not match numbering.pattern: " + skippedPath + "\n" +
		skippedArchiveHint +
		"Created CUST-001-005.yaml for CUST-001 (CUST-001-005)\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestIncrementWarnsAboutArchivedInvoicesThatDoNotMatchThePattern(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	testfixture.WriteNumberedInvoice(t, archiveDir, "a.yaml", "CUST-001-004", "archived")
	skippedPath := testfixture.WriteNumberedInvoice(t, archiveDir, "old-format.yaml", "CUST-001/0009", "archived")
	workDir := t.TempDir()
	x.Chdir(workDir)
	testfixture.WriteNumberedInvoice(t, workDir, "invoice.yaml", "CUST-001-001", "draft")

	exitCode, stdout, stderr := x.Run([]string{"increment", "-i", "invoice.yaml", "-c", draft.Customers})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "invoice.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "invoice.yaml\n")
	}
	want := "warning: numbering ignored 1 archived invoice(s) for CUST-001 that do not match numbering.pattern: " + skippedPath + "\n" +
		skippedArchiveHint +
		"Incremented invoice.yaml for CUST-001: CUST-001-001 -> CUST-001-005\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestNewListsAtMostFiveSkippedArchivedInvoices(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	var listed []string
	for _, name := range []string{"s1", "s2", "s3", "s4", "s5", "s6", "s7"} {
		path := testfixture.WriteNumberedInvoice(t, archiveDir, name+".yaml", "CUST-001/"+name, "archived")
		if len(listed) < 5 {
			listed = append(listed, path)
		}
	}
	x.Chdir(t.TempDir())

	exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "CUST-001-001.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-001.yaml\n")
	}
	want := "warning: numbering ignored 7 archived invoice(s) for CUST-001 that do not match numbering.pattern: " +
		strings.Join(listed, ", ") + " and 2 more\n" +
		skippedArchiveHint +
		"Created CUST-001-001.yaml for CUST-001 (CUST-001-001)\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestNewDoesNotWarnWhenNoArchivedInvoiceOfTheCustomerIsSkipped(t *testing.T) {
	for _, tc := range []struct {
		name  string
		files map[string]string
		want  string
	}{
		{
			name: "every invoice matches",
			files: map[string]string{
				"a.yaml": testfixture.NumberedInvoiceSource("CUST-001", "CUST-001-004", "2026-03-06"),
				"b.yaml": testfixture.NumberedInvoiceSource("CUST-001", "CUST-001-002", "2026-03-06"),
			},
			want: "CUST-001-005",
		},
		{
			name: "only unrelated files",
			files: map[string]string{
				"a.yaml":         testfixture.NumberedInvoiceSource("CUST-001", "CUST-001-004", "2026-03-06"),
				"other.yaml":     testfixture.NumberedInvoiceSource("CUST-002", "OTHER/17", "2026-03-06"),
				"notes.yaml":     "todo: call the accountant\n",
				"no-number.yaml": "customer_id: CUST-001\ninvoice:\n  issue_date: \"2026-03-06\"\n",
				"readme.md":      "# Archive\n",
				"a.pdf":          "%PDF-1.4\n",
				".DS_Store":      "junk",
			},
			want: "CUST-001-005",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			draft := testfixture.WriteDraft(t)
			archiveDir := t.TempDir()
			x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			for name, source := range tc.files {
				if err := os.WriteFile(filepath.Join(archiveDir, name), []byte(source), 0o644); err != nil {
					t.Fatalf("WriteFile(%s) returned error: %v", name, err)
				}
			}
			x.Chdir(t.TempDir())

			exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults})
			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
			}
			if want := tc.want + ".yaml\n"; stdout != want {
				t.Fatalf("stdout = %q, want %q", stdout, want)
			}
			if want := "Created " + tc.want + ".yaml for CUST-001 (" + tc.want + ")\n"; stderr != want {
				t.Fatalf("stderr = %q, want %q", stderr, want)
			}
		})
	}
}
