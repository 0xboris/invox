package cli

import (
	"fmt"
	"io"

	"invox/internal/invoice"
)

type templatePlaceholder struct {
	Token       string
	Description string
}

type customerField struct {
	Path        string
	Description string
}

type issuerField struct {
	Path        string
	Description string
}

type invoiceDefaultsField struct {
	Path        string
	Description string
}

var templatePlaceholderGroups = []struct {
	Title   string
	Entries []templatePlaceholder
}{
	{
		Title: "Issuer",
		Entries: []templatePlaceholder{
			{Token: "@@ISSUER_NAME@@", Description: "issuer.company.legal_company_name"},
			{Token: "@@ISSUER_COMPANY_REG_NO@@", Description: "issuer.company.company_registration_number"},
			{Token: "@@ISSUER_VAT_TAX_ID@@", Description: "issuer.company.vat_tax_id"},
			{Token: "@@ISSUER_WEBSITE@@", Description: "issuer.company.website"},
			{Token: "@@ISSUER_EMAIL@@", Description: "issuer.company.email"},
			{Token: "@@ISSUER_STREET@@", Description: "issuer.company.address.street"},
			{Token: "@@ISSUER_CITY@@", Description: "issuer.company.address.city"},
			{Token: "@@ISSUER_POSTAL_CODE@@", Description: "issuer.company.address.postal_code"},
			{Token: "@@ISSUER_COUNTRY@@", Description: "issuer.company.address.country"},
		},
	},
	{
		Title: "Customer",
		Entries: []templatePlaceholder{
			{Token: "@@CUSTOMER_NAME@@", Description: "Preferred customer name"},
			{Token: "@@CUSTOMER_STREET@@", Description: "customer.address.street"},
			{Token: "@@CUSTOMER_CITY@@", Description: "customer.address.city"},
			{Token: "@@CUSTOMER_POSTAL_CODE@@", Description: "customer.address.postal_code"},
			{Token: "@@CUSTOMER_COUNTRY@@", Description: "customer.address.country"},
			{Token: "@@CUSTOMER_VAT_TAX_ID@@", Description: "customer.tax.vat_tax_id"},
			{Token: "@@CUSTOMER_EMAIL@@", Description: "Preferred invoice email"},
		},
	},
	{
		Title: "Invoice metadata",
		Entries: []templatePlaceholder{
			{Token: "@@INVOICE_NUMBER@@", Description: "invoice.number"},
			{Token: "@@ISSUE_DATE@@", Description: "invoice.issue_date formatted as DD.MM.YYYY"},
			{Token: "@@DUE_DATE@@", Description: "invoice.due_date formatted as DD.MM.YYYY"},
			{Token: "@@PERIOD_LABEL@@", Description: "invoice.period"},
		},
	},
	{
		Title: "Line items",
		Entries: []templatePlaceholder{
			{Token: "@@LINE_ITEMS_ROWS@@", Description: "Structured rows: name, description, unit price, quantity, line total"},
			{Token: "@@LINE_ITEMS_ROWS_WITH_VAT@@", Description: "Structured rows: name, description, unit price, quantity, VAT rate, line total"},
			{Token: "@@LINE_ITEMS_BEGIN@@", Description: "Begin custom line-item block; repeat enclosed snippet once per position"},
			{Token: "@@LINE_ITEMS_END@@", Description: "End custom line-item block"},
			{Token: "@@LINE_ITEM_NAME@@", Description: "Line-item name; only inside @@LINE_ITEMS_BEGIN@@ ... @@LINE_ITEMS_END@@"},
			{Token: "@@LINE_ITEM_DESCRIPTION@@", Description: "Line-item description; only inside the custom line-item block"},
			{Token: "@@LINE_ITEM_UNIT_PRICE@@", Description: "Formatted unit price; only inside the custom line-item block"},
			{Token: "@@LINE_ITEM_QUANTITY@@", Description: "Formatted quantity; only inside the custom line-item block"},
			{Token: "@@LINE_ITEM_VAT_RATE@@", Description: "Formatted effective VAT rate; only inside the custom line-item block"},
			{Token: "@@LINE_ITEM_LINE_TOTAL@@", Description: "Formatted line total; only inside the custom line-item block"},
			{Token: "@@LINE_ITEM_RULE@@", Description: "Line separator rule; only inside the custom line-item block"},
		},
	},
	{
		Title: "Totals",
		Entries: []templatePlaceholder{
			{Token: "@@SUBTOTAL@@", Description: "Formatted net subtotal"},
			{Token: "@@VAT_SUMMARY_ROWS@@", Description: "Structured VAT summary rows, one row per VAT bucket"},
			{Token: "@@TOTAL@@", Description: "Formatted invoice total"},
			{Token: "@@PAID_AMOUNT@@", Description: "Formatted paid amount"},
			{Token: "@@OUTSTANDING_AMOUNT@@", Description: "Formatted outstanding amount"},
			{Token: "@@INVOICE_TOTAL@@", Description: "Alias for @@TOTAL@@"},
			{Token: "@@OUTSTANDING_TOTAL@@", Description: "Alias for @@OUTSTANDING_AMOUNT@@"},
		},
	},
	{
		Title: "Payment",
		Entries: []templatePlaceholder{
			{Token: "@@PAYMENT_TERMS_TEXT@@", Description: "issuer.payment.payment_terms_text"},
			{Token: "@@VAT_LABEL@@", Description: "VAT label from issuer.payment.vat_label, defaults to VAT"},
			{Token: "@@BANK_NAME@@", Description: "issuer.payment.bank_name"},
			{Token: "@@IBAN@@", Description: "issuer.payment.iban"},
			{Token: "@@BIC@@", Description: "issuer.payment.bic"},
			{Token: "@@EPC_QR_AVAILABLE@@", Description: "1 when an EPC QR code will be rendered, otherwise 0"},
			{Token: "@@EPC_QR_LABEL@@", Description: "EPC QR label text"},
			{Token: "@@EPC_QR_CODE@@", Description: "EPC QR code placeholder"},
		},
	},
}

