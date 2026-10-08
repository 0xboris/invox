package cmdutil

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/store"
)

// supportFlags names, for each support file, the flag that sets it, its
// config key, its file name and where invox looks for it by default.
var supportFlags = map[store.SupportFile]struct {
	label, flag, key, name string
	global                 func(store.Host) string
}{
	store.Customers: {"customers", "-c/--customers", "paths.customers", "customers.yaml", store.Host.GlobalCustomersPath},
	store.Issuer:    {"issuer", "-u/--issuer", "paths.issuer", "issuer.yaml", store.Host.GlobalIssuerPath},
	store.Defaults:  {"defaults", "--defaults", "paths.defaults", "invoice_defaults.yaml", store.Host.GlobalInvoiceDefaultsPath},
	store.Template:  {"template", "-t/--template", "paths.template", "template.tex", store.Host.GlobalTemplatePath},
}

// SupportPath returns the absolute path of the support file kind that
// command reads: flagValue made absolute against baseDir when it is set,
// else the file found from baseDir. Finding none is a usage error that names
// the ways to provide it.
func SupportPath(h store.Host, command string, kind store.SupportFile, flagValue, baseDir string) (string, error) {
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
	return store.AbsPath(filepath.Clean(baseDir), path), nil
}

// TemplatePath returns the absolute path of the template that command
// renders with: flagValue, a path or the name of a template in the catalog,
// when it is set, else the template found from baseDir.
func TemplatePath(h store.Host, command, flagValue, baseDir string) (string, error) {
	reference := flagValue
	if strings.TrimSpace(reference) == "" {
		found, err := SupportPath(h, command, store.Template, "", baseDir)
		if err != nil {
			return "", err
		}
		reference = found
	}
	path, err := h.ResolveTemplateReference(baseDir, reference)
	var notFound *store.TemplateNotFoundError
	if errors.As(err, &notFound) {
		return "", FlagErrorf(command, "%s; run 'invox template list' to see the templates", notFound)
	}
	if err != nil {
		return "", &FlagError{Command: command, Err: err}
	}
	return store.AbsPath(filepath.Clean(baseDir), path), nil
}
