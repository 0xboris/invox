package invoice

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	yaml "gopkg.in/yaml.v3"

	"github.com/0xboris/invox/internal/fsutil"
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

	_, root, err := h.loadConfigRoot()
	if err != nil || root == nil {
		return settings, err
	}

	numbering, ok := root["numbering"].(map[string]any)
	if !ok {
		return settings, nil
	}

	if pattern := strings.TrimSpace(asString(numbering["pattern"])); pattern != "" {
		settings.Pattern = pattern
	}
	if rawStart := strings.TrimSpace(asString(numbering["start"])); rawStart != "" {
		start, err := strconv.ParseInt(rawStart, 10, 64)
		if err != nil {
			return NumberingSettings{}, fmt.Errorf("config.yaml: numbering.start: expected a positive integer, got %q", rawStart)
		}
		settings.Start = start
	}

	if err := validateNumberingSettings(settings); err != nil {
		return NumberingSettings{}, fmt.Errorf("config.yaml: %w", err)
	}
	return settings, nil
}

func (h Host) NextInvoiceNumber(customerID, issueDate string, customer map[string]any, minimumCounter int64) (string, int64, error) {
	settings, err := h.ResolveNumberingSettings()
	if err != nil {
		return "", 0, err
	}
	start, err := effectiveNumberingStart(customerID, customer, settings.Start)
	if err != nil {
		return "", 0, err
	}

	baseCounter, err := h.highestArchivedCounter(settings.Pattern, customerID, issueDate, customer)
	if err != nil {
		return "", 0, err
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
		return "", 0, err
	}
	return invoiceNumber, nextCounter, nil
}

func effectiveNumberingStart(customerID string, customer map[string]any, globalStart int64) (int64, error) {
	rawStart := strings.TrimSpace(asString(getPath(customer, "numbering.start")))
	if rawStart == "" {
		return globalStart, nil
	}

	start, err := strconv.ParseInt(rawStart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("customers.%s.numbering.start: expected a positive integer, got %q", customerID, rawStart)
	}
	if start <= 0 {
		return 0, fmt.Errorf("customers.%s.numbering.start: must be >= 1", customerID)
	}
	return start, nil
}

func (h Host) CounterFromInvoiceNumber(invoiceNumber, customerID, issueDate string, customer map[string]any) (int64, error) {
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

func (h Host) highestArchivedCounter(pattern, customerID, issueDate string, customer map[string]any) (int64, error) {
	archiveDir, err := h.ResolveArchiveDir()
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(archiveDir) == "" {
		return 0, nil
	}
	info, err := os.Stat(archiveDir)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, fmt.Errorf("%s: archive.dir must point to a directory", archiveDir)
	}

	var highest int64
	err = walkArchiveDir(archiveDir, func(path string) error {
		invoiceNumber, ok, err := archivedInvoiceNumber(path)
		if err != nil || !ok {
			return err
		}

		counter, err := parseInvoiceCounter(pattern, invoiceNumber, customerID, issueDate, customer)
		if err != nil {
			return nil
		}
		if counter > highest {
			highest = counter
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return highest, nil
}

// highestDraftCounter returns the highest counter used by unarchived invoices
// (status draft or built) directly inside dirs, so that two drafts created
// before either is archived do not get the same number. Files that cannot be
// parsed, are not invoices, or do not match the numbering pattern are ignored.
func (h Host) highestDraftCounter(dirs []string, customerID, issueDate string, customer map[string]any) (int64, error) {
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
	value, err := loadYAML(path)
	if err != nil {
		return "", false
	}
	root, ok := value.(map[string]any)
	if !ok {
		return "", false
	}
	invoice, ok := root["invoice"].(map[string]any)
	if !ok {
		return "", false
	}
	switch strings.TrimSpace(asString(invoice["status"])) {
	case "draft", "built":
	default:
		return "", false
	}
	invoiceNumber := strings.TrimSpace(asString(invoice["number"]))
	return invoiceNumber, invoiceNumber != ""
}

func isArchivedInvoicePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func archivedInvoiceNumber(path string) (string, bool, error) {
	value, ok, err := archivedInvoiceValue(path)
	if err != nil || !ok {
		return "", ok, err
	}
	invoiceNumber := invoiceNumberFromValue(value)
	return invoiceNumber, invoiceNumber != "", nil
}

func markdownFrontMatter(source []byte) ([]byte, bool) {
	text := strings.ReplaceAll(string(source), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, false
	}
	remainder := text[len("---\n"):]
	end := strings.Index(remainder, "\n---\n")
	if end < 0 {
		return nil, false
	}
	// The leading newline stands in for the opening `---`, so YAML line
	// numbers in errors match the lines of the Markdown file.
	return []byte("\n" + remainder[:end]), true
}

func invoiceNumberFromValue(value any) string {
	root, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	invoice, ok := root["invoice"].(map[string]any)
	if !ok {
		return ""
	}
	return strings.TrimSpace(asString(invoice["number"]))
}

func formatInvoiceNumber(pattern, customerID string, customer map[string]any, issueDate string, counter int64) (string, error) {
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

func parseInvoiceCounter(pattern, invoiceNumber, customerID, issueDate string, customer map[string]any) (int64, error) {
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

func numberingValues(customerID string, customer map[string]any, issueDate string) (time.Time, string, error) {
	issueTime, err := time.Parse("2006-01-02", issueDate)
	if err != nil {
		return time.Time{}, "", fmt.Errorf("invoice.issue_date: expected YYYY-MM-DD, got %q", issueDate)
	}

	customerCode := strings.TrimSpace(asString(getPath(customer, "numbering.code")))
	if customerCode == "" {
		customerCode = customerID
	}
	return issueTime, customerCode, nil
}

func writeInvoiceNumber(path, invoiceNumber string) error {
	document, err := loadYAMLDocument(path)
	if err != nil {
		return err
	}
	if len(document.Content) == 0 {
		return fmt.Errorf("%s: root value must be a mapping", path)
	}

	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: root value must be a mapping", path)
	}

	invoiceNode := findMappingValue(root, "invoice")
	if invoiceNode == nil {
		return fmt.Errorf("%s: missing `invoice` mapping", path)
	}
	if invoiceNode.Kind != yaml.MappingNode {
		return fmt.Errorf("%s: `invoice` must be a mapping", path)
	}

	numberNode := findMappingValue(invoiceNode, "number")
	if numberNode == nil {
		appendMappingNode(invoiceNode, "number", scalarNode(invoiceNumber))
	} else {
		numberNode.Kind = yaml.ScalarNode
		numberNode.Tag = "!!str"
		numberNode.Value = invoiceNumber
	}

	clearYAMLMergeTags(document)
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	return fsutil.WriteFile(path, buffer.Bytes(), fsutil.Public)
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func appendMappingNode(node *yaml.Node, key string, value *yaml.Node) {
	node.Content = append(node.Content, scalarNode(key), value)
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
	}
}