var customerFieldGroups = []struct {
	Title   string
	Entries []customerField
}{
	{
		Title: "Preferred fields",
		Entries: []customerField{
			{Path: "<customer>.name", Description: "Preferred display name used on invoices and in emails"},
			{Path: "<customer>.status", Description: "Optional status shown by customer list"},
			{Path: "<customer>.contact_person", Description: "Preferred contact used by email templates"},
			{Path: "<customer>.email_greeting", Description: "Preferred greeting used by email templates"},
			{Path: "<customer>.email", Description: "Invoice email when billing.send_invoice_to is unset"},
			{Path: "<customer>.address.street", Description: "Billing address street"},
			{Path: "<customer>.address.postal_code", Description: "Billing address postal code"},
			{Path: "<customer>.address.city", Description: "Billing address city"},
			{Path: "<customer>.address.country", Description: "Billing address country"},
			{Path: "<customer>.tax.vat_tax_id", Description: "VAT number shown on the invoice"},
			{Path: "<customer>.tax.default_vat_rate", Description: "Optional default VAT used by new/validate/render/build/email"},
			{Path: "<customer>.billing.send_invoice_to", Description: "Preferred invoice-recipient email override"},
			{Path: "<customer>.billing.currency", Description: "Optional billing currency, defaults to EUR"},
			{Path: "<customer>.numbering.code", Description: "Value used by {customer_code}"},
			{Path: "<customer>.numbering.start", Description: "Override numbering.start for this customer"},
		},
	},
	{
		Title: "Alternate supported paths",
		Entries: []customerField{
			{Path: "<customer>.legal_company_name", Description: "Alternate path for the customer display name"},
			{Path: "<customer>.billing.email", Description: "Alternate path for the invoice email"},
			{Path: "<customer>.billing.contact_person", Description: "Alternate path for the customer contact"},
			{Path: "<customer>.billing.email_greeting", Description: "Alternate path for the email greeting"},
			{Path: "<customer>.currency", Description: "Alternate path for billing.currency"},
		},
	},
}

var issuerFieldGroups = []struct {
	Title   string
	Entries []issuerField
}{
	{
		Title: "Required company fields",
		Entries: []issuerField{
			{Path: "company.legal_company_name", Description: "Company name used on invoices, in email placeholders, and as the default EPC QR recipient name"},
			{Path: "company.company_registration_number", Description: "Company registration number shown on the invoice"},
			{Path: "company.vat_tax_id", Description: "VAT/tax number shown on the invoice"},
			{Path: "company.website", Description: "Website shown on the invoice"},
			{Path: "company.email", Description: "Sender/contact email shown on the invoice"},
			{Path: "company.address.street", Description: "Business address street"},
			{Path: "company.address.postal_code", Description: "Business address postal code"},
			{Path: "company.address.city", Description: "Business address city"},
			{Path: "company.address.country", Description: "Business address country"},
		},
	},
	{
		Title: "Required payment fields",
		Entries: []issuerField{
			{Path: "payment.bank_name", Description: "Bank name rendered into the invoice template"},
			{Path: "payment.iban", Description: "Bank account IBAN used on the invoice and for EPC QR generation"},
			{Path: "payment.bic", Description: "Bank identifier code shown on the invoice"},
			{Path: "payment.due_days", Description: "Non-negative integer day count used by `new` to prefill invoice.due_date"},
			{Path: "payment.payment_terms_text", Description: "Payment terms text used by templates and email placeholders"},
		},
	},
	{
		Title: "Optional payment fields",
		Entries: []issuerField{
			{Path: "payment.vat_label", Description: "Overrides the VAT label used by @@VAT_SUMMARY_ROWS@@, defaults to VAT"},
			{Path: "payment.epc_qr.label", Description: "Overrides the EPC QR label, defaults to Pay via EPC-QR"},
			{Path: "payment.epc_qr.name", Description: "Overrides the EPC QR recipient name, defaults to company.legal_company_name"},
			{Path: "payment.epc_qr.purpose", Description: "Optional EPC QR purpose code, must be 1-4 letters or digits"},
			{Path: "payment.epc_qr.text", Description: "Optional EPC QR text line, defaults to invoice.number"},
			{Path: "payment.epc_qr.information", Description: "Optional EPC QR unstructured remittance information"},
		},
	},
}

