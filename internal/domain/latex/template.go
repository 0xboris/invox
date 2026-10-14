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
	var validationErrors []string
	for placeholder, replacement := range map[string]string{
		"@@VAT_RATE@@":                      "@@VAT_SUMMARY_ROWS@@",
		"@@VAT_AMOUNT@@":                    "@@VAT_SUMMARY_ROWS@@",
		"@@ISSUER_CITY_AND_POSTAL_CODE@@":   "@@ISSUER_POSTAL_CODE@@ @@ISSUER_CITY@@",
		"@@CUSTOMER_CITY_AND_POSTAL_CODE@@": "@@CUSTOMER_POSTAL_CODE@@ @@CUSTOMER_CITY@@",
	} {
		if strings.Contains(template, placeholder) {
			validationErrors = append(validationErrors, fmt.Sprintf("%s: unsupported placeholder; use %s", placeholder, replacement))
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
