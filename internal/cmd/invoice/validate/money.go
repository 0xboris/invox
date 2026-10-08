package validate

import "github.com/0xboris/invox/internal/money"

// formatMoney is the terminal form of an amount. invoice.FormatCurrency is
// LaTeX (\euro) and belongs in rendered invoices only.
func formatMoney(cents int64, currency string) string {
	if currency == "EUR" {
		return money.FormatCents(cents) + " €"
	}
	return money.FormatCents(cents) + " " + currency
}
