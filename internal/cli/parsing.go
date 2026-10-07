package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
)

func reorderArgs(args []string, flagSpecs map[string]bool) []string {
	if len(args) <= 1 {
		return args
	}

	flags := make([]string, 0, len(args))
	positionals := make([]string, 0, len(args))
	for index := 0; index < len(args); index++ {
		arg := args[index]
		if takesValue, ok := flagSpecs[arg]; ok {
			flags = append(flags, arg)
			if takesValue && index+1 < len(args) {
				index++
				flags = append(flags, args[index])
			}
			continue
		}
		if hasInlineFlagValue(arg, flagSpecs) {
			flags = append(flags, arg)
			continue
		}
		positionals = append(positionals, arg)
	}
	return append(flags, positionals...)
}

func hasInlineFlagValue(arg string, flagSpecs map[string]bool) bool {
	if !strings.HasPrefix(arg, "-") {
		return false
	}
	for flagName := range flagSpecs {
		if strings.HasPrefix(arg, flagName+"=") {
			return true
		}
	}
	return false
}

// parseCommand parses args for spec into options and the remaining positional
// arguments. When args ask for help it prints the help and returns flag.ErrHelp.
func parseCommand(f *cmdutil.Factory, spec commandSpec, args []string) (invoice.Options, []string, error) {
	ios := f.IOStreams
	e := f.Env
	h := f.Host()
	if wantsHelp(args) {
		printCommandHelp(ios.Out, h, spec)
		return invoice.Options{}, nil, flag.ErrHelp
	}

	cwd, err := e.Getwd()
	if err != nil {
		return invoice.Options{}, nil, err
	}
	opts := invoice.Options{BaseDir: cwd}

	fs := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	bindCommandFlags(fs, &opts, spec)
	if err := fs.Parse(args); err != nil {
		return invoice.Options{}, nil, &cmdutil.FlagError{Command: spec.Name, Err: err}
	}

	remainingArgs := fs.Args()
	if spec.AcceptsPositionalInput && strings.TrimSpace(opts.InvoicePath) == "" && len(remainingArgs) > 0 {
		opts.InvoicePath = remainingArgs[0]
		remainingArgs = remainingArgs[1:]
	}
	if err := validatePositionalArgs(spec, remainingArgs); err != nil {
		return invoice.Options{}, nil, &cmdutil.FlagError{Command: spec.Name, Err: err}
	}
	if spec.InputBasedOutput && strings.TrimSpace(opts.OutputPath) == "" {
		opts.OutputPath = replacePathExtension(opts.InvoicePath, spec.OutputExtension)
	}
	if spec.NeedsPDF && strings.TrimSpace(opts.PDFPath) == "" {
		opts.PDFPath = replacePathExtension(opts.InvoicePath, ".pdf")
	}
	if spec.DefaultOutput != "" && !spec.DynamicDefaultOutput && !spec.InputBasedOutput && strings.TrimSpace(opts.OutputPath) == "" {
		opts.OutputPath = filepath.Join(opts.BaseDir, spec.DefaultOutput)
	}

	if err := validateRequiredInputs(spec, opts); err != nil {
		return invoice.Options{}, nil, &cmdutil.FlagError{Command: spec.Name, Err: err}
	}
	if err := resolveDefaultSupportPaths(h, spec, &opts); err != nil {
		return invoice.Options{}, nil, err
	}
	if err := validateSupportPaths(h, spec, opts); err != nil {
		return invoice.Options{}, nil, &cmdutil.FlagError{Command: spec.Name, Err: err}
	}
	if spec.NeedsTemplate && strings.TrimSpace(opts.TemplatePath) != "" {
		resolvedTemplatePath, err := h.ResolveTemplateReference(opts.BaseDir, opts.TemplatePath)
		var notFound *invoice.TemplateNotFoundError
		if errors.As(err, &notFound) {
			return invoice.Options{}, nil, cmdutil.FlagErrorf(spec.Name, "%s; run '%s template list' to see the templates", notFound, commandName)
		}
		if err != nil {
			return invoice.Options{}, nil, &cmdutil.FlagError{Command: spec.Name, Err: err}
		}
		opts.TemplatePath = resolvedTemplatePath
	}
	if err := validateCommandOptions(spec, opts); err != nil {
		return invoice.Options{}, nil, &cmdutil.FlagError{Command: spec.Name, Err: err}
	}
	invoice.NormalizeOptions(&opts)

	return opts, remainingArgs, nil
}

