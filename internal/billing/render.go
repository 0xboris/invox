package billing

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
)

// RenderRequest says which invoice Render fills into which template.
type RenderRequest struct {
	Invoice string
	// Template is a path or a template name, "" for the default.
	Template string
	Output   string
	// DryRun checks the invoice and the template and writes nothing.
	DryRun bool
}

// Render writes the invoice's LaTeX source to req.Output.
func (s *Service) Render(req RenderRequest) (*invoice.Context, error) {
	t, ctx, source, err := s.renderSource(req.Invoice, req.Template)
	if err != nil {
		return nil, err
	}
	if req.DryRun {
		return ctx, nil
	}
	return ctx, s.Renderer.Write(t, source, req.Output)
}

func (s *Service) renderSource(path, ref string) (Template, *invoice.Context, string, error) {
	customersPath, issuerPath, err := s.locateParties()
	if err != nil {
		return Template{}, nil, "", err
	}
	t, err := s.Directory.Template(ref)
	if err != nil {
		return Template{}, nil, "", err
	}
	ctx, err := s.loadContext(customersPath, issuerPath, path)
	if err != nil {
		return Template{}, nil, "", err
	}
	source, err := s.Renderer.Render(t, ctx, EPCFor(ctx))
	if err != nil {
		return Template{}, nil, "", err
	}
	return t, ctx, source, nil
}

// BuildRequest says which invoice Build compiles, where to, and whether to
// archive it.
type BuildRequest struct {
	Invoice  string
	Template string
	Output   string
	// Archive archives the invoice after a successful build, with
	// ArchiveOptions Replace and Confirm.
	Archive bool
	Replace bool
	Confirm func(*ArchiveReplaceError) error
	// DryRun checks the invoice, the template and, with Archive, the
	// archive step, and changes nothing.
	DryRun bool
}

// BuildResult is a built invoice.
type BuildResult struct {
	Context *invoice.Context
	// Archived is set when the invoice was archived, or would be.
	Archived *ArchiveResult
}

// Step names what failed after the PDF was built.
type Step int

const (
	// StepMark is setting invoice.status to built.
	StepMark Step = iota
	// StepArchive is archiving the invoice.
	StepArchive
)

// StepError is a failure after the PDF was built, or, in a dry run, of the
// archive check.
type StepError struct {
	Step Step
	Err  error
}

func (e *StepError) Error() string { return e.Err.Error() }
func (e *StepError) Unwrap() error { return e.Err }

// Build renders the invoice, compiles it to req.Output, sets its status to
// built and, with req.Archive, archives it.
func (s *Service) Build(ctx context.Context, req BuildRequest) (BuildResult, error) {
	t, inv, source, err := s.renderSource(req.Invoice, req.Template)
	if err != nil {
		return BuildResult{}, err
	}
	result := BuildResult{Context: inv}
	if req.DryRun {
		if req.Archive {
			archived, err := s.Archive(req.Invoice, ArchiveOptions{Replace: true, DryRun: true, AssumeBuilt: true})
			if err != nil {
				return BuildResult{}, &StepError{Step: StepArchive, Err: err}
			}
			result.Archived = &archived
		}
		return result, nil
	}

	if err := s.compile(ctx, t, source, req.Output); err != nil {
		return BuildResult{}, err
	}
	if err := s.markBuilt(req.Invoice); err != nil {
		return BuildResult{}, &StepError{Step: StepMark, Err: err}
	}
	if req.Archive {
		archived, err := s.Archive(req.Invoice, ArchiveOptions{Replace: req.Replace, Confirm: req.Confirm})
		if err != nil {
			return BuildResult{}, &StepError{Step: StepArchive, Err: err}
		}
		result.Archived = &archived
	}
	return result, nil
}

// compile writes source into a scratch directory, compiles it there, and
// copies the PDF to output.
func (s *Service) compile(ctx context.Context, t Template, source, output string) error {
	dir, remove, err := s.Renderer.Scratch()
	if err != nil {
		return err
	}
	defer remove()

	sourcePath := filepath.Join(dir, strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))+".tex")
	if err := s.Renderer.Write(t, source, sourcePath); err != nil {
		return err
	}
	pdf, err := s.Compiler.Compile(ctx, sourcePath)
	if err != nil {
		return err
	}
	return s.Renderer.Copy(pdf, output)
}

// markBuilt sets invoice.status to built after a successful PDF build. An
// archived invoice keeps `archived`: rebuilding its PDF does not take it
// out of the archive.
func (s *Service) markBuilt(path string) error {
	head, err := s.Invoices.Head(path)
	if err != nil && !isDecodeError(err) {
		return err
	}
	if next, _ := head.Status.Apply(invoice.Building); next != invoice.Built {
		return nil
	}
	return s.Invoices.Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Status = invoice.Text(invoice.Built)
		return nil
	})
}
