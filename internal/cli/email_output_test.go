package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/iostreams"
)

// writeBuiltEmailFixture writes a built invoice YAML and its PDF into a fresh directory and
// returns the customers, issuer and invoice paths.
func writeBuiltEmailFixture(t *testing.T) (string, string, string) {
	t.Helper()

	customersPath, issuerPath, invoicePath, _ := writeContextFixtures(t)
	source, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)

	inputDir := t.TempDir()
	builtInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	if err := os.WriteFile(builtInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(builtInvoicePath) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(inputDir, "BL00210001.pdf"), []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	return customersPath, issuerPath, builtInvoicePath
}

// stubEmailOpeners replaces the document opener for the test and returns a
// pointer to the paths it was called with.
func stubEmailOpeners(t *testing.T) *[]string {
	t.Helper()

	var opened []string
	oldOpenDocument := openDocument
	oldPreferNativeMailCompose := preferNativeMailCompose
	openDocument = func(_ *iostreams.IOStreams, path string) error {
		opened = append(opened, path)
		return nil
	}
	preferNativeMailCompose = false
	t.Cleanup(func() {
		openDocument = oldOpenDocument
		preferNativeMailCompose = oldPreferNativeMailCompose
	})
	return &opened
}

func TestEmailRefusesExistingOutputWithoutForce(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	opened := stubEmailOpeners(t)

	outputPath := filepath.Join(t.TempDir(), "keep.eml")
	if err := os.WriteFile(outputPath, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"email", invoicePath,
		"-o", outputPath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "keep.eml already exists; pass --force or choose another -o path") {
		t.Fatalf("stderr = %q, want an already-exists error with a next step", stderr)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if string(content) != "keep\n" {
		t.Fatalf("outputPath content = %q, want it untouched", content)
	}
	if len(*opened) != 0 {
		t.Fatalf("opened = %q, want no call", *opened)
	}
}

func TestEmailKeepsExplicitOutputFile(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	opened := stubEmailOpeners(t)

	outputPath := filepath.Join(t.TempDir(), "drafts", "BL00210001.eml")
	exitCode, stdout, stderr := captureRun(t, []string{
		"email", invoicePath,
		"-o", outputPath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	if len(*opened) != 1 || (*opened)[0] != outputPath {
		t.Fatalf("opened = %q, want [%q]", *opened, outputPath)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("explicit -o draft is gone after the command returned: %v", err)
	}
	if !strings.Contains(string(content), `filename="BL00210001.pdf"`) {
		t.Fatalf("draft does not contain the attached PDF:\n%s", content)
	}
}

func TestEmailForceOverwritesExistingOutputFile(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	stubEmailOpeners(t)

	outputPath := filepath.Join(t.TempDir(), "draft.eml")
	if err := os.WriteFile(outputPath, []byte("old draft\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"email", invoicePath,
		"--force",
		"-o", outputPath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if !strings.Contains(string(content), `filename="BL00210001.pdf"`) {
		t.Fatalf("draft was not overwritten:\n%s", content)
	}
}

func TestEmailImplicitDraftLeavesSiblingEMLUntouched(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	opened := stubEmailOpeners(t)

	siblingPath := filepath.Join(filepath.Dir(invoicePath), "BL00210001.eml")
	if err := os.WriteFile(siblingPath, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(siblingPath) returned error: %v", err)
	}

	exitCode, _, stderr := captureRun(t, []string{
		"email", invoicePath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if len(*opened) != 1 {
		t.Fatalf("opened = %q, want one call", *opened)
	}
	draftDir := filepath.Dir((*opened)[0])
	t.Cleanup(func() { os.RemoveAll(draftDir) })
	if draftDir == filepath.Dir(invoicePath) {
		t.Fatalf("opened %q, want the draft in a temporary directory", (*opened)[0])
	}
	content, err := os.ReadFile(siblingPath)
	if err != nil {
		t.Fatalf("ReadFile(siblingPath) returned error: %v", err)
	}
	if string(content) != "keep\n" {
		t.Fatalf("sibling .eml content = %q, want it untouched", content)
	}
}

func TestEmailImplicitDraftOpenFailureRemovesDraftDirectory(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	stubEmailOpeners(t)

	openedPath := ""
	openDocument = func(_ *iostreams.IOStreams, path string) error {
		openedPath = path
		return errors.New("no mail app")
	}

	exitCode, stdout, stderr := captureRun(t, []string{
		"email", invoicePath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if want := "error: failed to open email draft: no mail app\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if _, err := os.Stat(filepath.Dir(openedPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(draft directory) error = %v, want not exists", err)
	}
}
