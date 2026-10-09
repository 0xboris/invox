package billing

import (
	"errors"
	"fmt"
	"strings"
)

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

// DecodeError is a value in a YAML file that does not fit the schema: an
// unknown key, a value of the wrong kind, or a malformed number or date.
type DecodeError struct {
	File string
	Line int
	// Path is the field, such as positions[2].unit_price. It is "" when
	// Problem names the field itself.
	Path    string
	Problem string
	// Field is the field that did not decode, which validation then
	// skips. It is "" for an unknown key.
	Field string
	// UnknownKey is set when the problem is a key the schema does not
	// define. Schema then names the kind of file: "customers", "issuer",
	// "invoice" or "config".
	UnknownKey bool
	Schema     string
}

// failedFields returns the Field of every *DecodeError in err.
func failedFields(err error) map[string]bool {
	failed := map[string]bool{}
	var walk func(error)
	walk = func(err error) {
		switch e := err.(type) {
		case *DecodeError:
			if e.Field != "" {
				failed[e.Field] = true
			}
		case interface{ Unwrap() []error }:
			for _, inner := range e.Unwrap() {
				walk(inner)
			}
		}
	}
	walk(err)
	return failed
}

// withoutUnknownKeys returns err, the decode problems of one entry, without
// the keys the schema does not define.
func withoutUnknownKeys(err error) error {
	problems := []error{err}
	if joined, ok := err.(interface{ Unwrap() []error }); ok {
		problems = joined.Unwrap()
	}
	var kept []error
	for _, problem := range problems {
		var decodeErr *DecodeError
		if !errors.As(problem, &decodeErr) || !decodeErr.UnknownKey {
			kept = append(kept, problem)
		}
	}
	return errors.Join(kept...)
}

// within reports whether path is one of fields or lies inside one of them.
func within(path string, fields map[string]bool) bool {
	for field := range fields {
		if path == field || strings.HasPrefix(path, field+".") || strings.HasPrefix(path, field+"[") {
			return true
		}
	}
	return false
}

func (e *DecodeError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Problem)
	}
	return fmt.Sprintf("%s:%d: %s: %s", e.File, e.Line, e.Path, e.Problem)
}

// ArchiveReplaceError reports that archiving the invoice would replace
// archived files and ArchiveOptions.Replace was not set. Nothing was written.
type ArchiveReplaceError struct {
	InvoicePath string
	// Paths are the archived files that would be replaced.
	Paths []string
	// HistoryDir is where their previous versions would be kept.
	HistoryDir string
}

func (e *ArchiveReplaceError) Error() string {
	return fmt.Sprintf("%s: archiving replaces archived invoice %s", e.InvoicePath, strings.Join(e.Paths, ", "))
}

// ConfigDirNotFoundError reports a config directory chosen by the user that
// does not exist.
type ConfigDirNotFoundError struct {
	Dir string
}

func (e *ConfigDirNotFoundError) Error() string {
	return fmt.Sprintf("config directory %s does not exist", e.Dir)
}

// ConfigError is a problem with config.yaml.
type ConfigError struct {
	Err error
}

func (e *ConfigError) Error() string { return e.Err.Error() }
func (e *ConfigError) Unwrap() error { return e.Err }

// ToolMissingError means a program invox runs is not installed. Hint says
// how to install it.
type ToolMissingError struct {
	Tool string
	Hint string
}

func (e *ToolMissingError) Error() string { return e.Tool + " not found in PATH" }

// FileNotFoundError means no support file of its kind was found. Default is
// where invox looks last, in the config directory.
type FileNotFoundError struct {
	File    File
	Default string
}

func (e *FileNotFoundError) Error() string {
	return fmt.Sprintf("%s file not found", e.File)
}

// TemplateLookupError is a template reference that could not be resolved:
// one no template has (*TemplateNotFoundError), an ambiguous name, or a
// failure to read the template catalog.
type TemplateLookupError struct {
	Err error
}

func (e *TemplateLookupError) Error() string { return e.Err.Error() }
func (e *TemplateLookupError) Unwrap() error { return e.Err }

// isDecodeError reports whether err holds a *DecodeError.
func isDecodeError(err error) bool {
	var decodeErr *DecodeError
	return errors.As(err, &decodeErr)
}

// lenient drops the unknown-key problems from err, the joined *DecodeError
// values of a strict decode, and returns what is left, or nil.
func lenient(err error) error {
	return keepDecodeErrors(err, func(e *DecodeError) bool { return !e.UnknownKey })
}

// keepDecodeErrors returns err without the *DecodeError values keep
// rejects, or nil when nothing is left.
func keepDecodeErrors(err error, keep func(*DecodeError) bool) error {
	var kept []error
	var walk func(error)
	walk = func(err error) {
		switch e := err.(type) {
		case nil:
		case *DecodeError:
			if keep(e) {
				kept = append(kept, e)
			}
		case interface{ Unwrap() []error }:
			for _, inner := range e.Unwrap() {
				walk(inner)
			}
		default:
			kept = append(kept, e)
		}
	}
	walk(err)
	return errors.Join(kept...)
}
