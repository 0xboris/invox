package cmdutil

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
)

// supportFlags names, for each support file, the flag that sets it, its
// config key, its file name and where invox looks for it by default.
var supportFlags = map[invoice.SupportFile]struct {
	label, flag, key, name string
	global                 func(invoice.Host) string
}{
	invoice.Customers: {"customers", "-c/--customers", "paths.customers", "customers.yaml", invoice.Host.GlobalCustomersPath},
	invoice.Issuer:    {"issuer", "-u/--issuer", "paths.issuer", "issuer.yaml", invoice.Host.GlobalIssuerPath},
	invoice.Defaults:  {"defaults", "-s/--source", "paths.defaults", "invoice_defaults.yaml", invoice.Host.GlobalInvoiceDefaultsPath},
	invoice.Template:  {"template", "-t/--template", "paths.template", "template.tex", invoice.Host.GlobalTemplatePath},
}

// SupportPath returns the absolute path of the support file kind that
// command reads: flagValue made absolute against baseDir when it is set,
// else the file found from baseDir. Finding none is a usage error that names
// the ways to provide it.
func SupportPath(h invoice.Host, command string, kind invoice.SupportFile, flagValue, baseDir string) (string, error) {
	path := flagValue
	if strings.TrimSpace(path) == "" {
		found, err := h.ResolveSupportFile(kind, baseDir)
		if err != nil {
			return "", err
		}
		path = found.Path
	}
	if strings.TrimSpace(path) == "" {
		s := supportFlags[kind]
		return "", FlagErrorf(command, "%s file not found; pass %s, set %s in config.yaml, or place %s at %s", s.label, s.flag, s.key, s.name, s.global(h))
	}
	return invoice.AbsPath(filepath.Clean(baseDir), path), nil
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
		return "", FlagErrorf(command, "%s; run 'invox template list' to see the templates", notFound)
	}
	if err != nil {
		return "", &FlagError{Command: command, Err: err}
	}
	return invoice.AbsPath(filepath.Clean(baseDir), path), nil
}
