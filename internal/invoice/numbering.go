package invoice

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type NumberingSettings struct {
	Pattern string
	Start   int64
}

const (
	defaultNumberingPattern = "{customer_code}-{counter:03}"
	defaultNumberingStart   = 1
)

var numberingTokenPattern = regexp.MustCompile(`\{([a-z_]+)(?::([0-9]+))?\}`)

func (h Host) ResolveNumberingSettings() (NumberingSettings, error) {
	settings := NumberingSettings{
		Pattern: defaultNumberingPattern,
		Start:   defaultNumberingStart,
	}

	cfg, err := h.Config()
	if err != nil {
		return NumberingSettings{}, err
	}
	if cfg.Numbering.Pattern != "" {
		settings.Pattern = string(cfg.Numbering.Pattern)
	}
	if cfg.Numbering.Start != nil {
		settings.Start = int64(*cfg.Numbering.Start)
	}

	if err := validateNumberingSettings(settings); err != nil {
		return NumberingSettings{}, fmt.Errorf("%s: %w", cfg.File, err)
	}
	return settings, nil
}

func (h Host) NextInvoiceNumber(customerID, issueDate string, customer Customer, minimumCounter int64) (string, []string, error) {
	settings, err := h.ResolveNumberingSettings()
	if err != nil {
		return "", nil, err
	}
	start, err := effectiveNumberingStart(customerID, customer, settings.Start)
	if err != nil {
		return "", nil, err
	}

	baseCounter, skipped, err := h.highestArchivedCounter(settings.Pattern, customerID, issueDate, customer)
	if err != nil {
		return "", nil, err
	}
	if startBase := start - 1; startBase > baseCounter {
		baseCounter = startBase
	}
	if minimumCounter > baseCounter {
		baseCounter = minimumCounter
	}

	nextCounter := baseCounter + 1
	invoiceNumber, err := formatInvoiceNumber(settings.Pattern, customerID, customer, issueDate, nextCounter)
	if err != nil {
		return "", nil, err
	}
	return invoiceNumber, skipped, nil
}

func effectiveNumberingStart(customerID string, customer Customer, globalStart int64) (int64, error) {
	if !customer.Numbering.Start.isSet() {
		return globalStart, nil
	}
	if start := customer.Numbering.Start.Int(); start > 0 {
		return start, nil
	}
	return 0, fmt.Errorf("customers.%s.numbering.start: must be >= 1", customerID)
}

func (h Host) CounterFromInvoiceNumber(invoiceNumber, customerID, issueDate string, customer Customer) (int64, error) {
	settings, err := h.ResolveNumberingSettings()
	if err != nil {
		return 0, err
	}
	return parseInvoiceCounter(settings.Pattern, invoiceNumber, customerID, issueDate, customer)
}

// maxCounterWidth bounds {counter:WIDTH}; an int64 counter has at most 19
// digits.
const maxCounterWidth = 20

func validateNumberingSettings(settings NumberingSettings) error {
	pattern := settings.Pattern
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

	matches := numberingTokenPattern.FindAllStringSubmatch(pattern, -1)
	counterTokens := 0
	hasCustomerToken := false
	consumed := numberingTokenPattern.ReplaceAllString(pattern, "")
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
	if err := validateCustomerCounterSeparator(pattern); err != nil {
		return err
	}
	if settings.Start <= 0 {
		return fmt.Errorf("numbering.start must be >= 1")
	}

	return nil
}

