package latex

import (
	"errors"
	"regexp"
	"slices"
	"sort"
	"strings"
)

// placeholderPattern matches every token written like a placeholder.
var placeholderPattern = regexp.MustCompile(`@@[A-Z0-9_]+@@`)

// Placeholders returns every placeholder a template may use: the ones
// filled from the invoice, the line item block and its placeholders, and
// the EPC QR ones.
func Placeholders() []string {
	names := make([]string, 0, len(invoicePlaceholders)+len(lineItemPlaceholders)+5)
	for _, p := range invoicePlaceholders {
		names = append(names, p.name)
	}
	names = append(names, lineItemsBeginPlaceholder, lineItemsEndPlaceholder)
	for _, p := range lineItemPlaceholders {
		names = append(names, p.name)
	}
	return append(names, epcQRAvailablePlaceholder, epcQRLabelPlaceholder, epcQRCodePlaceholder)
}

// ValidateTemplate reports every unknown placeholder and malformed line
// item block in template, one per line.
func ValidateTemplate(template string) error {
	var validationErrors []string
	known := Placeholders()
	var unknown []string
	for _, name := range placeholderPattern.FindAllString(template, -1) {
		if !slices.Contains(known, name) && !slices.Contains(unknown, name) {
			unknown = append(unknown, name)
			validationErrors = append(validationErrors, name+": unknown placeholder")
		}
	}
	if validateLineItemBlockPlaceholders(template, &validationErrors) {
		validateLineItemPlaceholdersOutsideBlocks(template, &validationErrors)
	}
	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	return nil
}

// sortedReplacementPairs flattens placeholder values into strings.NewReplacer
// arguments in sorted key order, so rendering does not depend on map order.
func sortedReplacementPairs(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		pairs = append(pairs, key, values[key])
	}
	return pairs
}
