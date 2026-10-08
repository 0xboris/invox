package latex

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// ValidateTemplate reports every unsupported placeholder and malformed line
// item block in template, one per line.
func ValidateTemplate(template string) error {
	type unsupportedPlaceholder struct {
		placeholder, replacement string
		index                    int
	}
	var found []unsupportedPlaceholder
	for _, p := range []unsupportedPlaceholder{
		{placeholder: "@@VAT_RATE@@", replacement: "@@VAT_SUMMARY_ROWS@@"},
		{placeholder: "@@VAT_AMOUNT@@", replacement: "@@VAT_SUMMARY_ROWS@@"},
		{placeholder: "@@ISSUER_CITY_AND_POSTAL_CODE@@", replacement: "@@ISSUER_POSTAL_CODE@@ @@ISSUER_CITY@@"},
		{placeholder: "@@CUSTOMER_CITY_AND_POSTAL_CODE@@", replacement: "@@CUSTOMER_POSTAL_CODE@@ @@CUSTOMER_CITY@@"},
	} {
		if p.index = strings.Index(template, p.placeholder); p.index >= 0 {
			found = append(found, p)
		}
	}
	// Report in template order so the first error points at the first problem.
	sort.Slice(found, func(i, j int) bool { return found[i].index < found[j].index })
	var validationErrors []string
	for _, p := range found {
		validationErrors = append(validationErrors, fmt.Sprintf("%s: unsupported placeholder; use %s", p.placeholder, p.replacement))
	}
	if validateLineItemBlockPlaceholders(template, &validationErrors) {
		validateLineItemPlaceholdersOutsideBlocks(template, &validationErrors)
	}
	if len(validationErrors) > 0 {
		return errors.New(strings.Join(validationErrors, "\n"))
	}
	return nil
}

// MigrateLegacyPlaceholders replaces the single VAT row of older starter
// templates with @@VAT_SUMMARY_ROWS@@.
func MigrateLegacyPlaceholders(template string) string {
	legacyVATRowPattern := regexp.MustCompile(`(?m)^([ \t]*)VAT \(@@VAT_RATE@@\\%\): & @@VAT_AMOUNT@@\\\\[ \t]*$`)
	return legacyVATRowPattern.ReplaceAllString(template, `${1}@@VAT_SUMMARY_ROWS@@`)
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
