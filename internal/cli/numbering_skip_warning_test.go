package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const skippedArchiveHint = "If numbering.pattern changed, set numbering.start (or customers.CUST-001.numbering.start) to continue the sequence.\n"

func TestNewWarnsAboutArchivedInvoicesThatDoNotMatchThePattern(t *testing.T) {
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeNumberedInvoice(t, archiveDir, "a.yaml", "CUST-001-004", "archived")
	skippedPath := writeNumberedInvoice(t, archiveDir, "old-format.yaml", "CUST-001/0009", "archived")
	chdirForTest(t, t.TempDir())

	exitCode, stdout, stderr := captureRun(t, []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "-s", defaultsPath})
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
	customersPath, _, _ := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	writeNumberedInvoice(t, archiveDir, "a.yaml", "CUST-001-004", "archived")
	skippedPath := writeNumberedInvoice(t, archiveDir, "old-format.yaml", "CUST-001/0009", "archived")
	workDir := t.TempDir()
	chdirForTest(t, workDir)
	writeNumberedInvoice(t, workDir, "invoice.yaml", "CUST-001-001", "draft")

	exitCode, stdout, stderr := captureRun(t, []string{"increment", "-i", "invoice.yaml", "-c", customersPath})
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
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	archiveDir := t.TempDir()
	writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	var listed []string
	for _, name := range []string{"s1", "s2", "s3", "s4", "s5", "s6", "s7"} {
		path := writeNumberedInvoice(t, archiveDir, name+".yaml", "CUST-001/"+name, "archived")
		if len(listed) < 5 {
			listed = append(listed, path)
		}
	}
	chdirForTest(t, t.TempDir())

	exitCode, stdout, stderr := captureRun(t, []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "-s", defaultsPath})
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
				"a.yaml": numberedInvoiceSource("CUST-001", "CUST-001-004", "2026-03-06"),
				"b.yaml": numberedInvoiceSource("CUST-001", "CUST-001-002", "2026-03-06"),
			},
			want: "CUST-001-005",
		},
		{
			name: "only unrelated files",
			files: map[string]string{
				"a.yaml":         numberedInvoiceSource("CUST-001", "CUST-001-004", "2026-03-06"),
				"other.yaml":     numberedInvoiceSource("CUST-002", "OTHER/17", "2026-03-06"),
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
			customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
			archiveDir := t.TempDir()
			writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			for name, source := range tc.files {
				if err := os.WriteFile(filepath.Join(archiveDir, name), []byte(source), 0o644); err != nil {
					t.Fatalf("WriteFile(%s) returned error: %v", name, err)
				}
			}
			chdirForTest(t, t.TempDir())

			exitCode, stdout, stderr := captureRun(t, []string{"new", "CUST-001", "-c", customersPath, "-u", issuerPath, "-s", defaultsPath})
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

func numberedInvoiceSource(customerID, invoiceNumber, issueDate string) string {
	return "customer_id: " + customerID + "\ninvoice:\n  number: " + invoiceNumber + "\n  issue_date: \"" + issueDate + "\"\n"
}