// validateCustomerCounterSeparator rejects a customer token directly next to
// the counter: with {customer_code}{counter}, customer A would read customer
// A1's invoice A1007 as its own counter 1007.
func validateCustomerCounterSeparator(pattern string) error {
	tokens := numberingTokenPattern.FindAllStringSubmatchIndex(pattern, -1)
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

func (h Host) highestArchivedCounter(pattern, customerID, issueDate string, customer Customer) (int64, []string, error) {
	store, err := h.archiveStore()
	if err != nil {
		return 0, nil, err
	}

	var highest int64
	var skipped []string
	err = store.Walk(func(path string) error {
		identity, ok, err := readArchivedIdentity(path)
		if err != nil || !ok || identity.InvoiceNumber == "" {
			return err
		}

		counter, err := parseInvoiceCounter(pattern, identity.InvoiceNumber, customerID, issueDate, customer)
		if err != nil {
			if identity.CustomerID == customerID && inNumberingPeriod(pattern, identity.IssueDate, issueDate) {
				skipped = append(skipped, path)
			}
			return nil
		}
		if counter > highest {
			highest = counter
		}
		return nil
	})
	if err != nil {
		return 0, nil, err
	}
	return highest, skipped, nil
}

// inNumberingPeriod reports whether date has the same {year}, {month} and
// {day} as issueDate, for the tokens pattern uses. Counters restart in each
// such period, so an invoice from another period never counts. A date that
// does not parse is treated as in the period.
func inNumberingPeriod(pattern, date, issueDate string) bool {
	own, err := time.Parse("2006-01-02", date)
	if err != nil {
		return true
	}
	requested, err := time.Parse("2006-01-02", issueDate)
	if err != nil {
		return true
	}
	layouts := map[string]string{"year": "2006", "month": "01", "day": "02"}
	for _, match := range numberingTokenPattern.FindAllStringSubmatch(pattern, -1) {
		if layout, ok := layouts[match[1]]; ok && own.Format(layout) != requested.Format(layout) {
			return false
		}
	}
	return true
}

// highestDraftCounter returns the highest counter used by unarchived invoices
// (status draft or built) directly inside dirs, so that two drafts created
// before either is archived do not get the same number. Files that cannot be
// parsed, are not invoices, or do not match the numbering pattern are ignored.
func (h Host) highestDraftCounter(dirs []string, customerID, issueDate string, customer Customer) (int64, error) {
	settings, err := h.ResolveNumberingSettings()
	if err != nil {
		return 0, err
	}

	seen := make(map[string]bool, len(dirs))
	var highest int64
	for _, dir := range dirs {
		if strings.TrimSpace(dir) == "" || seen[dir] {
			continue
		}
		seen[dir] = true

		entries, err := os.ReadDir(dir)
		if err != nil {
			// The draft scan is best effort: the archive check still
			// guarantees uniqueness, so an unreadable directory must not
			// stop `new`.
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil || info.Size() > maxDraftScanSize {
				continue
			}
			switch strings.ToLower(filepath.Ext(entry.Name())) {
			case ".yaml", ".yml":
			default:
				continue
			}

			invoiceNumber, ok := draftInvoiceNumber(filepath.Join(dir, entry.Name()))
			if !ok {
				continue
			}
			counter, err := parseInvoiceCounter(settings.Pattern, invoiceNumber, customerID, issueDate, customer)
			if err != nil {
				continue
			}
			if counter > highest {
				highest = counter
			}
		}
	}
	return highest, nil
}

// maxDraftScanSize skips large YAML files in the draft scan; invoices are
// far smaller.
const maxDraftScanSize = 1 << 20

func draftInvoiceNumber(path string) (string, bool) {
	var identity invoiceIdentity
	if err := decodeYAMLFile(path, &identity, false); err != nil || identity.Invoice == nil {
		return "", false
	}
	switch identity.Invoice.Status.Trim() {
	case "draft", "built":
	default:
		return "", false
	}
	invoiceNumber := identity.Invoice.Number.Trim()
	return invoiceNumber, invoiceNumber != ""
}

func formatInvoiceNumber(pattern, customerID string, customer Customer, issueDate string, counter int64) (string, error) {
	issueTime, customerCode, err := numberingValues(customerID, customer, issueDate)
	if err != nil {
		return "", err
	}

	replaced := numberingTokenPattern.ReplaceAllStringFunc(pattern, func(token string) string {
		matches := numberingTokenPattern.FindStringSubmatch(token)
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

func parseInvoiceCounter(pattern, invoiceNumber, customerID, issueDate string, customer Customer) (int64, error) {
	issueTime, customerCode, err := numberingValues(customerID, customer, issueDate)
	if err != nil {
		return 0, err
	}

	// formatInvoiceNumber trims its result, so match against the trimmed
	// pattern.
	pattern = strings.TrimSpace(pattern)

	var patternBuilder strings.Builder
	patternBuilder.WriteString("^")

	lastIndex := 0
	for _, match := range numberingTokenPattern.FindAllStringSubmatchIndex(pattern, -1) {
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

func numberingValues(customerID string, customer Customer, issueDate string) (time.Time, string, error) {
	issueTime, err := time.Parse("2006-01-02", issueDate)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invoice.issue_date: expected YYYY-MM-DD, got %q", issueDate)
	}

	customerCode := customer.Numbering.Code.Trim()
	if customerCode == "" {
		customerCode = customerID
	}
	return issueTime, customerCode, nil
}
