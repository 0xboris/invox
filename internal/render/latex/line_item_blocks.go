package latex

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/0xboris/invox/internal/money"
)

const (
	lineItemsBeginPlaceholder      = "@@LINE_ITEMS_BEGIN@@"
	lineItemsEndPlaceholder        = "@@LINE_ITEMS_END@@"
	lineItemNamePlaceholder        = "@@LINE_ITEM_NAME@@"
	lineItemDescriptionPlaceholder = "@@LINE_ITEM_DESCRIPTION@@"
	lineItemUnitPricePlaceholder   = "@@LINE_ITEM_UNIT_PRICE@@"
	lineItemQuantityPlaceholder    = "@@LINE_ITEM_QUANTITY@@"
	lineItemVATRatePlaceholder     = "@@LINE_ITEM_VAT_RATE@@"
	lineItemLineTotalPlaceholder   = "@@LINE_ITEM_LINE_TOTAL@@"
	lineItemRulePlaceholder        = "@@LINE_ITEM_RULE@@"
)

var (
	lineItemsBlockPattern    = regexp.MustCompile(`(?s)` + regexp.QuoteMeta(lineItemsBeginPlaceholder) + `(.*?)` + regexp.QuoteMeta(lineItemsEndPlaceholder))
	lineItemsBoundaryPattern = regexp.MustCompile(regexp.QuoteMeta(lineItemsBeginPlaceholder) + `|` + regexp.QuoteMeta(lineItemsEndPlaceholder))
)

func validateLineItemBlockPlaceholders(template string, validationErrors *[]string) bool {
	depth := 0
	for _, match := range lineItemsBoundaryPattern.FindAllStringIndex(template, -1) {
		placeholder := template[match[0]:match[1]]
		switch placeholder {
		case lineItemsBeginPlaceholder:
			if depth > 0 {
				*validationErrors = append(*validationErrors, lineItemsBeginPlaceholder+": nested line-item blocks are unsupported")
				return false
			}
			depth++
		case lineItemsEndPlaceholder:
			if depth == 0 {
				*validationErrors = append(*validationErrors, lineItemsEndPlaceholder+": missing matching "+lineItemsBeginPlaceholder)
				return false
			}
			depth--
		}
	}
	if depth > 0 {
		*validationErrors = append(*validationErrors, lineItemsBeginPlaceholder+": missing matching "+lineItemsEndPlaceholder)
		return false
	}
	return true
}

func validateLineItemPlaceholdersOutsideBlocks(template string, validationErrors *[]string) {
	stripped := lineItemsBlockPattern.ReplaceAllString(template, "")
	for _, placeholder := range []string{
		lineItemNamePlaceholder,
		lineItemDescriptionPlaceholder,
		lineItemUnitPricePlaceholder,
		lineItemQuantityPlaceholder,
		lineItemVATRatePlaceholder,
		lineItemLineTotalPlaceholder,
		lineItemRulePlaceholder,
	} {
		if strings.Contains(stripped, placeholder) {
			*validationErrors = append(*validationErrors, fmt.Sprintf("%s: only supported inside %s ... %s", placeholder, lineItemsBeginPlaceholder, lineItemsEndPlaceholder))
		}
	}
}

// Fill replaces every placeholder in template: values outside line item
// blocks, and values plus the item placeholders inside each block.
func Fill(template string, values map[string]string, items []Item, currency string) string {
	return renderLineItemTemplateBlocks(template, items, currency, sortedReplacementPairs(values))
}

