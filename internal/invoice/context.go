package invoice

import (
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Context is an invoice with its customer and issuer, validated, with its
// totals computed.
type Context struct {
	CustomerID       string
	Customer         Customer
	Company          Company
	Payment          Payment
	Invoice          InvoiceHeader
	LineItems        []LineItem
	Currency         string
	VATBreakdowns    []VATBreakdown
	SubtotalCents    int64
	VATAmountCents   int64
	TotalCents       int64
	PaidAmountCents  int64
	OutstandingCents int64
	CustomerEmail    string
	InvoiceNumber    string
}

type LineItem struct {
	Name           string
	Description    string
	UnitPrice      *big.Rat
	Quantity       *big.Rat
	VATRatePercent *big.Rat
	LineTotalCents int64
}

type VATBreakdown struct {
	RatePercent    *big.Rat
	NetCents       int64
	VATAmountCents int64
}

// LoadContext decodes the three files, validates them together and computes
// the invoice totals. It reports every problem at once: values that do not
// decode (*DecodeError, with file and line), then an unknown customer_id
// and the fields that are missing or out of range. Validation skips the
// fields that did not decode.
func LoadContext(customersPath, issuerPath, invoicePath string) (*Context, error) {
	customers, err := loadCustomerTable(customersPath)
	if err != nil {
		return nil, err
	}
	var issuer IssuerFile
	issuerErr := decodeYAMLFile(issuerPath, &issuer, true)
	if issuerErr != nil && !isDecodeError(issuerErr) {
		return nil, issuerErr
	}
	var invoiceFile InvoiceFile
	invoiceErr := decodeYAMLFile(invoicePath, &invoiceFile, true)
	if invoiceErr != nil && !isDecodeError(invoiceErr) {
		return nil, invoiceErr
	}

	var problems []string
	var unknownCustomer error
	customerID := invoiceFile.CustomerID.Trim()
	// customer stays nil when the invoice names no usable customer, so its
	// fields are not reported missing one by one.
	var customer *Customer
	var customerErr error
	if customerID == "" {
		if !within("customer_id", failedFields(invoiceErr)) {
			problems = append(problems, fmt.Sprintf("%s: missing `customer_id`", invoicePath))
		}
	} else if found, ok, err := customers.customer(customerID, true); !ok {
		unknownCustomer = &UnknownCustomerError{Path: invoicePath, CustomerID: customerID}
	} else {
		customer, customerErr = &found, err
	}
	decodeErr := errors.Join(customerErr, issuerErr, invoiceErr)
	failed := failedFields(decodeErr)

	header := invoiceFile.Invoice
	if header == nil {
		problems = append(problems, fmt.Sprintf("%s: missing `invoice` mapping", invoicePath))
		header = &InvoiceHeader{}
	}
	company := issuer.Company
	if company == nil {
		problems = append(problems, fmt.Sprintf("%s: missing `company` mapping", issuerPath))
		company = &Company{}
	}
	payment := issuer.Payment
	if payment == nil {
		problems = append(problems, fmt.Sprintf("%s: missing `payment` mapping", issuerPath))
		payment = &Payment{}
	}
	if len(invoiceFile.Positions) == 0 && !within("positions", failed) {
		problems = append(problems, fmt.Sprintf("%s: `positions` must be a non-empty list", invoicePath))
	}
	var fieldProblems []string
	if customer != nil {
		fieldProblems = append(fieldProblems, customer.validate()...)
	}
	fieldProblems = append(fieldProblems, company.validate()...)
	fieldProblems = append(fieldProblems, header.validate()...)
	fieldProblems = append(fieldProblems, payment.validate()...)

	var customerVATRate Rate
	if customer != nil {
		customerVATRate = customer.Tax.DefaultVATRate
	}
	// A rate that did not decode may be the one that applies, so a missing
	// rate is not reported while one did not decode.
	vatUndecoded := within("invoice.vat_percent", failed) || within("customer.tax.default_vat_rate", failed)
	items := make([]LineItem, 0, len(invoiceFile.Positions))
	missingVATReported := false
	for index, position := range invoiceFile.Positions {
		fieldProblems = append(fieldProblems, position.validate(index+1)...)
		rate := firstRate(position.VATPercent, header.VATPercent, customerVATRate)
		positionVATUndecoded := within(fmt.Sprintf("positions[%d].vat_percent", index+1), failed)
		if rate == nil && !missingVATReported && !vatUndecoded && !positionVATUndecoded {
			fieldProblems = append(fieldProblems, "invoice.vat_percent: missing value")
			missingVATReported = true
		}
		items = append(items, LineItem{
			Name:           string(position.Name),
			Description:    string(position.Description),
			UnitPrice:      position.UnitPrice.Rat(),
			Quantity:       position.Quantity.Rat(),
			VATRatePercent: rate,
		})
	}
	// Every field problem starts with the path of its field.
	for _, problem := range fieldProblems {
		if path, _, _ := strings.Cut(problem, ":"); !within(path, failed) {
			problems = append(problems, problem)
		}
	}

	var validationErr error
	if len(problems) > 0 {
		validationErr = errors.New(strings.Join(problems, "\n"))
	}
	if err := errors.Join(unknownCustomer, decodeErr, validationErr); err != nil {
		return nil, err
	}

	ctx := &Context{
		CustomerID:    customerID,
		Customer:      *customer,
		Company:       *company,
		Payment:       *payment,
		Invoice:       *header,
		Currency:      customer.BillingCurrency(),
		CustomerEmail: customer.InvoiceEmail(),
		InvoiceNumber: header.Number.Trim(),
	}
	// Validation bounded paid_amount, so it converts.
	ctx.PaidAmountCents, _ = moneyCents(header.PaidAmount.Rat())
	if err := ctx.computeTotals(items); err != nil {
		return nil, err
	}
	return ctx, nil
}

func isDecodeError(err error) bool {
	var decodeErr *DecodeError
	return errors.As(err, &decodeErr)
}

// firstRate returns the first rate that is set, nil when none is: the
// position's own, then the invoice's, then the customer's default.
func firstRate(rates ...Rate) *big.Rat {
	for _, rate := range rates {
		if rate.isSet() {
			return rate.Percent()
		}
	}
	return nil
}

// computeTotals sets the line totals, the VAT per rate and the invoice
// totals from items, whose prices, quantities and rates are validated.
func (ctx *Context) computeTotals(items []LineItem) error {
	var subtotalCents int64
	vatBuckets := make(map[string]*VATBreakdown, len(items))
	for index := range items {
		item := &items[index]
		lineTotal, ok := moneyCents(new(big.Rat).Mul(item.UnitPrice, item.Quantity))
		if !ok {
			return errAmountTooLarge(fmt.Sprintf("positions[%d]: unit_price × quantity", index+1))
		}
		if subtotalCents, ok = addMoneyCents(subtotalCents, lineTotal); !ok {
			return errAmountTooLarge("invoice subtotal")
		}
		item.LineTotalCents = lineTotal
		key := item.VATRatePercent.RatString()
		bucket, ok := vatBuckets[key]
		if !ok {
			bucket = &VATBreakdown{
				RatePercent: new(big.Rat).Set(item.VATRatePercent),
			}
			vatBuckets[key] = bucket
		}
		bucket.NetCents += lineTotal // bounded by the subtotal check above
	}

	vatBreakdowns := make([]VATBreakdown, 0, len(vatBuckets))
	var vatAmountCents int64
	for _, bucket := range vatBuckets {
		vatCents, ok := moneyCents(percentOfMoney(bucket.NetCents, bucket.RatePercent))
		if !ok {
			return errAmountTooLarge("invoice VAT amount")
		}
		bucket.VATAmountCents = vatCents
		if vatAmountCents, ok = addMoneyCents(vatAmountCents, vatCents); !ok {
			return errAmountTooLarge("invoice VAT amount")
		}
		vatBreakdowns = append(vatBreakdowns, *bucket)
	}
	sort.Slice(vatBreakdowns, func(left, right int) bool {
		return vatBreakdowns[left].RatePercent.Cmp(vatBreakdowns[right].RatePercent) < 0
	})

	totalCents, ok := addMoneyCents(subtotalCents, vatAmountCents)
	if !ok {
		return errAmountTooLarge("invoice total")
	}
	if ctx.PaidAmountCents > totalCents {
		return fmt.Errorf("invoice.paid_amount: `%s` exceeds total `%s`", FormatMoneyCents(ctx.PaidAmountCents), FormatMoneyCents(totalCents))
	}

	ctx.LineItems = items
	ctx.VATBreakdowns = vatBreakdowns
	ctx.SubtotalCents = subtotalCents
	ctx.VATAmountCents = vatAmountCents
	ctx.TotalCents = totalCents
	ctx.OutstandingCents = totalCents - ctx.PaidAmountCents
	return nil
}
