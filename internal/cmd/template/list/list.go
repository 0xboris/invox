// Package list is the `invox template list` command.
package list

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/tableprinter"
)

type ListOptions struct {
	IO      *iostreams.IOStreams
	Service func(cmdutil.Files) *billing.Service
	Getwd   func() (string, error)

	NamesOnly bool
	Exporter  *cmdutil.Exporter
}

// templateJSON is a template in --json output. Default marks the template
// that render and build use from the current directory without -t.
type templateJSON struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Default bool   `json:"default"`
}

// NewCmdList returns the template list command. runF replaces listRun in tests.
func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
	opts := &ListOptions{IO: f.IOStreams, Service: f.Service, Getwd: f.Env.Getwd}
	cmd := &cobra.Command{
		Use:               "list",
		Short:             "List available invoice templates",
		ValidArgsFunction: cobra.NoFileCompletions,
		Long: `List available LaTeX invoice templates from the same directory as the resolved default template.

Output:
  Default: NAME<TAB>ABSOLUTE_PATH
  --names: TEMPLATE_NAME per line
  --json: an array of the requested fields; default is true for the template
    render and build use from the current directory without -t
  On a terminal, aligned columns under a header. Piped, \, tab, CR and LF in a field are written as \\, \t, \r and \n.

Lookup:
  The directory is derived from the resolved default template path.
  Name-only -t/--template values are resolved in that same directory.
`,
		Example: `$ invox template list
$ invox template list --names
$ invox template list --json name,default
$ invox build invoice.yaml -t multi_vat.tex
`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return cmdutil.FlagErrorf("template list", "unexpected arguments: %s", strings.Join(args, " "))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if opts.NamesOnly && opts.Exporter != nil {
				return cmdutil.FlagErrorf("template list", "--names and --json cannot be used together")
			}
			if runF != nil {
				return runF(opts)
			}
			return listRun(opts)
		},
	}
	cmd.Flags().BoolVar(&opts.NamesOnly, "names", false, "Print only template names")
	cmdutil.AddJSONFlags(cmd, &opts.Exporter, templateJSON{})
	return cmd
}

func listRun(opts *ListOptions) error {
	svc := opts.Service(cmdutil.Files{})
	catalog, err := svc.ListTemplates()
	if err != nil {
		return err
	}
	templates, templateDir := catalog.Templates, catalog.Dir

	if opts.Exporter != nil {
		return exportTemplates(opts, svc, templates)
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

func exportTemplates(opts *ListOptions, svc *billing.Service, templates []billing.Template) error {
	if _, err := opts.Getwd(); err != nil {
		return err
	}
	defaultTemplate, err := svc.DefaultTemplate()
	if err != nil {
		return err
	}
	items := make([]templateJSON, 0, len(templates))
	for _, template := range templates {
		items = append(items, templateJSON{
			Name:    template.Name,
			Path:    template.Path,
			Default: defaultTemplate != "" && filepath.Clean(defaultTemplate) == filepath.Clean(template.Path),
		})
	}
	return opts.Exporter.Write(opts.IO, items)
}
