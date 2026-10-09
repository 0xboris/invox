package email_test

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestEmailHelpShowsDraftOutputAndFlags(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"email", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"INVOICE.yaml, INVOICE.pdf, or -i, --input PATH",
		"-p, --pdf string",
		"-o, --output string",
		"--to string",
		"--subject string",
		"--force",
		"<input name>.eml in a new temporary directory, removed after 24 hours",
		"Accepts either the invoice YAML file or the built PDF as input.",
		"The PDF lookup checks next to the PDF first, then archive.dir.",
		"Requires invoice.status to be built or archived and the PDF attachment to exist.",
		"On macOS, opens an editable compose window in Apple Mail with the PDF attached.",
		"If -o is set, or on non-macOS platforms, writes a .eml draft file and opens it.",
		"Does not send the email and does not change invoice.status.",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

func TestEmailDefaultsDraftPathFromInputFile(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(fx.Invoice, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	if err := os.WriteFile(customInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}
	pdfPath := filepath.Join(inputDir, "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	openedPath := x.ExpectOpener(nil)

	exitCode, stdout, stderr := x.Run([]string{
		"email",
		customInvoicePath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if filepath.Base(*openedPath) != "BL00210001.eml" || filepath.Dir(*openedPath) == inputDir {
		t.Fatalf("openedPath = %q, want BL00210001.eml in a temporary directory", *openedPath)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(*openedPath)) })
	if _, err := os.Stat(*openedPath); err != nil {
		t.Fatalf("Stat(openedPath) returned error: %v, want the draft kept for the mail app", err)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	openedSource, err := os.ReadFile(*openedPath)
	if err != nil {
		t.Fatalf("ReadFile(openedPath) returned error: %v", err)
	}
	if !strings.Contains(string(openedSource), `filename="BL00210001.pdf"`) {
		t.Fatalf("draft email does not contain attached PDF filename:\n%s", openedSource)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(outputPath) error = %v, want not exists", err)
	}
}

func TestEmailAcceptsPDFInputFile(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(fx.Invoice, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	if err := os.WriteFile(customInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}
	pdfPath := filepath.Join(inputDir, "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	openedPath := x.ExpectOpener(nil)

	exitCode, stdout, stderr := x.Run([]string{
		"email",
		pdfPath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if filepath.Base(*openedPath) != "BL00210001.eml" || filepath.Dir(*openedPath) == inputDir {
		t.Fatalf("openedPath = %q, want BL00210001.eml in a temporary directory", *openedPath)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(*openedPath)) })
	if _, err := os.Stat(*openedPath); err != nil {
		t.Fatalf("Stat(openedPath) returned error: %v, want the draft kept for the mail app", err)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	openedSource, err := os.ReadFile(*openedPath)
	if err != nil {
		t.Fatalf("ReadFile(openedPath) returned error: %v", err)
	}
	if !strings.Contains(string(openedSource), `filename="BL00210001.pdf"`) {
		t.Fatalf("draft email does not contain attached PDF filename:\n%s", openedSource)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(outputPath) error = %v, want not exists", err)
	}
}

func TestEmailFindsInvoiceYAMLInArchiveDirForPDFInput(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)

	archiveDir := t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")

	archivedInvoicePath := filepath.Join(archiveDir, "customer-a", "BL00210001.yaml")
	if err := os.MkdirAll(filepath.Dir(archivedInvoicePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(filepath.Dir(archivedInvoicePath)) returned error: %v", err)
	}
	if err := os.WriteFile(archivedInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedInvoicePath) returned error: %v", err)
	}

	inputDir := t.TempDir()
	pdfPath := filepath.Join(inputDir, "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	openedPath := x.ExpectOpener(nil)

	exitCode, stdout, stderr := x.Run([]string{
		"email",
		pdfPath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	outputPath := filepath.Join(inputDir, "BL00210001.eml")
	if filepath.Base(*openedPath) != "BL00210001.eml" || filepath.Dir(*openedPath) == inputDir {
		t.Fatalf("openedPath = %q, want BL00210001.eml in a temporary directory", *openedPath)
	}
	t.Cleanup(func() { _ = os.RemoveAll(filepath.Dir(*openedPath)) })
	if _, err := os.Stat(*openedPath); err != nil {
		t.Fatalf("Stat(openedPath) returned error: %v, want the draft kept for the mail app", err)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if _, err := os.Stat(outputPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(outputPath) error = %v, want not exists", err)
	}
}

func TestEmailUsesEditableNativeComposeByDefault(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(fx.Invoice, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}

	inputDir := t.TempDir()
	customInvoicePath := filepath.Join(inputDir, "BL00210001.yaml")
	if err := os.WriteFile(customInvoicePath, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(customInvoicePath) returned error: %v", err)
	}
	pdfPath := filepath.Join(inputDir, "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}

	// Apple Mail drafts the email on macOS.
	x.GOOS = "darwin"
	var opened billing.Message
	x.Stub.Register("osascript", func(cmd run.Cmd) error {
		message := cmd.Args[slices.Index(cmd.Args, "--")+1:]
		opened = billing.Message{To: message[0], Subject: message[1], Body: message[2], Attachment: message[3], FromAddress: message[4]}
		return nil
	})

	exitCode, stdout, stderr := x.Run([]string{
		"email",
		customInvoicePath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if opened.To != "office@appsters.example" {
		t.Fatalf("To = %q, want %q", opened.To, "office@appsters.example")
	}
	if opened.Subject != "Invoice CUST-001-001" {
		t.Fatalf("Subject = %q, want %q", opened.Subject, "Invoice CUST-001-001")
	}
	if opened.Attachment != pdfPath {
		t.Fatalf("Attachment = %q, want %q", opened.Attachment, pdfPath)
	}
	if !strings.Contains(opened.Body, "Please find attached invoice CUST-001-001.") {
		t.Fatalf("Body %q does not contain the default invoice text", opened.Body)
	}
	if strings.Contains(opened.Body, "\r") {
		t.Fatalf("Body = %q, want LF-only newlines", opened.Body)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if _, err := os.Stat(filepath.Join(inputDir, "BL00210001.eml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(default .eml path) error = %v, want not exists", err)
	}
}

func TestEmailRemovesStaleTemporaryDraftsBeforeWritingANewOne(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)
	tempDir := t.TempDir()
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, tempDir)
	}
	staleDir := filepath.Join(tempDir, "invox-email-1")
	if err := os.Mkdir(staleDir, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := factorytest.Now.Add(-48 * time.Hour)
	if err := os.Chtimes(staleDir, stale, stale); err != nil {
		t.Fatal(err)
	}
	opened := x.ExpectOpener(nil)

	exitCode, stdout, stderr := x.Run([]string{"email", fx.Invoice, "-c", fx.Customers, "-u", fx.Issuer})

	if exitCode != 0 || stdout != "" {
		t.Fatalf("exit code, stdout = %d, %q, want 0, empty; stderr=%q", exitCode, stdout, stderr)
	}
	if _, err := os.Stat(staleDir); !os.IsNotExist(err) {
		t.Fatalf("Stat(stale draft directory) error = %v, want it removed", err)
	}
	if filepath.Dir(filepath.Dir(*opened)) != tempDir {
		t.Fatalf("opened %q, want a new draft directory in %q", *opened, tempDir)
	}
	if _, err := os.Stat(*opened); err != nil {
		t.Fatalf("Stat(new draft) returned error: %v, want it kept for the mail app", err)
	}
}

func TestEmailRefusesExistingOutputWithoutForce(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)

	outputPath := filepath.Join(t.TempDir(), "keep.eml")
	if err := os.WriteFile(outputPath, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"email", fx.Invoice,
		"-o", outputPath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
	})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "keep.eml already exists; pass --force to replace it or choose a different -o/--output path") {
		t.Fatalf("stderr = %q, want an already-exists error with a next step", stderr)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if string(content) != "keep\n" {
		t.Fatalf("outputPath content = %q, want it untouched", content)
	}
}

func TestEmailKeepsExplicitOutputFile(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)
	opened := x.ExpectOpener(nil)

	outputPath := filepath.Join(t.TempDir(), "drafts", "BL00210001.eml")
	exitCode, stdout, stderr := x.Run([]string{
		"email", fx.Invoice,
		"-o", outputPath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
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
	if *opened != outputPath {
		t.Fatalf("opened = %q, want %q", *opened, outputPath)
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
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)
	x.ExpectOpener(nil)

	outputPath := filepath.Join(t.TempDir(), "draft.eml")
	if err := os.WriteFile(outputPath, []byte("old draft\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{
		"email", fx.Invoice,
		"--force",
		"-o", outputPath,
		"-c", fx.Customers,
		"-u", fx.Issuer,
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
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)
	opened := x.ExpectOpener(nil)

	siblingPath := filepath.Join(filepath.Dir(fx.Invoice), "BL00210001.eml")
	if err := os.WriteFile(siblingPath, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(siblingPath) returned error: %v", err)
	}

	exitCode, _, stderr := x.Run([]string{
		"email", fx.Invoice,
		"-c", fx.Customers,
		"-u", fx.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	draftDir := filepath.Dir(*opened)
	t.Cleanup(func() { _ = os.RemoveAll(draftDir) })
	if draftDir == filepath.Dir(fx.Invoice) {
		t.Fatalf("opened %q, want the draft in a temporary directory", *opened)
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
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)
	openedPath := x.ExpectOpener(errors.New("no mail app"))

	exitCode, stdout, stderr := x.Run([]string{
		"email", fx.Invoice,
		"-c", fx.Customers,
		"-u", fx.Issuer,
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
	if _, err := os.Stat(filepath.Dir(*openedPath)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Stat(draft directory) error = %v, want not exists", err)
	}
}
