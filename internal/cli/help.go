package cli

import (
	"fmt"
	"io"

	"invox/internal/invoice"
)

func printRootHelp(w io.Writer) {
	fmt.Fprintf(w, "%s generates LaTeX and PDF invoices from YAML data.\n\n", commandName)
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s <subcommand> [options]\n", commandName)
	fmt.Fprintf(w, "  %s help [subcommand]\n\n", commandName)
	fmt.Fprintf(w, "Subcommands:\n")
	fmt.Fprintf(w, "  customer       Customer-related commands\n")
	fmt.Fprintf(w, "  config         Open config.yaml in the default shell editor\n")
	fmt.Fprintf(w, "  init           Create starter support files in the global config directory\n")
	fmt.Fprintf(w, "  new            Create a new invoice YAML file with a generated number\n")
	fmt.Fprintf(w, "  increment      Increment the invoice number in an existing invoice YAML file\n")
	fmt.Fprintf(w, "  validate       Validate invoice YAML against customers and issuer data\n")
	fmt.Fprintf(w, "  render         Render a LaTeX invoice file\n")
	fmt.Fprintf(w, "  build          Render and compile an invoice PDF with Tectonic\n")
	fmt.Fprintf(w, "  archive        Archive invoices and manage archived invoices\n\n")
	fmt.Fprintf(w, "Required inputs by command:\n")
	fmt.Fprintf(w, "  new        CUSTOMER_ID\n")
	fmt.Fprintf(w, "  increment  -i, --input\n")
	fmt.Fprintf(w, "  validate   -i, --input\n")
	fmt.Fprintf(w, "  render     -i, --input\n")
	fmt.Fprintf(w, "  build      INVOICE.yaml or -i, --input\n")
	fmt.Fprintf(w, "  archive    INVOICE.yaml or -i, --input\n\n")
	fmt.Fprintf(w, "Optional flags:\n")
	fmt.Fprintf(w, "  -h, --help              Show help\n")
	fmt.Fprintf(w, "  -c, --customers PATH    Path to customers.yaml\n")
	fmt.Fprintf(w, "  -o, --output PATH       Output file path (defaults vary by command)\n")
	fmt.Fprintf(w, "  --archive               Archive after a successful PDF build (build)\n")
	fmt.Fprintf(w, "  --from-last             Use the latest archived invoice for CUSTOMER_ID (new)\n")
	fmt.Fprintf(w, "  -s, --source PATH       Path to invoice_defaults.yaml (new)\n")
	fmt.Fprintf(w, "  -u, --issuer PATH       Path to issuer.yaml\n")
	fmt.Fprintf(w, "  -t, --template PATH     Path to invoice_template.tex (render/build)\n\n")
	fmt.Fprintf(w, "Defaults:\n")
	fmt.Fprintf(w, "  customers.yaml: upward project search, then %s\n", invoice.GlobalCustomersPath())
	fmt.Fprintf(w, "  issuer.yaml: upward project search, then %s\n", invoice.GlobalIssuerPath())
	fmt.Fprintf(w, "  invoice_defaults.yaml: upward project search, then %s\n", invoice.GlobalInvoiceDefaultsPath())
	fmt.Fprintf(w, "  template.tex: upward project search, then %s\n", invoice.GlobalTemplatePath())
	fmt.Fprintf(w, "  new output: ./<invoice.number>.yaml\n")
	fmt.Fprintf(w, "  render output: ./invoice.tex\n")
	fmt.Fprintf(w, "  build output: input path with .pdf extension\n\n")
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("customer list"))
	fmt.Fprintf(w, "  %s\n", commandExample("config"))
	fmt.Fprintf(w, "  %s\n", commandExample("init"))
	fmt.Fprintf(w, "  %s\n", commandExample("new CUST-001"))
	fmt.Fprintf(w, "  %s\n", commandExample("new CUST-001 --from-last"))
	fmt.Fprintf(w, "  %s\n", commandExample("new CUST-001 -u issuer.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("increment -i invoice.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("validate -i invoice.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("render -i invoice.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("build invoice.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("archive invoice.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("archive edit 2026-03-06.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("archive list"))
}

func printCustomerHelp(w io.Writer) {
	fmt.Fprintf(w, "Customer-related commands.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s customer <subcommand> [options]\n", commandName)
	fmt.Fprintf(w, "  %s help customer [subcommand]\n\n", commandName)
	fmt.Fprintf(w, "Subcommands:\n")
	fmt.Fprintf(w, "  list          List all customers\n")
	fmt.Fprintf(w, "  config        Open customers.yaml in the default shell editor\n")
	fmt.Fprintf(w, "\n")
	fmt.Fprintf(w, "Optional flags:\n")
	fmt.Fprintf(w, "  -h, --help              Show this help page\n")
	fmt.Fprintf(w, "  -c, --customers PATH    Path to customers.yaml\n\n")
	fmt.Fprintf(w, "Default lookup:\n")
	fmt.Fprintf(w, "  customers.yaml: upward project search, then %s\n\n", invoice.GlobalCustomersPath())
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("customer list"))
	fmt.Fprintf(w, "  %s\n", commandExample("customer list -c customers.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("customer config"))
}

func printCommandHelp(w io.Writer, spec commandSpec) {
	if spec.Name == "config" {
		printConfigHelp(w)
		return
	}
	if spec.Name == "init" {
		printInitHelp(w)
		return
	}

	fmt.Fprintf(w, "%s\n\n", spec.Summary)
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s %s\n\n", commandName, spec.Usage)
	fmt.Fprintf(w, "Required inputs:\n")
	if spec.RequiresInput {
		if spec.AcceptsPositionalInput {
			fmt.Fprintf(w, "  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file\n")
		} else {
			fmt.Fprintf(w, "  -i, --input PATH        Path to the invoice YAML file\n")
		}
	}
	for _, arg := range spec.RequiredArgs {
		fmt.Fprintf(w, "  %s                  Required positional argument\n", arg)
	}
	if description := defaultOutputDescription(spec); description != "" {
		fmt.Fprintf(w, "\nDefault output:\n")
		fmt.Fprintf(w, "  %s\n", description)
	}
	fmt.Fprintf(w, "\nOptional flags:\n")
	fmt.Fprintf(w, "  -h, --help              Show this help page\n")
	if spec.NeedsCustomers {
		fmt.Fprintf(w, "  -c, --customers PATH    Path to customers.yaml\n")
	}
	if spec.NeedsIssuer {
		fmt.Fprintf(w, "  -u, --issuer PATH       Path to issuer.yaml\n")
	}
	if spec.NeedsDefaults {
		fmt.Fprintf(w, "  -s, --source PATH       Path to invoice_defaults.yaml\n")
	}
	if spec.OutputExtension != "" {
		fmt.Fprintf(w, "  -o, --output PATH       Output file path (must end with %s)\n", spec.OutputExtension)
	}
	if spec.SupportsFromLastFlag {
		fmt.Fprintf(w, "  --from-last             Use the latest archived invoice for CUSTOMER_ID as the source document\n")
	}
	if spec.SupportsArchiveFlag {
		fmt.Fprintf(w, "  --archive               Archive the invoice after a successful PDF build\n")
	}
	if spec.NeedsTemplate {
		fmt.Fprintf(w, "  -t, --template PATH     Path to invoice_template.tex\n")
	}
	fmt.Fprintf(w, "\nDefault lookup:\n")
	if spec.NeedsCustomers {
		fmt.Fprintf(w, "  customers.yaml: upward project search, then %s\n", invoice.GlobalCustomersPath())
	}
	if spec.NeedsIssuer {
		fmt.Fprintf(w, "  issuer.yaml: upward project search, then %s\n", invoice.GlobalIssuerPath())
	}
	if spec.NeedsDefaults {
		fmt.Fprintf(w, "  invoice_defaults.yaml: upward project search, then %s\n", invoice.GlobalInvoiceDefaultsPath())
	}
	if spec.NeedsTemplate {
		fmt.Fprintf(w, "  template.tex: upward project search, then %s\n", invoice.GlobalTemplatePath())
	}
	if spec.Name == "archive" || spec.Name == "archive edit" || spec.Name == "archive list" || spec.SupportsFromLastFlag {
		fmt.Fprintf(w, "  archive.dir: config.yaml, then %s\n", invoice.DefaultArchiveDir())
	}
	if spec.Name == "customer config" {
		fmt.Fprintf(w, "\nCommon customer fields:\n")
		fmt.Fprintf(w, "  <customer>.name             Preferred customer name\n")
		fmt.Fprintf(w, "  <customer>.email            Preferred invoice email\n")
		fmt.Fprintf(w, "  <customer>.status           Optional status shown by customer list\n")
		fmt.Fprintf(w, "  <customer>.address.*        Billing address used for rendering\n")
		fmt.Fprintf(w, "  <customer>.tax.vat_tax_id   VAT number shown on the invoice\n")
		fmt.Fprintf(w, "  <customer>.billing.currency Optional currency, defaults to EUR\n")
		fmt.Fprintf(w, "\nOptional numbering fields:\n")
		fmt.Fprintf(w, "  <customer>.numbering.code   Value used by {customer_code}\n")
		fmt.Fprintf(w, "  <customer>.numbering.start  Override numbering.start for this customer\n")
	}
	if spec.Name == "archive list" {
		fmt.Fprintf(w, "\nOutput:\n")
		fmt.Fprintf(w, "  One archived invoice per line as FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS\n")
	}
	if spec.Name == "archive edit" {
		fmt.Fprintf(w, "\nBehavior:\n")
		fmt.Fprintf(w, "  Copies the archived invoice from archive.dir into the current directory.\n")
		fmt.Fprintf(w, "  The working copy is written as YAML with invoice.status set to editing.\n")
		fmt.Fprintf(w, "  Re-running %s archive on that working copy replaces the archived invoice.\n", commandName)
	}
	fmt.Fprintf(w, "\nExamples:\n")
	for _, example := range spec.Examples {
		fmt.Fprintf(w, "  %s\n", example)
	}
}

func defaultOutputDescription(spec commandSpec) string {
	if spec.DynamicDefaultOutput {
		return "<invoice.number>" + spec.OutputExtension + " in the current directory"
	}
	if spec.InputBasedOutput {
		return "the input path with " + spec.OutputExtension + " extension"
	}
	if spec.DefaultOutput == "" {
		return ""
	}
	return spec.DefaultOutput + " in the current directory"
}

func printConfigHelp(w io.Writer) {
	fmt.Fprintf(w, "Open config.yaml in the default shell editor.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s config\n", commandName)
	fmt.Fprintf(w, "  %s help config\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Opens the resolved config.yaml in the default shell editor.\n")
	fmt.Fprintf(w, "  If config.yaml does not exist yet, creates it from the template below.\n")
	fmt.Fprintf(w, "  Existing config.yaml files are left unchanged.\n\n")
	fmt.Fprintf(w, "Config paths:\n")
	fmt.Fprintf(w, "  preferred: %s\n", invoice.GlobalConfigPath())
	fmt.Fprintf(w, "  fallback read path: ~/.config/invoice-tool/config.yaml\n\n")
	fmt.Fprintf(w, "Formatting:\n")
	fmt.Fprintf(w, "  Top-level keys must start at column 1 with no leading spaces.\n\n")
	fmt.Fprintf(w, "Supported settings:\n")
	fmt.Fprintf(w, "  paths.customers    Override the default customers.yaml lookup path\n")
	fmt.Fprintf(w, "  paths.issuer       Override the default issuer.yaml lookup path\n")
	fmt.Fprintf(w, "  paths.defaults     Override the default invoice_defaults.yaml lookup path\n")
	fmt.Fprintf(w, "  paths.template     Override the default template.tex lookup path\n")
	fmt.Fprintf(w, "  numbering.pattern  Override the invoice-number pattern\n")
	fmt.Fprintf(w, "  numbering.start    Global starting counter when no archived invoice matches\n")
	fmt.Fprintf(w, "  archive.dir        Override the archive directory for archived invoice files\n\n")
	fmt.Fprintf(w, "Customer overrides:\n")
	fmt.Fprintf(w, "  customers.<CUSTOMER_ID>.numbering.start  Override numbering.start for one customer\n\n")
	fmt.Fprintf(w, "Support file precedence:\n")
	fmt.Fprintf(w, "  1. explicit CLI flag\n")
	fmt.Fprintf(w, "  2. upward project search\n")
	fmt.Fprintf(w, "  3. paths.* in config.yaml\n")
	fmt.Fprintf(w, "  4. conventional files in %s\n\n", invoice.ConfigDir())
	fmt.Fprintf(w, "Template:\n")
	fmt.Fprint(w, invoice.ConfigTemplate())
	fmt.Fprintf(w, "\nExamples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("config"))
	fmt.Fprintf(w, "  %s\n", commandExample("help config"))
}

func printInitHelp(w io.Writer) {
	fmt.Fprintf(w, "Create starter support files in the global config directory.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s init\n", commandName)
	fmt.Fprintf(w, "  %s help init\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Creates the global config directory if it does not exist yet.\n")
	fmt.Fprintf(w, "  Writes starter versions of config.yaml, customers.yaml, issuer.yaml,\n")
	fmt.Fprintf(w, "  invoice_defaults.yaml, and template.tex.\n")
	fmt.Fprintf(w, "  Existing non-empty files are left unchanged.\n\n")
	fmt.Fprintf(w, "Config directory:\n")
	fmt.Fprintf(w, "  %s\n\n", invoice.ConfigDir())
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("init"))
}

func printCommandError(w io.Writer, spec commandSpec, message string) {
	fmt.Fprintf(w, "error: %s\n\n", message)
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s %s\n\n", commandName, spec.Usage)
	fmt.Fprintf(w, "Examples:\n")
	for index, example := range spec.Examples {
		if index == 2 {
			break
		}
		fmt.Fprintf(w, "  %s\n", example)
	}
	fmt.Fprintf(w, "\nUse '%s %s --help' for more information.\n", commandName, spec.Name)
}