var invoiceDefaultsFieldGroups = []struct {
	Title   string
	Entries []invoiceDefaultsField
}{
	{
		Title: "Top-level keys",
		Entries: []invoiceDefaultsField{
			{Path: "invoice", Description: "Mapping of invoice defaults used as the source document for `new`"},
			{Path: "positions", Description: "Line-item list copied into the created invoice; if omitted `new` creates an empty list"},
		},
	},
	{
		Title: "Invoice fields",
		Entries: []invoiceDefaultsField{
			{Path: "invoice.number", Description: "Usually blank in defaults; `new` always replaces it with the next generated invoice number"},
			{Path: "invoice.issue_date", Description: "Usually blank in defaults; `new` always replaces it with the current date"},
			{Path: "invoice.due_date", Description: "Usually blank in defaults; `new` always replaces it using issuer.payment.due_days"},
			{Path: "invoice.status", Description: "Usually `draft`; `new` always resets it to `draft`"},
			{Path: "invoice.period", Description: "Invoice period label copied into the created invoice and required by validate/render/build"},
			{Path: "invoice.vat_percent", Description: "Optional default VAT rate for the whole invoice; can be filled from customer.tax.default_vat_rate"},
			{Path: "invoice.paid_amount", Description: "Usually `0`; `new` always resets it to `0`"},
		},
	},
	{
		Title: "Position fields",
		Entries: []invoiceDefaultsField{
			{Path: "positions[].name", Description: "Line-item name"},
			{Path: "positions[].description", Description: "Line-item description"},
			{Path: "positions[].unit_price", Description: "Line-item net unit price; must be >= 0 on the final invoice"},
			{Path: "positions[].quantity", Description: "Line-item quantity; must be > 0 on the final invoice"},
			{Path: "positions[].vat_percent", Description: "Optional per-line VAT override"},
		},
	},
	{
		Title: "Unsupported legacy keys",
		Entries: []invoiceDefaultsField{
			{Path: "line_items", Description: "Unsupported; use positions"},
			{Path: "invoice.period_label", Description: "Unsupported; use invoice.period"},
			{Path: "invoice.vat_rate_percent", Description: "Unsupported; use invoice.vat_percent"},
		},
	},
}

const customerYAMLExample = `CUST-001:
  name: Appsters GmbH
  status: active
  contact_person: Jane Doe
  email_greeting: Dear Jane Doe,
  email: office@appsters.example
  address:
    street: Hauptstrasse 1
    postal_code: "1010"
    city: Vienna
    country: Austria
  tax:
    vat_tax_id: ATU12345678
    default_vat_rate: 20
  billing:
    send_invoice_to: accounting@appsters.example
    currency: EUR
    # email: invoices@appsters.example
    # contact_person: Jane Billing
    # email_greeting: Dear Accounts Team,
  numbering:
    code: APP
    start: 100
  # legal_company_name: Appsters GmbH
  # currency: EUR
`

const issuerYAMLExample = `company:
  legal_company_name: Boris Consulting
  company_registration_number: FN 123456a
  vat_tax_id: ATU87654321
  website: https://example.com
  email: hello@example.com
  address:
    street: Ring 1
    postal_code: "1010"
    city: Vienna
    country: Austria
payment:
  bank_name: Test Bank
  iban: AT611904300234573201
  bic: BKAUATWW
  due_days: 30
  payment_terms_text: Pay within 30 days
  vat_label: VAT
  epc_qr:
    label: Pay via EPC-QR
    purpose: SUPP
    information: Scan to pay this invoice
    # name: Boris Consulting
    # text: 2026-0001
`

const invoiceDefaultsYAMLExample = `invoice:
  number: ""
  issue_date: ""
  due_date: ""
  status: draft
  period: "Leistungszeitraum: "
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Example position
    description: Description of the delivered service
    unit_price: 100
    quantity: 1
    # vat_percent: 20
`

func printCustomerFieldReference(w io.Writer) {
	fmt.Fprintf(w, "Customer fields:\n")
	fmt.Fprintf(w, "  customers.yaml maps CUSTOMER_ID keys to customer data.\n")
	fmt.Fprintf(w, "  Use the preferred paths below unless you need an alternate supported path.\n\n")
	for groupIndex, group := range customerFieldGroups {
		if groupIndex > 0 {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "%s:\n", group.Title)
		for _, entry := range group.Entries {
			fmt.Fprintf(w, "  %-35s %s\n", entry.Path, entry.Description)
		}
	}
}

