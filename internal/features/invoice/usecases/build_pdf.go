package usecases

import (
	"context"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/ports"
)

// BuildPDF compiles inv into the PDF at outputPath and then marks the invoice
// at invoicePath as built.
type BuildPDF struct {
	hosts    *invoice.HostSource
	compiler ports.PDFCompiler
}

func NewBuildPDF(hosts *invoice.HostSource, compiler ports.PDFCompiler) BuildPDF {
	return BuildPDF{hosts: hosts, compiler: compiler}
}

// MarkBuiltError reports that the PDF was built but the invoice could not be
// marked as built.
type MarkBuiltError struct {
	Err error
}

func (e *MarkBuiltError) Error() string { return e.Err.Error() }
func (e *MarkBuiltError) Unwrap() error { return e.Err }

func (u BuildPDF) Run(ctx context.Context, templatePath, outputPath, invoicePath string, inv *invoice.Context) error {
	h := u.hosts.Host()
	if err := h.BuildInvoicePDF(ctx, u.compiler.Build, templatePath, outputPath, inv); err != nil {
		return err
	}
	if err := invoice.MarkInvoiceBuilt(h.Files(), invoicePath); err != nil {
		return &MarkBuiltError{Err: err}
	}
	return nil
}
