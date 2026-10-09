package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

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
// archive edit. A null one counts as missing, a scalar is a decode error
// with its line, and an alias to a mapping is refused because archiving
// rewrites the mapping in place.
func TestArchiveRefusesInvoiceKeyThatIsNotAMapping(t *testing.T) {
	tests := []struct {
		name    string
		command string
		flags   []string
		invoice string
		want    string
	}{
		{name: "add null", command: "add", invoice: "invoice:\n", want: ": missing `invoice` mapping"},
		{name: "add tilde", command: "add", invoice: "invoice: ~\n", want: ": missing `invoice` mapping"},
		{name: "add scalar", command: "add", invoice: "invoice: 5\n", want: ":2: invoice must be a mapping, got an integer"},
		{name: "add alias", command: "add", invoice: archiveShapeAlias, want: ": `invoice` must be a mapping"},
		{name: "add dry-run alias", command: "add", flags: []string{"--dry-run"}, invoice: archiveShapeAlias, want: ": `invoice` must be a mapping"},
		{name: "add missing", command: "add", invoice: "", want: ": missing `invoice` mapping"},
		{name: "edit null", command: "edit", invoice: "invoice:\n", want: ": missing `invoice` mapping"},
		{name: "edit alias", command: "edit", invoice: archiveShapeAlias, want: ": `invoice` must be a mapping"},
		{name: "edit missing", command: "edit", invoice: "", want: ": missing `invoice` mapping"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := clitest.New(t)

			archiveDir := t.TempDir()
			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
			workDir := t.TempDir()
			x.Chdir(workDir)

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

			exitCode, stdout, stderr := x.Run(append([]string{"archive", tt.command, arg}, tt.flags...))
			if want := "error: " + label + tt.want + "\n"; exitCode != 1 || stdout != "" || stderr != want {
				t.Fatalf("archive %s = exit %d, stdout %q, stderr %q; want exit 1, no stdout, stderr %q", tt.command, exitCode, stdout, stderr, want)
			}
			if got := testfixture.ReadFile(t, path); got != source {
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

// Writes other than archiving keep an `_invox` key that names no file as
// written.
func TestIncrementKeepsArchiveLinkThatNamesNoFile(t *testing.T) {
	for name, link := range map[string]string{
		"empty mapping": "_invox: {}\n",
		"null":          "_invox:\n",
	} {
		t.Run(name, func(t *testing.T) {
			x := clitest.New(t)

			x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(t.TempDir()) + "\n")
			workDir := t.TempDir()
			x.Chdir(workDir)
			customersPath := filepath.Join(t.TempDir(), "customers.yaml")
			if err := os.WriteFile(customersPath, []byte("CUST-003:\n  name: Third Customer KG\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(workDir, "inv.yaml"), []byte(testfixture.Source("cust003-built.yaml")+link), 0o644); err != nil {
				t.Fatal(err)
			}

			exitCode, stdout, stderr := x.Run([]string{"increment", "inv.yaml", "-c", customersPath})
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
			if got := testfixture.ReadFile(t, filepath.Join(workDir, "inv.yaml")); got != want {
				t.Fatalf("inv.yaml =\n%s\nwant\n%s", got, want)
			}
		})
	}
}