func printIssuerFieldReference(w io.Writer) {
	fmt.Fprintf(w, "Issuer fields:\n")
	fmt.Fprintf(w, "  issuer.yaml contains your own company and payment details.\n")
	fmt.Fprintf(w, "  Required fields are validated by new, validate, render, build, and email.\n\n")
	for groupIndex, group := range issuerFieldGroups {
		if groupIndex > 0 {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "%s:\n", group.Title)
		for _, entry := range group.Entries {
			fmt.Fprintf(w, "  %-35s %s\n", entry.Path, entry.Description)
		}
	}
}

func printInvoiceDefaultsFieldReference(w io.Writer) {
	fmt.Fprintf(w, "invoice_defaults.yaml fields:\n")
	fmt.Fprintf(w, "  invoice_defaults.yaml is the source document for `invox new`.\n")
	fmt.Fprintf(w, "  The created invoice later also gains a top-level customer_id.\n\n")
	for groupIndex, group := range invoiceDefaultsFieldGroups {
		if groupIndex > 0 {
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "%s:\n", group.Title)
		for _, entry := range group.Entries {
			fmt.Fprintf(w, "  %-35s %s\n", entry.Path, entry.Description)
		}
	}
}

func printCustomerYAMLExample(w io.Writer) {
	fmt.Fprintf(w, "customers.yaml example:\n")
	fmt.Fprint(w, customerYAMLExample)
}

func printIssuerYAMLExample(w io.Writer) {
	fmt.Fprintf(w, "issuer.yaml example:\n")
	fmt.Fprint(w, issuerYAMLExample)
}

func printInvoiceDefaultsYAMLExample(w io.Writer) {
	fmt.Fprintf(w, "invoice_defaults.yaml example:\n")
	fmt.Fprint(w, invoiceDefaultsYAMLExample)
}

func printRootHelp(w io.Writer) {
	fmt.Fprintf(w, "%s generates LaTeX and PDF invoices from YAML data.\n\n", commandName)
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s <subcommand> [options]\n", commandName)
	fmt.Fprintf(w, "  %s help [subcommand]\n\n", commandName)
	fmt.Fprintf(w, "Subcommands:\n")
	fmt.Fprintf(w, "  customer       Customer-related commands\n")
	fmt.Fprintf(w, "  config         Open config.yaml in the default shell editor\n")
	fmt.Fprintf(w, "  init           Create starter support files in the global config directory\n")
	fmt.Fprintf(w, "  template       Template-related commands\n")
	fmt.Fprintf(w, "  completion     Generate shell completion scripts\n")
	fmt.Fprintf(w, "  new            Create a new invoice YAML file with a generated number\n")
	fmt.Fprintf(w, "  increment      Increment the invoice number in an existing invoice YAML file\n")
	fmt.Fprintf(w, "  validate       Validate invoice YAML against customers and issuer data\n")
	fmt.Fprintf(w, "  render         Render a LaTeX invoice file\n")
	fmt.Fprintf(w, "  email          Create an email draft with the invoice PDF attached\n")
	fmt.Fprintf(w, "  build          Render and compile an invoice PDF with Tectonic\n")
	fmt.Fprintf(w, "  archive        Archive invoices and manage archived invoices\n\n")
	fmt.Fprintf(w, "Required inputs by command:\n")
	fmt.Fprintf(w, "  new        CUSTOMER_ID\n")
	fmt.Fprintf(w, "  increment  INVOICE.yaml or -i, --input\n")
	fmt.Fprintf(w, "  validate   INVOICE.yaml or -i, --input\n")
	fmt.Fprintf(w, "  render     INVOICE.yaml or -i, --input\n")
	fmt.Fprintf(w, "  email      INVOICE.yaml, INVOICE.pdf, or -i, --input\n")
	fmt.Fprintf(w, "  build      INVOICE.yaml or -i, --input\n")
	fmt.Fprintf(w, "  archive    INVOICE.yaml or -i, --input\n\n")
	fmt.Fprintf(w, "Optional flags:\n")
	fmt.Fprintf(w, "  -h, --help              Show help\n")
	fmt.Fprintf(w, "  --version               Show version\n")
	fmt.Fprintf(w, "  --json                  Structured output for customer list, template list, archive list, and validate\n")
	fmt.Fprintf(w, "  -c, --customers PATH    Path to customers.yaml\n")
	fmt.Fprintf(w, "  -o, --output PATH       Output file path (defaults vary by command)\n")
	fmt.Fprintf(w, "  -p, --pdf PATH          Path to the invoice PDF (email)\n")
	fmt.Fprintf(w, "  --archive               Archive after a successful PDF build (build)\n")
	fmt.Fprintf(w, "  --from-last             Use the latest archived invoice for CUSTOMER_ID (new)\n")
	fmt.Fprintf(w, "  -e, --edit              Open the created invoice in the default shell editor (new)\n")
	fmt.Fprintf(w, "  --to EMAIL              Recipient email override (email)\n")
	fmt.Fprintf(w, "  --subject TEXT          Email subject override, supports placeholders (email)\n")
	fmt.Fprintf(w, "  --no-open               Create the .eml draft without opening it (email)\n")
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
	fmt.Fprintf(w, "  email draft path: input path with .eml extension\n")
	fmt.Fprintf(w, "  build output: input path with .pdf extension\n\n")
	fmt.Fprintf(w, "Documentation topics:\n")
	fmt.Fprintf(w, "  %s help config      config.yaml keys, precedence, and email placeholders\n", commandName)
	fmt.Fprintf(w, "  %s help customers   customers.yaml fields, aliases, and example\n", commandName)
	fmt.Fprintf(w, "  %s help issuer      issuer.yaml fields, validation rules, and example\n", commandName)
	fmt.Fprintf(w, "  %s help defaults    invoice_defaults.yaml shape and new-command behavior\n", commandName)
	fmt.Fprintf(w, "  %s help template    template placeholders and authoring rules\n", commandName)
	fmt.Fprintf(w, "  Direct aliases: `%s customers -h`, `%s issuer -h`, `%s defaults -h`\n\n", commandName, commandName, commandName)
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("customer list"))
	fmt.Fprintf(w, "  %s\n", commandExample("config"))
	fmt.Fprintf(w, "  %s\n", commandExample("init"))
	fmt.Fprintf(w, "  %s\n", commandExample("template list"))
	fmt.Fprintf(w, "  %s\n", commandExample("completion zsh"))
	fmt.Fprintf(w, "  %s\n", commandExample("new CUST-001"))
	fmt.Fprintf(w, "  %s\n", commandExample("new CUST-001 -e"))
	fmt.Fprintf(w, "  %s\n", commandExample("new CUST-001 --from-last"))
	fmt.Fprintf(w, "  %s\n", commandExample("new CUST-001 -u issuer.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("increment invoice.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("validate invoice.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("render invoice.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("email invoice.pdf"))
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
	fmt.Fprintf(w, "  -c, --customers PATH    Path to customers.yaml\n")
	fmt.Fprintf(w, "  --json                  Print JSON output (customer list)\n\n")
	fmt.Fprintf(w, "Default lookup:\n")
	fmt.Fprintf(w, "  customers.yaml: upward project search, then %s\n\n", invoice.GlobalCustomersPath())
	fmt.Fprintf(w, "Documentation:\n")
	fmt.Fprintf(w, "  Run `%s help customers` for the customers.yaml schema reference.\n\n", commandName)
	printCustomerFieldReference(w)
	fmt.Fprintf(w, "\n\n")
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("customer list"))
	fmt.Fprintf(w, "  %s\n", commandExample("customer list -c customers.yaml"))
	fmt.Fprintf(w, "  %s\n", commandExample("customer config"))
	fmt.Fprintf(w, "\n")
	printCustomerYAMLExample(w)
}

