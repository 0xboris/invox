package invoice

import (
	"errors"
	"fmt"
	"math/big"
	"sort"

	"github.com/0xboris/invox/internal/money"
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

	var unknownCustomer error
	customerID := invoiceFile.CustomerID.Trim()
	// customer stays nil when the invoice names no usable customer, so its
	// fields are not reported missing one by one.
	var customer *Customer
	var customerErr error
	if customerID != "" {
		if found, ok, err := customers.customer(customerID, true); !ok {
			unknownCustomer = &UnknownCustomerError{Path: invoicePath, CustomerID: customerID}
		} else {
			customer, customerErr = &found, err
		}
	}
	decodeErr := errors.Join(customerErr, issuerErr, invoiceErr)
	failed := failedFields(decodeErr)

	problems := Validate(Bundle{
		Invoice:     invoiceFile,
		InvoicePath: invoicePath,
		Customer:    customer,
		Issuer:      issuer,
		IssuerPath:  issuerPath,
		Undecoded:   func(field string) bool { return within(field, failed) },
	})
	var validationErr error
	if len(problems) > 0 {
		validationErr = &ValidationError{Problems: problems}
	}
	if err := errors.Join(unknownCustomer, decodeErr, validationErr); err != nil {
		return nil, err
	}

	header := invoiceFile.Invoice
	items := make([]LineItem, 0, len(invoiceFile.Positions))
	for _, position := range invoiceFile.Positions {
		items = append(items, LineItem{
			Name:           string(position.Name),
			Description:    string(position.Description),
			UnitPrice:      position.UnitPrice.Rat(),
			Quantity:       position.Quantity.Rat(),
			VATRatePercent: firstRate(position.VATPercent, header.VATPercent, customer.Tax.DefaultVATRate),
		})
	}
	company, payment := issuer.Company, issuer.Payment
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
	ctx.PaidAmountCents, _ = money.Cents(header.PaidAmount.Rat())
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
		lineTotal, ok := money.Cents(new(big.Rat).Mul(item.UnitPrice, item.Quantity))
		if !ok {
			return errAmountTooLarge(fmt.Sprintf("positions[%d]: unit_price × quantity", index+1))
		}
		if subtotalCents, ok = money.AddCents(subtotalCents, lineTotal); !ok {
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
		vatCents, ok := money.Cents(money.PercentOf(bucket.NetCents, bucket.RatePercent))
		if !ok {
			return errAmountTooLarge("invoice VAT amount")
		}
		bucket.VATAmountCents = vatCents
		if vatAmountCents, ok = money.AddCents(vatAmountCents, vatCents); !ok {
			return errAmountTooLarge("invoice VAT amount")
		}
		vatBreakdowns = append(vatBreakdowns, *bucket)
	}
	sort.Slice(vatBreakdowns, func(left, right int) bool {
		return vatBreakdowns[left].RatePercent.Cmp(vatBreakdowns[right].RatePercent) < 0
	})

	totalCents, ok := money.AddCents(subtotalCents, vatAmountCents)
	if !ok {
		return errAmountTooLarge("invoice total")
	}
	if ctx.PaidAmountCents > totalCents {
		return fmt.Errorf("invoice.paid_amount: `%s` exceeds total `%s`", money.FormatCents(ctx.PaidAmountCents), money.FormatCents(totalCents))
	}

	ctx.LineItems = items
	ctx.VATBreakdowns = vatBreakdowns
	ctx.SubtotalCents = subtotalCents
	ctx.VATAmountCents = vatAmountCents
	ctx.TotalCents = totalCents
	ctx.OutstandingCents = totalCents - ctx.PaidAmountCents
	return nil
}
