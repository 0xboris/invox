package invoice

import (
	"errors"
	"fmt"
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

// TemplateNotFoundError reports a template name that no known template has.
type TemplateNotFoundError struct {
	Name string
}

func (e *TemplateNotFoundError) Error() string {
	return fmt.Sprintf("template %q not found", e.Name)
}

// ErrTectonicNotFound means the tectonic program is not on PATH.
var ErrTectonicNotFound = errors.New("tectonic not found in PATH")