func printCustomersHelp(w io.Writer) {
	fmt.Fprintf(w, "customers.yaml reference.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s help customers\n\n", commandName)
	fmt.Fprintf(w, "Behavior:\n")
	fmt.Fprintf(w, "  Shows the supported customers.yaml shape used by new, validate, render, build, and email.\n")
	fmt.Fprintf(w, "  `invox customer config` opens the resolved file for editing.\n\n")
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
	fmt.Fprintf(w, "  customers.yaml: upward project search, then %s\n\n", invoice.GlobalCustomersPath())
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help customers"))
	fmt.Fprintf(w, "  %s\n", commandExample("customer config"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 -c customers.yaml"))
	printCustomerYAMLExample(w)
}

func printIssuerHelp(w io.Writer) {
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
	fmt.Fprintf(w, "  issuer.yaml: upward project search, then %s\n\n", invoice.GlobalIssuerPath())
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help issuer"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 -u issuer.yaml"))
	printIssuerYAMLExample(w)
}

func printDefaultsHelp(w io.Writer) {
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
	fmt.Fprintf(w, "  invoice_defaults.yaml: upward project search, then %s\n\n", invoice.GlobalInvoiceDefaultsPath())
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("help defaults"))
	fmt.Fprintf(w, "  %s\n\n", commandExample("new CUST-001 -s invoice_defaults.yaml"))
	printInvoiceDefaultsYAMLExample(w)
}

