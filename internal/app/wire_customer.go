package app

import (
	"github.com/spf13/cobra"

	customer "github.com/0xboris/invox/internal/features/customer/ui"
	customershared "github.com/0xboris/invox/internal/features/customer/ui/shared"
	customerusecases "github.com/0xboris/invox/internal/features/customer/usecases"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

func customerFactory(d *Dependencies) *customershared.Factory {
	return &customershared.Factory{
		IOStreams:     d.IOStreams,
		Env:           d.Env,
		Hosts:         d.Hosts,
		EditCustomers: customerusecases.NewEditCustomers(d.Editor),
		ListCustomers: customerusecases.NewListCustomers(d.Hosts),
	}
}

func customerCommands(d *Dependencies) []*cobra.Command {
	return []*cobra.Command{customer.NewCmdCustomer(customerFactory(d))}
}

func customerTopics(d *Dependencies) []cmdutil.Topic {
	return customershared.Topics(customerFactory(d))
}
