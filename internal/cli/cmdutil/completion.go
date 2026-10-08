package cmdutil

import (
	"strings"

	"github.com/0xboris/invox/internal/store"
	"github.com/spf13/cobra"
)

// The completion funcs below read the same files as the commands. Any error,
// such as a missing or broken file, completes nothing rather than failing,
// and none of them prompts or prints.

// CompleteCustomerIDs completes the CUSTOMER_ID argument from customers.yaml,
// the -c, --customers file or the one the command would find.
func CompleteCustomerIDs(f *Factory) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		cwd, err := f.Env.Getwd()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		flagValue, _ := cmd.Flags().GetString("customers")
		path, err := SupportPath(completionHost(f, cmd), "", store.Customers, flagValue, cwd)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		customers, err := store.ListCustomers(path)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var ids []cobra.Completion
		for _, customer := range customers {
			ids = append(ids, cobra.CompletionWithDesc(customer.ID, customer.LegalCompanyName))
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleteTemplates completes -t, --template with the names `template list`
// shows, and with file names when no name matches.
func CompleteTemplates(f *Factory) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		templates, err := completionHost(f, cmd).ListTemplates()
		if err != nil {
			return nil, cobra.ShellCompDirectiveDefault
		}
		var names []cobra.Completion
		for _, template := range templates {
			if strings.HasPrefix(template.Name, toComplete) {
				names = append(names, template.Name)
			}
		}
		if len(names) == 0 {
			return nil, cobra.ShellCompDirectiveDefault
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleteArchivedInvoices completes the FILENAME argument of `archive edit`
// with the files `archive list` shows.
func CompleteArchivedInvoices(f *Factory) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		archived, err := completionHost(f, cmd).ListArchivedInvoices()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var files []cobra.Completion
		for _, entry := range archived {
			files = append(files, cobra.CompletionWithDesc(entry.Filename, entry.CustomerID))
		}
		return files, cobra.ShellCompDirectiveNoFileComp
	}
}

// CompleteInputFile completes the optional positional input of a command
// that also takes -i, --input: files with one of exts, and nothing once the
// input is given.
func CompleteInputFile(exts ...string) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
		if len(args) > 0 || cmd.Flags().Changed("input") {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return exts, cobra.ShellCompDirectiveFilterFileExt
	}
}

// completionHost returns the Host for a completion request on cmd. Main
// sets ConfigFile from --config before a command runs, but a completion
// request runs no command, so this reads the flag itself.
func completionHost(f *Factory, cmd *cobra.Command) store.Host {
	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		f.ConfigFile = configFile
	}
	return f.Host()
}
