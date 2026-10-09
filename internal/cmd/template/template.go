// Package template is the `invox template` noun.
package template

import (
	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/cmd/template/list"
)

// NewCmdTemplate returns the template command and its subcommands. Without a
// subcommand it prints its help.
func NewCmdTemplate(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template <subcommand>",
		Short: "Template-related commands",
		Long: `Template-related commands.

Description:
  Author and discover the LaTeX templates used by render and build.
  Templates are regular .tex files with literal @@PLACEHOLDER@@ tokens.

Important rules:
  Placeholder names are case-sensitive and must match exactly.
  An unknown placeholder fails render and build; the error names it.
  Most placeholders are LaTeX-escaped automatically.
  Dates render as DD.MM.YYYY.
  Money renders as 1.234,56 \euro for EUR, or with the currency code otherwise.

Structured placeholders:
  @@LINE_ITEMS_ROWS@@ requires a five-column table.
  @@LINE_ITEMS_ROWS_WITH_VAT@@ requires a six-column table.
  @@LINE_ITEMS_BEGIN@@ ... @@LINE_ITEMS_END@@ repeats a custom snippet once per position.
  Inside that block, use @@LINE_ITEM_NAME@@, @@LINE_ITEM_DESCRIPTION@@, @@LINE_ITEM_UNIT_PRICE@@,
  @@LINE_ITEM_QUANTITY@@, @@LINE_ITEM_VAT_RATE@@, @@LINE_ITEM_LINE_TOTAL@@, and @@LINE_ITEM_RULE@@.
  @@VAT_SUMMARY_ROWS@@ belongs inside a two-column totals table.
  Use @@VAT_LABEL@@ anywhere you want the same VAT label text in the template.

Custom line-item block example:
  @@LINE_ITEMS_BEGIN@@
  @@LINE_ITEM_NAME@@ & @@LINE_ITEM_UNIT_PRICE@@ & @@LINE_ITEM_VAT_RATE@@ & @@LINE_ITEM_LINE_TOTAL@@\\
  @@LINE_ITEM_RULE@@
  @@LINE_ITEMS_END@@

Template workflow:
  Run ` + "`" + `invox render -i invoice.yaml` + "`" + ` to inspect the generated .tex.
  Run ` + "`" + `invox build invoice.yaml` + "`" + ` after the rendered LaTeX looks correct.
  Run ` + "`" + `invox template list` + "`" + ` to discover templates addressable by name with -t/--template.

` +
			helptext.TemplatePlaceholderReference(),
		Example: `$ invox template list
$ invox template list --names
$ invox render -i invoice.yaml -t multi_vat.tex
$ invox build invoice.yaml -t multi_vat.tex
`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return cmdutil.UnknownSubcommandError(cmd, args[0])
		},
	}
	cmd.AddCommand(list.NewCmdList(f, nil))
	return cmd
}
