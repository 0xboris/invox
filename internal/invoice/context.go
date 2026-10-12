package invoice

import (
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
	Header           Header
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

// firstRate returns the first rate that is set, nil when none is: the
// position's own, then the invoice's, then the customer's default.
func firstRate(rates ...Rate) *big.Rat {
	for _, rate := range rates {
		if rate.IsSet() {
			return rate.Percent()
		}
	}
	return nil
}

// NewContext computes the totals of b, which Validate found no problems
// with. Its customer and its issuer's company and payment are set.
func NewContext(b Bundle) (*Context, error) {
	header, customer := b.Invoice.Header, b.Customer
	items := make([]LineItem, 0, len(b.Invoice.Positions))
	for _, position := range b.Invoice.Positions {
		items = append(items, LineItem{
			Name:           string(position.Name),
			Description:    string(position.Description),
			UnitPrice:      position.UnitPrice.Rat(),
			Quantity:       position.Quantity.Rat(),
			VATRatePercent: firstRate(position.VATPercent, header.VATPercent, customer.Tax.DefaultVATRate),
		})
	}
	ctx := &Context{
		CustomerID:    b.Invoice.CustomerID.Trim(),
		Customer:      *customer,
		Company:       *b.Issuer.Company,
		Payment:       *b.Issuer.Payment,
		Header:        *header,
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
