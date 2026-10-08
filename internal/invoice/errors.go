package invoice

import (
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/money"
)

// UnknownCustomerError reports a customer_id that customers.yaml does not
// define. Path is the file the error is about: the invoice that names the
// customer_id, or customers.yaml when the ID came from the command line.
type UnknownCustomerError struct {
	Path       string
	CustomerID string
}

func (e *UnknownCustomerError) Error() string {
	return fmt.Sprintf("%s: unknown customer_id `%s`", e.Path, e.CustomerID)
}

// OutputExistsError reports that a command would overwrite an existing file.
type OutputExistsError struct {
	Path string
}

func (e *OutputExistsError) Error() string {
	return e.Path + " already exists"
}

// OutputIsDirError reports that a command's output path is a directory,
// which is never replaced, even when overwriting is allowed.
type OutputIsDirError struct {
	Path string
}

func (e *OutputIsDirError) Error() string {
	return e.Path + " is a directory"
}

// ArchivedOutputError reports that a command would overwrite a file in the
// archive directory.
type ArchivedOutputError struct {
	Path string
}

func (e *ArchivedOutputError) Error() string {
	return e.Path + " is in the archive directory and is never overwritten; archived invoices change only by re-archiving an edited copy"
}

// TemplateNotFoundError reports a template name that no known template has.
type TemplateNotFoundError struct {
	Name string
}

func (e *TemplateNotFoundError) Error() string {
	return fmt.Sprintf("template %q not found", e.Name)
}

// errAmountTooLarge reports an amount above money.MaxCents. subject names the
// amount, such as "invoice.paid_amount:" or "invoice total".
func errAmountTooLarge(subject string) error {
	return fmt.Errorf("%s exceeds the maximum amount of `%s`", subject, money.FormatCents(money.MaxCents))
}

// Problem is a field of an invoice, or of the customer or issuer it uses,
// that is missing or out of range.
type Problem struct {
	// File is set when the problem concerns the file as a whole, such as a
	// missing mapping, and is then named in place of Field.
	File string
	// Field is the field, such as positions[1].quantity.
	Field   string
	Message string
}

func (p Problem) String() string {
	if p.File != "" {
		return p.File + ": " + p.Message
	}
	return p.Field + ": " + p.Message
}

// ValidationError lists the problems LoadContext found once the files
// decoded, one per line.
type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	lines := make([]string, 0, len(e.Problems))
	for _, problem := range e.Problems {
		lines = append(lines, problem.String())
	}
	return strings.Join(lines, "\n")
}
