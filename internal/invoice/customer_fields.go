package invoice

import "strings"

const defaultCustomerCurrency = "EUR"

func customerName(customer map[string]any) string {
	for _, value := range []any{
		customer["name"],
		customer["legal_company_name"],
	} {
		if name := strings.TrimSpace(asString(value)); name != "" {
			return name
		}
	}
	return ""
}

func customerEmail(customer map[string]any) string {
	for _, path := range []string{
		"billing.send_invoice_to",
		"billing.email",
		"email",
	} {
		if email := strings.TrimSpace(asString(getPath(customer, path))); email != "" {
			return email
		}
	}
	return ""
}

func customerCurrency(customer map[string]any) string {
	for _, path := range []string{
		"billing.currency",
		"currency",
	} {
		if currency := strings.TrimSpace(asString(getPath(customer, path))); currency != "" {
			return currency
		}
	}
	return defaultCustomerCurrency
}
