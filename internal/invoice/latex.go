package invoice

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/0xboris/invox/internal/money"
)

func FormatCurrency(cents int64, currency string) string {
	return withCurrency(money.FormatCents(cents), currency)
}

func formatUnitPrice(value *big.Rat, currency string) string {
	return withCurrency(money.FormatUnitPrice(value), currency)
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
			latexEscape(money.FormatQuantity(item.Quantity)),
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
	return latexEscape(money.FormatQuantity(value)) + `\%`
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
