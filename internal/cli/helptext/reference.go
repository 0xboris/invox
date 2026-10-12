// Package helptext holds the help content that several commands and help
// topics share: the topic pages (topics/*.tmpl), the reference tables for
// the support files and templates with their YAML examples
// (reference.tmpl), and the Default lookup lines.
package helptext

// A group is one titled block of a reference table.
type group struct {
	Title string
	Rows  []row
}

// A row names a field or placeholder and says what it holds.
type row struct {
	Name        string
	Description string
}

var customerFields = []group{
	{"Preferred fields", []row{
		{"<customer>.name", "Preferred display name used on invoices and in emails"},
		{"<customer>.status", "Optional status shown by customer list"},
		{"<customer>.contact_person", "Preferred contact used by email templates"},
		{"<customer>.email_greeting", "Preferred greeting used by email templates"},
		{"<customer>.email", "Invoice email when billing.send_invoice_to is unset"},
		{"<customer>.address.street", "Billing address street"},
		{"<customer>.address.postal_code", "Billing address postal code"},
		{"<customer>.address.city", "Billing address city"},
		{"<customer>.address.country", "Billing address country"},
		{"<customer>.tax.vat_tax_id", "VAT number shown on the invoice"},
		{"<customer>.tax.default_vat_rate", "Optional default VAT used by new/validate/render/build/email"},
		{"<customer>.billing.send_invoice_to", "Preferred invoice-recipient email override"},
		{"<customer>.billing.currency", "Optional billing currency, defaults to EUR"},
		{"<customer>.numbering.code", "Value used by {customer_code}"},
		{"<customer>.numbering.start", "Override numbering.start for this customer"},
	}},
	{"Alternate supported paths", []row{
		{"<customer>.legal_company_name", "Alternate path for the customer display name"},
		{"<customer>.billing.email", "Alternate path for the invoice email"},
		{"<customer>.billing.contact_person", "Alternate path for the customer contact"},
		{"<customer>.billing.email_greeting", "Alternate path for the email greeting"},
		{"<customer>.currency", "Alternate path for billing.currency"},
	}},
}

var issuerFields = []group{
	{"Required company fields", []row{
		{"company.legal_company_name", "Company name used on invoices, in email placeholders, and as the default EPC QR recipient name"},
		{"company.company_registration_number", "Company registration number shown on the invoice"},
		{"company.vat_tax_id", "VAT/tax number shown on the invoice"},
		{"company.website", "Website shown on the invoice"},
		{"company.email", "Sender/contact email shown on the invoice"},
		{"company.address.street", "Business address street"},
		{"company.address.postal_code", "Business address postal code"},
		{"company.address.city", "Business address city"},
		{"company.address.country", "Business address country"},
	}},
	{"Required payment fields", []row{
		{"payment.bank_name", "Bank name rendered into the invoice template"},
		{"payment.iban", "Bank account IBAN used on the invoice and for EPC QR generation"},
		{"payment.bic", "Bank identifier code shown on the invoice"},
		{"payment.due_days", "Non-negative integer day count used by `new` to prefill invoice.due_date"},
		{"payment.payment_terms_text", "Payment terms text used by templates and email placeholders"},
	}},
	{"Optional payment fields", []row{
		{"payment.vat_label", "Overrides the VAT label used by @@VAT_SUMMARY_ROWS@@, defaults to VAT"},
		{"payment.epc_qr.label", "Overrides the EPC QR label, defaults to Pay via EPC-QR"},
		{"payment.epc_qr.name", "Overrides the EPC QR recipient name, defaults to company.legal_company_name"},
		{"payment.epc_qr.purpose", "Optional EPC QR purpose code, must be 1-4 letters or digits"},
		{"payment.epc_qr.text", "Optional EPC QR text line, defaults to invoice.number"},
		{"payment.epc_qr.information", "Optional EPC QR unstructured remittance information"},
	}},
}

var defaultsFields = []group{
	{"Top-level keys", []row{
		{"invoice", "Mapping of invoice defaults used as the source document for `new`"},
		{"positions", "Line-item list copied into the created invoice; if omitted `new` creates an empty list"},
	}},
	{"Invoice fields", []row{
		{"invoice.number", "Usually blank in defaults; `new` always replaces it with the next generated invoice number"},
		{"invoice.issue_date", "Usually blank in defaults; `new` always replaces it with the current date"},
		{"invoice.due_date", "Usually blank in defaults; `new` always replaces it using issuer.payment.due_days"},
		{"invoice.status", "Usually `draft`; `new` always resets it to `draft`"},
		{"invoice.period", "Invoice period label copied into the created invoice and required by validate/render/build"},
		{"invoice.vat_percent", "Optional default VAT rate for the whole invoice; can be filled from customer.tax.default_vat_rate"},
		{"invoice.paid_amount", "Usually `0`; must be >= 0 and <= the invoice total; `new` always resets it to `0`"},
	}},
	{"Position fields", []row{
		{"positions[].name", "Line-item name"},
		{"positions[].description", "Line-item description"},
		{"positions[].unit_price", "Line-item net unit price; must be >= 0 on the final invoice"},
		{"positions[].quantity", "Line-item quantity; must be > 0 on the final invoice"},
		{"positions[].vat_percent", "Optional per-line VAT override"},
	}},
}