// resolveDefaultSupportPaths fills in the support files the command needs and
// no flag provided. It is the only step of argument parsing that reads
// config.yaml, so usage errors, and commands whose support files all came from
// flags, do not depend on a readable config.
func resolveDefaultSupportPaths(h invoice.Host, spec commandSpec, opts *invoice.Options) error {
	resolvers := []struct {
		needed  bool
		path    *string
		resolve func(string) (string, error)
	}{
		{spec.NeedsCustomers, &opts.CustomersPath, h.ResolveDefaultCustomersPath},
		{spec.NeedsIssuer, &opts.IssuerPath, h.ResolveDefaultIssuerPath},
		{spec.NeedsDefaults, &opts.DefaultsPath, h.ResolveDefaultInvoiceDefaultsPath},
		{spec.NeedsTemplate, &opts.TemplatePath, h.ResolveDefaultTemplatePath},
	}
	for _, r := range resolvers {
		if !r.needed || strings.TrimSpace(*r.path) != "" {
			continue
		}
		path, err := r.resolve(opts.BaseDir)
		if err != nil {
			return err
		}
		*r.path = path
	}
	return nil
}

func bindCommandFlags(fs *flag.FlagSet, opts *invoice.Options, spec commandSpec) {
	if spec.RequiresInput {
		description := "input invoice YAML file"
		if spec.AcceptsPDFInput {
			description = "input invoice YAML or PDF file"
		}
		fs.StringVar(&opts.InvoicePath, "i", opts.InvoicePath, description)
		fs.StringVar(&opts.InvoicePath, "input", opts.InvoicePath, description)
	}
	if spec.NeedsCustomers {
		fs.StringVar(&opts.CustomersPath, "c", opts.CustomersPath, "path to customers.yaml")
		fs.StringVar(&opts.CustomersPath, "customers", opts.CustomersPath, "path to customers.yaml")
	}
	if spec.NeedsIssuer {
		fs.StringVar(&opts.IssuerPath, "u", opts.IssuerPath, "path to issuer.yaml")
		fs.StringVar(&opts.IssuerPath, "issuer", opts.IssuerPath, "path to issuer.yaml")
	}
	if spec.NeedsPDF {
		fs.StringVar(&opts.PDFPath, "p", opts.PDFPath, "path to invoice PDF")
		fs.StringVar(&opts.PDFPath, "pdf", opts.PDFPath, "path to invoice PDF")
	}
	if spec.NeedsDefaults {
		fs.StringVar(&opts.DefaultsPath, "s", opts.DefaultsPath, "path to invoice_defaults.yaml")
		fs.StringVar(&opts.DefaultsPath, "source", opts.DefaultsPath, "path to invoice_defaults.yaml")
	}
	if spec.NeedsTemplate {
		fs.StringVar(&opts.TemplatePath, "t", opts.TemplatePath, "path to invoice_template.tex")
		fs.StringVar(&opts.TemplatePath, "template", opts.TemplatePath, "path to invoice_template.tex")
	}
	if spec.OutputExtension != "" {
		fs.StringVar(&opts.OutputPath, "o", opts.OutputPath, "output file path")
		fs.StringVar(&opts.OutputPath, "output", opts.OutputPath, "output file path")
	}
	if spec.SupportsFromLastFlag {
		fs.BoolVar(&opts.FromLastInvoice, "from-last", opts.FromLastInvoice, "use the latest archived invoice for this customer as the source document")
	}
	if spec.SupportsEditFlag {
		fs.BoolVar(&opts.EditNewInvoice, "e", opts.EditNewInvoice, "open the created invoice in your editor")
		fs.BoolVar(&opts.EditNewInvoice, "edit", opts.EditNewInvoice, "open the created invoice in your editor")
	}
	if spec.SupportsEmailToFlag {
		fs.StringVar(&opts.EmailTo, "to", opts.EmailTo, "recipient email override")
	}
	if spec.SupportsSubjectFlag {
		fs.StringVar(&opts.EmailSubject, "subject", opts.EmailSubject, "email subject override")
	}
	if spec.SupportsForceFlag {
		fs.BoolVar(&opts.OverwriteOutput, "force", opts.OverwriteOutput, "overwrite an existing output file")
	}
	if spec.SupportsArchiveFlag {
		fs.BoolVar(&opts.ArchiveAfterBuild, "archive", opts.ArchiveAfterBuild, "archive the invoice after a successful build")
	}
	if spec.SupportsYesFlag {
		fs.BoolVar(&opts.AssumeYes, "yes", opts.AssumeYes, "replace an archived invoice without asking")
	}
}

