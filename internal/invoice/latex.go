package invoice

import (
	"fmt"
	"math/big"
	"strings"
)

func FormatCurrency(cents int64, currency string) string {
	return withCurrency(FormatMoneyCents(cents), currency)
}

// Unit prices are shown with as many decimals as they need, at least
// minUnitPriceDecimals and at most maxUnitPriceDecimals (rounded half up beyond
// that), so that unit price × quantity matches the line total.
const (
	minUnitPriceDecimals = 2
	maxUnitPriceDecimals = 4
)

func formatUnitPrice(value *big.Rat, currency string) string {
	decimals := unitPriceDecimals(value)
	if decimals == minUnitPriceDecimals {
		return FormatCurrency(quantizeMoney(value), currency)
	}
	scale := int64(1)
	for range decimals {
		scale *= 10
	}
	units := roundHalfUpToInt(new(big.Rat).Mul(value, new(big.Rat).SetInt64(scale)))
	sign := ""
	if units < 0 {
		sign = "-"
		units = -units
	}
	formatted := fmt.Sprintf("%s%s,%0*d", sign, groupThousands(units/scale), decimals, units%scale)
	return withCurrency(formatted, currency)
}

func unitPriceDecimals(value *big.Rat) int {
	if value == nil {
		return minUnitPriceDecimals
	}
	scaled := new(big.Rat).Set(value)
	scaled.Mul(scaled, big.NewRat(100, 1))
	for decimals := minUnitPriceDecimals; decimals < maxUnitPriceDecimals; decimals++ {
		if scaled.IsInt() {
			return decimals
		}
		scaled.Mul(scaled, big.NewRat(10, 1))
	}
	return maxUnitPriceDecimals
}

func withCurrency(formatted, currency string) string {
	if currency == "EUR" {
		return formatted + " \\euro"
	}
	return formatted + " " + latexEscape(currency)
}

func renderQRCodePayload(payload []byte) string {
	var rendered strings.Builder
	rendered.WriteString("{%\n")
	for value := 0x80; value <= 0xFF; value++ {
		fmt.Fprintf(&rendered, "\\catcode`\\^^%02x=12\\relax\n", value)
	}
	rendered.WriteString(`\edef\invoxqrcodepayload{`)
	rendered.WriteString(qrcodePayloadTeXSource(payload))
	rendered.WriteString("}%\n")
	rendered.WriteString(`\qrcode{\invoxqrcodepayload}`)
	rendered.WriteString("}")
	return rendered.String()
}

// qrcode parses a limited verbatim syntax in its argument. When the QR command
// is nested inside another macro, the package documentation requires spaces,
// reserved characters, and LF to reach \qrcode as escaped control sequences
// like \ , \%, \^, \~, \\, \{, \}, and \? rather than as raw TeX tokens.
func qrcodePayloadTeXSource(payload []byte) string {
	var source strings.Builder
	for _, value := range payload {
		switch value {
		case ' ':
			source.WriteString(`\noexpand\ `)
		case '\n':
			source.WriteString(`\noexpand\?`)
		case '\\':
			source.WriteString(`\noexpand\\`)
		case '%':
			source.WriteString(`\noexpand\%`)
		case '#':
			source.WriteString(`\noexpand\#`)
		case '&':
			source.WriteString(`\noexpand\&`)
		case '^':
			source.WriteString(`\noexpand\^`)
		case '_':
			source.WriteString(`\noexpand\_`)
		case '~':
			source.WriteString(`\noexpand\~`)
		case '$':
			source.WriteString(`\noexpand\$`)
		case '{':
			source.WriteString(`\noexpand\{`)
		case '}':
			source.WriteString(`\noexpand\}`)
		default:
			if value >= 0x80 {
				fmt.Fprintf(&source, "^^%02x", value)
				continue
			}
			source.WriteByte(value)
		}
	}
	return source.String()
}

func renderLineItems(items []LineItem, currency string) string {
	return renderLineItemRows(items, currency, false)
}

func renderLineItemsWithVAT(items []LineItem, currency string) string {
	return renderLineItemRows(items, currency, true)
}

func renderLineItemRows(items []LineItem, currency string, includeVAT bool) string {
	rows := make([]string, 0, len(items)*2)
	lastIndex := len(items) - 1
	for index, item := range items {
		parts := []string{
			latexEscape(item.Name),
			latexEscape(item.Description),
			formatUnitPrice(item.UnitPrice, currency),
			latexEscape(formatQuantity(item.Quantity)),
		}
		if includeVAT {
			parts = append(parts, formatVATRate(item.VATRatePercent))
		}
		parts = append(parts, FormatCurrency(item.LineTotalCents, currency))
		rows = append(rows, "    "+strings.Join(parts, " & ")+`\\`)
		rows = append(rows, "    "+lineItemRule(index, lastIndex))
	}
	return strings.Join(rows, "\n")
}

func lineItemRule(index, lastIndex int) string {
	ruleWidth := "0.2pt"
	if index == lastIndex {
		ruleWidth = "0.4pt"
	}
	return fmt.Sprintf(`\specialrule{%s}{0pt}{0pt}`, ruleWidth)
}

func renderVATSummaryRows(label string, breakdowns []VATBreakdown, currency string) string {
	rows := make([]string, 0, len(breakdowns))
	escapedLabel := latexEscape(label)
	for _, breakdown := range breakdowns {
		rows = append(rows, fmt.Sprintf(
			"%s (%s): & %s\\\\",
			escapedLabel,
			formatVATRate(breakdown.RatePercent),
			FormatCurrency(breakdown.VATAmountCents, currency),
		))
	}
	return strings.Join(rows, "\n")
}

func formatVATRate(value *big.Rat) string {
	return latexEscape(formatQuantity(value)) + `\%`
}

func latexEscape(text string) string {
	replacer := strings.NewReplacer(
		`\`, `\textbackslash{}`,
		`&`, `\&`,
		`%`, `\%`,
		`$`, `\$`,
		`#`, `\#`,
		`_`, `\_`,
		`{`, `\{`,
		`}`, `\}`,
		`~`, `\textasciitilde{}`,
		`^`, `\textasciicircum{}`,
	)
	escaped := replacer.Replace(text)
	// Values often follow \\ (line ends in addresses). LaTeX would read a
	// leading * as the starred form and a leading [ as an optional argument,
	// even after spaces, so an empty group shields them.
	if trimmed := strings.TrimLeft(escaped, " \t\r\n"); strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "*") {
		return "{}" + escaped
	}
	return escaped
}