// renderLineItemTemplateBlocks substitutes every placeholder in a single pass:
// text outside line item blocks uses the template pairs, and each block body
// uses the line item pairs plus the template pairs. Substituted values are
// never scanned again.
func renderLineItemTemplateBlocks(template string, items []Item, currency string, templatePairs []string) string {
	replacer := strings.NewReplacer(templatePairs...)
	matches := lineItemsBlockPattern.FindAllStringSubmatchIndex(template, -1)
	if len(matches) == 0 {
		return replacer.Replace(template)
	}
	var builder strings.Builder
	lastEnd := 0
	for _, match := range matches {
		bounds := lineItemTemplateBlockBounds(template, match)
		builder.WriteString(replacer.Replace(template[lastEnd:bounds.renderStart]))
		body := template[bounds.bodyStart:bounds.bodyEnd]
		builder.WriteString(renderLineItemTemplateBlock(body, items, currency, templatePairs))
		lastEnd = bounds.renderEnd
	}
	builder.WriteString(replacer.Replace(template[lastEnd:]))
	return builder.String()
}

type lineItemBlockBounds struct {
	renderStart int
	bodyStart   int
	bodyEnd     int
	renderEnd   int
}

func lineItemTemplateBlockBounds(template string, match []int) lineItemBlockBounds {
	bounds := lineItemBlockBounds{
		renderStart: match[0],
		bodyStart:   match[2],
		bodyEnd:     match[3],
		renderEnd:   match[1],
	}

	if lineStart, lineAfter, ok := standaloneTemplateLineBounds(template, match[0], match[2]); ok {
		bounds.renderStart = lineStart
		bounds.bodyStart = lineAfter
	}
	if lineStart, lineAfter, ok := standaloneTemplateLineBounds(template, match[3], match[1]); ok {
		bounds.bodyEnd = lineStart
		bounds.renderEnd = lineAfter
	}

	return bounds
}

func standaloneTemplateLineBounds(template string, placeholderStart, placeholderEnd int) (int, int, bool) {
	lineStart := templateLineStart(template, placeholderStart)
	if !templateLineHasOnlyIndentation(template[lineStart:placeholderStart]) {
		return 0, 0, false
	}

	lineEnd := templateLineEnd(template, placeholderEnd)
	if !templateLineHasOnlyIndentation(template[placeholderEnd:lineEnd]) {
		return 0, 0, false
	}

	return lineStart, templateLineAfterBreak(template, lineEnd), true
}

func templateLineStart(text string, index int) int {
	for index > 0 {
		switch text[index-1] {
		case '\n', '\r':
			return index
		default:
			index--
		}
	}
	return 0
}

func templateLineEnd(text string, index int) int {
	for index < len(text) {
		switch text[index] {
		case '\n', '\r':
			return index
		default:
			index++
		}
	}
	return len(text)
}

func templateLineAfterBreak(text string, lineEnd int) int {
	if lineEnd >= len(text) {
		return lineEnd
	}
	if text[lineEnd] == '\r' && lineEnd+1 < len(text) && text[lineEnd+1] == '\n' {
		return lineEnd + 2
	}
	return lineEnd + 1
}

func templateLineHasOnlyIndentation(text string) bool {
	for _, r := range text {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}

func renderLineItemTemplateBlock(body string, items []Item, currency string, templatePairs []string) string {
	var builder strings.Builder
	lastIndex := len(items) - 1
	for index, item := range items {
		builder.WriteString(renderLineItemTemplate(body, item, currency, lineItemRule(index, lastIndex), templatePairs))
	}
	return builder.String()
}

func renderLineItemTemplate(body string, item Item, currency, rule string, templatePairs []string) string {
	pairs := []string{
		lineItemNamePlaceholder, Escape(item.Name),
		lineItemDescriptionPlaceholder, Escape(item.Description),
		lineItemUnitPricePlaceholder, formatUnitPrice(item.UnitPrice, currency),
		lineItemQuantityPlaceholder, Escape(money.FormatQuantity(item.Quantity)),
		lineItemVATRatePlaceholder, formatVATRate(item.VATRatePercent),
		lineItemLineTotalPlaceholder, FormatCurrency(item.LineTotalCents, currency),
		lineItemRulePlaceholder, rule,
	}
	return strings.NewReplacer(append(pairs, templatePairs...)...).Replace(body)
}
