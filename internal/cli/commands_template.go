package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
)

func runTemplate(f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	if len(args) == 0 {
		printTemplateHelp(ios.Out)
		return nil
	}
	if len(args) == 1 && wantsHelp(args) {
		printTemplateHelp(ios.Out)
		return nil
	}

	switch args[0] {
	case "list":
		return runTemplateList(f, args[1:])
	default:
		return cmdutil.FlagErrorf("template", "unknown template subcommand %q", args[0])
	}
}

func runTemplateList(f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	h := f.Host()
	if wantsHelp(args) {
		printTemplateListHelp(ios.Out)
		return nil
	}

	spec := templateListSpec()
	fs := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var namesOnly bool
	fs.BoolVar(&namesOnly, "names", false, "print only template names")

	if err := fs.Parse(args); err != nil {
		return &cmdutil.FlagError{Command: spec.Name, Err: err}
	}
	if len(fs.Args()) > 0 {
		return cmdutil.FlagErrorf(spec.Name, "unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	templates, err := h.ListTemplates()
	if err != nil {
		return err
	}
	templateDir, err := h.TemplateCatalogDir()
	if err != nil {
		return err
	}

	list := table{
		columns:   []column{{header: "NAME"}, {header: "PATH"}},
		emptyHint: "No templates found in " + templateDir,
	}
	if namesOnly {
		list.columns = list.columns[:1]
	}
	for _, template := range templates {
		if namesOnly {
			list.addRow(template.Name)
			continue
		}
		list.addRow(template.Name, template.Path)
	}
	list.print(ios)
	return nil
}

func runCompletion(ios *iostreams.IOStreams, args []string) error {
	if len(args) == 0 || wantsHelp(args) {
		printCompletionHelp(ios.Out)
		return nil
	}
	if len(args) != 1 {
		return cmdutil.FlagErrorf(completionSpec().Name, "unexpected arguments: %s", strings.Join(args, " "))
	}

	switch args[0] {
	case "zsh":
		fmt.Fprint(ios.Out, zshCompletionScript())
		return nil
	default:
		return cmdutil.FlagErrorf(completionSpec().Name, "unsupported shell %q", args[0])
	}
}

func zshCompletionScript() string {
	return `#compdef invox

_invox_template_names() {
  local -a templates
  templates=(${(f)"$(${invox_command:-$words[1]} template list --names 2>/dev/null)"})
  (( ${#templates[@]} == 0 )) && return 1
  _describe 'template' templates
}

_invox_template_values() {
  _alternative \
    'templates:template name:_invox_template_names' \
    'files:template file:_files -g "*.tex"'
}

_invox_invoice_files() {
  _files -g "*.y(|a)ml(-.)"
}

_invox_shift_words() {
  local skip=$1
  local -a subwords
  subwords=("${(@)words[$((skip + 1)),-1]}")
  words=("${(@)subwords}")
  (( CURRENT -= skip ))
}

_invox() {
  local context state line
  local invox_command=$words[1]
  typeset -A opt_args

  if (( CURRENT == 2 )); then
    _values 'subcommand' \
      'customer[Customer-related commands]' \
      'config[Open config.yaml in your editor]' \
      'init[Create starter support files in the global config directory]' \
      'template[List available templates]' \
      'completion[Generate shell completion scripts]' \
      'new[Create a new invoice YAML file]' \
      'increment[Increment the invoice number in an invoice YAML file]' \
      'validate[Validate invoice YAML against customers and issuer data]' \
      'render[Render a LaTeX invoice file]' \
      'email[Create and open an email draft]' \
      'build[Render and compile an invoice PDF]' \
      'archive[Archive invoices and manage archived invoices]'
    return
  fi

  case $words[2] in
    template)
      if (( CURRENT == 3 )); then
        _values 'template command' 'list[List available templates]'
        return
      fi
      case $words[3] in
        list)
          _invox_shift_words 2
          _arguments '--names[print only template names]'
          return
          ;;
      esac
      ;;
    completion)
      if (( CURRENT == 3 )); then
        _values 'shell' 'zsh[Zsh completion script]'
        return
      fi
      ;;
    render)
      if (( CURRENT == 3 )) && [[ ${words[3]:-} != -* ]]; then
        _invox_invoice_files
        return
      fi
      _invox_shift_words 1
      _arguments -s \
        '(-i --input)'{-i+,--input=}'[invoice YAML file]:invoice:_files -g "*.y(|a)ml"' \
        '(-o --output)'{-o+,--output=}'[output TeX path]:output:_files' \
        '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_files -g "*.yaml"' \
        '(-u --issuer)'{-u+,--issuer=}'[issuer.yaml]:issuer:_files -g "*.yaml"' \
        '(-t --template)'{-t+,--template=}'[template file or known template name]:template:_invox_template_values' \
        '(-i --input)1::invoice:_files -g "*.y(|a)ml(-.)"'
      return
      ;;
    build)
      if (( CURRENT == 3 )) && [[ ${words[3]:-} != -* ]]; then
        _invox_invoice_files
        return
      fi
      _invox_shift_words 1
      _arguments -s \
        '(-i --input)'{-i+,--input=}'[invoice YAML file]:invoice:_files -g "*.y(|a)ml"' \
        '(-o --output)'{-o+,--output=}'[output PDF path]:output:_files' \
        '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_files -g "*.yaml"' \
        '(-u --issuer)'{-u+,--issuer=}'[issuer.yaml]:issuer:_files -g "*.yaml"' \
        '(-t --template)'{-t+,--template=}'[template file or known template name]:template:_invox_template_values' \
        '(-i --input)1::invoice:_files -g "*.y(|a)ml(-.)"' \
        '--archive[archive after a successful PDF build]' \
        '--yes[replace an archived invoice without asking]'
      return
      ;;
    archive)
      if (( CURRENT == 3 )); then
        _alternative \
          'archive-commands:archive command:((edit\:"Copy an archived invoice for editing" list\:"List archived invoices"))' \
          'files:invoice:_invox_invoice_files'
        return
      fi
      case $words[3] in
        edit|list)
          return
          ;;
      esac
      _invox_shift_words 1
      _arguments -s \
        '(-i --input)'{-i+,--input=}'[invoice YAML file]:invoice:_files -g "*.y(|a)ml"' \
        '--yes[replace an archived invoice without asking]' \
        '(-i --input)1::invoice:_files -g "*.y(|a)ml(-.)"'
      return
      ;;
  esac

  _files
}

compdef _invox invox
`
}
