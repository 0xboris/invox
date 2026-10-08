package validate

import "github.com/0xboris/invox/internal/invoice"

// formatMoney is the terminal form of an amount. invoice.FormatCurrency is
// LaTeX (\euro) and belongs in rendered invoices only.
func formatMoney(cents int64, currency string) string {
	if currency == "EUR" {
		return invoice.FormatMoneyCents(cents) + " €"
	}
	return invoice.FormatMoneyCents(cents) + " " + currency
}
