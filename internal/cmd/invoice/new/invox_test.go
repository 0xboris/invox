package newcmd_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestNewRequiresCustomerID(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"new"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "missing required arguments: CUSTOMER_ID") {
		t.Fatalf("stderr %q does not contain missing CUSTOMER_ID message", stderr)
	}
}

func TestNewHelpShowsShortFlags(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"new", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"CUSTOMER_ID",
		"-c, --customers string",
		"-u, --issuer string",
		"      --defaults string",
		"-o, --output string",
		"-e, --edit",
		"--from-last",
		"<invoice.number>.yaml in the current directory",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestNewCreatesDefaultOutputInvoiceFile(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	workDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{
		"new",
		"CUST-001",
		"-c", draft.Customers,
		"-u", draft.Issuer,
		"--defaults", draft.Defaults,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-002.yaml for CUST-001 (CUST-001-002)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-002.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-002.yaml\n")
	}
	if _, err := os.Stat(filepath.Join(workDir, "CUST-001-002.yaml")); err != nil {
		t.Fatalf("default invoice file was not created: %v", err)
	}
}

func TestNewEditOpensCreatedInvoiceFile(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	workDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	x.Chdir(workDir)
	openedPath := x.ExpectEditor(nil)

	exitCode, stdout, stderr := x.Run([]string{
		"new",
		"CUST-001",
		"-e",
		"-c", draft.Customers,
		"-u", draft.Issuer,
		"--defaults", draft.Defaults,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	wantPath := filepath.Join(workDir, "CUST-001-002.yaml")
	if *openedPath != wantPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, wantPath)
	}
	if want := "Created CUST-001-002.yaml for CUST-001 (CUST-001-002)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-002.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-002.yaml\n")
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("edited invoice file was not created: %v", err)
	}
}

func TestNewEditReportsFailureAfterCreatingInvoiceFile(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	workDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	x.Chdir(workDir)
	x.ExpectEditor(errors.New("editor unavailable"))

	exitCode, stdout, stderr := x.Run([]string{
		"new",
		"CUST-001",
		"--edit",
		"-c", draft.Customers,
		"-u", draft.Issuer,
		"--defaults", draft.Defaults,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "created CUST-001-002.yaml but failed to open it: editor unavailable") {
		t.Fatalf("stderr %q does not contain editor error", stderr)
	}

	wantPath := filepath.Join(workDir, "CUST-001-002.yaml")
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("invoice file should still have been created: %v", err)
	}
}

func TestNewUsesCustomerSpecificStartFromCustomersFile(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraftWithStart(t, "7")
	workDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{
		"new",
		"CUST-001",
		"-c", draft.Customers,
		"-u", draft.Issuer,
		"--defaults", draft.Defaults,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-007.yaml for CUST-001 (CUST-001-007)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-007.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-007.yaml\n")
	}
}

