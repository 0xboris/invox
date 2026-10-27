// Package usecases holds the operations of the customer commands. Each one
// reaches the outside of invox through the domain's ports, so the commands
// never name a port.
package usecases

import (
	"github.com/0xboris/invox/internal/domain/invoice"
)

// ListCustomers reads the customers in customers.yaml.
type ListCustomers struct {
	hosts *invoice.HostSource
}

func NewListCustomers(hosts *invoice.HostSource) ListCustomers {
	return ListCustomers{hosts: hosts}
}

func (u ListCustomers) Run(customersPath string) ([]invoice.CustomerSummary, error) {
	return invoice.ListCustomers(u.hosts.Host().Files(), customersPath)
}
