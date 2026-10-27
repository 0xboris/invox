package latex

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
	"unicode/utf8"
)

func FuzzLatexEscape(f *testing.F) {
	for _, seed := range []string{
		"",
		"Hauptstrasse 1",
		`100% & $5 #1 a_b {c} ~d ^e \f`,
		`\textbf{bold}`,
		`\\`,
		"Café Zürich",
		"[Q1] 2026",
		"*Street 1",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, text string) {
		escaped := Escape(text)
		decoded, err := decodeLatexEscape(escaped)
		if err != nil {
			t.Fatalf("Escape(%q) = %q: %v", text, escaped, err)
		}
		if decoded != text {
			t.Fatalf("Escape(%q) = %q, which reads back as %q", text, escaped, decoded)
		}
	})
}

// latexEscapeSequences are the control sequences Escape may emit, and the
// input text each one stands for.
var latexEscapeSequences = []struct{ sequence, text string }{
	{`\textbackslash{}`, `\`},
	{`\textasciitilde{}`, `~`},
	{`\textasciicircum{}`, `^`},
	{`\&`, `&`},
	{`\%`, `%`},
	{`\$`, `$`},
	{`\#`, `#`},
	{`\_`, `_`},
	{`\{`, `{`},
	{`\}`, `}`},
}

// decodeLatexEscape reads Escape output back into text. It fails on any
// TeX special character that is not part of an escape sequence, so a
// successful round trip shows that every special in the input was escaped
// and that no control sequence in the output came from the input. A brace
// group holding at most one plain character, such as `{}` or `{[}`, is
// allowed so the escaper may protect `[` and `*`.
func decodeLatexEscape(escaped string) (string, error) {
	var decoded strings.Builder
	rest := escaped
next:
	for rest != "" {
		for _, escape := range latexEscapeSequences {
			if strings.HasPrefix(rest, escape.sequence) {
				decoded.WriteString(escape.text)
				rest = rest[len(escape.sequence):]
				continue next
			}
		}
		if rest[0] == '{' {
			body, after, ok := strings.Cut(rest[1:], "}")
			if ok && utf8.RuneCountInString(body) <= 1 && !strings.ContainsAny(body, `\&%$#_{}~^`) {
				decoded.WriteString(body)
				rest = after
				continue
			}
		}
		if strings.ContainsRune(`\&%$#_{}~^`, rune(rest[0])) {
			return "", fmt.Errorf("unescaped TeX special character at byte %d", len(escaped)-len(rest))
		}
		decoded.WriteByte(rest[0])
		rest = rest[1:]
	}
	return decoded.String(), nil
}

func FuzzValidateTemplatePlaceholders(f *testing.F) {
	for _, seed := range []string{
		"",
		"Invoice @@INVOICE_NUMBER@@\n@@LINE_ITEMS_ROWS@@\n",
		"@@LINE_ITEMS_BEGIN@@\n@@LINE_ITEM_NAME@@ & @@LINE_ITEM_LINE_TOTAL@@\\\\\n@@LINE_ITEM_RULE@@\n@@LINE_ITEMS_END@@\n",
		"  @@LINE_ITEMS_BEGIN@@\r\n@@LINE_ITEM_NAME@@\r\n  @@LINE_ITEMS_END@@\r\n",
		"@@LINE_ITEMS_BEGIN@@@@LINE_ITEMS_BEGIN@@@@LINE_ITEMS_END@@@@LINE_ITEMS_END@@",
		"@@LINE_ITEMS_END@@",
		"@@LINE_ITEMS_BEGIN@@",
		"@@LINE_ITEM_NAME@@",
		"VAT (@@VAT_RATE@@\\%): & @@VAT_AMOUNT@@\\\\",
		"@@CUSTOMER_CITY_AND_POSTAL_CODE@@ @@ISSUER_CITY_AND_POSTAL_CODE@@",
	} {
		f.Add(seed)
	}

	pairs := sortedReplacementPairs(map[string]string{
		"@@INVOICE_NUMBER@@":  "INV-1",
		"@@LINE_ITEMS_ROWS@@": `Consulting & 100,00 \euro\\`,
	})
	items := []Item{{
		Name:           "Consulting",
		UnitPrice:      big.NewRat(100, 1),
		Quantity:       big.NewRat(2, 1),
		VATRatePercent: big.NewRat(20, 1),
		LineTotalCents: 20000,
	}}
	f.Fuzz(func(t *testing.T, template string) {
		err := ValidateTemplate(template)
		if err != nil {
			if strings.TrimSpace(err.Error()) == "" {
				t.Fatalf("ValidateTemplate(%q) returned an empty error", template)
			}
			return
		}

		// The renderer expands exactly the blocks the validator approved:
		// each block starts at a BEGIN and ends at the END that the
		// boundary scan paired with it.
		var boundaries []int
		for _, match := range lineItemsBoundaryPattern.FindAllStringIndex(template, -1) {
			boundaries = append(boundaries, match[0], match[1])
		}
		var blocks []int
		for _, match := range lineItemsBlockPattern.FindAllStringSubmatchIndex(template, -1) {
			blocks = append(blocks, match[0], match[2], match[3], match[1])
		}
		if len(blocks) != len(boundaries) {
			t.Fatalf("template %q: renderer finds blocks %v, validator finds boundaries %v", template, blocks, boundaries)
		}
		for index := range blocks {
			if blocks[index] != boundaries[index] {
				t.Fatalf("template %q: renderer finds blocks %v, validator finds boundaries %v", template, blocks, boundaries)
			}
		}

		rendered := renderLineItemTemplateBlocks(template, items, "EUR", pairs)
		if !strings.Contains(template, "@@") && rendered != template {
			t.Fatalf("template %q without placeholders rendered as %q", template, rendered)
		}
	})
}
