package invoice

import (
	"fmt"

	"github.com/0xboris/invox/internal/money"
)

// The schema of customers.yaml, issuer.yaml and invoice files. Each struct
// maps YAML keys to fields through yaml tags; decodeYAMLNode fills them.

// Customer is one entry of customers.yaml.
type Customer struct {
	Name             Text              `yaml:"name"`
	LegalCompanyName Text              `yaml:"legal_company_name"`
	Status           Text              `yaml:"status"`
	Email            Text              `yaml:"email"`
	EmailGreeting    Text              `yaml:"email_greeting"`
	ContactPerson    Text              `yaml:"contact_person"`
	Currency         Text              `yaml:"currency"`
	Address          Address           `yaml:"address"`
	Tax              CustomerTax       `yaml:"tax"`
	Billing          CustomerBilling   `yaml:"billing"`
	Numbering        CustomerNumbering `yaml:"numbering"`
}

type Address struct {
	Street     Text `yaml:"street"`
	PostalCode Text `yaml:"postal_code"`
	City       Text `yaml:"city"`
	Country    Text `yaml:"country"`
}

type CustomerTax struct {
	VATTaxID       Text `yaml:"vat_tax_id"`
	DefaultVATRate Rate `yaml:"default_vat_rate"`
}

type CustomerBilling struct {
	SendInvoiceTo Text `yaml:"send_invoice_to"`
	Email         Text `yaml:"email"`
	ContactPerson Text `yaml:"contact_person"`
	EmailGreeting Text `yaml:"email_greeting"`
	Currency      Text `yaml:"currency"`
}

type CustomerNumbering struct {
	Code  Text  `yaml:"code"`
	Start Count `yaml:"start"`
}

// IssuerFile is issuer.yaml. Company and Payment are nil when the file does not
// have them.
type IssuerFile struct {
	Company *Company `yaml:"company"`
	Payment *Payment `yaml:"payment"`
}

type Company struct {
	LegalCompanyName          Text    `yaml:"legal_company_name"`
	CompanyRegistrationNumber Text    `yaml:"company_registration_number"`
	VATTaxID                  Text    `yaml:"vat_tax_id"`
	Website                   Text    `yaml:"website"`
	Email                     Text    `yaml:"email"`
	Address                   Address `yaml:"address"`
}

type Payment struct {
	BankName         Text  `yaml:"bank_name"`
	IBAN             Text  `yaml:"iban"`
	BIC              Text  `yaml:"bic"`
	DueDays          Count `yaml:"due_days"`
	PaymentTermsText Text  `yaml:"payment_terms_text"`
	VATLabel         Text  `yaml:"vat_label"`
	EPCQR            EPCQR `yaml:"epc_qr"`
}

type EPCQR struct {
	Label       Text `yaml:"label"`
	Name        Text `yaml:"name"`
	Purpose     Text `yaml:"purpose"`
	Text        Text `yaml:"text"`
	Information Text `yaml:"information"`
}

// InvoiceFile is an invoice: a working file, invoice_defaults.yaml, or an
// archived invoice. Invoice is nil when the file has no `invoice` mapping.
type InvoiceFile struct {
	CustomerID Text           `yaml:"customer_id"`
	Invoice    *InvoiceHeader `yaml:"invoice"`
	Positions  []Position     `yaml:"positions"`
	Archive    ArchiveLink    `yaml:"_invox"`
	LineItems  removedKey     `yaml:"line_items" replacement:"positions"`
}

type InvoiceHeader struct {
	Number         Text       `yaml:"number"`
	IssueDate      Date       `yaml:"issue_date"`
	DueDate        Date       `yaml:"due_date"`
	Status         Text       `yaml:"status"`
	Period         Text       `yaml:"period"`
	VATPercent     Rate       `yaml:"vat_percent"`
	PaidAmount     Decimal    `yaml:"paid_amount"`
	PeriodLabel    removedKey `yaml:"period_label" replacement:"invoice.period"`
	VATRatePercent removedKey `yaml:"vat_rate_percent" replacement:"invoice.vat_percent"`
}

type Position struct {
	Name        Text    `yaml:"name"`
	Description Text    `yaml:"description"`
	UnitPrice   Decimal `yaml:"unit_price"`
	Quantity    Decimal `yaml:"quantity"`
	VATPercent  Rate    `yaml:"vat_percent"`
}

// ArchiveLink is the `_invox` mapping of a working copy made by `archive
// edit`: the archived file it replaces when it is archived again.
type ArchiveLink struct {
	ArchivePath        Text `yaml:"archive_path"`
	ArchiveReplacePath Text `yaml:"archive_replace_path"`
}

// invoiceIdentity is the part of an invoice that numbering and the archive
// read from every file they scan. It is decoded leniently, so files with
// keys invox no longer knows still count.
type invoiceIdentity struct {
	CustomerID Text `yaml:"customer_id"`
	Invoice    *struct {
		Number    Text `yaml:"number"`
		IssueDate Text `yaml:"issue_date"`
		Status    Text `yaml:"status"`
	} `yaml:"invoice"`
}

const (
	defaultCustomerCurrency = "EUR"
	defaultVATLabel         = "VAT"
)

// DisplayName is the name invoices and emails show: name, else
// legal_company_name.
func (c Customer) DisplayName() string {
	return firstText(c.Name, c.LegalCompanyName)
}

