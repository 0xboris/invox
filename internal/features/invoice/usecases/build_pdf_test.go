package usecases_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/programs/run"
	"github.com/0xboris/invox/internal/adapters/programs/tectonic"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/features/invoice/usecases"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// newInvoice writes a draft invoice for the starter customer into a new
// directory and returns its path.
func newInvoice(t *testing.T, hosts *invoice.HostSource) string {
	h := hosts.Host()
	if _, _, err := h.InitializeConfigDir(); err != nil {
		t.Fatal(err)
	}
	work := t.TempDir()
	path := filepath.Join(work, "inv.yaml")
	if _, err := h.CreateNewInvoice(invoice.NewInvoiceParams{Now: time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC), WorkDir: work, DefaultsPath: h.GlobalInvoiceDefaultsPath(), OutputPath: path, CustomersPath: h.GlobalCustomersPath(), IssuerPath: h.GlobalIssuerPath(), CustomerID: "CUST-001"}); err != nil {
		t.Fatal(err)
	}
	return path
}

func buildPDF(t *testing.T, compile func(run.Cmd) error) (invoicePath string, err error) {
	hosts := newHosts(t)
	invoicePath = newInvoice(t, hosts)
	h := hosts.Host()
	inv, err := usecases.NewLoadInvoice(hosts).Run(h.GlobalCustomersPath(), h.GlobalIssuerPath(), invoicePath)
	if err != nil {
		t.Fatal(err)
	}
	ios, _, _, _ := iostreams.Test()
	stub := run.NewStub(t)
	stub.Register("tectonic", compile)
	template := h.GlobalTemplatePath()
	outputPath := invoice.ReplaceExt(invoicePath, ".pdf")
	return invoicePath, usecases.NewBuildPDF(hosts, tectonic.New(stub, ios)).Run(context.Background(), template, outputPath, invoicePath, inv)
}

func invoiceStatus(t *testing.T, invoicePath string) string {
	data, err := os.ReadFile(invoicePath)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if status, ok := strings.CutPrefix(strings.TrimSpace(line), "status:"); ok {
			return strings.Trim(strings.TrimSpace(status), `"'`)
		}
	}
	return ""
}

func TestBuildPDFMarksTheInvoiceBuilt(t *testing.T) {
	invoicePath, err := buildPDF(t, func(cmd run.Cmd) error {
		pdf := strings.TrimSuffix(cmd.Args[0], ".tex") + ".pdf"
		return os.WriteFile(filepath.Join(cmd.Dir, pdf), []byte("%PDF\n"), 0o644)
	})

	if err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if got := invoiceStatus(t, invoicePath); got != "built" {
		t.Fatalf("invoice.status = %q; want built", got)
	}
}

func TestBuildPDFLeavesAFailedBuildUnmarked(t *testing.T) {
	failed := errors.New("tectonic failed")
	invoicePath, err := buildPDF(t, func(run.Cmd) error { return failed })

	if !errors.Is(err, failed) {
		t.Fatalf("Run() = %v; want %v", err, failed)
	}
	if got := invoiceStatus(t, invoicePath); got == "built" {
		t.Fatalf("invoice.status = %q after a failed build", got)
	}
}
