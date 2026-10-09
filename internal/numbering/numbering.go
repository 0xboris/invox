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

	matches := tokenPattern.FindAllStringSubmatch(pattern, -1)
	counterTokens := 0
	hasCustomerToken := false
	consumed := tokenPattern.ReplaceAllString(pattern, "")
	if strings.Contains(consumed, "{") || strings.Contains(consumed, "}") {
		return fmt.Errorf("numbering.pattern contains unsupported placeholders: %q", pattern)
	}

	for _, match := range matches {
		token := match[1]
		format := match[2]
		switch token {
		case "customer_id", "customer_code", "year", "month", "day":
			if format != "" {
				return fmt.Errorf("numbering.pattern token {%s} does not support a format", token)
			}
			if token == "customer_id" || token == "customer_code" {
				hasCustomerToken = true
			}
		case "counter":
			counterTokens++
			if format != "" {
				if width, err := strconv.Atoi(format); err != nil || width > maxCounterWidth {
					return fmt.Errorf("numbering.pattern token {counter:%s} uses an invalid width; use at most %d", format, maxCounterWidth)
				}
			}
		default:
			return fmt.Errorf("numbering.pattern uses unsupported token {%s}", token)
		}
	}

	if counterTokens == 0 {
		return fmt.Errorf("numbering.pattern must contain {counter} or {counter:WIDTH}")
	}
	// Two counters only parse back when they hold the same digits, which a
	// regular expression cannot check.
	if counterTokens > 1 {
		return fmt.Errorf("numbering.pattern must contain {counter} only once")
	}
	if !hasCustomerToken {
		return fmt.Errorf("numbering.pattern must contain {customer_id} or {customer_code}")
	}
	if err := checkCustomerCounterSeparator(pattern); err != nil {
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
func checkCustomerCounterSeparator(pattern string) error {
	tokens := tokenPattern.FindAllStringSubmatchIndex(pattern, -1)
	for index := 1; index < len(tokens); index++ {
		previous, current := tokens[index-1], tokens[index]
		if previous[1] != current[0] {
			continue
		}
		left := pattern[previous[2]:previous[3]]
		right := pattern[current[2]:current[3]]
		for _, pair := range [][2]string{{left, right}, {right, left}} {
			if (pair[0] == "customer_id" || pair[0] == "customer_code") && pair[1] == "counter" {
				suggestion := pattern[:current[0]] + "-" + pattern[current[0]:]
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
	layouts := map[string]string{"year": "2006", "month": "01", "day": "02"}
	for _, match := range tokenPattern.FindAllStringSubmatch(pattern, -1) {
		if layout, ok := layouts[match[1]]; ok && own.Format(layout) != requested.Format(layout) {
			return false
		}
	}
	return true
}

// Format returns the invoice number pattern gives counter for the customer
// on issueDate. customerCode stands in for {customer_code}; when it is ""
// the customer ID does.
func Format(pattern, customerID, customerCode, issueDate string, counter int64) (string, error) {
	issueTime, customerCode, err := values(customerID, customerCode, issueDate)
	if err != nil {
		return "", err
	}

	replaced := tokenPattern.ReplaceAllStringFunc(pattern, func(token string) string {
		matches := tokenPattern.FindStringSubmatch(token)
		if len(matches) != 3 {
			return token
		}
		name := matches[1]
		format := matches[2]
		switch name {
		case "customer_id":
			return customerID
		case "customer_code":
			return customerCode
		case "year":
			return issueTime.Format("2006")
		case "month":
			return issueTime.Format("01")
		case "day":
			return issueTime.Format("02")
		case "counter":
			if format == "" {
				return strconv.FormatInt(counter, 10)
			}
			width, _ := strconv.Atoi(format)
			return fmt.Sprintf("%0*d", width, counter)
		default:
			return token
		}
	})

	return strings.TrimSpace(replaced), nil
}

// Parse returns the counter of invoiceNumber, which must be a number that
// Format gives for the same pattern, customer and issue date.
func Parse(pattern, invoiceNumber, customerID, customerCode, issueDate string) (int64, error) {
	issueTime, customerCode, err := values(customerID, customerCode, issueDate)
	if err != nil {
		return 0, err
	}

	// Format trims its result, so match against the trimmed pattern.
	pattern = strings.TrimSpace(pattern)

	var patternBuilder strings.Builder
	patternBuilder.WriteString("^")

	lastIndex := 0
	for _, match := range tokenPattern.FindAllStringSubmatchIndex(pattern, -1) {
		start := match[0]
		end := match[1]
		tokenStart := match[2]
		tokenEnd := match[3]

		patternBuilder.WriteString(regexp.QuoteMeta(pattern[lastIndex:start]))

		token := pattern[tokenStart:tokenEnd]
		switch token {
		case "customer_id":
			patternBuilder.WriteString(regexp.QuoteMeta(customerID))
		case "customer_code":
			patternBuilder.WriteString(regexp.QuoteMeta(customerCode))
		case "year":
			patternBuilder.WriteString(regexp.QuoteMeta(issueTime.Format("2006")))
		case "month":
			patternBuilder.WriteString(regexp.QuoteMeta(issueTime.Format("01")))
		case "day":
			patternBuilder.WriteString(regexp.QuoteMeta(issueTime.Format("02")))
		case "counter":
			patternBuilder.WriteString(`([0-9]+)`)
		default:
			return 0, fmt.Errorf("numbering.pattern uses unsupported token {%s}", token)
		}

		lastIndex = end
	}
	patternBuilder.WriteString(regexp.QuoteMeta(pattern[lastIndex:]))
	patternBuilder.WriteString("$")

	numberPattern, err := regexp.Compile(patternBuilder.String())
	if err != nil {
		return 0, fmt.Errorf("numbering.pattern %q: %w", pattern, err)
	}
	matches := numberPattern.FindStringSubmatch(invoiceNumber)
	if len(matches) != 2 {
		return 0, fmt.Errorf("invoice.number %q does not match numbering pattern %q", invoiceNumber, pattern)
	}

	counter, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invoice.number %q contains an invalid counter", invoiceNumber)
	}
	return counter, nil
}

func values(customerID, customerCode, issueDate string) (time.Time, string, error) {
	issueTime, err := time.Parse(time.DateOnly, issueDate)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invoice.issue_date: expected YYYY-MM-DD, got %q", issueDate)
	}
	if customerCode == "" {
		customerCode = customerID
	}
	return issueTime, customerCode, nil
}
