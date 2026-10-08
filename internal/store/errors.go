package store

import "fmt"

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