func printTemplateHelp(w io.Writer) {
	fmt.Fprintf(w, "Template-related commands.\n\n")
	fmt.Fprintf(w, "Description:\n")
	fmt.Fprintf(w, "  Author and discover the LaTeX templates used by render and build.\n")
	fmt.Fprintf(w, "  Templates are regular .tex files with literal @@PLACEHOLDER@@ tokens.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s template <subcommand> [options]\n", commandName)
	fmt.Fprintf(w, "  %s help template [subcommand]\n\n", commandName)
	fmt.Fprintf(w, "Subcommands:\n")
	fmt.Fprintf(w, "  list          List available invoice templates\n\n")
	fmt.Fprintf(w, "Important rules:\n")
	fmt.Fprintf(w, "  Placeholder names are case-sensitive and must match exactly.\n")
	fmt.Fprintf(w, "  Unknown placeholders are left unchanged in the rendered TeX.\n")
	fmt.Fprintf(w, "  @@VAT_RATE@@ and @@VAT_AMOUNT@@ are unsupported; use @@VAT_SUMMARY_ROWS@@.\n")
	fmt.Fprintf(w, "  Most placeholders are LaTeX-escaped automatically.\n")
	fmt.Fprintf(w, "  Dates render as DD.MM.YYYY.\n")
	fmt.Fprintf(w, "  Money renders as 1.234,56 \\euro for EUR, or with the currency code otherwise.\n\n")
	fmt.Fprintf(w, "Structured placeholders:\n")
	fmt.Fprintf(w, "  @@LINE_ITEMS_ROWS@@ requires a five-column table.\n")
	fmt.Fprintf(w, "  @@LINE_ITEMS_ROWS_WITH_VAT@@ requires a six-column table.\n")
	fmt.Fprintf(w, "  @@LINE_ITEMS_BEGIN@@ ... @@LINE_ITEMS_END@@ repeats a custom snippet once per position.\n")
	fmt.Fprintf(w, "  Inside that block, use @@LINE_ITEM_NAME@@, @@LINE_ITEM_DESCRIPTION@@, @@LINE_ITEM_UNIT_PRICE@@,\n")
	fmt.Fprintf(w, "  @@LINE_ITEM_QUANTITY@@, @@LINE_ITEM_VAT_RATE@@, @@LINE_ITEM_LINE_TOTAL@@, and @@LINE_ITEM_RULE@@.\n")
	fmt.Fprintf(w, "  @@VAT_SUMMARY_ROWS@@ belongs inside a two-column totals table.\n")
	fmt.Fprintf(w, "  Use @@VAT_LABEL@@ anywhere you want the same VAT label text in the template.\n\n")
	fmt.Fprintf(w, "Custom line-item block example:\n")
	fmt.Fprintf(w, "  @@LINE_ITEMS_BEGIN@@\n")
	fmt.Fprintf(w, "  @@LINE_ITEM_NAME@@ & @@LINE_ITEM_UNIT_PRICE@@ & @@LINE_ITEM_VAT_RATE@@ & @@LINE_ITEM_LINE_TOTAL@@\\\\\n")
	fmt.Fprintf(w, "  @@LINE_ITEM_RULE@@\n")
	fmt.Fprintf(w, "  @@LINE_ITEMS_END@@\n\n")
	fmt.Fprintf(w, "Template workflow:\n")
	fmt.Fprintf(w, "  Run `%s render invoice.yaml` to inspect the generated .tex.\n", commandName)
	fmt.Fprintf(w, "  Run `%s build invoice.yaml` after the rendered LaTeX looks correct.\n", commandName)
	fmt.Fprintf(w, "  Run `%s template list` to discover templates addressable by name with -t/--template.\n\n", commandName)
	printTemplatePlaceholderReference(w)
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("template list"))
	fmt.Fprintf(w, "  %s\n", commandExample("template list --names"))
	fmt.Fprintf(w, "  %s\n", commandExample("render invoice.yaml -t multi_vat.tex"))
	fmt.Fprintf(w, "  %s\n", commandExample("build invoice.yaml -t multi_vat.tex"))
}

