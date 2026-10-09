package invoice

import (
	"fmt"

	"github.com/0xboris/invox/internal/money"
)

// The schema of customers.yaml, issuer.yaml and invoice files. store maps
// their YAML keys to these fields.

// Customer is one entry of customers.yaml.
type Customer struct {
	Name             Text
	LegalCompanyName Text
	Status           Text
	Email            Text
	EmailGreeting    Text
	ContactPerson    Text
	Currency         Text
	Address          Address
	Tax              CustomerTax
	Billing          CustomerBilling
	Numbering        CustomerNumbering
}

type Address struct {
	Street     Text
	PostalCode Text
	City       Text
	Country    Text
}

type CustomerTax struct {
	VATTaxID       Text
	DefaultVATRate Rate
}

type CustomerBilling struct {
	SendInvoiceTo Text
	Email         Text
	ContactPerson Text
	EmailGreeting Text
	Currency      Text
}

type CustomerNumbering struct {
	Code  Text
	Start Count
}

// Issuer is issuer.yaml. Company and Payment are nil when the file does not
// have them.
type Issuer struct {
	Company *Company
	Payment *Payment
}

type Company struct {
	LegalCompanyName          Text
	CompanyRegistrationNumber Text
	VATTaxID                  Text
	Website                   Text
	Email                     Text
	Address                   Address
}

type Payment struct {
	BankName         Text
	IBAN             Text
	BIC              Text
	DueDays          Count
	PaymentTermsText Text
	VATLabel         Text
	EPCQR            EPCQR
}

type EPCQR struct {
	Label       Text
	Name        Text
	Purpose     Text
	Text        Text
	Information Text
}

// Invoice is an invoice: a working file, invoice_defaults.yaml, or an
// archived invoice. Header is nil when the file has no `invoice` mapping,
// and Archive when it has no `_invox` mapping.
type Invoice struct {
	CustomerID Text
	Header     *Header
	Positions  []Position
	Archive    *ArchiveLink
}

type Header struct {
	Number     Text
	IssueDate  Date
	DueDate    Date
	Status     Status
	Period     Text
	VATPercent Rate
	PaidAmount Decimal
}

type Position struct {
	Name        Text
	Description Text
	UnitPrice   Decimal
	Quantity    Decimal
	VATPercent  Rate
}

// ArchiveLink is the `_invox` mapping of a working copy made by `archive
// edit`: the archived file it replaces when it is archived again.
type ArchiveLink struct {
	ArchivePath Text
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

// Contact is billing.contact_person, else contact_person.
func (c Customer) Contact() string {
	return firstText(c.Billing.ContactPerson, c.ContactPerson)
}

// Greeting is billing.email_greeting, else email_greeting, else "Hello,".
func (c Customer) Greeting() string {
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

// VATName is vat_label, else VAT.
func (p Payment) VATName() string {
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
		requiredField{"address.street", c.Address.Street.IsSet()},
		requiredField{"address.postal_code", c.Address.PostalCode.IsSet()},
		requiredField{"address.city", c.Address.City.IsSet()},
		requiredField{"address.country", c.Address.Country.IsSet()},
		requiredField{"tax.vat_tax_id", c.Tax.VATTaxID.IsSet()},
	)
	if rate := c.Tax.DefaultVATRate.Percent(); rate != nil && rate.Sign() < 0 {
		problems = append(problems, "customer.tax.default_vat_rate: must be >= 0")
	}
	return problems
}

func (c Company) validate() []string {
	return missingFields("issuer.company",
		requiredField{"legal_company_name", c.LegalCompanyName.IsSet()},
		requiredField{"company_registration_number", c.CompanyRegistrationNumber.IsSet()},
		requiredField{"vat_tax_id", c.VATTaxID.IsSet()},
		requiredField{"website", c.Website.IsSet()},
		requiredField{"email", c.Email.IsSet()},
		requiredField{"address.street", c.Address.Street.IsSet()},
		requiredField{"address.postal_code", c.Address.PostalCode.IsSet()},
		requiredField{"address.city", c.Address.City.IsSet()},
		requiredField{"address.country", c.Address.Country.IsSet()},
	)
}

func (h Header) validate() []string {
	problems := missingFields("invoice",
		requiredField{"number", h.Number.IsSet()},
		requiredField{"issue_date", h.IssueDate.IsSet()},
		requiredField{"due_date", h.DueDate.IsSet()},
		requiredField{"period", h.Period.IsSet()},
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
		requiredField{"bank_name", p.BankName.IsSet()},
		requiredField{"iban", p.IBAN.IsSet()},
		requiredField{"bic", p.BIC.IsSet()},
		requiredField{"due_days", p.DueDays.IsSet()},
		requiredField{"payment_terms_text", p.PaymentTermsText.IsSet()},
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
		requiredField{"name", p.Name.IsSet()},
		requiredField{"description", p.Description.IsSet()},
		requiredField{"unit_price", p.UnitPrice.IsSet()},
		requiredField{"quantity", p.Quantity.IsSet()},
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
