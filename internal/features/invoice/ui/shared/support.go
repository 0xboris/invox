package shared

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

// supportFlags names the flag that sets each support file.
var supportFlags = map[invoice.SupportFile]string{
	invoice.Customers: "-c/--customers",
	invoice.Issuer:    "-u/--issuer",
	invoice.Defaults:  "--defaults",
	invoice.Template:  "-t/--template",
}

// SupportPath returns the absolute path of the support file kind that
// command reads: flagValue made absolute against baseDir when it is set,
// else the file found from baseDir. Finding none is a usage error that names
// the ways to provide it.
func SupportPath(h invoice.Host, command string, kind invoice.SupportFile, flagValue, baseDir string) (string, error) {
	path, err := h.SupportPath(kind, flagValue, baseDir)
	var notFound *invoice.SupportFileNotFoundError
	if errors.As(err, &notFound) {
		return "", cmdutil.FlagErrorf(command, "%s; pass %s, %s", notFound, supportFlags[kind], notFound.Remedy())
	}
	return path, err
}

// TemplatePath returns the absolute path of the template that command
// renders with: flagValue, a path or the name of a template in the catalog,
// when it is set, else the template found from baseDir.
func TemplatePath(h invoice.Host, command, flagValue, baseDir string) (string, error) {
	reference := flagValue
	if strings.TrimSpace(reference) == "" {
		found, err := SupportPath(h, command, invoice.Template, "", baseDir)
		if err != nil {
			return "", err
		}
		reference = found
	}
	path, err := h.ResolveTemplateReference(baseDir, reference)
	var notFound *invoice.TemplateNotFoundError
	if errors.As(err, &notFound) {
		return "", cmdutil.FlagErrorf(command, "%s; run 'invox template list' to see the templates", notFound)
	}
	if err != nil {
		return "", &cmdutil.FlagError{Command: command, Err: err}
	}
	return invoice.AbsPath(filepath.Clean(baseDir), path), nil
}