func TestNewAcceptsInlineLongFlagsAfterCustomerID(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	outputPath := filepath.Join(t.TempDir(), "out.yaml")

	exitCode, stdout, stderr := x.Run([]string{
		"new",
		"CUST-001",
		"--output=" + outputPath,
		"--customers=" + draft.Customers,
		"--issuer=" + draft.Issuer,
		"--defaults=" + draft.Defaults,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created " + outputPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	if _, err := os.Stat(outputPath); err != nil {
		t.Fatalf("output file was not created: %v", err)
	}
}

func TestNewFromLastUsesLatestArchivedInvoiceForCustomer(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	if err := os.WriteFile(filepath.Join(archiveDir, "2026-03-01.yaml"), []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-01
  due_date: 2026-03-31
  status: archived
  period: February 2026
  vat_percent: 19
  paid_amount: 500
positions:
  - name: Older position
    description: Keep me old
    unit_price: 50
    quantity: 1
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(older archived invoice) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(archiveDir, "2026-03-08.yaml"), []byte(strings.TrimSpace(`
customer_id: CUST-001
invoice:
  number: CUST-001-002
  issue_date: 2026-03-08
  due_date: 2026-04-07
  status: archived
  period: March 2026
  vat_percent: 10
  paid_amount: 999
positions:
  - name: Latest position
    description: Keep me latest
    unit_price: 120
    quantity: 2
`)+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(latest archived invoice) returned error: %v", err)
	}

	workDir := t.TempDir()
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{
		"new",
		"CUST-001",
		"--from-last",
		"-c", draft.Customers,
		"-u", draft.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-003.yaml for CUST-001 (CUST-001-003)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-003.yaml\n" {
		t.Fatalf("stdout = %q, want %q", stdout, "CUST-001-003.yaml\n")
	}

	createdPath := filepath.Join(workDir, "CUST-001-003.yaml")
	createdSource, err := os.ReadFile(createdPath)
	if err != nil {
		t.Fatalf("ReadFile(createdPath) returned error: %v", err)
	}
	createdText := string(createdSource)
	for _, want := range []string{
		"number: CUST-001-003",
		"status: draft",
		"paid_amount: \"0\"",
		"period: March 2026",
		"name: Latest position",
		"description: Keep me latest",
		"unit_price: 120",
		"quantity: 2",
	} {
		if !strings.Contains(createdText, want) {
			t.Fatalf("created invoice does not contain %q:\n%s", want, createdText)
		}
	}
	for _, forbidden := range []string{
		"Older position",
		"Keep me old",
		"paid_amount: 999",
		"status: archived",
	} {
		if strings.Contains(createdText, forbidden) {
			t.Fatalf("created invoice should not contain %q:\n%s", forbidden, createdText)
		}
	}
}

func TestNewHelpShowsSupportFileDocumentationHints(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"new", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"-c, --customers string",
		"-u, --issuer string",
		"      --defaults string",
		"schema/docs: run `invox help customers`",
		"schema/docs: run `invox help issuer`",
		"schema/docs: run `invox help defaults`",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestNewTwiceWithoutArchivingAllocatesDifferentNumbers(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	workDir := t.TempDir()
	x.Chdir(workDir)

	for _, tc := range []struct {
		output     string
		wantStdout string
		wantStderr string
	}{
		{output: "a.yaml", wantStdout: "a.yaml\n", wantStderr: "Created a.yaml for CUST-001 (CUST-001-001)\n"},
		{output: "b.yaml", wantStdout: "b.yaml\n", wantStderr: "Created b.yaml for CUST-001 (CUST-001-002)\n"},
		{output: "", wantStdout: "CUST-001-003.yaml\n", wantStderr: "Created CUST-001-003.yaml for CUST-001 (CUST-001-003)\n"},
	} {
		args := []string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults}
		if tc.output != "" {
			args = append(args, "-o", tc.output)
		}
		exitCode, stdout, stderr := x.Run(args)
		if exitCode != 0 {
			t.Fatalf("new -o %q: exitCode = %d, want 0, stderr=%q", tc.output, exitCode, stderr)
		}
		if stderr != tc.wantStderr {
			t.Fatalf("new -o %q: stderr = %q, want %q", tc.output, stderr, tc.wantStderr)
		}
		if stdout != tc.wantStdout {
			t.Fatalf("new -o %q: stdout = %q, want %q", tc.output, stdout, tc.wantStdout)
		}
	}
}

func TestNewSkipsNumbersOfDraftsInOutputDirectory(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	x.Chdir(t.TempDir())

	outputDir := t.TempDir()
	testfixture.WriteNumberedInvoice(t, outputDir, "built.yaml", "CUST-001-004", "built")
	testfixture.WriteNumberedInvoice(t, outputDir, "editing.yaml", "CUST-001-009", "editing")

	exitCode, stdout, stderr := x.Run([]string{
		"new", "CUST-001",
		"-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults,
		"-o", filepath.Join(outputDir, "next.yaml"),
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stderr, "(CUST-001-005)") {
		t.Fatalf("stderr = %q, want number CUST-001-005", stderr)
	}
	if want := filepath.Join(outputDir, "next.yaml") + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestNewIgnoresUnrelatedAndOversizedYAMLInWorkingDirectory(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	archiveDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 1\narchive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	workDir := t.TempDir()
	x.Chdir(workDir)

	for name, source := range map[string]string{
		"broken.yaml":    "invoice: [unclosed\n",
		"customers.yaml": "CUST-001:\n  name: Appsters GmbH\n",
		"list.yml":       "- one\n- two\n",
		"other.yaml":     "customer_id: CUST-002\ninvoice:\n  number: CUST-002-007\n  status: draft\n",
	} {
		if err := os.WriteFile(filepath.Join(workDir, name), []byte(source), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", name, err)
		}
	}
	oversized := testfixture.ReadFile(t, testfixture.WriteNumberedInvoice(t, workDir, "huge.yaml", "CUST-001-009", "draft")) +
		"# " + strings.Repeat("x", 1<<20) + "\n"
	if err := os.WriteFile(filepath.Join(workDir, "huge.yaml"), []byte(oversized), 0o644); err != nil {
		t.Fatalf("WriteFile(huge.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-001.yaml for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if want := "CUST-001-001.yaml\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
}

func TestNewForceReplacesExistingOutput(t *testing.T) {
	x := clitest.New(t)

	draft := testfixture.WriteDraft(t)
	workDir := t.TempDir()
	x.WriteConfig("numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
	outputPath := filepath.Join(workDir, "mine.yaml")
	if err := os.WriteFile(outputPath, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	x.Chdir(workDir)

	exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001", "-c", draft.Customers, "-u", draft.Issuer, "--defaults", draft.Defaults, "-o", "mine.yaml", "--force"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "mine.yaml\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if want := "Created mine.yaml for CUST-001 (CUST-001-002)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if got := testfixture.ReadFile(t, outputPath); !strings.Contains(got, "number: CUST-001-002") {
		t.Fatalf("mine.yaml was not replaced:\n%s", got)
	}
}
