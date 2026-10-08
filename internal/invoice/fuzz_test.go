package invoice

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/0xboris/invox/internal/money"
)

// The fuzz targets below run their seeds, and every crasher committed under
// testdata/fuzz/<Target>/, as part of `go test`. Run one for longer with
//
//	go test -run='^$' -fuzz='^FuzzLoadContext$' -fuzztime=60s ./internal/invoice
//
// or all of them with `make fuzz`.

func FuzzInvoiceNumberRoundTrip(f *testing.F) {
	for _, seed := range []struct {
		pattern, customerID, customerCode, issueDate string
		counter                                      int64
	}{
		{"{customer_id}-{counter:03}", "CUST-001", "", "2026-03-06", 1},
		{"{customer_code}-{counter:03}", "CUST-001", "APP", "2026-03-06", 7},
		{"{customer_id}-{year}-{counter:04}", "CUST-001", "", "2026-12-31", 12345},
		{"{year}{month}{day}/{customer_code}/{counter}", "1001", "", "2026-01-02", 0},
		{"{counter}.{customer_id}", "A1", "", "2026-03-06", 9223372036854775807},
		// #17: patterns that formatted but did not parse back.
		{" {customer_id}-{counter}", "CUST-001", "", "2026-03-06", 1},
		{"{customer_id}-{counter:03}/{counter}", "CUST-001", "", "2026-03-06", 1},
		{"{customer_code}{counter:03}", "A", "", "2026-03-06", 7},
		// #17: invalid UTF-8 made parseInvoiceCounter's regexp.MustCompile panic.
		{"\xff{customer_id}-{counter}", "CUST-001", "", "2026-03-06", 1},
		{"{customer_id}-{counter:999999999}", "CUST-001", "", "2026-03-06", 1},
	} {
		f.Add(seed.pattern, seed.customerID, seed.customerCode, seed.issueDate, seed.counter)
	}

	f.Fuzz(func(t *testing.T, pattern, customerID, customerCode, issueDate string, counter int64) {
		customer := Customer{Numbering: CustomerNumbering{Code: Text(customerCode)}}

		// Parsing never panics, whatever the pattern and number.
		_, _ = parseInvoiceCounter(pattern, customerID, customerID, issueDate, customer)

		// ResolveNumberingSettings trims the configured pattern, then
		// validates it; only patterns that pass are ever formatted.
		pattern = strings.TrimSpace(pattern)
		if err := validateNumberingSettings(NumberingSettings{Pattern: pattern, Start: 1}); err != nil {
			return
		}
		// Customer IDs and codes come from YAML, which is valid UTF-8, and
		// IDs are trimmed when an invoice is loaded. Counters are never
		// negative.
		customerID = strings.TrimSpace(customerID)
		if customerID == "" || !utf8.ValidString(customerID) || !utf8.ValidString(customerCode) || counter < 0 {
			return
		}

		number, err := formatInvoiceNumber(pattern, customerID, customer, issueDate, counter)
		if _, dateErr := time.Parse("2006-01-02", issueDate); dateErr != nil {
			if err == nil {
				t.Fatalf("formatInvoiceNumber accepted invalid issue date %q", issueDate)
			}
			return
		}
		if err != nil {
			t.Fatalf("formatInvoiceNumber(%q, %q, %q, %d) returned error: %v", pattern, customerID, issueDate, counter, err)
		}

		got, err := parseInvoiceCounter(pattern, number, customerID, issueDate, customer)
		if err != nil {
			t.Fatalf("pattern %q formatted counter %d as %q, which does not parse back: %v", pattern, counter, number, err)
		}
		if got != counter {
			t.Fatalf("pattern %q formatted counter %d as %q, which parses back as %d", pattern, counter, number, got)
		}
	})
}

func FuzzIsValidIBAN(f *testing.F) {
	for _, seed := range []string{
		"AT611904300234573201",
		"PL61109010140000071219812874",
		"DE89370400440532013000",
		"GI75NWBK000000007099453",
		"ZZ6600000000000",
		"AT61190430023457320",
		"ATAA1904300234573201",
		"at611904300234573201",
		// #17: check digits 00, 01 and 99 never occur in a valid IBAN.
		"DE01370400440000000042",
		"DE00370400440000000",
		"DE99370400440000000000",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		got := isValidIBAN(value)
		if want := ibanOracle(value); got != want {
			t.Fatalf("isValidIBAN(%q) = %v, want %v", value, got, want)
		}
	})
}