func validateRequiredInputs(spec commandSpec, opts invoice.Options) error {
	if spec.RequiresInput && strings.TrimSpace(opts.InvoicePath) == "" {
		if spec.AcceptsPositionalInput {
			return fmt.Errorf("missing required input: %s", requiredInputSynopsis(spec))
		}
		return fmt.Errorf("missing required flags: -i, --input")
	}
	return nil
}

func validatePositionalArgs(spec commandSpec, args []string) error {
	if len(args) < len(spec.RequiredArgs) {
		missing := strings.Join(spec.RequiredArgs[len(args):], ", ")
		return fmt.Errorf("missing required arguments: %s", missing)
	}
	if len(args) > len(spec.RequiredArgs) {
		return fmt.Errorf("unexpected arguments: %s", strings.Join(args[len(spec.RequiredArgs):], " "))
	}
	return nil
}

func validateSupportPaths(h invoice.Host, spec commandSpec, opts invoice.Options) error {
	if spec.NeedsCustomers && strings.TrimSpace(opts.CustomersPath) == "" {
		return fmt.Errorf("customers file not found; pass -c/--customers, set paths.customers in config.yaml, or place customers.yaml at %s", h.GlobalCustomersPath())
	}
	if spec.NeedsIssuer && strings.TrimSpace(opts.IssuerPath) == "" {
		return fmt.Errorf("issuer file not found; pass -u/--issuer, set paths.issuer in config.yaml, or place issuer.yaml at %s", h.GlobalIssuerPath())
	}
	if spec.NeedsDefaults && !opts.FromLastInvoice && strings.TrimSpace(opts.DefaultsPath) == "" {
		return fmt.Errorf("defaults file not found; pass -s/--source, set paths.defaults in config.yaml, or place invoice_defaults.yaml at %s", h.GlobalInvoiceDefaultsPath())
	}
	if spec.NeedsTemplate && strings.TrimSpace(opts.TemplatePath) == "" {
		return fmt.Errorf("template file not found; pass -t/--template, set paths.template in config.yaml, or place template.tex at %s", h.GlobalTemplatePath())
	}
	return nil
}

func validateCommandOptions(spec commandSpec, opts invoice.Options) error {
	if spec.OutputExtension == "" || strings.TrimSpace(opts.OutputPath) == "" {
	} else {
		ext := filepath.Ext(opts.OutputPath)
		if ext != spec.OutputExtension {
			return fmt.Errorf("-o, --output must end with %s", spec.OutputExtension)
		}
	}
	if spec.AcceptsPDFInput && strings.TrimSpace(opts.InvoicePath) != "" {
		switch strings.ToLower(filepath.Ext(opts.InvoicePath)) {
		case ".yaml", ".yml", ".pdf":
		default:
			return fmt.Errorf("input must end with .yaml, .yml, or .pdf")
		}
	}
	if spec.NeedsPDF && strings.TrimSpace(opts.PDFPath) != "" && filepath.Ext(opts.PDFPath) != ".pdf" {
		return fmt.Errorf("-p, --pdf must end with .pdf")
	}
	return nil
}

func requiredInputSynopsis(spec commandSpec) string {
	if spec.AcceptsPDFInput {
		return "INVOICE.yaml, INVOICE.pdf, or -i, --input"
	}
	return "INVOICE.yaml or -i, --input"
}

func replacePathExtension(path, ext string) string {
	if strings.TrimSpace(path) == "" || strings.TrimSpace(ext) == "" {
		return ""
	}
	currentExt := filepath.Ext(path)
	if currentExt == "" {
		return path + ext
	}
	return strings.TrimSuffix(path, currentExt) + ext
}
