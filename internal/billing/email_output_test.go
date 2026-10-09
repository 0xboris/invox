package billing_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestCreateInvoiceEmailDraftRefusesExistingOutputUnlessOverwrite(t *testing.T) {
	t.Parallel()

	h := testfixture.NewHost(t)

	fx := testfixture.WriteContext(t)
	source, err := os.ReadFile(fx.Invoice)
	if err != nil {
		t.Fatalf("ReadFile(invoicePath) returned error: %v", err)
	}
	mutated := strings.Replace(string(source), "  paid_amount: 0", "  paid_amount: 0\n  status: built", 1)
	if err := os.WriteFile(fx.Invoice, []byte(mutated), 0o644); err != nil {
		t.Fatalf("WriteFile(invoicePath) returned error: %v", err)
	}
	pdfPath := filepath.Join(t.TempDir(), "invoice.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\nfake"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	outputPath := filepath.Join(t.TempDir(), "invoice.eml")
	if err := os.WriteFile(outputPath, []byte("keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(outputPath) returned error: %v", err)
	}

	_, err = service(t, h, cmdutil.Files{Customers: fx.Customers, Issuer: fx.Issuer}, t.TempDir(), time.Now()).DraftEmail(context.Background(), billing.EmailRequest{Invoice: fx.Invoice, PDF: pdfPath, Output: outputPath, Keep: true})
	var exists *billing.OutputExistsError
	if !errors.As(err, &exists) || exists.Path != outputPath {
		t.Fatalf("CreateInvoiceEmailDraft error = %v, want *billing.OutputExistsError for %s", err, outputPath)
	}
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if string(content) != "keep\n" {
		t.Fatalf("outputPath content = %q, want it untouched", content)
	}

	svc, _ := mailService(t, h, cmdutil.Files{Customers: fx.Customers, Issuer: fx.Issuer}, t.TempDir(), time.Now())
	if _, err := svc.DraftEmail(context.Background(), billing.EmailRequest{Invoice: fx.Invoice, PDF: pdfPath, Output: outputPath, Keep: true, Overwrite: true}); err != nil {
		t.Fatalf("CreateInvoiceEmailDraft with overwrite returned error: %v", err)
	}
	content, err = os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile(outputPath) returned error: %v", err)
	}
	if !strings.Contains(string(content), "X-Unsent: 1") {
		t.Fatalf("outputPath was not overwritten with the draft:\n%s", content)
	}
}
