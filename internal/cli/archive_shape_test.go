package cli

import (
	"os"
	"path/filepath"
	"testing"
)

const archiveShapeInvoice = `customer_id: CUST-003
invoice:
  number: CUST-003-001
  issue_date: 2026-03-07
  due_date: 2026-04-06
  status: built
  period: March
  vat_percent: 20
positions:
  - name: Consulting
    description: Workshop
    unit_price: 500
    quantity: 1
`

const archiveShapePositions = `positions:
  - name: Consulting
    description: Workshop
    unit_price: 500
    quantity: 1
`

const archiveShapeAlias = `base: &base
  number: CUST-003-001
  issue_date: 2026-03-07
  due_date: 2026-04-06
  status: built
  period: March
  vat_percent: 20
invoice: *base
`

// An `invoice` key that holds no mapping is refused by archive add and
// archive edit with its own message, apart from a missing key.
func TestArchiveRefusesInvoiceKeyThatIsNotAMapping(t *testing.T) {
	tests := []struct {
		name    string
		command string
		invoice string
		want    string
	}{
		{name: "add null", command: "add", invoice: "invoice:\n", want: "`invoice` must be a mapping"},
		{name: "add tilde", command: "add", invoice: "invoice: ~\n", want: "`invoice` must be a mapping"},
		{name: "add scalar", command: "add", invoice: "invoice: 5\n", want: "`invoice` must be a mapping"},
		{name: "add alias", command: "add", invoice: archiveShapeAlias, want: "`invoice` must be a mapping"},
		{name: "add missing", command: "add", invoice: "", want: "missing `invoice` mapping"},
		{name: "edit null", command: "edit", invoice: "invoice:\n", want: "`invoice` must be a mapping"},
		{name: "edit alias", command: "edit", invoice: archiveShapeAlias, want: "`invoice` must be a mapping"},
		{name: "edit missing", command: "edit", invoice: "", want: "missing `invoice` mapping"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			workDir := t.TempDir()
			chdirForTest(t, workDir)

			source := "customer_id: CUST-003\n" + tt.invoice + archiveShapePositions
			path, arg := filepath.Join(workDir, "inv.yaml"), "inv.yaml"
			if tt.command == "edit" {
				path = filepath.Join(archiveDir, "inv.yaml")
			}
			if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
				t.Fatal(err)
			}
			label := arg
			if tt.command == "edit" {
				label = path
			}

			exitCode, stdout, stderr := captureRun(t, []string{"archive", tt.command, arg})
			if want := "error: " + label + ": " + tt.want + "\n"; exitCode != 1 || stdout != "" || stderr != want {
				t.Fatalf("archive %s = exit %d, stdout %q, stderr %q; want exit 1, no stdout, stderr %q", tt.command, exitCode, stdout, stderr, want)
			}
			if got := readFileForTest(t, path); got != source {
				t.Fatalf("%s changed:\n%s", path, got)
			}
			entries, err := os.ReadDir(workDir)
			if err != nil {
				t.Fatal(err)
			}
			if tt.command == "edit" && len(entries) != 0 {
				t.Fatalf("archive edit left %d files in the work dir", len(entries))
			}
		})
	}
}

// Archiving removes the `_invox` key whatever it holds, including a link
// that names no archived file.
func TestArchiveRemovesArchiveLinkThatNamesNoFile(t *testing.T) {
	want := `customer_id: CUST-003
invoice:
  number: CUST-003-001
  issue_date: 2026-03-07
  due_date: 2026-04-06
  status: archived
  period: March
  vat_percent: 20
positions:
  - name: Consulting
    description: Workshop
    unit_price: 500
    quantity: 1
`
	for name, link := range map[string]string{
		"empty mapping": "_invox: {}\n",
		"null":          "_invox:\n",
		"empty path":    "_invox: {archive_path: \"\"}\n",
	} {
		t.Run(name, func(t *testing.T) {
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			if err := os.WriteFile(filepath.Join(workDir, "inv.yaml"), []byte(archiveShapeInvoice+link), 0o644); err != nil {
				t.Fatal(err)
			}

			exitCode, stdout, stderr := captureRun(t, []string{"archive", "add", "inv.yaml"})
			archived := filepath.Join(archiveDir, "inv.yaml")
			if wantErr := "Archived inv.yaml -> " + archived + "\n"; exitCode != 0 || stdout != archived+"\n" || stderr != wantErr {
				t.Fatalf("archive add = exit %d, stdout %q, stderr %q; want exit 0, stdout %q, stderr %q", exitCode, stdout, stderr, archived+"\n", wantErr)
			}
			if got := readFileForTest(t, archived); got != want {
				t.Fatalf("archived invoice =\n%s\nwant\n%s", got, want)
			}
		})
	}
}

// Writes other than archiving keep an `_invox` key that names no file as
// written.
func TestIncrementKeepsArchiveLinkThatNamesNoFile(t *testing.T) {
	for name, link := range map[string]string{
		"empty mapping": "_invox: {}\n",
		"null":          "_invox:\n",
	} {
		t.Run(name, func(t *testing.T) {
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(t.TempDir())+"\n")
			workDir := t.TempDir()
			chdirForTest(t, workDir)
			customersPath := filepath.Join(t.TempDir(), "customers.yaml")
			if err := os.WriteFile(customersPath, []byte("CUST-003:\n  name: Third Customer KG\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workDir, "inv.yaml"), []byte(archiveShapeInvoice+link), 0o644); err != nil {
				t.Fatal(err)
			}

			exitCode, stdout, stderr := captureRun(t, []string{"increment", "inv.yaml", "-c", customersPath})
			if exitCode != 0 || stdout != "inv.yaml\n" {
				t.Fatalf("increment = exit %d, stdout %q, stderr %q; want exit 0, stdout %q", exitCode, stdout, stderr, "inv.yaml\n")
			}
			want := `customer_id: CUST-003
invoice:
  number: CUST-003-002
  issue_date: 2026-03-07
  due_date: 2026-04-06
  status: built
  period: March
  vat_percent: 20
positions:
  - name: Consulting
    description: Workshop
    unit_price: 500
    quantity: 1
` + link
			if got := readFileForTest(t, filepath.Join(workDir, "inv.yaml")); got != want {
				t.Fatalf("inv.yaml =\n%s\nwant\n%s", got, want)
			}
		})
	}
}
