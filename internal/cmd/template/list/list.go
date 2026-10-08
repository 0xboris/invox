// Package list is the `invox template list` command.
package list

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/tableprinter"
)

type ListOptions struct {
	IO   *iostreams.IOStreams
	Host func() invoice.Host

	NamesOnly bool
}

// NewCmdList returns the template list command. runF replaces listRun in tests.
func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{IO: f.IOStreams, Host: f.Host}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available invoice templates",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("template list", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}
	cmd.Flags().BoolVar(&opts.NamesOnly, "names", false, "Print only template names")
	return cmd
}

func listRun(opts *ListOptions) error {
	h := opts.Host()
	templates, err := h.ListTemplates()
	if err != nil {
		return err
	}
	templateDir, err := h.TemplateCatalogDir()
	if err != nil {
		return err
	}

	list := tableprinter.Table{
		Columns:   []tableprinter.Column{{Header: "NAME"}, {Header: "PATH"}},
		EmptyHint: "No templates found in " + templateDir,
	}
	if opts.NamesOnly {
		list.Columns = list.Columns[:1]
	}
	for _, template := range templates {
		if opts.NamesOnly {
			list.AddRow(template.Name)
			continue
		}
		list.AddRow(template.Name, template.Path)
	}
	list.Print(opts.IO)
	return nil
}
