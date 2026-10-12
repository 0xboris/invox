// Package numbering formats and parses invoice numbers from a pattern such
// as {customer_code}-{counter:03}, and picks the next counter.
package numbering

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Settings is the numbering.pattern and numbering.start of config.yaml.
type Settings struct {
	Pattern string
	Start   int64
}

const (
	DefaultPattern = "{customer_code}-{counter:03}"
	DefaultStart   = 1
)

var tokenPattern = regexp.MustCompile(`\{([a-z_]+)(?::([0-9]+))?\}`)

// counterToken holds the counter. It is the only token that takes a width,
// {counter:WIDTH}.
const counterToken = "counter"

// fields are what the tokens of a pattern stand for in one invoice number.
type fields struct {
	customerID   string
	customerCode string
	issue        time.Time
}

// tokens holds every token but {counter}.
var tokens = map[string]struct {
	// customer marks a token that names the customer; {counter} needs a
	// separator from it.
	customer bool
	value    func(fields) string
}{
	"customer_id":   {customer: true, value: func(f fields) string { return f.customerID }},
	"customer_code": {customer: true, value: func(f fields) string { return f.customerCode }},
	"year":          {value: func(f fields) string { return f.issue.Format("2006") }},
	"month":         {value: func(f fields) string { return f.issue.Format("01") }},
	"day":           {value: func(f fields) string { return f.issue.Format("02") }},
}

// token is one {name} or {name:width} in a pattern, at pattern[start:end].
type token struct {
	name, width string
	start, end  int
}

func scan(pattern string) []token {
	var found []token
	for _, match := range tokenPattern.FindAllStringSubmatchIndex(pattern, -1) {
		t := token{name: pattern[match[2]:match[3]], start: match[0], end: match[1]}
		if match[4] >= 0 {
			t.width = pattern[match[4]:match[5]]
		}
		found = append(found, t)
	}
	return found
}

// maxCounterWidth bounds {counter:WIDTH}; an int64 counter has at most 19
// digits.
const maxCounterWidth = 20

// Validate reports a pattern that cannot be formatted and parsed back, or a
// start below 1.
func (s Settings) Validate() error {
	pattern := s.Pattern
	if strings.TrimSpace(pattern) == "" {
		return fmt.Errorf("numbering.pattern: missing value")
	}
	// Invoice numbers are trimmed when they are formatted and read back, so
	// surrounding whitespace in the pattern could never be parsed again.
	if strings.TrimSpace(pattern) != pattern {
		return fmt.Errorf("numbering.pattern must not start or end with whitespace: %q", pattern)
	}
	if !utf8.ValidString(pattern) {
		return fmt.Errorf("numbering.pattern must be valid UTF-8: %q", pattern)
	}

	consumed := tokenPattern.ReplaceAllString(pattern, "")
	if strings.Contains(consumed, "{") || strings.Contains(consumed, "}") {
		return fmt.Errorf("numbering.pattern contains unsupported placeholders: %q", pattern)
	}

	found := scan(pattern)
	counterTokens := 0
	hasCustomerToken := false
	for _, t := range found {
		if t.name == counterToken {
			counterTokens++
			if t.width != "" {
				if width, err := strconv.Atoi(t.width); err != nil || width > maxCounterWidth {
					return fmt.Errorf("numbering.pattern token {counter:%s} uses an invalid width; use at most %d", t.width, maxCounterWidth)
				}
			}
			continue
		}
		spec, ok := tokens[t.name]
		if !ok {
			return fmt.Errorf("numbering.pattern uses unsupported token {%s}", t.name)
		}
		if t.width != "" {
			return fmt.Errorf("numbering.pattern token {%s} does not support a format", t.name)
		}
		hasCustomerToken = hasCustomerToken || spec.customer
	}

	if counterTokens == 0 {
		return fmt.Errorf("numbering.pattern must contain {counter} or {counter:WIDTH}")
	}
	// Two counters only parse back when they hold the same digits, which
	// Parse does not check.
	if counterTokens > 1 {
		return fmt.Errorf("numbering.pattern must contain {counter} only once")
	}
	if !hasCustomerToken {
		return fmt.Errorf("numbering.pattern must contain {customer_id} or {customer_code}")
	}
	if err := checkCustomerCounterSeparator(pattern, found); err != nil {
		return err
	}
	if s.Start <= 0 {
		return fmt.Errorf("numbering.start must be >= 1")
	}

	return nil
}

