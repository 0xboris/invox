package store

// The tests in this package predate internal/billing and call the API that
// Host had then. This file implements that API on billing.Service with the
// real adapters, so each test keeps pinning the behavior it pinned before.

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/archive"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/email"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/numbering"
	"github.com/0xboris/invox/internal/render/latex"
)

type (
	ArchiveOptions      = billing.ArchiveOptions
	ArchiveResult       = billing.ArchiveResult
	ArchiveReplaceError = billing.ArchiveReplaceError
	DecodeError         = billing.DecodeError
)

// service returns the use cases on h, with the support files named as a
// command line would name them, run in workDir at now.
func (h Host) service(files Files, workDir string, now time.Time) *billing.Service {
	st := &Store{Host: h, Getwd: func() (string, error) { return workDir, nil }, Files: files}
	return &billing.Service{
		Invoices:  st,
		Directory: st,
		Archives:  archive.Archive{Locate: h.ResolveArchiveDir, Read: ReadArchived, Rewrite: st.Rewrite},
		Renderer:  latex.Renderer{},
		Mailer:    email.Mailer{},
		Settings:  h.Settings,
		Now:       func() time.Time { return now },
	}
}

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return dir
}

// LoadContext loads the invoice at invoicePath with its customer and
// issuer, as validate does.
func LoadContext(customersPath, issuerPath, invoicePath string) (*invoice.Context, error) {
	result, err := NewHost(HostInputs{GOOS: "linux"}).service(Files{Customers: customersPath, Issuer: issuerPath}, cwd(), time.Time{}).Validate(invoicePath)
	return result.Context, err
}

type NewInvoice struct {
	Number              string
	Path                string
	SkippedArchiveFiles []string
}

type NewInvoiceParams struct {
	Now           time.Time
	WorkDir       string
	DefaultsPath  string
	OutputPath    string
	CustomersPath string
	IssuerPath    string
	CustomerID    string
	FromLast      bool
	Overwrite     bool
	DryRun        bool
}

func (h Host) CreateNewInvoice(p NewInvoiceParams) (NewInvoice, error) {
	svc := h.service(Files{Customers: p.CustomersPath, Issuer: p.IssuerPath, Defaults: p.DefaultsPath}, p.WorkDir, p.Now)
	created, err := svc.New(billing.NewRequest{
		CustomerID: p.CustomerID,
		WorkDir:    p.WorkDir,
		Output:     absOrEmpty(p.WorkDir, p.OutputPath),
		FromLast:   p.FromLast,
		Overwrite:  p.Overwrite,
		DryRun:     p.DryRun,
	})
	return NewInvoice{Number: created.Number, Path: created.Path, SkippedArchiveFiles: created.Skipped}, err
}

func absOrEmpty(base, path string) string {
	if path == "" {
		return ""
	}
	return absPath(base, path)
}

type IncrementedInvoice struct {
	CustomerID          string
	OldNumber           string
	NewNumber           string
	SkippedArchiveFiles []string
}

func (h Host) IncrementInvoiceNumber(invoicePath, customersPath string, dryRun bool) (IncrementedInvoice, error) {
	incremented, err := h.service(Files{Customers: customersPath}, filepath.Dir(invoicePath), time.Time{}).Increment(invoicePath, dryRun)
	return IncrementedInvoice{CustomerID: incremented.CustomerID, OldNumber: incremented.OldNumber, NewNumber: incremented.NewNumber, SkippedArchiveFiles: incremented.Skipped}, err
}

func (h Host) NextInvoiceNumber(customerID, issueDate string, customer invoice.Customer, minimumCounter int64) (string, []string, error) {
	return h.service(Files{}, cwd(), time.Time{}).NextNumber(customerID, issueDate, customer, minimumCounter)
}

func (h Host) ResolveNumberingSettings() (numbering.Settings, error) {
	settings, err := h.Settings()
	if err != nil {
		return numbering.Settings{}, err
	}
	if err := settings.Numbering.Validate(); err != nil {
		return numbering.Settings{}, &wrapped{settings.File, err}
	}
	return settings.Numbering, nil
}

type wrapped struct {
	file string
	err  error
}

func (w *wrapped) Error() string { return w.file + ": " + w.err.Error() }
func (w *wrapped) Unwrap() error { return w.err }

func (h Host) ArchiveInvoice(now time.Time, invoicePath string, opts ArchiveOptions) (ArchiveResult, error) {
	return h.service(Files{}, filepath.Dir(invoicePath), now).Archive(invoicePath, opts)
}

