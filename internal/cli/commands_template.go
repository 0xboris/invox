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
	var jsonOutput bool
	fs.BoolVar(&namesOnly, "names", false, "print only template names")
	fs.BoolVar(&jsonOutput, "json", false, "print JSON output")

	if err := fs.Parse(args); err != nil {
		printCommandError(os.Stderr, spec, err.Error())
		return 2
	}
	if len(fs.Args()) > 0 {
		printCommandError(os.Stderr, spec, "unexpected arguments: "+strings.Join(fs.Args(), " "))
		return 2
	}
	if namesOnly && jsonOutput {
		printCommandError(os.Stderr, spec, "cannot combine --names with --json")
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

	if jsonOutput {
		if err := writeJSON(os.Stdout, templateListJSON(templates)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0
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

_invox_customer_ids() {
  local -a customer_ids
  customer_ids=(${(f)"$(${invox_command:-$words[1]} customer list 2>/dev/null | cut -f1)"})
  (( ${#customer_ids[@]} == 0 )) && return 1
  _describe 'customer' customer_ids
}

_invox_archive_names() {
  local -a archived
  archived=(${(f)"$(${invox_command:-$words[1]} archive list 2>/dev/null | cut -f1)"})
  (( ${#archived[@]} == 0 )) && return 1
  _describe 'archived invoice' archived
}

_invox_template_values() {
  _alternative \
    'templates:template name:_invox_template_names' \
    'files:template file:_files -g "*.tex"'
}

_invox_invoice_files() {
  _files -g "*.y(|a)ml(-.)"
}

_invox_yaml_files() {
  _files -g "*.yaml(-.)"
}

_invox_pdf_files() {
  _files -g "*.pdf(-.)"
}

_invox_email_input_files() {
  _alternative \
    'yaml:invoice YAML:_invox_invoice_files' \
    'pdf:invoice PDF:_invox_pdf_files'
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
    _alternative \
      'flags:root flag:((-h\:show\ help --help\:show\ help --version\:show\ version))' \
      'subcommands:subcommand:((help\:Show\ help\ topics customer\:Customer-related\ commands config\:Open\ config.yaml\ in\ the\ default\ shell\ editor init\:Create\ starter\ support\ files\ in\ the\ global\ config\ directory template\:List\ available\ templates completion\:Generate\ shell\ completion\ scripts new\:Create\ a\ new\ invoice\ YAML\ file increment\:Increment\ the\ invoice\ number\ in\ an\ invoice\ YAML\ file validate\:Validate\ invoice\ YAML\ against\ customers\ and\ issuer\ data render\:Render\ a\ LaTeX\ invoice\ file email\:Create\ and\ open\ an\ email\ draft build\:Render\ and\ compile\ an\ invoice\ PDF archive\:Archive\ invoices\ and\ manage\ archived\ invoices))'
    return
  fi

  case $words[2] in
    help)
      if (( CURRENT == 3 )); then
        _values 'help topic' \
          'customer[Customer-related commands]' \
          'config[config.yaml reference]' \
          'init[Starter file initialization]' \
          'template[Template-related commands]' \
          'completion[Shell completion]' \
          'new[Create a new invoice YAML file]' \
          'increment[Increment the invoice number in an existing invoice YAML file]' \
          'validate[Validate invoice YAML against customers and issuer data]' \
          'render[Render a LaTeX invoice file]' \
          'email[Create and open an email draft]' \
          'build[Render and compile an invoice PDF]' \
          'archive[Archive invoices and manage archived invoices]' \
          'customers[customers.yaml reference]' \
          'issuer[issuer.yaml reference]' \
          'defaults[invoice_defaults.yaml reference]' \
          'invoice-defaults[invoice_defaults.yaml reference]' \
          'invoice_defaults[invoice_defaults.yaml reference]'
        return
      fi
      case $words[3] in
        customer)
          if (( CURRENT == 4 )); then
            _values 'customer help topic' \
              'list[List all customers]' \
              'config[Open customers.yaml in the default shell editor]'
            return
          fi
          ;;
        template)
          if (( CURRENT == 4 )); then
            _values 'template help topic' \
              'list[List available invoice templates]'
            return
          fi
          ;;
        completion)
          if (( CURRENT == 4 )); then
            _values 'completion help topic' \
              'zsh[Zsh completion script]'
            return
          fi
          ;;
        archive)
          if (( CURRENT == 4 )); then
            _values 'archive help topic' \
              'edit[Copy an archived invoice into the current directory]' \
              'list[List archived invoices]'
            return
          fi
          ;;
      esac
      ;;
    customer)
      if (( CURRENT == 3 )); then
        _values 'customer command' \
          'list[List all customers]' \
          'config[Open customers.yaml in the default shell editor]'
        return
      fi
      case $words[3] in
        list|config)
          _invox_shift_words 2
          if [[ $words[3] == list ]]; then
            _arguments -s \
              '(-h --help)'{-h,--help}'[show this help page]' \
              '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_invox_yaml_files' \
              '--json[print JSON output]'
            return
          fi
          _arguments -s \
            '(-h --help)'{-h,--help}'[show this help page]' \
            '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_invox_yaml_files'
          return
          ;;
      esac
      ;;
    config)
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]'
      return
      ;;
    init)
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]'
      return
      ;;
    template)
      if (( CURRENT == 3 )); then
        _values 'template command' 'list[List available templates]'
        return
      fi
      case $words[3] in
        list)
          _invox_shift_words 2
          _arguments -s \
            '(-h --help)'{-h,--help}'[show this help page]' \
            '--names[print only template names]' \
            '--json[print JSON output]'
          return
          ;;
      esac
      ;;
    completion)
      if (( CURRENT == 3 )); then
        _values 'shell' 'zsh[Zsh completion script]'
        return
      fi
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]' \
        '1:shell:(zsh)'
      return
      ;;
    new)
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]' \
        '(-o --output)'{-o+,--output=}'[output YAML path]:output:_files' \
        '(-s --source)'{-s+,--source=}'[invoice_defaults.yaml]:defaults:_invox_yaml_files' \
        '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_invox_yaml_files' \
        '(-u --issuer)'{-u+,--issuer=}'[issuer.yaml]:issuer:_invox_yaml_files' \
        '--from-last[use the latest archived invoice for this customer as the source document]' \
        '(-e --edit)'{-e,--edit}'[open the created invoice in the default shell editor]' \
        '1:customer:_invox_customer_ids'
      return
      ;;
    increment)
      if (( CURRENT == 3 )) && [[ ${words[3]:-} != -* ]]; then
        _invox_invoice_files
        return
      fi
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]' \
        '(-i --input)'{-i+,--input=}'[invoice YAML file]:invoice:_invox_invoice_files' \
        '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_invox_yaml_files' \
        '(-i --input)1::invoice:_invox_invoice_files'
      return
      ;;
    validate)
      if (( CURRENT == 3 )) && [[ ${words[3]:-} != -* ]]; then
        _invox_invoice_files
        return
      fi
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]' \
        '(-i --input)'{-i+,--input=}'[invoice YAML file]:invoice:_invox_invoice_files' \
        '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_invox_yaml_files' \
        '(-u --issuer)'{-u+,--issuer=}'[issuer.yaml]:issuer:_invox_yaml_files' \
        '--json[print JSON output]' \
        '(-i --input)1::invoice:_invox_invoice_files'
      return
      ;;
    render)
      if (( CURRENT == 3 )) && [[ ${words[3]:-} != -* ]]; then
        _invox_invoice_files
        return
      fi
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]' \
        '(-i --input)'{-i+,--input=}'[invoice YAML file]:invoice:_invox_invoice_files' \
        '(-o --output)'{-o+,--output=}'[output TeX path]:output:_files' \
        '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_invox_yaml_files' \
        '(-u --issuer)'{-u+,--issuer=}'[issuer.yaml]:issuer:_invox_yaml_files' \
        '(-t --template)'{-t+,--template=}'[template file or known template name]:template:_invox_template_values' \
        '(-i --input)1::invoice:_invox_invoice_files'
      return
      ;;
    build)
      if (( CURRENT == 3 )) && [[ ${words[3]:-} != -* ]]; then
        _invox_invoice_files
        return
      fi
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]' \
        '(-i --input)'{-i+,--input=}'[invoice YAML file]:invoice:_invox_invoice_files' \
        '(-o --output)'{-o+,--output=}'[output PDF path]:output:_files' \
        '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_invox_yaml_files' \
        '(-u --issuer)'{-u+,--issuer=}'[issuer.yaml]:issuer:_invox_yaml_files' \
        '(-t --template)'{-t+,--template=}'[template file or known template name]:template:_invox_template_values' \
        '(-i --input)1::invoice:_invox_invoice_files' \
        '--archive[archive after a successful PDF build]'
      return
      ;;
    email|send)
      if (( CURRENT == 3 )) && [[ ${words[3]:-} != -* ]]; then
        _invox_email_input_files
        return
      fi
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]' \
        '(-i --input)'{-i+,--input=}'[invoice YAML or PDF file]:input:_invox_email_input_files' \
        '(-p --pdf)'{-p+,--pdf=}'[invoice PDF path]:pdf:_invox_pdf_files' \
        '(-o --output)'{-o+,--output=}'[output EML path]:output:_files' \
        '(-c --customers)'{-c+,--customers=}'[customers.yaml]:customers:_invox_yaml_files' \
        '(-u --issuer)'{-u+,--issuer=}'[issuer.yaml]:issuer:_invox_yaml_files' \
        '--to[recipient email override]:email address:' \
        '--subject[email subject override]:subject:' \
        '--no-open[create the draft file without opening it]' \
        '(-i --input)1::input:_invox_email_input_files'
      return
      ;;
    archive)
      if (( CURRENT == 3 )); then
        _alternative \
          'commands:archive command:((edit\:copy\ an\ archived\ invoice\ into\ the\ current\ directory list\:list\ archived\ invoices))' \
          'invoice:invoice:_invox_invoice_files'
        return
      fi
      case $words[3] in
        edit)
          _invox_shift_words 2
          _arguments -s \
            '(-h --help)'{-h,--help}'[show this help page]' \
            '1:archived invoice:_invox_archive_names'
          return
          ;;
        list)
          _invox_shift_words 2
          _arguments -s \
            '(-h --help)'{-h,--help}'[show this help page]' \
            '--json[print JSON output]'
          return
          ;;
      esac
      _invox_shift_words 1
      _arguments -s \
        '(-h --help)'{-h,--help}'[show this help page]' \
        '(-i --input)'{-i+,--input=}'[invoice YAML file]:invoice:_invox_invoice_files' \
        '(-i --input)1::invoice:_invox_invoice_files'
      return
      ;;
  esac

  _files
}

compdef _invox invox
`
}