// InvoiceEmail is where invoices go: billing.send_invoice_to, else
// billing.email, else email.
func (c Customer) InvoiceEmail() string {
	return firstText(c.Billing.SendInvoiceTo, c.Billing.Email, c.Email)
}

func (c Customer) contactPerson() string {
	return firstText(c.Billing.ContactPerson, c.ContactPerson)
}

func (c Customer) emailGreeting() string {
	if greeting := firstText(c.Billing.EmailGreeting, c.EmailGreeting); greeting != "" {
		return greeting
	}
	return "Hello,"
}

// BillingCurrency is billing.currency, else currency, else EUR.
func (c Customer) BillingCurrency() string {
	if currency := firstText(c.Billing.Currency, c.Currency); currency != "" {
		return currency
	}
	return defaultCustomerCurrency
}

func (p Payment) vatLabel() string {
	if label := p.VATLabel.Trim(); label != "" {
		return label
	}
	return defaultVATLabel
}

func firstText(values ...Text) string {
	for _, value := range values {
		if text := value.Trim(); text != "" {
			return text
		}
	}
	return ""
}

// requiredField is a field a check reports as missing when it is not set.
type requiredField struct {
	path string
	set  bool
}

func missingFields(prefix string, fields ...requiredField) []string {
	var problems []string
	for _, field := range fields {
		if !field.set {
			problems = append(problems, fmt.Sprintf("%s.%s: missing value", prefix, field.path))
		}
	}
	return problems
}

func (c Customer) validate() []string {
	problems := missingFields("customer",
		requiredField{"name", c.DisplayName() != ""},
		requiredField{"email", c.InvoiceEmail() != ""},
		requiredField{"address.street", c.Address.Street.isSet()},
		requiredField{"address.postal_code", c.Address.PostalCode.isSet()},
		requiredField{"address.city", c.Address.City.isSet()},
		requiredField{"address.country", c.Address.Country.isSet()},
		requiredField{"tax.vat_tax_id", c.Tax.VATTaxID.isSet()},
	)
	if rate := c.Tax.DefaultVATRate.Percent(); rate != nil && rate.Sign() < 0 {
		problems = append(problems, "customer.tax.default_vat_rate: must be >= 0")
	}
	return problems
}

func (c Company) validate() []string {
	return missingFields("issuer.company",
		requiredField{"legal_company_name", c.LegalCompanyName.isSet()},
		requiredField{"company_registration_number", c.CompanyRegistrationNumber.isSet()},
		requiredField{"vat_tax_id", c.VATTaxID.isSet()},
		requiredField{"website", c.Website.isSet()},
		requiredField{"email", c.Email.isSet()},
		requiredField{"address.street", c.Address.Street.isSet()},
		requiredField{"address.postal_code", c.Address.PostalCode.isSet()},
		requiredField{"address.city", c.Address.City.isSet()},
		requiredField{"address.country", c.Address.Country.isSet()},
	)
}

func (h InvoiceHeader) validate() []string {
	problems := missingFields("invoice",
		requiredField{"number", h.Number.isSet()},
		requiredField{"issue_date", h.IssueDate.isSet()},
		requiredField{"due_date", h.DueDate.isSet()},
		requiredField{"period", h.Period.isSet()},
	)
	if paid := h.PaidAmount.Rat(); paid != nil {
		if paid.Sign() < 0 {
			problems = append(problems, "invoice.paid_amount: must not be negative")
		} else if _, ok := money.Cents(paid); !ok {
			problems = append(problems, errAmountTooLarge("invoice.paid_amount:").Error())
		}
	}
	if rate := h.VATPercent.Percent(); rate != nil && rate.Sign() < 0 {
		problems = append(problems, "invoice.vat_percent: must be >= 0")
	}
	return problems
}

func (p Payment) validate() []string {
	problems := missingFields("issuer.payment",
		requiredField{"bank_name", p.BankName.isSet()},
		requiredField{"iban", p.IBAN.isSet()},
		requiredField{"bic", p.BIC.isSet()},
		requiredField{"due_days", p.DueDays.isSet()},
		requiredField{"payment_terms_text", p.PaymentTermsText.isSet()},
	)
	if p.DueDays.Int() < 0 {
		problems = append(problems, "issuer.payment.due_days: must be >= 0")
	}
	return problems
}

// validate checks the position at index (from 1) in positions.
func (p Position) validate(index int) []string {
	prefix := fmt.Sprintf("positions[%d]", index)
	problems := missingFields(prefix,
		requiredField{"name", p.Name.isSet()},
		requiredField{"description", p.Description.isSet()},
		requiredField{"unit_price", p.UnitPrice.isSet()},
		requiredField{"quantity", p.Quantity.isSet()},
	)
	if price := p.UnitPrice.Rat(); price != nil {
		if price.Sign() < 0 {
			problems = append(problems, prefix+".unit_price: must be >= 0")
		} else if _, ok := money.Cents(price); !ok {
			problems = append(problems, errAmountTooLarge(prefix+".unit_price:").Error())
		}
	}
	if quantity := p.Quantity.Rat(); quantity != nil && quantity.Sign() <= 0 {
		problems = append(problems, prefix+".quantity: must be > 0")
	}
	if rate := p.VATPercent.Percent(); rate != nil && rate.Sign() < 0 {
		problems = append(problems, prefix+".vat_percent: must be >= 0")
	}
	return problems
}