type EditArchiveOptions = billing.EditOptions

func (h Host) EditArchivedInvoice(archiveName, workDir string, opts EditArchiveOptions) (string, string, error) {
	edited, err := h.service(Files{}, workDir, time.Time{}).EditArchived(archiveName, workDir, opts)
	return edited.Path, edited.Archived, err
}

func (h Host) CheckArchivedNumberUnique(invoicePath string) error {
	return h.service(Files{}, filepath.Dir(invoicePath), time.Time{}).CheckNumberUnique(invoicePath)
}

type ArchivedInvoiceSummary = billing.ArchiveEntry

func (h Host) ListArchivedInvoices() ([]ArchivedInvoiceSummary, error) {
	list, err := h.service(Files{}, cwd(), time.Time{}).ListArchive()
	return list.Entries, err
}

type CustomerSummary struct {
	ID               string
	LegalCompanyName string
	Status           string
	Email            string
	Currency         string
}

func ListCustomers(customersPath string) ([]CustomerSummary, error) {
	list, err := NewHost(HostInputs{GOOS: "linux"}).service(Files{Customers: customersPath}, cwd(), time.Time{}).ListCustomers()
	summaries := make([]CustomerSummary, 0, len(list.Customers))
	for _, c := range list.Customers {
		summaries = append(summaries, CustomerSummary{ID: c.ID, LegalCompanyName: c.Name, Status: c.Status, Email: c.Email, Currency: c.Currency})
	}
	return summaries, err
}

func (h Host) templateFor(path string) billing.Template {
	return (&Store{Host: h}).template(path)
}

// RenderTeX checks the template at templatePath and fills it in with ctx.
func RenderTeX(templatePath string, ctx *invoice.Context) (string, error) {
	return NewHost(HostInputs{GOOS: "linux"}).renderTeX(templatePath, ctx)
}

func (h Host) renderTeX(templatePath string, ctx *invoice.Context) (string, error) {
	return latex.Renderer{}.Render(h.templateFor(templatePath), ctx, billing.EPCFor(ctx))
}

func (h Host) RenderInvoice(templatePath, outputPath string, ctx *invoice.Context) error {
	source, err := h.renderTeX(templatePath, ctx)
	if err != nil {
		return err
	}
	return latex.Renderer{}.Write(h.templateFor(templatePath), source, outputPath)
}

func (h Host) copyTemplateAssets(templatePath, outputPath, rendered string) error {
	return latex.Renderer{}.Write(h.templateFor(templatePath), rendered, outputPath)
}

// BuildInvoicePDF renders the invoice into a temporary directory, runs
// compile on the .tex file there, and copies the PDF to outputPath, as
// billing.Service.Build does.
func (h Host) BuildInvoicePDF(ctx context.Context, compile func(ctx context.Context, texPath string) error, templatePath, outputPath string, inv *invoice.Context) error {
	svc := h.service(Files{}, cwd(), time.Time{})
	svc.Compiler = compilerFunc(func(ctx context.Context, sourcePath string) (string, error) {
		if err := compile(ctx, sourcePath); err != nil {
			return "", err
		}
		return sourcePath[:len(sourcePath)-len(filepath.Ext(sourcePath))] + ".pdf", nil
	})
	source, err := h.renderTeX(templatePath, inv)
	if err != nil {
		return err
	}
	return svc.Renderer.Build(ctx, svc.Compiler, h.templateFor(templatePath), source, outputPath)
}

type compilerFunc func(ctx context.Context, sourcePath string) (string, error)

func (f compilerFunc) Compile(ctx context.Context, sourcePath string) (string, error) {
	return f(ctx, sourcePath)
}

func buildEPCPayload(ctx *invoice.Context) ([]byte, error) { return billing.EPCPayload(ctx) }

func epcQRCodeEligible(ctx *invoice.Context) bool {
	code := billing.EPCFor(ctx)
	return code.Payload != nil || code.Err != nil
}

func writeInvoiceNumber(path, invoiceNumber string) error {
	return (&Store{}).Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Number = invoice.Text(invoiceNumber)
		return nil
	})
}

func SetInvoiceStatus(invoicePath, status string) error {
	return writeInvoiceStringField(invoicePath, "status", status)
}

func writeInvoiceStringField(path, key, value string) error {
	if key != "status" {
		panic("writeInvoiceStringField: only status is supported")
	}
	return (&Store{}).Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Status = invoice.Text(value)
		return nil
	})
}

