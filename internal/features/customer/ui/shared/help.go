package shared

import (
	"fmt"
	"io"
	"strings"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/schema"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

const commandName = "invox"

// Topics returns the help topics about the customer slice's files.
func Topics(f *Factory) []cmdutil.Topic {
	return []cmdutil.Topic{
		{Name: "customers", Short: "customers.yaml fields, aliases, and example", Print: func(w io.Writer) { printCustomersHelp(w, f.Host()) }},
	}
}

func commandExample(args string) string {
	return commandName + " " + args
}

func printCustomersHelp(w io.Writer, h invoice.Host) {
	fmt.Fprintf(w, "customers.yaml reference.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help customers\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Shows the supported customers.yaml shape used by new, validate, render, build, and email.\n")
	fmt.Fprintf(w, "  `invox customer edit` opens the resolved file for editing.\n\n")
	fmt.Fprintf(w, "Formatting:\n")
	fmt.Fprintf(w, "  Top-level customer IDs must start at column 1 with no leading spaces.\n\n")
	printCustomerFieldReference(w)
	fmt.Fprintf(w, "\n\nRules:\n")
	fmt.Fprintf(w, "  Preferred display name path is <customer>.name; <customer>.legal_company_name is also accepted.\n")
	fmt.Fprintf(w, "  Email lookup order is billing.send_invoice_to, billing.email, then email.\n")
	fmt.Fprintf(w, "  email_greeting defaults to Hello, when omitted.\n")
	fmt.Fprintf(w, "  billing.currency defaults to EUR.\n")
	fmt.Fprintf(w, "  numbering.code feeds {customer_code}; numbering.start overrides config.numbering.start for one customer.\n\n")
	fmt.Fprintf(w, "Lookup:\n")
	fmt.Fprintf(w, "  customers.yaml: upward project search, then %s\n\n", h.GlobalCustomersPath())
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help customers"))
	fmt.Fprintf(w, "  %s\n", commandExample("customer edit"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 -c customers.yaml"))
	printCustomerYAMLExample(w)
}

// CustomerFieldReference is the customers.yaml field table.
func CustomerFieldReference() string {
	var b strings.Builder
	printCustomerFieldReference(&b)
	return b.String()
}

// CustomerYAMLExample is the titled customers.yaml example.
func CustomerYAMLExample() string {
	var b strings.Builder
	printCustomerYAMLExample(&b)
	return b.String()
}

func printCustomerFieldReference(w io.Writer) {
	fmt.Fprintf(w, "Customer fields:\n")
	fmt.Fprintf(w, "  customers.yaml maps CUSTOMER_ID keys to customer data.\n")
	fmt.Fprintf(w, "  Use the preferred paths below unless you need an alternate supported path.\n\n")
	for groupIndex, group := range schema.CustomerFields {
		if groupIndex > 0 {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "%s:\n", group.Title)
		for _, field := range group.Fields {
			fmt.Fprintf(w, "  %-35s %s\n", field.Path, field.Description)
		}
	}
}

func printCustomerYAMLExample(w io.Writer) {
	fmt.Fprintf(w, "customers.yaml example:\n")
	fmt.Fprint(w, schema.CustomerYAMLExample)
}
