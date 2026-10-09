package store

import (
	"reflect"

	"github.com/0xboris/invox/internal/invoice"
)

// invoiceIdentity is the part of an invoice that numbering and the archive
// read from every file they scan. It is decoded leniently, so files with
// keys invox no longer knows still count.
type invoiceIdentity struct {
	CustomerID invoice.Text `yaml:"customer_id"`
	Invoice    *struct {
		Number    invoice.Text `yaml:"number"`
		IssueDate invoice.Text `yaml:"issue_date"`
		Status    invoice.Text `yaml:"status"`
	} `yaml:"invoice"`
}

// schemaKeys maps the YAML keys of each schema struct to the names of its
// fields. A pointer field stays nil when its key is missing or null.
var schemaKeys = map[reflect.Type]map[string]string{
	reflect.TypeFor[invoice.Customer](): {
		"name":               "Name",
		"legal_company_name": "LegalCompanyName",
		"status":             "Status",
		"email":              "Email",
		"email_greeting":     "EmailGreeting",
		"contact_person":     "ContactPerson",
		"currency":           "Currency",
		"address":            "Address",
		"tax":                "Tax",
		"billing":            "Billing",
		"numbering":          "Numbering",
	},
	reflect.TypeFor[invoice.Address](): {
		"street":      "Street",
		"postal_code": "PostalCode",
		"city":        "City",
		"country":     "Country",
	},
	reflect.TypeFor[invoice.CustomerTax](): {
		"vat_tax_id":       "VATTaxID",
		"default_vat_rate": "DefaultVATRate",
	},
	reflect.TypeFor[invoice.CustomerBilling](): {
		"send_invoice_to": "SendInvoiceTo",
		"email":           "Email",
		"contact_person":  "ContactPerson",
		"email_greeting":  "EmailGreeting",
		"currency":        "Currency",
	},
	reflect.TypeFor[invoice.CustomerNumbering](): {
		"code":  "Code",
		"start": "Start",
	},
	reflect.TypeFor[invoice.Issuer](): {
		"company": "Company",
		"payment": "Payment",
	},
	reflect.TypeFor[invoice.Company](): {
		"legal_company_name":          "LegalCompanyName",
		"company_registration_number": "CompanyRegistrationNumber",
		"vat_tax_id":                  "VATTaxID",
		"website":                     "Website",
		"email":                       "Email",
		"address":                     "Address",
	},
	reflect.TypeFor[invoice.Payment](): {
		"bank_name":          "BankName",
		"iban":               "IBAN",
		"bic":                "BIC",
		"due_days":           "DueDays",
		"payment_terms_text": "PaymentTermsText",
		"vat_label":          "VATLabel",
		"epc_qr":             "EPCQR",
	},
	reflect.TypeFor[invoice.EPCQR](): {
		"label":       "Label",
		"name":        "Name",
		"purpose":     "Purpose",
		"text":        "Text",
		"information": "Information",
	},
	reflect.TypeFor[invoice.Invoice](): {
		"customer_id": "CustomerID",
		"invoice":     "Header",
		"positions":   "Positions",
		"_invox":      "Archive",
	},
	reflect.TypeFor[invoice.Header](): {
		"number":      "Number",
		"issue_date":  "IssueDate",
		"due_date":    "DueDate",
		"status":      "Status",
		"period":      "Period",
		"vat_percent": "VATPercent",
		"paid_amount": "PaidAmount",
	},
	reflect.TypeFor[invoice.Position](): {
		"name":        "Name",
		"description": "Description",
		"unit_price":  "UnitPrice",
		"quantity":    "Quantity",
		"vat_percent": "VATPercent",
	},
	reflect.TypeFor[invoice.ArchiveLink](): {
		"archive_path": "ArchivePath",
	},
}