// ibanOracle checks an IBAN with big-integer arithmetic: a known country and
// length, upper-case letters and digits only, check digits 02-98, and the
// rearranged number mod 97 == 1.
func ibanOracle(value string) bool {
	if len(value) < 4 || ibanCountryLengths[value[:2]] != len(value) {
		return false
	}
	checkDigits := value[2:4]
	if checkDigits[0] < '0' || checkDigits[0] > '9' || checkDigits[1] < '0' || checkDigits[1] > '9' {
		return false
	}
	if checkDigits < "02" || checkDigits > "98" {
		return false
	}
	var numeric strings.Builder
	for _, r := range value[4:] + value[:4] {
		switch {
		case r >= '0' && r <= '9':
			numeric.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			numeric.WriteString(strconv.Itoa(int(r-'A') + 10))
		default:
			return false
		}
	}
	number, ok := new(big.Int).SetString(numeric.String(), 10)
	if !ok {
		return false
	}
	return new(big.Int).Mod(number, big.NewInt(97)).Int64() == 1
}

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
		escaped := latexEscape(text)
		decoded, err := decodeLatexEscape(escaped)
		if err != nil {
			t.Fatalf("latexEscape(%q) = %q: %v", text, escaped, err)
		}
		if decoded != text {
			t.Fatalf("latexEscape(%q) = %q, which reads back as %q", text, escaped, decoded)
		}
	})
}

// latexEscapeSequences are the control sequences latexEscape may emit, and the
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

// decodeLatexEscape reads latexEscape output back into text. It fails on any
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

	ctx := fuzzContext(f)
	pairs := sortedReplacementPairs(buildTemplateValues(ctx))
	f.Fuzz(func(t *testing.T, template string) {
		err := validateTemplatePlaceholders(template)
		if err != nil {
			if strings.TrimSpace(err.Error()) == "" {
				t.Fatalf("validateTemplatePlaceholders(%q) returned an empty error", template)
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

		rendered := renderLineItemTemplateBlocks(template, ctx.LineItems, ctx.Currency, pairs)
		if !strings.Contains(template, "@@") && rendered != template {
			t.Fatalf("template %q without placeholders rendered as %q", template, rendered)
		}
	})
}

func FuzzRenderEmailTemplate(f *testing.F) {
	for _, seed := range []string{
		"",
		defaultEmailBodyTemplate,
		"{customer_name} | {email_greeting} | {contact_person} | {customer_id} | {invoice_number} | {issue_date} | {due_date} | {total_amount} | {outstanding_amount} | {payment_terms_text} | {issuer_name}",
		"Invoice {invoice_number}\r\nTotal {total_amount}\r",
		"{{invoice_number}}",
		"{unknown}",
	} {
		f.Add(seed)
	}

	ctx := fuzzContext(f)
	placeholders := []string{
		"{customer_name}", "{email_greeting}", "{contact_person}", "{customer_id}", "{invoice_number}",
		"{issue_date}", "{due_date}", "{total_amount}", "{outstanding_amount}", "{payment_terms_text}", "{issuer_name}",
	}
	f.Fuzz(func(t *testing.T, template string) {
		rendered := renderEmailTemplate(template, ctx)
		if strings.Contains(rendered, "\r") {
			t.Fatalf("renderEmailTemplate(%q) = %q, which still contains a carriage return", template, rendered)
		}
		// The fixture's values contain no braces, so every placeholder is
		// replaced and none can be formed by a replacement.
		for _, placeholder := range placeholders {
			if strings.Contains(rendered, placeholder) {
				t.Fatalf("renderEmailTemplate(%q) = %q, which still contains %s", template, rendered, placeholder)
			}
		}
		if !strings.Contains(template, "{") {
			normalized := strings.ReplaceAll(strings.ReplaceAll(template, "\r\n", "\n"), "\r", "\n")
			if rendered != normalized {
				t.Fatalf("renderEmailTemplate(%q) = %q, want it unchanged", template, rendered)
			}
		}
	})
}

func FuzzBuildEPCPayload(f *testing.F) {
	for _, seed := range []struct {
		name, iban, bic, ref string
		cents                int64
	}{
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "CUST-001-001", 25200},
		{"Boris Consulting", "AT61 1904 3002 3457 3201", "", "", 1},
		{"Zürich Café GmbH", "GI75NWBK000000007099453", "NWBKGI2G", "Rechnung №1", 99999999999},
		{strings.Repeat("N", 70), "AT611904300234573201", "BKAUATWW", strings.Repeat("R", 140), 100},
		{"Boris Consulting", "DE01370400440000000042", "", "", 100},
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "line\nbreak", 100},
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "", 0},
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "", -500},
		{"Boris Consulting", "AT611904300234573201", "BKAUATWW", "", -9223372036854775808},
		{"\xff", "AT611904300234573201", "BKAUATWW", "", 100},
	} {
		f.Add(seed.name, seed.iban, seed.bic, seed.ref, seed.cents)
	}

	f.Fuzz(func(t *testing.T, name, iban, bic, ref string, cents int64) {
		ctx := &Context{
			Currency:         "EUR",
			TotalCents:       cents,
			OutstandingCents: cents,
			Company:          Company{LegalCompanyName: "Fallback Name"},
			Payment: Payment{
				IBAN:  Text(iban),
				BIC:   Text(bic),
				EPCQR: EPCQR{Name: Text(name), Text: Text(ref)},
			},
			Invoice: InvoiceHeader{Number: "CUST-001-001"},
		}
		payload, err := buildEPCPayload(ctx)
		if err != nil {
			return
		}
		if len(payload) > epcQRMaxPayloadBytes {
			t.Fatalf("payload is %d bytes, more than %d: %q", len(payload), epcQRMaxPayloadBytes, payload)
		}
		if !utf8.Valid(payload) {
			t.Fatalf("payload is not valid UTF-8: %q", payload)
		}
		lines := strings.Split(string(payload), "\n")
		if len(lines) < 8 || len(lines) > 12 || lines[0] != "BCD" || lines[3] != "SCT" {
			t.Fatalf("payload does not have the EPC layout: %q", payload)
		}
		if !isValidIBAN(lines[6]) {
			t.Fatalf("payload carries invalid IBAN %q", lines[6])
		}
		if cents < 1 || cents > epcQRMaxAmountCents {
			t.Fatalf("payload accepted amount %d cents outside 0.01-999999999.99: %q", cents, payload)
		}
		if want := fmt.Sprintf("EUR%d.%02d", cents/100, cents%100); lines[7] != want {
			t.Fatalf("payload amount line = %q, want %q", lines[7], want)
		}
	})
}

