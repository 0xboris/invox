package cli

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"invox/internal/invoice"
)

func runTemplate(args []string) int {
	if len(args) == 0 {
		printTemplateHelp(os.Stdout)
		return 0
	}
	if len(args) == 1 && wantsHelp(args) {
		printTemplateHelp(os.Stdout)
		return 0
	}

	switch args[0] {
	case "list":
		return runTemplateList(args[1:])
	default:
		return templateUsageError(fmt.Sprintf("unknown template subcommand %q", args[0]))
	}
}

func runTemplateList(args []string) int {
	if wantsHelp(args) {
		printTemplateListHelp(os.Stdout)
		return 0
	}

	spec := templateListSpec()
	fs := flag.NewFlagSet(spec.Name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var namesOnly bool
	fs.BoolVar(&namesOnly, "names", false, "print only template names")

	if err := fs.Parse(args); err != nil {
		printCommandError(os.Stderr, spec, err.Error())
		return 2
	}
	if len(fs.Args()) > 0 {
		printCommandError(os.Stderr, spec, "unexpected arguments: "+strings.Join(fs.Args(), " "))
		return 2
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	templates, err := invoice.ListTemplates(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	for _, template := range templates {
		if namesOnly {
			fmt.Println(template.Name)
			continue
		}
		fmt.Printf("%s\t%s\n", template.Name, template.Path)
	}
	return 0
}

func runCompletion(args []string) int {
	if len(args) == 0 || wantsHelp(args) {
		printCompletionHelp(os.Stdout)
		return 0
	}
	if len(args) != 1 {
		printCommandError(os.Stderr, completionSpec(), "unexpected arguments: "+strings.Join(args, " "))
		return 2
	}

	switch args[0] {
	case "zsh":
		fmt.Fprint(os.Stdout, zshCompletionScript())
		return 0
	default:
		printCommandError(os.Stderr, completionSpec(), fmt.Sprintf("unsupported shell %q", args[0]))
		return 2
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
      'config[Open config.yaml in the default shell editor]' \
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
        '--archive[archive after a successful PDF build]'
      return
      ;;
  esac

  _files
}

compdef _invox invox
`
}