func printTemplateListHelp(w io.Writer) {
	fmt.Fprintf(w, "List available LaTeX invoice templates from the same directory as the resolved default template.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s template list [--names | --json]\n\n", commandName)
	fmt.Fprintf(w, "Optional flags:\n")
	fmt.Fprintf(w, "  -h, --help              Show this help page\n")
	fmt.Fprintf(w, "  --names                 Print only template names\n")
	fmt.Fprintf(w, "  --json                  Print JSON output\n\n")
	fmt.Fprintf(w, "Output:\n")
	fmt.Fprintf(w, "  Default: NAME<TAB>ABSOLUTE_PATH\n")
	fmt.Fprintf(w, "  --names: TEMPLATE_NAME per line\n")
	fmt.Fprintf(w, "  --json: array of objects with name and path\n\n")
	fmt.Fprintf(w, "Lookup:\n")
	fmt.Fprintf(w, "  The directory is derived from the resolved default template path.\n")
	fmt.Fprintf(w, "  Name-only -t/--template values are resolved in that same directory.\n\n")
	fmt.Fprintf(w, "Examples:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("template list"))
	fmt.Fprintf(w, "  %s\n", commandExample("template list --names"))
	fmt.Fprintf(w, "  %s\n", commandExample("template list --json"))
	fmt.Fprintf(w, "  %s\n", commandExample("build invoice.yaml -t multi_vat.tex"))
}

func printCompletionHelp(w io.Writer) {
	fmt.Fprintf(w, "Generate shell completion scripts.\n\n")
	fmt.Fprintf(w, "Usage:\n")
	fmt.Fprintf(w, "  %s completion zsh\n\n", commandName)
	fmt.Fprintf(w, "Supported shells:\n")
	fmt.Fprintf(w, "  zsh\n\n")
	fmt.Fprintf(w, "Notes:\n")
	fmt.Fprintf(w, "  The generated Zsh completion script completes the documented command tree and common flags.\n")
	fmt.Fprintf(w, "  It suggests invoice, PDF, YAML, and template paths where relevant.\n")
	fmt.Fprintf(w, "  Customer-ID completion for `new` is backed by `%s customer list`.\n", commandName)
	fmt.Fprintf(w, "  Template-name completion for render/build is backed by `%s template list --names`.\n", commandName)
	fmt.Fprintf(w, "  Archived-invoice completion for `archive edit` is backed by `%s archive list`.\n\n", commandName)
	fmt.Fprintf(w, "Quick start:\n")
	fmt.Fprintf(w, "  source <(%s completion zsh)\n\n", commandName)
	fmt.Fprintf(w, "Persistent Zsh install:\n")
	fmt.Fprintf(w, "  mkdir -p ~/.zsh/completions\n")
	fmt.Fprintf(w, "  %s completion zsh > ~/.zsh/completions/_%s\n", commandName, commandName)
	fmt.Fprintf(w, "  Add this before compinit in ~/.zshrc:\n")
	fmt.Fprintf(w, "    fpath=(~/.zsh/completions $fpath)\n")
	fmt.Fprintf(w, "    autoload -Uz compinit\n")
	fmt.Fprintf(w, "    compinit\n\n")
	fmt.Fprintf(w, "Example:\n")
	fmt.Fprintf(w, "  %s\n", commandExample("completion zsh"))
}

func printTemplatePlaceholderReference(w io.Writer) {
	fmt.Fprintf(w, "Available .tex placeholders:\n")
	for _, group := range templatePlaceholderGroups {
		fmt.Fprintf(w, "  %s:\n", group.Title)
		for _, entry := range group.Entries {
			fmt.Fprintf(w, "    %-32s %s\n", entry.Token, entry.Description)
		}
	}
	fmt.Fprintf(w, "\n")
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
			if spec.AcceptsPDFInput {
				fmt.Fprintf(w, "  INVOICE.yaml, INVOICE.pdf, or -i, --input PATH  Path to the invoice YAML or built PDF file\n")
			} else {
				fmt.Fprintf(w, "  INVOICE.yaml or -i, --input PATH  Path to the invoice YAML file\n")
			}
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
	if spec.NeedsPDF {
		fmt.Fprintf(w, "  -p, --pdf PATH          Path to the invoice PDF\n")
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
	if spec.SupportsEditFlag {
		fmt.Fprintf(w, "  -e, --edit              Open the created invoice in the default shell editor\n")
	}
	if spec.SupportsEmailToFlag {
		fmt.Fprintf(w, "  --to EMAIL              Recipient email override\n")
	}
	if spec.SupportsSubjectFlag {
		fmt.Fprintf(w, "  --subject TEXT          Email subject override, supports placeholders\n")
	}
	if spec.SupportsNoOpenFlag {
		fmt.Fprintf(w, "  --no-open               Create the .eml draft without opening it\n")
	}
	if spec.SupportsJSONFlag {
		fmt.Fprintf(w, "  --json                  Print JSON output\n")
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
		fmt.Fprintf(w, "  schema/docs: run `%s help customers`\n", commandName)
	}
	if spec.NeedsIssuer {
		fmt.Fprintf(w, "  issuer.yaml: upward project search, then %s\n", invoice.GlobalIssuerPath())
		fmt.Fprintf(w, "  schema/docs: run `%s help issuer`\n", commandName)
	}
	if spec.NeedsPDF {
		if spec.AcceptsPDFInput {
			fmt.Fprintf(w, "  invoice PDF: input path with .pdf extension by default, or the input itself when the input is a PDF\n")
		} else {
			fmt.Fprintf(w, "  invoice PDF: input path with .pdf extension by default\n")
		}
	}
	if spec.NeedsDefaults {
		fmt.Fprintf(w, "  invoice_defaults.yaml: upward project search, then %s\n", invoice.GlobalInvoiceDefaultsPath())
		fmt.Fprintf(w, "  schema/docs: run `%s help defaults`\n", commandName)
	}
	if spec.NeedsTemplate {
		fmt.Fprintf(w, "  template.tex: upward project search, then %s\n", invoice.GlobalTemplatePath())
	}
	if spec.Name == "archive" || spec.Name == "archive edit" || spec.Name == "archive list" || spec.SupportsFromLastFlag {
		fmt.Fprintf(w, "  archive.dir: config.yaml, then %s\n", invoice.DefaultArchiveDir())
	}
	if spec.Name == "customer config" {
		fmt.Fprintf(w, "\n")
		printCustomerFieldReference(w)
	}
	if spec.Name == "customer list" {
		fmt.Fprintf(w, "\nOutput:\n")
		fmt.Fprintf(w, "  Default: CUSTOMER_ID<TAB>NAME<TAB>STATUS\n")
		fmt.Fprintf(w, "  --json: array of objects with id, name, and status\n")
	}
	if spec.Name == "archive list" {
		fmt.Fprintf(w, "\nOutput:\n")
		fmt.Fprintf(w, "  One archived invoice per line as FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS\n")
		fmt.Fprintf(w, "  --json: array of objects with filename, customer_id, issue_date, and status\n")
	}
	if spec.Name == "validate" {
		fmt.Fprintf(w, "\nOutput:\n")
		fmt.Fprintf(w, "  Default: Validation OK summary line for humans\n")
		fmt.Fprintf(w, "  --json: object with invoice_number, customer_id, line_item_count, currency,\n")
		fmt.Fprintf(w, "          subtotal_cents, vat_amount_cents, total_cents, paid_amount_cents, and outstanding_cents\n")
	}
	if spec.Name == "archive edit" {
		fmt.Fprintf(w, "\nBehavior:\n")
		fmt.Fprintf(w, "  Copies the archived invoice from archive.dir into the current directory.\n")
		fmt.Fprintf(w, "  The working copy is written as YAML with invoice.status set to editing.\n")
		fmt.Fprintf(w, "  Re-running %s archive on that working copy replaces the archived invoice.\n", commandName)
	}
	if spec.Name == "email" {
		fmt.Fprintf(w, "\nBehavior:\n")
		fmt.Fprintf(w, "  Accepts either the invoice YAML file or the built PDF as input.\n")
		fmt.Fprintf(w, "  When the input is a PDF, the matching YAML file is resolved from the same basename.\n")
		fmt.Fprintf(w, "  The PDF lookup checks next to the PDF first, then archive.dir.\n")
		fmt.Fprintf(w, "  Requires invoice.status to be built or archived and the PDF attachment to exist.\n")
		fmt.Fprintf(w, "  On macOS, opens an editable compose window in Apple Mail with the PDF attached.\n")
		fmt.Fprintf(w, "  If -o is set, or on non-macOS platforms, writes a .eml draft file and opens it.\n")
		fmt.Fprintf(w, "  --no-open always writes a .eml draft file and leaves it on disk.\n")
		fmt.Fprintf(w, "  File-based drafts opened by the CLI are scheduled for cleanup shortly after they are opened.\n")
		fmt.Fprintf(w, "  Does not send the email and does not change invoice.status.\n")
	}
	fmt.Fprintf(w, "\nExamples:\n")
	for _, example := range spec.Examples {
		fmt.Fprintf(w, "  %s\n", example)
	}
	if spec.Name == "customer config" {
		fmt.Fprintf(w, "\n")
		printCustomerYAMLExample(w)
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
	fmt.Fprintf(w, "  paths.template     Set the default invoice template file, for example 'multi_vat.tex'\n")
	fmt.Fprintf(w, "  numbering.pattern  Override the invoice-number pattern\n")
	fmt.Fprintf(w, "  numbering.start    Global starting counter when no archived invoice matches\n")
	fmt.Fprintf(w, "  archive.dir        Override the archive directory for archived invoice files\n")
	fmt.Fprintf(w, "  email.subject      Override the draft email subject template for the email command\n")
	fmt.Fprintf(w, "  email.body         Override the plain-text body template for the email command\n\n")
	fmt.Fprintf(w, "email template placeholders:\n")
	fmt.Fprintf(w, "  {customer_name}        Customer display name\n")
	fmt.Fprintf(w, "  {email_greeting}       Customer-specific greeting, defaults to Hello,\n")
	fmt.Fprintf(w, "  {contact_person}       Customer contact person\n")
	fmt.Fprintf(w, "  {customer_id}          Customer ID from the invoice\n")
	fmt.Fprintf(w, "  {invoice_number}       Invoice number\n")
	fmt.Fprintf(w, "  {issue_date}           Invoice issue date\n")
	fmt.Fprintf(w, "  {due_date}             Invoice due date\n")
	fmt.Fprintf(w, "  {total_amount}         Invoice total with currency\n")
	fmt.Fprintf(w, "  {outstanding_amount}   Outstanding amount with currency\n")
	fmt.Fprintf(w, "  {payment_terms_text}   issuer.payment.payment_terms_text\n")
	fmt.Fprintf(w, "  {issuer_name}          issuer.company.legal_company_name\n\n")
	fmt.Fprintf(w, "Customer overrides:\n")
	fmt.Fprintf(w, "  customers.<CUSTOMER_ID>.numbering.start  Override numbering.start for one customer\n\n")
	fmt.Fprintf(w, "Support file precedence:\n")
	fmt.Fprintf(w, "  1. explicit CLI flag\n")
	fmt.Fprintf(w, "  2. upward project search\n")
	fmt.Fprintf(w, "  3. paths.* in config.yaml\n")
	fmt.Fprintf(w, "  4. conventional files in %s\n\n", invoice.ConfigDir())
	fmt.Fprintf(w, "Template selection:\n")
	fmt.Fprintf(w, "  Set paths.template in config.yaml to choose the default .tex file.\n")
	fmt.Fprintf(w, "  Relative values like 'multi_vat.tex' resolve next to config.yaml.\n\n")
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
