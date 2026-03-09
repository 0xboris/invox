package invoice

import (
	"fmt"
	"sort"
	"strings"
)

type CustomerSummary struct {
	ID               string
	LegalCompanyName string
	Status           string
}

func ListCustomers(customersPath string) ([]CustomerSummary, error) {
	customersValue, err := loadYAML(customersPath)
	if err != nil {
		return nil, err
	}

	customers, ok := customersValue.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s: root value must be a mapping", customersPath)
	}

	customerIDs := make([]string, 0, len(customers))
	for customerID := range customers {
		customerIDs = append(customerIDs, customerID)
	}
	sort.Strings(customerIDs)

	summaries := make([]CustomerSummary, 0, len(customerIDs))
	for _, customerID := range customerIDs {
		rawCustomer := customers[customerID]
		customer, ok := rawCustomer.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: customer `%s` must be a mapping", customersPath, customerID)
		}
		summaries = append(summaries, CustomerSummary{
			ID:               customerID,
			LegalCompanyName: customerName(customer),
			Status:           strings.TrimSpace(asString(customer["status"])),
		})
	}

	return summaries, nil
}
