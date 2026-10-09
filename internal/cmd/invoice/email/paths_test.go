package email

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestResolveEmailDraftPaths(t *testing.T) {
	t.Parallel()

	h := isolatedHost(t)
	rootDir := t.TempDir()
	yamlInput := filepath.Join(rootDir, "BL00210001.yaml")
	pdfInput := filepath.Join(rootDir, "BL00210001.pdf")
	if err := os.WriteFile(yamlInput, []byte(builtInvoice), 0o644); err != nil {
		t.Fatalf("WriteFile(yamlInput) returned error: %v", err)
	}
	for _, pdf := range []string{pdfInput, filepath.Join(rootDir, "outgoing.pdf")} {
		if err := os.WriteFile(pdf, []byte("%PDF-1.4\n"), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", pdf, err)
		}
	}

	tests := []struct {
		name          string
		inputPath     string
		pdfPath       string
		outputPath    string
		wantInvoice   string
		wantPDF       string
		wantOutput    string
		wantErrSubstr string
	}{
		{
			name:        "yaml input derives sibling pdf and eml",
			inputPath:   yamlInput,
			wantInvoice: yamlInput,
			wantPDF:     pdfInput,
			wantOutput:  filepath.Join(rootDir, "BL00210001.eml"),
		},
		{
			name:        "pdf input derives sibling yaml and eml",
			inputPath:   pdfInput,
			wantInvoice: yamlInput,
			wantPDF:     pdfInput,
			wantOutput:  filepath.Join(rootDir, "BL00210001.eml"),
		},
		{
			name:        "explicit overrides are preserved",
			inputPath:   pdfInput,
			pdfPath:     filepath.Join(rootDir, "outgoing.pdf"),
			outputPath:  filepath.Join(rootDir, "drafts", "outgoing.eml"),
			wantInvoice: yamlInput,
			wantPDF:     filepath.Join(rootDir, "outgoing.pdf"),
			wantOutput:  filepath.Join(rootDir, "drafts", "outgoing.eml"),
		},
		{
			name:          "unsupported input extension",
			inputPath:     filepath.Join(rootDir, "BL00210001.txt"),
			wantErrSubstr: "input must end with .yaml, .yml, or .pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			paths, err := h.emailDraftPaths(t, tt.inputPath, tt.pdfPath, tt.outputPath)
			if tt.wantErrSubstr != "" {
				if err == nil {
					t.Fatal("ResolveEmailDraftPaths returned nil error")
				}
				if !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ResolveEmailDraftPaths returned error: %v", err)
			}
			if paths.InvoicePath != tt.wantInvoice {
				t.Fatalf("InvoicePath = %q, want %q", paths.InvoicePath, tt.wantInvoice)
			}
			if paths.PDFPath != tt.wantPDF {
				t.Fatalf("PDFPath = %q, want %q", paths.PDFPath, tt.wantPDF)
			}
			if paths.OutputPath != tt.wantOutput {
				t.Fatalf("OutputPath = %q, want %q", paths.OutputPath, tt.wantOutput)
			}
		})
	}
}

func TestResolveEmailDraftPathsFallsBackToArchiveDir(t *testing.T) {
	t.Parallel()

	archiveDir := t.TempDir()
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	archivedInvoicePath := filepath.Join(archiveDir, "customer-a", "BL00210001.yaml")
	if err := os.MkdirAll(filepath.Dir(archivedInvoicePath), 0o755); err != nil {
		t.Fatalf("MkdirAll(filepath.Dir(archivedInvoicePath)) returned error: %v", err)
	}
	if err := os.WriteFile(archivedInvoicePath, []byte(builtInvoice), 0o644); err != nil {
		t.Fatalf("WriteFile(archivedInvoicePath) returned error: %v", err)
	}

	pdfPath := filepath.Join(t.TempDir(), "BL00210001.pdf")
	if err := os.WriteFile(pdfPath, []byte("%PDF-1.4\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(pdfPath) returned error: %v", err)
	}
	paths, err := h.emailDraftPaths(t, pdfPath, "", "")
	if err != nil {
		t.Fatalf("ResolveEmailDraftPaths returned error: %v", err)
	}
	if paths.InvoicePath != archivedInvoicePath {
		t.Fatalf("InvoicePath = %q, want %q", paths.InvoicePath, archivedInvoicePath)
	}
	if paths.PDFPath != pdfPath {
		t.Fatalf("PDFPath = %q, want %q", paths.PDFPath, pdfPath)
	}
	if paths.OutputPath != filepath.Join(filepath.Dir(pdfPath), "BL00210001.eml") {
		t.Fatalf("OutputPath = %q, want %q", paths.OutputPath, filepath.Join(filepath.Dir(pdfPath), "BL00210001.eml"))
	}
}

