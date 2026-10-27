package shared

import (
	"fmt"
	"io"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/schema"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

const commandName = "invox"

// Topics returns the help topics about the invoice slice's files.
func Topics(f *Factory) []cmdutil.Topic {
	return []cmdutil.Topic{
		{Name: "issuer", Short: "issuer.yaml fields, validation rules, and example", Print: func(w io.Writer) { printIssuerHelp(w, f.Host()) }},
		{Name: "defaults", Aliases: []string{"invoice-defaults", "invoice_defaults"}, Short: "invoice_defaults.yaml shape and new-command behavior", Print: func(w io.Writer) { printDefaultsHelp(w, f.Host()) }},
	}
}

// ReplacingArchived is the section on replacing an archived invoice, for
// archive, and for build when withArchiveFlag is set.
func ReplacingArchived(withArchiveFlag bool) string {
	text := "Replacing an archived invoice:\n"
	if withArchiveFlag {
		text += "  An invoice with invoice.status archived keeps that status when its PDF is rebuilt.\n" +
			"  With --archive, a working copy from `invox archive edit` replaces the archived invoice it came from.\n"
	} else {
		text += "  Archiving a working copy from `invox archive edit` replaces the archived invoice it came from.\n"
	}
	return text + "  On a terminal you are asked to confirm; otherwise pass --yes. Declining exits with status 2.\n" +
		"  The previous version is kept as archive.dir/.history/<path>.<UTC timestamp>.<ext>,\n" +
		"  which archive list, numbering and the duplicate-number check ignore.\n" +
		"  --yes only answers the question; every other check still applies.\n"
}

func commandExample(args string) string {
	return commandName + " " + args
}

func printIssuerHelp(w io.Writer, h invoice.Host) {
	fmt.Fprintf(w, "issuer.yaml reference.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help issuer\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Shows the supported issuer.yaml shape used by new, validate, render, build, and email.\n")
	fmt.Fprintf(w, "  `invox init` writes a starter issuer.yaml with this structure.\n\n")
	fmt.Fprintf(w, "Formatting:\n")
	fmt.Fprintf(w, "  Top-level keys must start at column 1 with no leading spaces.\n\n")
	printIssuerFieldReference(w)
	fmt.Fprintf(w, "\n\nRules:\n")
	fmt.Fprintf(w, "  payment.due_days must be a non-negative integer.\n")
	fmt.Fprintf(w, "  payment.vat_label defaults to VAT when omitted.\n")
	fmt.Fprintf(w, "  payment.epc_qr.name defaults to company.legal_company_name.\n")
	fmt.Fprintf(w, "  payment.epc_qr.text defaults to invoice.number.\n")
	fmt.Fprintf(w, "  payment.epc_qr.label defaults to Pay via EPC-QR.\n")
	fmt.Fprintf(w, "  EPC QR generation requires a valid SEPA-scope payment.iban.\n\n")
	fmt.Fprintf(w, "Lookup:\n")
	fmt.Fprintf(w, "  issuer.yaml: upward project search, then %s\n\n", h.GlobalIssuerPath())
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help issuer"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 -u issuer.yaml"))
	printIssuerYAMLExample(w)
}

func printDefaultsHelp(w io.Writer, h invoice.Host) {
	fmt.Fprintf(w, "invoice_defaults.yaml reference.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help defaults\n", commandName)
	fmt.Fprintf(w, "  %s help invoice-defaults\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Shows the supported invoice_defaults.yaml shape used by `invox new`.\n")
	fmt.Fprintf(w, "  `invox init` writes a starter invoice_defaults.yaml with this structure.\n")
	fmt.Fprintf(w, "  `invox new --from-last` bypasses invoice_defaults.yaml and clones the latest archived invoice for that customer.\n\n")
	fmt.Fprintf(w, "Formatting:\n")
	fmt.Fprintf(w, "  Top-level keys must start at column 1 with no leading spaces.\n\n")
	printInvoiceDefaultsFieldReference(w)
	fmt.Fprintf(w, "\n\nRules:\n")
	fmt.Fprintf(w, "  `new` sets customer_id, invoice.number, invoice.issue_date, invoice.due_date, invoice.status, and invoice.paid_amount.\n")
	fmt.Fprintf(w, "  If positions is omitted, `new` creates an empty list.\n")
	fmt.Fprintf(w, "  The final invoice used by validate/render/build/email still needs a non-empty positions list.\n")
	fmt.Fprintf(w, "  Canonical keys are positions, invoice.period, and invoice.vat_percent.\n\n")
	fmt.Fprintf(w, "Lookup:\n")
	fmt.Fprintf(w, "  invoice_defaults.yaml: upward project search, then %s\n\n", h.GlobalInvoiceDefaultsPath())
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help defaults"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 --defaults invoice_defaults.yaml"))
	printInvoiceDefaultsYAMLExample(w)
}

func printIssuerFieldReference(w io.Writer) {
	fmt.Fprintf(w, "Issuer fields:\n")
	fmt.Fprintf(w, "  issuer.yaml contains your own company and payment details.\n")
	fmt.Fprintf(w, "  Required fields are validated by new, validate, render, build, and email.\n\n")
	printFieldGroups(w, schema.IssuerFields)
}

func printInvoiceDefaultsFieldReference(w io.Writer) {
	fmt.Fprintf(w, "invoice_defaults.yaml fields:\n")
	fmt.Fprintf(w, "  invoice_defaults.yaml is the source document for `invox new`.\n")
	fmt.Fprintf(w, "  The created invoice later also gains a top-level customer_id.\n\n")
	printFieldGroups(w, schema.InvoiceDefaultsFields)
}

func printFieldGroups(w io.Writer, groups []schema.FieldGroup) {
	for groupIndex, group := range groups {
		if groupIndex > 0 {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "%s:\n", group.Title)
		for _, field := range group.Fields {
			fmt.Fprintf(w, "  %-35s %s\n", field.Path, field.Description)
		}
	}
}

func printIssuerYAMLExample(w io.Writer) {
	fmt.Fprintf(w, "issuer.yaml example:\n")
	fmt.Fprint(w, schema.IssuerYAMLExample)
}

func printInvoiceDefaultsYAMLExample(w io.Writer) {
	fmt.Fprintf(w, "invoice_defaults.yaml example:\n")
	fmt.Fprint(w, schema.InvoiceDefaultsYAMLExample)
}
