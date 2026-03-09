package invoice

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"
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

func ResolveNumberingSettings() (NumberingSettings, error) {
	settings := NumberingSettings{
		Pattern: defaultNumberingPattern,
		Start:   defaultNumberingStart,
	}

	_, root, err := loadConfigRoot()
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

func NextInvoiceNumber(customerID, issueDate string, customer map[string]any, minimumCounter int64) (string, int64, error) {
	settings, err := ResolveNumberingSettings()
	if err != nil {
		return "", 0, err
	}
	start, err := effectiveNumberingStart(customerID, customer, settings.Start)
	if err != nil {
		return "", 0, err
	}

	baseCounter, err := highestArchivedCounter(settings.Pattern, customerID, issueDate, customer)
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

func CounterFromInvoiceNumber(invoiceNumber, customerID, issueDate string, customer map[string]any) (int64, error) {
	settings, err := ResolveNumberingSettings()
	if err != nil {
		return 0, err
	}
	return parseInvoiceCounter(settings.Pattern, invoiceNumber, customerID, issueDate, customer)
}

func validateNumberingSettings(settings NumberingSettings) error {
	pattern := strings.TrimSpace(settings.Pattern)
	if pattern == "" {
		return fmt.Errorf("numbering.pattern: missing value")
	}

	matches := numberingTokenPattern.FindAllStringSubmatch(pattern, -1)
	hasCounterToken := false
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
			hasCounterToken = true
			if format != "" {
				if _, err := strconv.Atoi(format); err != nil {
					return fmt.Errorf("numbering.pattern token {counter:%s} uses an invalid width", format)
				}
			}
		default:
			return fmt.Errorf("numbering.pattern uses unsupported token {%s}", token)
		}
	}

	if !hasCounterToken {
		return fmt.Errorf("numbering.pattern must contain {counter} or {counter:WIDTH}")
	}
	if !hasCustomerToken {
		return fmt.Errorf("numbering.pattern must contain {customer_id} or {customer_code}")
	}
	if settings.Start <= 0 {
		return fmt.Errorf("numbering.start must be >= 1")
	}

	return nil
}

func highestArchivedCounter(pattern, customerID, issueDate string, customer map[string]any) (int64, error) {
	archiveDir, err := ResolveArchiveDir()
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
	err = filepath.WalkDir(archiveDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !isArchivedInvoicePath(path) {
			return nil
		}

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

func isArchivedInvoicePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown", ".yaml", ".yml":
		return true
	default:
		return false
	}
}

func archivedInvoiceNumber(path string) (string, bool, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		value, err := loadYAML(path)
		if err != nil {
			return "", false, nil
		}
		return invoiceNumberFromValue(value), invoiceNumberFromValue(value) != "", nil
	case ".md", ".markdown":
		source, err := os.ReadFile(path)
		if err != nil {
			return "", false, err
		}
		frontMatter, ok := markdownFrontMatter(source)
		if !ok {
			return "", false, nil
		}

		value, err := parseYAMLSource(frontMatter, path)
		if err != nil {
			return "", false, nil
		}
		return invoiceNumberFromValue(value), invoiceNumberFromValue(value) != "", nil
	default:
		return "", false, nil
	}
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
	return []byte(remainder[:end]), true
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

	matches := regexp.MustCompile(patternBuilder.String()).FindStringSubmatch(invoiceNumber)
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
	source, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var document yaml.Node
	if err := yaml.Unmarshal(source, &document); err != nil {
		return fmt.Errorf("%s: %w", path, err)
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

	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(&document); err != nil {
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	return writeFileAtomic(path, buffer.Bytes(), 0o644)
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

func writeFileAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tempFile, err := os.CreateTemp(filepath.Dir(path), ".invox-*")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	success := false
	defer func() {
		if !success {
			_ = os.Remove(tempPath)
		}
	}()
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Chmod(mode); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	success = true
	return nil
}