// host is a user's config home and home directory.
type host struct {
	configHome string
	home       string
}

func isolatedHost(t *testing.T) host {
	t.Helper()
	root := t.TempDir()
	return host{configHome: filepath.Join(root, "config-home"), home: filepath.Join(root, "home")}
}

// writeConfigFile returns a host under fresh temporary directories whose
// config.yaml is source.
func writeConfigFile(t *testing.T, source string) host {
	t.Helper()
	h := isolatedHost(t)
	configDir := filepath.Join(h.configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(config.yaml) returned error: %v", err)
	}
	return h
}

func quoteYAMLString(value string) string {
	return strconv.Quote(value)
}

// draftPaths are the invoice an email is drafted for, the PDF attached to
// it and the draft file.
type draftPaths struct {
	InvoicePath string
	PDFPath     string
	OutputPath  string
}

// emailDraftPaths runs email on input with -p pdf and -o output, each ""
// to leave the flag out, and returns the paths the draft was made with.
// The draft itself is recorded instead of written.
func (h host) emailDraftPaths(t *testing.T, input, pdf, output string) (draftPaths, error) {
	t.Helper()
	parties := t.TempDir()
	customersPath := filepath.Join(parties, "customers.yaml")
	issuerPath := filepath.Join(parties, "issuer.yaml")
	for path, source := range map[string]string{customersPath: customersYAML, issuerPath: issuerYAML} {
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) returned error: %v", path, err)
		}
	}

	var paths draftPaths
	f := factorytest.New(t, nil, factorytest.Options{Home: h.home, Vars: map[string]string{"XDG_CONFIG_HOME": h.configHome}})
	ios, _, _, _ := iostreams.Test()
	opts := &EmailOptions{
		IO: ios,
		Service: func(files cmdutil.Files) *billing.Service {
			svc := f.Service(files)
			svc.Invoices = loadRecorder{Invoices: svc.Invoices, path: &paths.InvoicePath}
			svc.Mailer = draftRecorder{Mailer: svc.Mailer, paths: &paths}
			return svc
		},
		Getwd:       func() (string, error) { return parties, nil },
		InvoicePath: input,
		PDFPath:     pdf,
		OutputPath:  output,
		KeepDraft:   output != "",
		Support:     cmdutil.SupportPaths{Customers: customersPath, Issuer: issuerPath},
	}
	if err := validate(opts); err != nil {
		return draftPaths{}, err
	}
	err := emailRun(context.Background(), opts)
	return paths, err
}

// loadRecorder records the invoice file the use case loads.
type loadRecorder struct {
	billing.Invoices
	path *string
}

func (r loadRecorder) Load(path string) (invoice.Invoice, error) {
	*r.path = path
	return r.Invoices.Load(path)
}

// draftRecorder records the attachment and the draft file of a message
// instead of drafting it.
type draftRecorder struct {
	billing.Mailer
	paths *draftPaths
}

func (r draftRecorder) Draft(_ context.Context, m billing.Message, _ bool) (string, error) {
	r.paths.PDFPath, r.paths.OutputPath = m.Attachment, m.Output
	return "", nil
}

const customersYAML = `CUST-001:
  name: Appsters GmbH
  status: active
  email: office@appsters.example
  address:
    street: Hauptstrasse 1
    postal_code: 1010
    city: Vienna
    country: Austria
  tax:
    vat_tax_id: ATU12345678
`

const issuerYAML = `company:
  legal_company_name: Boris Consulting
  company_registration_number: FN 123456a
  vat_tax_id: ATU87654321
  website: https://example.com
  email: hello@example.com
  address:
    street: Ring 1
    postal_code: 1010
    city: Vienna
    country: Austria
payment:
  bank_name: Test Bank
  iban: AT611904300234573201
  bic: BKAUATWW
  due_days: 30
  payment_terms_text: Pay within 30 days
`

const builtInvoice = `customer_id: CUST-001
invoice:
  number: BL00210001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  status: built
  period: Leistungszeitraum
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
`