// checkCustomerCounterSeparator rejects a customer token directly next to
// the counter: with {customer_code}{counter}, customer A would read customer
// A1's invoice A1007 as its own counter 1007.
func checkCustomerCounterSeparator(pattern string, found []token) error {
	for index := 1; index < len(found); index++ {
		previous, current := found[index-1], found[index]
		if previous.end != current.start {
			continue
		}
		for _, pair := range [][2]string{{previous.name, current.name}, {current.name, previous.name}} {
			if tokens[pair[0]].customer && pair[1] == counterToken {
				suggestion := pattern[:current.start] + "-" + pattern[current.start:]
				return fmt.Errorf(
					"numbering.pattern %q needs a separator between {%s} and {counter}, such as %q; "+
						"invoice numbers in the old format no longer count towards the next number, so set "+
						"numbering.start (or customers.<id>.numbering.start) to continue the sequence",
					pattern, pair[0], suggestion,
				)
			}
		}
	}
	return nil
}

// Next returns the counter after the highest one seen, and at least start.
func Next(start, highest int64) int64 {
	return max(highest, start-1) + 1
}

// InPeriod reports whether date has the same {year}, {month} and {day} as
// issueDate, for the tokens pattern uses. Counters restart in each such
// period, so an invoice from another period never counts. A date that does
// not parse is treated as in the period.
func InPeriod(pattern, date, issueDate string) bool {
	own, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return true
	}
	requested, err := time.Parse(time.DateOnly, issueDate)
	if err != nil {
		return true
	}
	// The customer tokens stand for "" on both sides, so only the date
	// tokens can differ.
	for _, t := range scan(pattern) {
		if spec, ok := tokens[t.name]; ok && spec.value(fields{issue: own}) != spec.value(fields{issue: requested}) {
			return false
		}
	}
	return true
}

// Format returns the invoice number pattern gives counter for the customer
// on issueDate. customerCode stands in for {customer_code}; when it is ""
// the customer ID does.
func Format(pattern, customerID, customerCode, issueDate string, counter int64) (string, error) {
	f, err := newFields(customerID, customerCode, issueDate)
	if err != nil {
		return "", err
	}
	pieces, widths, err := expand(pattern, f)
	if err != nil {
		return "", err
	}

	var number strings.Builder
	for index, width := range widths {
		number.WriteString(pieces[index])
		if width == "" {
			number.WriteString(strconv.FormatInt(counter, 10))
			continue
		}
		digits, _ := strconv.Atoi(width)
		fmt.Fprintf(&number, "%0*d", digits, counter)
	}
	number.WriteString(pieces[len(widths)])
	return strings.TrimSpace(number.String()), nil
}

// Parse returns the counter of invoiceNumber, which must be a number that
// Format gives for the same pattern, customer and issue date.
func Parse(pattern, invoiceNumber, customerID, customerCode, issueDate string) (int64, error) {
	f, err := newFields(customerID, customerCode, issueDate)
	if err != nil {
		return 0, err
	}

	// Format trims its result, so match against the trimmed pattern.
	pattern = strings.TrimSpace(pattern)
	pieces, _, err := expand(pattern, f)
	if err != nil {
		return 0, err
	}
	for _, piece := range pieces {
		if !utf8.ValidString(piece) {
			return 0, fmt.Errorf("numbering.pattern %q: invalid UTF-8", pattern)
		}
	}

	mismatch := fmt.Errorf("invoice.number %q does not match numbering pattern %q", invoiceNumber, pattern)
	if len(pieces) != 2 {
		return 0, mismatch
	}
	digits, ok := strings.CutPrefix(invoiceNumber, pieces[0])
	if ok {
		digits, ok = strings.CutSuffix(digits, pieces[1])
	}
	if !ok || digits == "" || strings.Trim(digits, "0123456789") != "" {
		return 0, mismatch
	}

	counter, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invoice.number %q contains an invalid counter", invoiceNumber)
	}
	return counter, nil
}

// expand fills in every token of pattern but {counter}. It returns the text
// around the counters, one piece more than there are counters, and the width
// of each counter ("" for a plain {counter}).
func expand(pattern string, f fields) (pieces, widths []string, err error) {
	var piece strings.Builder
	last := 0
	for _, t := range scan(pattern) {
		piece.WriteString(pattern[last:t.start])
		last = t.end
		if t.name == counterToken {
			pieces = append(pieces, piece.String())
			widths = append(widths, t.width)
			piece.Reset()
			continue
		}
		spec, ok := tokens[t.name]
		if !ok {
			return nil, nil, fmt.Errorf("numbering.pattern uses unsupported token {%s}", t.name)
		}
		piece.WriteString(spec.value(f))
	}
	piece.WriteString(pattern[last:])
	return append(pieces, piece.String()), widths, nil
}

func newFields(customerID, customerCode, issueDate string) (fields, error) {
	issue, err := time.Parse(time.DateOnly, issueDate)
	if err != nil {
		return fields{}, fmt.Errorf("invoice.issue_date: expected YYYY-MM-DD, got %q", issueDate)
	}
	if customerCode == "" {
		customerCode = customerID
	}
	return fields{customerID: customerID, customerCode: customerCode, issue: issue}, nil
}
