// Package usecases holds the operations of the invoice commands. Each one
// reaches the outside of invox through the domain's ports, so the commands
// never name a port.
package usecases

import (
	"github.com/0xboris/invox/internal/domain/invoice"
)

// LoadInvoice reads an invoice with its customer and issuer, and checks it.
type LoadInvoice struct {
	hosts *invoice.HostSource
}

func NewLoadInvoice(hosts *invoice.HostSource) LoadInvoice {
	return LoadInvoice{hosts: hosts}
}

func (u LoadInvoice) Run(customersPath, issuerPath, invoicePath string) (*invoice.Context, error) {
	return invoice.LoadContext(u.hosts.Host().Files(), customersPath, issuerPath, invoicePath)
}
