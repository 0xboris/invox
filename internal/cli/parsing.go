package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"invox/internal/invoice"
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

func parseCommand(spec commandSpec, args []string) (invoice.Options, []string, int, bool) {
	if wantsHelp(args) {
		printCommandHelp(os.Stdout, spec)
		return invoice.Options{}, nil, 0, false
	}

	opts, err := invoice.DefaultOptions()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return invoice.Options{}, nil, 1, false
	}
	opts.InvoicePath = ""
	opts.OutputPath = ""

	fs := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	bindCommandFlags(fs, &opts, spec)
	if err := fs.Parse(args); err != nil {
		printCommandError(os.Stderr, spec, err.Error())
		return invoice.Options{}, nil, 2, false
	}

	remainingArgs := fs.Args()
	if spec.AcceptsPositionalInput && strings.TrimSpace(opts.InvoicePath) == "" && len(remainingArgs) > 0 {
		opts.InvoicePath = remainingArgs[0]
		remainingArgs = remainingArgs[1:]
	}
	if err := validatePositionalArgs(spec, remainingArgs); err != nil {
		printCommandError(os.Stderr, spec, err.Error())
		return invoice.Options{}, nil, 2, false
	}
	if spec.InputBasedOutput && strings.TrimSpace(opts.OutputPath) == "" {
		opts.OutputPath = replacePathExtension(opts.InvoicePath, spec.OutputExtension)
	}
	if spec.DefaultOutput != "" && !spec.DynamicDefaultOutput && !spec.InputBasedOutput && strings.TrimSpace(opts.OutputPath) == "" {
		opts.OutputPath = filepath.Join(opts.BaseDir, spec.DefaultOutput)
	}

	if err := validateRequiredInputs(spec, opts); err != nil {
		printCommandError(os.Stderr, spec, err.Error())
		return invoice.Options{}, nil, 2, false
	}
	if err := validateSupportPaths(spec, opts); err != nil {
		printCommandError(os.Stderr, spec, err.Error())
		return invoice.Options{}, nil, 2, false
	}
	if err := validateCommandOptions(spec, opts); err != nil {
		printCommandError(os.Stderr, spec, err.Error())
		return invoice.Options{}, nil, 2, false
	}
	if err := invoice.NormalizeOptions(&opts); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return invoice.Options{}, nil, 1, false
	}

	return opts, remainingArgs, 0, true
}

func bindCommandFlags(fs *flag.FlagSet, opts *invoice.Options, spec commandSpec) {
	if spec.RequiresInput {
		fs.StringVar(&opts.InvoicePath, "i", opts.InvoicePath, "input invoice YAML file")
		fs.StringVar(&opts.InvoicePath, "input", opts.InvoicePath, "input invoice YAML file")
	}
	if spec.NeedsCustomers {
		fs.StringVar(&opts.CustomersPath, "c", opts.CustomersPath, "path to customers.yaml")
		fs.StringVar(&opts.CustomersPath, "customers", opts.CustomersPath, "path to customers.yaml")
	}
	if spec.NeedsIssuer {
		fs.StringVar(&opts.IssuerPath, "u", opts.IssuerPath, "path to issuer.yaml")
		fs.StringVar(&opts.IssuerPath, "issuer", opts.IssuerPath, "path to issuer.yaml")
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
	if spec.SupportsArchiveFlag {
		fs.BoolVar(&opts.ArchiveAfterBuild, "archive", opts.ArchiveAfterBuild, "archive the invoice after a successful build")
	}
}

func validateRequiredInputs(spec commandSpec, opts invoice.Options) error {
	if spec.RequiresInput && strings.TrimSpace(opts.InvoicePath) == "" {
		if spec.AcceptsPositionalInput {
			return fmt.Errorf("missing required input: INVOICE.yaml or -i, --input")
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

func validateSupportPaths(spec commandSpec, opts invoice.Options) error {
	if spec.NeedsCustomers && strings.TrimSpace(opts.CustomersPath) == "" {
		return fmt.Errorf("customers file not found; pass -c/--customers, set paths.customers in config.yaml, or place customers.yaml at %s", invoice.GlobalCustomersPath())
	}
	if spec.NeedsIssuer && strings.TrimSpace(opts.IssuerPath) == "" {
		return fmt.Errorf("issuer file not found; pass -u/--issuer, set paths.issuer in config.yaml, or place issuer.yaml at %s", invoice.GlobalIssuerPath())
	}
	if spec.NeedsDefaults && !opts.FromLastInvoice && strings.TrimSpace(opts.DefaultsPath) == "" {
		return fmt.Errorf("defaults file not found; pass -s/--source, set paths.defaults in config.yaml, or place invoice_defaults.yaml at %s", invoice.GlobalInvoiceDefaultsPath())
	}
	if spec.NeedsTemplate && strings.TrimSpace(opts.TemplatePath) == "" {
		return fmt.Errorf("template file not found; pass -t/--template, set paths.template in config.yaml, or place template.tex at %s", invoice.GlobalTemplatePath())
	}
	return nil
}

func validateCommandOptions(spec commandSpec, opts invoice.Options) error {
	if spec.OutputExtension == "" || strings.TrimSpace(opts.OutputPath) == "" {
		return nil
	}
	ext := filepath.Ext(opts.OutputPath)
	if ext != spec.OutputExtension {
		return fmt.Errorf("-o, --output must end with %s", spec.OutputExtension)
	}
	return nil
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
