// Package schema documents the files invox reads: the fields of
// customers.yaml, issuer.yaml and invoice_defaults.yaml, their examples, and
// the template placeholders. The help pages print it.
package schema

// Field is one documented key, written as a dotted path.
type Field struct {
	Path        string
	Description string
}

// FieldGroup is a titled list of fields.
type FieldGroup struct {
	Title  string
	Fields []Field
}

// Placeholder is one template placeholder and what it is replaced with.
type Placeholder struct {
	Token       string
	Description string
}

// PlaceholderGroup is a titled list of placeholders.
type PlaceholderGroup struct {
	Title        string
	Placeholders []Placeholder
}

// TemplatePlaceholders lists the placeholders a template may use, grouped
// by what they show.
var TemplatePlaceholders = []PlaceholderGroup{
	{
		Title: "Issuer",
		Placeholders: []Placeholder{
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
		Placeholders: []Placeholder{
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
		Placeholders: []Placeholder{
			{Token: "@@INVOICE_NUMBER@@", Description: "invoice.number"},
			{Token: "@@ISSUE_DATE@@", Description: "invoice.issue_date formatted as DD.MM.YYYY"},
			{Token: "@@DUE_DATE@@", Description: "invoice.due_date formatted as DD.MM.YYYY"},
			{Token: "@@PERIOD_LABEL@@", Description: "invoice.period"},
		},
	},
	{
		Title: "Line items",
		Placeholders: []Placeholder{
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
		Placeholders: []Placeholder{
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
		Placeholders: []Placeholder{
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

// CustomerFields documents the fields of a customer in customers.yaml.
var CustomerFields = []FieldGroup{
	{
		Title: "Preferred fields",
		Fields: []Field{
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
		Fields: []Field{
			{Path: "<customer>.legal_company_name", Description: "Alternate path for the customer display name"},
			{Path: "<customer>.billing.email", Description: "Alternate path for the invoice email"},
			{Path: "<customer>.billing.contact_person", Description: "Alternate path for the customer contact"},
			{Path: "<customer>.billing.email_greeting", Description: "Alternate path for the email greeting"},
			{Path: "<customer>.currency", Description: "Alternate path for billing.currency"},
		},
	},
}

// IssuerFields documents the fields of issuer.yaml.
var IssuerFields = []FieldGroup{
	{
		Title: "Required company fields",
		Fields: []Field{
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
		Fields: []Field{
			{Path: "payment.bank_name", Description: "Bank name rendered into the invoice template"},
			{Path: "payment.iban", Description: "Bank account IBAN used on the invoice and for EPC QR generation"},
			{Path: "payment.bic", Description: "Bank identifier code shown on the invoice"},
			{Path: "payment.due_days", Description: "Non-negative integer day count used by `new` to prefill invoice.due_date"},
			{Path: "payment.payment_terms_text", Description: "Payment terms text used by templates and email placeholders"},
		},
	},
	{
		Title: "Optional payment fields",
		Fields: []Field{
			{Path: "payment.vat_label", Description: "Overrides the VAT label used by @@VAT_SUMMARY_ROWS@@, defaults to VAT"},
			{Path: "payment.epc_qr.label", Description: "Overrides the EPC QR label, defaults to Pay via EPC-QR"},
			{Path: "payment.epc_qr.name", Description: "Overrides the EPC QR recipient name, defaults to company.legal_company_name"},
			{Path: "payment.epc_qr.purpose", Description: "Optional EPC QR purpose code, must be 1-4 letters or digits"},
			{Path: "payment.epc_qr.text", Description: "Optional EPC QR text line, defaults to invoice.number"},
			{Path: "payment.epc_qr.information", Description: "Optional EPC QR unstructured remittance information"},
		},
	},
}

// InvoiceDefaultsFields documents the fields of invoice_defaults.yaml.
var InvoiceDefaultsFields = []FieldGroup{
	{
		Title: "Top-level keys",
		Fields: []Field{
			{Path: "invoice", Description: "Mapping of invoice defaults used as the source document for `new`"},
			{Path: "positions", Description: "Line-item list copied into the created invoice; if omitted `new` creates an empty list"},
		},
	},
	{
		Title: "Invoice fields",
		Fields: []Field{
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
		Fields: []Field{
			{Path: "positions[].name", Description: "Line-item name"},
			{Path: "positions[].description", Description: "Line-item description"},
			{Path: "positions[].unit_price", Description: "Line-item net unit price; must be >= 0 on the final invoice"},
			{Path: "positions[].quantity", Description: "Line-item quantity; must be > 0 on the final invoice"},
			{Path: "positions[].vat_percent", Description: "Optional per-line VAT override"},
		},
	},
	{
		Title: "Unsupported legacy keys",
		Fields: []Field{
			{Path: "line_items", Description: "Unsupported; use positions"},
			{Path: "invoice.period_label", Description: "Unsupported; use invoice.period"},
			{Path: "invoice.vat_rate_percent", Description: "Unsupported; use invoice.vat_percent"},
		},
	},
}

// CustomerYAMLExample is a customers.yaml with one customer.
const CustomerYAMLExample = `CUST-001:
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

// IssuerYAMLExample is an issuer.yaml.
const IssuerYAMLExample = `company:
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

// InvoiceDefaultsYAMLExample is an invoice_defaults.yaml.
const InvoiceDefaultsYAMLExample = `invoice:
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