func FuzzLoadContext(f *testing.F) {
	for _, seed := range []string{
		fuzzInvoiceYAML,
		strings.Replace(fuzzInvoiceYAML, "paid_amount: 0", "paid_amount: 252", 1),
		strings.Replace(fuzzInvoiceYAML, "vat_percent: 20", "vat_percent: 0.125", 1),
		// #17: amounts that overflowed int64 cents.
		strings.Replace(fuzzInvoiceYAML, "unit_price: 100", `unit_price: "100000000000000000"`, 1),
		strings.Replace(fuzzInvoiceYAML, "quantity: 2", "quantity: 1000000000000000000000000", 1),
		strings.Replace(fuzzInvoiceYAML, "paid_amount: 0", "paid_amount: 1000000000000000000000000000000", 1),
		strings.Replace(fuzzInvoiceYAML, "vat_percent: 20", "vat_percent: 100000000000000000000", 1),
		"",
		"[]",
		strings.Replace(strings.Replace(fuzzInvoiceYAML, "  - name: Development", "  - &dev\n    name: Development", 1),
			"  - name: Support\n    description: QA\n    unit_price: 10\n    quantity: 1\n", "  - *dev\n", 1),
		// #17: an alias to its enclosing node, and a billion laughs.
		"customer_id: CUST-001\npositions: &p [*p]\n",
		billionLaughs(),
	} {
		f.Add([]byte(seed))
	}

	dir := f.TempDir()
	customersPath, issuerPath := writeFuzzCustomerAndIssuer(f, dir)
	f.Fuzz(func(t *testing.T, source []byte) {
		invoicePath := filepath.Join(t.TempDir(), "invoice.yaml")
		if err := os.WriteFile(invoicePath, source, 0o644); err != nil {
			t.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
		}
		ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
		if err != nil {
			return
		}
		checkContextTotals(t, ctx)
	})
}

