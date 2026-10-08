package invoice

import (
	"fmt"
	"strings"
)

// Bundle is an invoice with the customer and issuer it is checked against.
type Bundle struct {
	Invoice InvoiceFile
	// Customer is nil when the invoice names no customer that customers.yaml
	// has.
	Customer *Customer
	Issuer   IssuerFile
	// InvoicePath and IssuerPath name the files in problems about a whole
	// mapping.
	InvoicePath string
	IssuerPath  string
	// Undecoded reports a field whose value did not decode. Validate skips
	// it and every field inside it. Nil skips nothing.
	Undecoded func(field string) bool
}

// Validate returns every field of b that is missing or out of range, in
// file order: the invoice's own mappings, then the fields of the customer,
// the issuer's company, the invoice header, the issuer's payment and each
// position.
func Validate(b Bundle) []Problem {
	undecoded := b.Undecoded
	if undecoded == nil {
		undecoded = func(string) bool { return false }
	}
	var problems []Problem
	if b.Invoice.CustomerID.Trim() == "" && !undecoded("customer_id") {
		problems = append(problems, Problem{File: b.InvoicePath, Field: "customer_id", Message: "missing `customer_id`"})
	}
	header := b.Invoice.Invoice
	if header == nil {
		problems = append(problems, Problem{File: b.InvoicePath, Field: "invoice", Message: "missing `invoice` mapping"})
		header = &InvoiceHeader{}
	}
	company := b.Issuer.Company
	if company == nil {
		problems = append(problems, Problem{File: b.IssuerPath, Field: "issuer.company", Message: "missing `company` mapping"})
		company = &Company{}
	}
	payment := b.Issuer.Payment
	if payment == nil {
		problems = append(problems, Problem{File: b.IssuerPath, Field: "issuer.payment", Message: "missing `payment` mapping"})
		payment = &Payment{}
	}
	if len(b.Invoice.Positions) == 0 && !undecoded("positions") {
		problems = append(problems, Problem{File: b.InvoicePath, Field: "positions", Message: "`positions` must be a non-empty list"})
	}
	var fieldProblems []string
	if b.Customer != nil {
		fieldProblems = append(fieldProblems, b.Customer.validate()...)
	}
	fieldProblems = append(fieldProblems, company.validate()...)
	fieldProblems = append(fieldProblems, header.validate()...)
	fieldProblems = append(fieldProblems, payment.validate()...)

	var customerVATRate Rate
	if b.Customer != nil {
		customerVATRate = b.Customer.Tax.DefaultVATRate
	}
	// A rate that did not decode may be the one that applies, so a missing
	// rate is not reported while one did not decode.
	vatUndecoded := undecoded("invoice.vat_percent") || undecoded("customer.tax.default_vat_rate")
	missingVATReported := false
	for index, position := range b.Invoice.Positions {
		fieldProblems = append(fieldProblems, position.validate(index+1)...)
		rate := firstRate(position.VATPercent, header.VATPercent, customerVATRate)
		positionVATUndecoded := undecoded(fmt.Sprintf("positions[%d].vat_percent", index+1))
		if rate == nil && !missingVATReported && !vatUndecoded && !positionVATUndecoded {
			fieldProblems = append(fieldProblems, "invoice.vat_percent: missing value")
			missingVATReported = true
		}
	}
	// Every field problem starts with the path of its field.
	for _, problem := range fieldProblems {
		if field, message, _ := strings.Cut(problem, ": "); !undecoded(field) {
			problems = append(problems, Problem{Field: field, Message: message})
		}
	}
	return problems
}
