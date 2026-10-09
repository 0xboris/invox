package cmdutil

import (
	"errors"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
)

// SupportPaths are the support files a command line names, as typed; ""
// looks the file up.
type SupportPaths struct {
	Customers string
	Issuer    string
	Defaults  string
	// Template is a template path, or a name in the template directory.
	Template string
}

// Files returns the support files relative to cwd as absolute paths. The
// template is not one of them: a request takes it as typed.
func (p SupportPaths) Files(cwd string) Files {
	return Files{
		Customers: AbsFlag(cwd, p.Customers),
		Issuer:    AbsFlag(cwd, p.Issuer),
		Defaults:  AbsFlag(cwd, p.Defaults),
	}
}

// supportFlag is the flag that names a support file. key and file are its
// config key and file name, which UsageError names.
type supportFlag struct {
	name, shorthand, usage string
	// exts are the file extensions the flag completes. A flag without
	// them completes the names `template list` shows.
	exts  []string
	value func(*SupportPaths) *string
	key   string
	file  string
}

// supportFlags are the flags of the support files.
var supportFlags = map[billing.File]supportFlag{
	billing.CustomersFile: {
		name: "customers", shorthand: "c", usage: "Path to customers.yaml", exts: []string{"yaml", "yml"},
		value: func(p *SupportPaths) *string { return &p.Customers },
		key:   "paths.customers", file: "customers.yaml",
	},
	billing.IssuerFile: {
		name: "issuer", shorthand: "u", usage: "Path to issuer.yaml", exts: []string{"yaml", "yml"},
		value: func(p *SupportPaths) *string { return &p.Issuer },
		key:   "paths.issuer", file: "issuer.yaml",
	},
	billing.DefaultsFile: {
		name: "defaults", usage: "Path to invoice_defaults.yaml", exts: []string{"yaml", "yml"},
		value: func(p *SupportPaths) *string { return &p.Defaults },
		key:   "paths.defaults", file: "invoice_defaults.yaml",
	},
	billing.TemplateFile: {
		name: "template", shorthand: "t", usage: "Template path or name",
		value: func(p *SupportPaths) *string { return &p.Template },
		key:   "paths.template", file: "template.tex",
	},
}

// AddSupportFlags gives cmd the flag of each of files, which sets the file
// in paths, and its completion.
func AddSupportFlags(cmd *cobra.Command, f *Factory, paths *SupportPaths, files ...billing.File) {
	for _, file := range files {
		s := supportFlags[file]
		cmd.Flags().StringVarP(s.value(paths), s.name, s.shorthand, "", s.usage)
		if len(s.exts) > 0 {
			_ = cmd.MarkFlagFilename(s.name, s.exts...)
		} else {
			_ = cmd.RegisterFlagCompletionFunc(s.name, CompleteTemplates(f))
		}
	}
}

// flags writes the flag as UsageError names it: -c/--customers, or
// --defaults without a shorthand.
func (s supportFlag) flags() string {
	if s.shorthand == "" {
		return "--" + s.name
	}
	return "-" + s.shorthand + "/--" + s.name
}

// UsageError returns err as a usage error when it is about a file the
// command line can name: a support file that was not found, which
// names the ways to provide it, or a template reference that resolves to
// nothing. Any other err comes back as it is.
func UsageError(err error) error {
	var notFound *billing.FileNotFoundError
	if errors.As(err, &notFound) {
		s := supportFlags[notFound.File]
		return FlagErrorf("%s file not found; pass %s, set %s in config.yaml, or place %s at %s", notFound.File, s.flags(), s.key, s.file, notFound.Default)
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