// checkContextTotals asserts the money invariants of a loaded invoice.
func checkContextTotals(t *testing.T, ctx *Context) {
	t.Helper()

	inRange := func(label string, cents int64) {
		if cents < 0 || cents > money.MaxCents {
			t.Fatalf("%s = %d cents, outside 0..%d", label, cents, money.MaxCents)
		}
	}
	if len(ctx.LineItems) == 0 {
		t.Fatal("loaded an invoice without line items")
	}
	var subtotal int64
	for index, item := range ctx.LineItems {
		inRange("line total", item.LineTotalCents)
		if _, ok := money.Cents(item.UnitPrice); !ok {
			t.Fatalf("positions[%d].unit_price %s does not fit in cents", index+1, item.UnitPrice.RatString())
		}
		subtotal += item.LineTotalCents
	}
	inRange("subtotal", ctx.SubtotalCents)
	if subtotal != ctx.SubtotalCents {
		t.Fatalf("line totals add up to %d, subtotal is %d", subtotal, ctx.SubtotalCents)
	}
	var net, vat int64
	for _, breakdown := range ctx.VATBreakdowns {
		inRange("VAT amount", breakdown.VATAmountCents)
		net += breakdown.NetCents
		vat += breakdown.VATAmountCents
	}
	if net != ctx.SubtotalCents || vat != ctx.VATAmountCents {
		t.Fatalf("VAT breakdowns add up to net %d and VAT %d, want %d and %d", net, vat, ctx.SubtotalCents, ctx.VATAmountCents)
	}
	inRange("total", ctx.TotalCents)
	if ctx.TotalCents != ctx.SubtotalCents+ctx.VATAmountCents {
		t.Fatalf("total %d != subtotal %d + VAT %d", ctx.TotalCents, ctx.SubtotalCents, ctx.VATAmountCents)
	}
	inRange("paid amount", ctx.PaidAmountCents)
	inRange("outstanding amount", ctx.OutstandingCents)
	if ctx.OutstandingCents != ctx.TotalCents-ctx.PaidAmountCents {
		t.Fatalf("outstanding %d != total %d - paid %d", ctx.OutstandingCents, ctx.TotalCents, ctx.PaidAmountCents)
	}
}

const fuzzInvoiceYAML = `customer_id: CUST-001
invoice:
  number: CUST-001-001
  issue_date: 2026-03-06
  due_date: 2026-04-05
  period: Leistungszeitraum
  vat_percent: 20
  paid_amount: 0
positions:
  - name: Development
    description: Sprint work
    unit_price: 100
    quantity: 2
  - name: Support
    description: QA
    unit_price: 10
    quantity: 1
`

// writeFuzzCustomerAndIssuer writes the customers.yaml and issuer.yaml that
// writeContextFixtures uses, for targets that cannot take a *testing.T.
func writeFuzzCustomerAndIssuer(tb testing.TB, dir string) (string, string) {
	tb.Helper()

	customersPath := filepath.Join(dir, "customers.yaml")
	issuerPath := filepath.Join(dir, "issuer.yaml")
	files := map[string]string{
		customersPath: `CUST-001:
  name: Appsters GmbH
  email: office@appsters.example
  email_greeting: Dear Jane Doe,
  contact_person: Jane Doe
  address:
    street: Hauptstrasse 1
    postal_code: 1010
    city: Vienna
    country: Austria
  tax:
    vat_tax_id: ATU12345678
`,
		issuerPath: `company:
  legal_company_name: Boris Consulting
  company_registration_number: FN 123456a
  vat_tax_id: ATU87654321
  website: https://example.com
  email: hello@example.com
  address:
    street: Ring 1
    postal_code: 1010
    city: Vienna
    country: Austria
payment:
  bank_name: Test Bank
  iban: AT611904300234573201
  bic: BKAUATWW
  due_days: 30
  payment_terms_text: Pay within 30 days
`,
	}
	for path, source := range files {
		if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
			tb.Fatalf("WriteFile(%s) returned error: %v", filepath.Base(path), err)
		}
	}
	return customersPath, issuerPath
}

// fuzzContext loads the fixture invoice for targets that render with it.
func fuzzContext(tb testing.TB) *Context {
	tb.Helper()

	dir := tb.TempDir()
	customersPath, issuerPath := writeFuzzCustomerAndIssuer(tb, dir)
	invoicePath := filepath.Join(dir, "invoice.yaml")
	if err := os.WriteFile(invoicePath, []byte(fuzzInvoiceYAML), 0o644); err != nil {
		tb.Fatalf("WriteFile(invoice.yaml) returned error: %v", err)
	}
	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		tb.Fatalf("LoadContext returned error: %v", err)
	}
	return ctx
}