// MarkInvoiceBuilt sets invoice.status to built as a successful build does.
func MarkInvoiceBuilt(invoicePath string) error {
	head, err := (&Store{}).Head(invoicePath)
	if err != nil && !isDecodeError(err) {
		return err
	}
	if next, _ := head.Status.Apply(invoice.Building); next != invoice.Built {
		return nil
	}
	return SetInvoiceStatus(invoicePath, string(invoice.Built))
}

type EmailDraftPaths struct {
	InvoicePath string
	PDFPath     string
	OutputPath  string
}

// ResolveEmailDraftPaths derives the invoice, PDF and draft of an email for
// input as the email command does: the PDF and draft default to input's
// name, and the invoice of a PDF is looked up next to it or in the archive.
func (h Host) ResolveEmailDraftPaths(inputPath, pdfPath, outputPath string) (EmailDraftPaths, error) {
	paths := EmailDraftPaths{InvoicePath: inputPath, PDFPath: pdfPath, OutputPath: outputPath}
	if paths.OutputPath == "" {
		paths.OutputPath = replaceExt(inputPath, ".eml")
	}
	switch strings.ToLower(filepath.Ext(inputPath)) {
	case ".pdf":
		found, err := h.service(Files{}, cwd(), time.Time{}).Archives.Source(inputPath)
		if err != nil {
			return EmailDraftPaths{}, err
		}
		if found == "" {
			return EmailDraftPaths{}, fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", inputPath)
		}
		paths.InvoicePath = found
		if paths.PDFPath == "" {
			paths.PDFPath = inputPath
		}
	case ".yaml", ".yml":
		if paths.PDFPath == "" {
			paths.PDFPath = replaceExt(inputPath, ".pdf")
		}
	default:
		return EmailDraftPaths{}, fmt.Errorf("%s: input must end with .yaml, .yml, or .pdf", inputPath)
	}
	return paths, nil
}

func replaceExt(path, ext string) string {
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}

type EmailParams struct {
	CustomersPath string
	IssuerPath    string
	InvoicePath   string
	PDFPath       string
	Recipient     string
	Subject       string
}

// EmailMessage is the draft of an invoice email and the invoice it is for.
type EmailMessage struct {
	email.Draft
	CustomerID    string
	InvoiceNumber string
}

func (h Host) PrepareInvoiceEmail(p EmailParams) (EmailMessage, error) {
	svc := h.service(Files{Customers: p.CustomersPath, Issuer: p.IssuerPath}, cwd(), time.Time{})
	pdf := p.PDFPath
	if pdf == "" {
		pdf = replaceExt(p.InvoicePath, ".pdf")
	}
	result, err := svc.DraftEmail(context.Background(), billing.EmailRequest{
		Invoice: p.InvoicePath,
		PDF:     pdf,
		To:      p.Recipient,
		Subject: p.Subject,
		DryRun:  true,
	})
	if err != nil {
		return EmailMessage{}, err
	}
	m := result.Message
	return EmailMessage{
		Draft: email.Draft{
			Recipient:      m.To,
			Subject:        m.Subject,
			Body:           m.Body,
			SenderName:     m.FromName,
			SenderAddress:  m.FromAddress,
			AttachmentPath: m.Attachment,
		},
		CustomerID:    result.CustomerID,
		InvoiceNumber: result.Number,
	}, nil
}

type EmailDraftResult struct {
	OutputPath    string
	Recipient     string
	Subject       string
	CustomerID    string
	InvoiceNumber string
}

func (h Host) CreateInvoiceEmailDraft(now time.Time, m EmailMessage, outputPath string, overwrite bool) (EmailDraftResult, error) {
	_, err := email.Mailer{}.Draft(context.Background(), billing.Message{
		To:          m.Recipient,
		Subject:     m.Subject,
		Body:        m.Body,
		FromName:    m.SenderName,
		FromAddress: m.SenderAddress,
		Attachment:  m.AttachmentPath,
		Output:      outputPath,
		Overwrite:   overwrite,
		Date:        now,
	})
	if err != nil {
		return EmailDraftResult{}, err
	}
	return EmailDraftResult{OutputPath: outputPath, Recipient: m.Recipient, Subject: m.Subject, CustomerID: m.CustomerID, InvoiceNumber: m.InvoiceNumber}, nil
}

func CheckEmailDraftOutput(outputPath string, overwrite bool) error {
	return email.Mailer{}.Check(billing.Message{Output: outputPath, Overwrite: overwrite})
}
