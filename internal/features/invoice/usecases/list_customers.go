package usecases

import (
	"github.com/0xboris/invox/internal/domain/invoice"
)

// ListCustomers reads the customers in customers.yaml, for the completion of
// customer IDs.
type ListCustomers struct {
	hosts *invoice.HostSource
}

func NewListCustomers(hosts *invoice.HostSource) ListCustomers {
	return ListCustomers{hosts: hosts}
}

func (u ListCustomers) Run(customersPath string) ([]invoice.CustomerSummary, error) {
	return invoice.ListCustomers(u.hosts.Host().Files(), customersPath)
}
