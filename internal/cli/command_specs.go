package cli

const commandName = "invox"

type commandSpec struct {
	Name                   string
	Summary                string
	Usage                  string
	Examples               []string
	RequiresInput          bool
	AcceptsPDFInput        bool
	RequiredArgs           []string
	NeedsCustomers         bool
	NeedsIssuer            bool
	NeedsDefaults          bool
	NeedsPDF               bool
	NeedsTemplate          bool
	SupportsArchiveFlag    bool
	SupportsFromLastFlag   bool
	SupportsEditFlag       bool
	SupportsEmailToFlag    bool
	SupportsSubjectFlag    bool
	AcceptsPositionalInput bool
	DynamicDefaultOutput   bool
	InputBasedOutput       bool
	DefaultOutput          string
	OutputExtension        string
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

func initSpec() commandSpec {
	return commandSpec{
		Name:    "init",
		Summary: "Create starter support files in the global config directory.",
		Usage:   "init",
		Examples: []string{
			commandExample("init"),
		},
	}
}

func templateListSpec() commandSpec {
	return commandSpec{
		Name:    "template list",
		Summary: "List available LaTeX invoice templates from the current project and config directories.",
		Usage:   "template list [--names]",
		Examples: []string{
			commandExample("template list"),
			commandExample("template list --names"),
		},
	}
}

func completionSpec() commandSpec {
	return commandSpec{
		Name:    "completion",
		Summary: "Generate shell completion scripts.",
		Usage:   "completion zsh",
		Examples: []string{
			commandExample("completion zsh"),
		},
	}
}

func newSpec() commandSpec {
	return commandSpec{
		Name:                 "new",
		Summary:              "Create a new invoice YAML file with a generated number and prefilled defaults.",
		Usage:                "new CUSTOMER_ID [-o OUTPUT.yaml] [-s SOURCE.yaml] [-c CUSTOMERS.yaml] [-u ISSUER.yaml] [-e] [--from-last]",
		RequiredArgs:         []string{"CUSTOMER_ID"},
		NeedsCustomers:       true,
		NeedsIssuer:          true,
		NeedsDefaults:        true,
		SupportsFromLastFlag: true,
		SupportsEditFlag:     true,
		DynamicDefaultOutput: true,
		OutputExtension:      ".yaml",
		Examples: []string{
			commandExample("new CUST-001"),
			commandExample("new CUST-001 -e"),
			commandExample("new CUST-001 --from-last"),
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

func emailSpec() commandSpec {
	return commandSpec{
		Name:                   "email",
		Summary:                "Create an email draft, open it in the default mail app, and remove the draft file.",
		Usage:                  "email (INVOICE.yaml | INVOICE.pdf | -i INPUT) [-p INVOICE.pdf] [-o OUTPUT.eml] [-c CUSTOMERS.yaml] [-u ISSUER.yaml] [--to EMAIL] [--subject TEXT]",
		RequiresInput:          true,
		AcceptsPDFInput:        true,
		NeedsCustomers:         true,
		NeedsIssuer:            true,
		NeedsPDF:               true,
		SupportsEmailToFlag:    true,
		SupportsSubjectFlag:    true,
		AcceptsPositionalInput: true,
		InputBasedOutput:       true,
		OutputExtension:        ".eml",
		Examples: []string{
			commandExample("email invoice.yaml"),
			commandExample("email invoice.pdf"),
			commandExample("email invoice.yaml --to billing@example.com"),
			commandExample("email invoices/2026-0021.yaml -p out/2026-0021.pdf -o drafts/2026-0021.eml -c customers.yaml -u issuer.yaml"),
		},
	}
}

func buildSpec() commandSpec {
	return commandSpec{
		Name:                   "build",
		Summary:                "Render and compile an invoice PDF with Tectonic.",
		Usage:                  "build (INVOICE.yaml | -i INVOICE.yaml) [-o OUTPUT.pdf] [-c CUSTOMERS.yaml] [-u ISSUER.yaml] [-t TEMPLATE.tex] [--archive]",
		RequiresInput:          true,
		NeedsCustomers:         true,
		NeedsIssuer:            true,
		NeedsTemplate:          true,
		SupportsArchiveFlag:    true,
		AcceptsPositionalInput: true,
		InputBasedOutput:       true,
		OutputExtension:        ".pdf",
		Examples: []string{
			commandExample("build invoice.yaml"),
			commandExample("build invoice.yaml --archive"),
			commandExample("build invoices/2026-0021.yaml -o out/2026-0021.pdf -c customers.yaml -u issuer.yaml -t invoice_template.tex"),
		},
	}
}

func archiveSpec() commandSpec {
	return commandSpec{
		Name:                   "archive",
		Summary:                "Archive a built or edited invoice YAML file into the configured archive directory.",
		Usage:                  "archive (INVOICE.yaml | -i INVOICE.yaml)",
		RequiresInput:          true,
		AcceptsPositionalInput: true,
		Examples: []string{
			commandExample("archive invoice.yaml"),
			commandExample("archive invoices/2026-0021.yaml"),
		},
	}
}

func archiveEditSpec() commandSpec {
	return commandSpec{
		Name:         "archive edit",
		Summary:      "Copy an archived invoice into the current directory and mark it as editing.",
		Usage:        "archive edit FILENAME",
		RequiredArgs: []string{"FILENAME"},
		Examples: []string{
			commandExample("archive edit 2026-03-06.yaml"),
			commandExample("archive edit customer-a/2026-03-06.yaml"),
		},
	}
}

func archiveListSpec() commandSpec {
	return commandSpec{
		Name:    "archive list",
		Summary: "List archived invoices from the configured archive directory.",
		Usage:   "archive list",
		Examples: []string{
			commandExample("archive list"),
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
	case "init":
		return initSpec(), true
	case "template list":
		return templateListSpec(), true
	case "completion":
		return completionSpec(), true
	case "new":
		return newSpec(), true
	case "increment":
		return incrementSpec(), true
	case "validate":
		return validateSpec(), true
	case "render":
		return renderSpec(), true
	case "email", "send":
		return emailSpec(), true
	case "build":
		return buildSpec(), true
	case "archive":
		return archiveSpec(), true
	case "archive edit":
		return archiveEditSpec(), true
	case "archive list":
		return archiveListSpec(), true
	default:
		return commandSpec{}, false
	}
}
