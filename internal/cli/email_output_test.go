package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

// stubEmailOpeners replaces the document opener and cleanup hooks for the test and
// returns pointers to the paths they were called with.
func stubEmailOpeners(t *testing.T) (*[]string, *[]string) {
	t.Helper()

	var opened, cleaned []string
	oldOpenDocument := openDocument
	oldCleanupOpenedDocument := cleanupOpenedDocument
	oldPreferNativeMailCompose := preferNativeMailCompose
	openDocument = func(path string) error {
		opened = append(opened, path)
		return nil
	}
	cleanupOpenedDocument = func(file, dir string) error {
		if filepath.Dir(file) != dir {
			t.Fatalf("cleanupOpenedDocument(%q, %q): file is not in dir", file, dir)
		}
		cleaned = append(cleaned, dir)
		return os.RemoveAll(dir)
	}
	preferNativeMailCompose = false
	t.Cleanup(func() {
		openDocument = oldOpenDocument
		cleanupOpenedDocument = oldCleanupOpenedDocument
		preferNativeMailCompose = oldPreferNativeMailCompose
	})
	return &opened, &cleaned
}

func TestEmailRefusesExistingOutputWithoutForce(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	opened, cleaned := stubEmailOpeners(t)

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
	if len(*opened) != 0 || len(*cleaned) != 0 {
		t.Fatalf("opened = %q, cleaned = %q, want neither called", *opened, *cleaned)
	}
}

func TestEmailKeepsExplicitOutputFile(t *testing.T) {
	customersPath, issuerPath, invoicePath := writeBuiltEmailFixture(t)
	opened, cleaned := stubEmailOpeners(t)

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
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if !strings.Contains(stdout, "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example") {
		t.Fatalf("stdout %q does not contain email summary", stdout)
	}
	if len(*opened) != 1 || (*opened)[0] != outputPath {
		t.Fatalf("opened = %q, want [%q]", *opened, outputPath)
	}
	if len(*cleaned) != 0 {
		t.Fatalf("cleanupOpenedDocument called with %q, want no cleanup for an explicit -o", *cleaned)
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
	_, cleaned := stubEmailOpeners(t)

	outputPath := filepath.Join(t.TempDir(), "draft.eml")
	if err := os.WriteFile(outputPath, []byte("old draft\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	exitCode, _, stderr := captureRun(t, []string{
		"email", invoicePath,
		"--force",
		"-o", outputPath,
		"-c", customersPath,
		"-u", issuerPath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	if len(*cleaned) != 0 {
		t.Fatalf("cleanupOpenedDocument called with %q, want no cleanup for an explicit -o", *cleaned)
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
	opened, cleaned := stubEmailOpeners(t)

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
	if len(*opened) != 1 || len(*cleaned) != 1 {
		t.Fatalf("opened = %q, cleaned = %q, want one call each", *opened, *cleaned)
	}
	if (*cleaned)[0] != filepath.Dir((*opened)[0]) {
		t.Fatalf("cleaned %q, want only the draft's temporary directory %q", (*cleaned)[0], filepath.Dir((*opened)[0]))
	}
	if _, err := os.Stat((*cleaned)[0]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(draft directory) error = %v, want not exists", err)
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
	_, cleaned := stubEmailOpeners(t)

	openedPath := ""
	openDocument = func(path string) error {
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
	if stderr != "failed to open email draft: no mail app\n" {
		t.Fatalf("stderr = %q, want %q", stderr, "failed to open email draft: no mail app\n")
	}
	if len(*cleaned) != 0 {
		t.Fatalf("cleanupOpenedDocument called with %q, want no delayed cleanup after a failed open", *cleaned)
	}
	if _, err := os.Stat(filepath.Dir(openedPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(draft directory) error = %v, want not exists", err)
	}
}