var templatePlaceholders = []group{
	{"Issuer", []row{
		{"@@ISSUER_NAME@@", "issuer.company.legal_company_name"},
		{"@@ISSUER_COMPANY_REG_NO@@", "issuer.company.company_registration_number"},
		{"@@ISSUER_VAT_TAX_ID@@", "issuer.company.vat_tax_id"},
		{"@@ISSUER_WEBSITE@@", "issuer.company.website"},
		{"@@ISSUER_EMAIL@@", "issuer.company.email"},
		{"@@ISSUER_STREET@@", "issuer.company.address.street"},
		{"@@ISSUER_CITY@@", "issuer.company.address.city"},
		{"@@ISSUER_POSTAL_CODE@@", "issuer.company.address.postal_code"},
		{"@@ISSUER_COUNTRY@@", "issuer.company.address.country"},
	}},
	{"Customer", []row{
		{"@@CUSTOMER_NAME@@", "Preferred customer name"},
		{"@@CUSTOMER_STREET@@", "customer.address.street"},
		{"@@CUSTOMER_CITY@@", "customer.address.city"},
		{"@@CUSTOMER_POSTAL_CODE@@", "customer.address.postal_code"},
		{"@@CUSTOMER_COUNTRY@@", "customer.address.country"},
		{"@@CUSTOMER_VAT_TAX_ID@@", "customer.tax.vat_tax_id"},
		{"@@CUSTOMER_EMAIL@@", "Preferred invoice email"},
	}},
	{"Invoice metadata", []row{
		{"@@INVOICE_NUMBER@@", "invoice.number"},
		{"@@ISSUE_DATE@@", "invoice.issue_date formatted as DD.MM.YYYY"},
		{"@@DUE_DATE@@", "invoice.due_date formatted as DD.MM.YYYY"},
		{"@@PERIOD_LABEL@@", "invoice.period"},
	}},
	{"Line items", []row{
		{"@@LINE_ITEMS_ROWS@@", "Structured rows: name, description, unit price, quantity, line total"},
		{"@@LINE_ITEMS_ROWS_WITH_VAT@@", "Structured rows: name, description, unit price, quantity, VAT rate, line total"},
		{"@@LINE_ITEMS_BEGIN@@", "Begin custom line-item block; repeat enclosed snippet once per position"},
		{"@@LINE_ITEMS_END@@", "End custom line-item block"},
		{"@@LINE_ITEM_NAME@@", "Line-item name; only inside @@LINE_ITEMS_BEGIN@@ ... @@LINE_ITEMS_END@@"},
		{"@@LINE_ITEM_DESCRIPTION@@", "Line-item description; only inside the custom line-item block"},
		{"@@LINE_ITEM_UNIT_PRICE@@", "Formatted unit price with 2 to 4 decimals, as many as it needs (rounded half up beyond 4); only inside the custom line-item block"},
		{"@@LINE_ITEM_QUANTITY@@", "Formatted quantity; only inside the custom line-item block"},
		{"@@LINE_ITEM_VAT_RATE@@", "Formatted effective VAT rate; only inside the custom line-item block"},
		{"@@LINE_ITEM_LINE_TOTAL@@", "Formatted line total; only inside the custom line-item block"},
		{"@@LINE_ITEM_RULE@@", "Line separator rule; only inside the custom line-item block"},
	}},
	{"Totals", []row{
		{"@@SUBTOTAL@@", "Formatted net subtotal"},
		{"@@VAT_SUMMARY_ROWS@@", "Structured VAT summary rows, one row per VAT bucket"},
		{"@@TOTAL@@", "Formatted invoice total"},
		{"@@PAID_AMOUNT@@", "Formatted paid amount"},
		{"@@OUTSTANDING_AMOUNT@@", "Formatted outstanding amount"},
		{"@@INVOICE_TOTAL@@", "Alias for @@TOTAL@@"},
		{"@@OUTSTANDING_TOTAL@@", "Alias for @@OUTSTANDING_AMOUNT@@"},
	}},
	{"Payment", []row{
		{"@@PAYMENT_TERMS_TEXT@@", "issuer.payment.payment_terms_text"},
		{"@@VAT_LABEL@@", "VAT label from issuer.payment.vat_label, defaults to VAT"},
		{"@@BANK_NAME@@", "issuer.payment.bank_name"},
		{"@@IBAN@@", "issuer.payment.iban"},
		{"@@BIC@@", "issuer.payment.bic"},
		{"@@EPC_QR_AVAILABLE@@", "1 when an EPC QR code will be rendered, otherwise 0"},
		{"@@EPC_QR_LABEL@@", "EPC QR label text"},
		{"@@EPC_QR_CODE@@", "EPC QR code placeholder"},
	}},
}

// TemplatePlaceholders lists the placeholders the template help documents,
// in the order it shows them.
func TemplatePlaceholders() []string {
	var names []string
	for _, group := range templatePlaceholders {
		for _, row := range group.Rows {
			names = append(names, row.Name)
		}
	}
	return names
}
