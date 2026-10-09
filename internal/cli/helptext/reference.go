// Package helptext holds the help content that several commands and help
// topics share: the reference tables for the support files and templates,
// their YAML examples, the topic pages, and the Default lookup lines.
package helptext

import (
	"fmt"
	"io"
	"strings"
)

const commandName = "invox"

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
			{Token: "@@LINE_ITEM_UNIT_PRICE@@", Description: "Formatted unit price with 2 to 4 decimals, as many as it needs (rounded half up beyond 4); only inside the custom line-item block"},
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
			{Path: "invoice.paid_amount", Description: "Usually `0`; must be >= 0 and <= the invoice total; `new` always resets it to `0`"},
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

// CustomerFieldReference is the customers.yaml field table.
func CustomerFieldReference() string {
	return sprint(printCustomerFieldReference)
}

// CustomerYAMLExample is the titled customers.yaml example.
func CustomerYAMLExample() string {
	return sprint(printCustomerYAMLExample)
}

// TemplatePlaceholderReference is the table of template placeholders.
func TemplatePlaceholderReference() string {
	return sprint(printTemplatePlaceholderReference)
}

func sprint(write func(io.Writer)) string {
	var b strings.Builder
	write(&b)
	return b.String()
}

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
