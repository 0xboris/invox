package cmdutil

import (
	"errors"

	"github.com/0xboris/invox/internal/billing"
)

// supportFlags names, for each support file, the flag that sets it, its
// config key and its file name.
var supportFlags = map[billing.File]struct{ flag, key, name string }{
	billing.CustomersFile: {"-c/--customers", "paths.customers", "customers.yaml"},
	billing.IssuerFile:    {"-u/--issuer", "paths.issuer", "issuer.yaml"},
	billing.DefaultsFile:  {"--defaults", "paths.defaults", "invoice_defaults.yaml"},
	billing.TemplateFile:  {"-t/--template", "paths.template", "template.tex"},
}

// UsageError returns err as a usage error when it is about a file the
// command line can name: a support file that was not found, which
// names the ways to provide it, or a template reference that resolves to
// nothing. Any other err comes back as it is.
func UsageError(err error) error {
	var notFound *billing.FileNotFoundError
	if errors.As(err, &notFound) {
		s := supportFlags[notFound.File]
		return FlagErrorf("%s file not found; pass %s, set %s in config.yaml, or place %s at %s", notFound.File, s.flag, s.key, s.name, notFound.Default)
	}
	var lookup *billing.TemplateLookupError
	if !errors.As(err, &lookup) {
		return err
	}
	var missing *billing.TemplateNotFoundError
	if errors.As(lookup.Err, &missing) {
		return FlagErrorf("%s; run 'invox template list' to see the templates", missing)
	}
	return &FlagError{Err: lookup.Err}
}
