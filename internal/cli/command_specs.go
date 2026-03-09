package cli

const commandName = "invox"

type commandSpec struct {
	Name            string
	Summary         string
	Usage           string
	Examples        []string
	RequiresInput   bool
	RequiredArgs    []string
	NeedsCustomers  bool
	NeedsIssuer     bool
	NeedsDefaults   bool
	NeedsTemplate   bool
	DefaultOutput   string
	OutputExtension string
}

func commandExample(args string) string {
	return commandName + " " + args
}

func customerListSpec() commandSpec {
	return commandSpec{
		Name:           "customer list",
		Summary:        "List all customers from customers.yaml.",
		Usage:          "customer list [-c CUSTOMERS.yaml]",
		NeedsCustomers: true,
		Examples: []string{
			commandExample("customer list"),
			commandExample("customer list -c customers.yaml"),
		},
	}
}

func customerConfigSpec() commandSpec {
	return commandSpec{
		Name:           "customer config",
		Summary:        "Open customers.yaml in the default shell editor.",
		Usage:          "customer config [-c CUSTOMERS.yaml]",
		NeedsCustomers: true,
		Examples: []string{
			commandExample("customer config"),
			commandExample("customer config -c customers.yaml"),
		},
	}
}

func configSpec() commandSpec {
	return commandSpec{
		Name:    "config",
		Summary: "Open config.yaml in the default shell editor.",
		Usage:   "config",
		Examples: []string{
			commandExample("config"),
		},
	}
}

func newSpec() commandSpec {
	return commandSpec{
		Name:            "new",
		Summary:         "Create a new invoice YAML file with a generated number and prefilled defaults.",
		Usage:           "new CUSTOMER_ID [-o OUTPUT.yaml] [-s SOURCE.yaml] [-c CUSTOMERS.yaml] [-u ISSUER.yaml]",
		RequiredArgs:    []string{"CUSTOMER_ID"},
		NeedsCustomers:  true,
		NeedsIssuer:     true,
		NeedsDefaults:   true,
		DefaultOutput:   "invoice.yaml",
		OutputExtension: ".yaml",
		Examples: []string{
			commandExample("new CUST-001"),
			commandExample("new CUST-001 -o invoices/2026-0022.yaml -s invoice_defaults.yaml -c customers.yaml -u issuer.yaml"),
		},
	}
}

func incrementSpec() commandSpec {
	return commandSpec{
		Name:           "increment",
		Summary:        "Increment the invoice number in an existing invoice YAML file.",
		Usage:          "increment -i INVOICE.yaml [-c CUSTOMERS.yaml]",
		RequiresInput:  true,
		NeedsCustomers: true,
		Examples: []string{
			commandExample("increment -i invoice.yaml"),
			commandExample("increment -i invoices/2026-0022.yaml -c customers.yaml"),
		},
	}
}

func validateSpec() commandSpec {
	return commandSpec{
		Name:           "validate",
		Summary:        "Validate invoice YAML against customers and issuer data.",
		Usage:          "validate -i INVOICE.yaml [-c CUSTOMERS.yaml] [-u ISSUER.yaml]",
		RequiresInput:  true,
		NeedsCustomers: true,
		NeedsIssuer:    true,
		Examples: []string{
			commandExample("validate -i invoice.yaml"),
			commandExample("validate -i invoices/2026-0021.yaml -c customers.yaml -u issuer.yaml"),
		},
	}
}

func renderSpec() commandSpec {
	return commandSpec{
		Name:            "render",
		Summary:         "Render a LaTeX invoice file from YAML data.",
		Usage:           "render -i INVOICE.yaml [-o OUTPUT.tex] [-c CUSTOMERS.yaml] [-u ISSUER.yaml] [-t TEMPLATE.tex]",
		RequiresInput:   true,
		NeedsCustomers:  true,
		NeedsIssuer:     true,
		NeedsTemplate:   true,
		DefaultOutput:   "invoice.tex",
		OutputExtension: ".tex",
		Examples: []string{
			commandExample("render -i invoice.yaml"),
			commandExample("render -i invoices/2026-0021.yaml -o out/2026-0021.tex -c customers.yaml -u issuer.yaml -t invoice_template.tex"),
		},
	}
}

func buildSpec() commandSpec {
	return commandSpec{
		Name:            "build",
		Summary:         "Render and compile an invoice PDF with Tectonic.",
		Usage:           "build -i INVOICE.yaml [-o OUTPUT.pdf] [-c CUSTOMERS.yaml] [-u ISSUER.yaml] [-t TEMPLATE.tex]",
		RequiresInput:   true,
		NeedsCustomers:  true,
		NeedsIssuer:     true,
		NeedsTemplate:   true,
		DefaultOutput:   "invoice.pdf",
		OutputExtension: ".pdf",
		Examples: []string{
			commandExample("build -i invoice.yaml"),
			commandExample("build -i invoices/2026-0021.yaml -o out/2026-0021.pdf -c customers.yaml -u issuer.yaml -t invoice_template.tex"),
		},
	}
}

func lookupCommand(name string) (commandSpec, bool) {
	switch name {
	case "customer list":
		return customerListSpec(), true
	case "customer config":
		return customerConfigSpec(), true
	case "config":
		return configSpec(), true
	case "new":
		return newSpec(), true
	case "increment":
		return incrementSpec(), true
	case "validate":
		return validateSpec(), true
	case "render":
		return renderSpec(), true
	case "build":
		return buildSpec(), true
	default:
		return commandSpec{}, false
	}
}
