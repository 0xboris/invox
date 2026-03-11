package invoice

import "strings"

const defaultVATLabel = "VAT"

func issuerVATLabel(payment map[string]any) string {
	if label := strings.TrimSpace(asString(getPath(payment, "vat_label"))); label != "" {
		return label
	}
	return defaultVATLabel
}
