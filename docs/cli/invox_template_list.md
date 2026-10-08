# invox template list

```text
List available LaTeX invoice templates from the same directory as the resolved default template.

Output:
  Default: NAME<TAB>ABSOLUTE_PATH
  --names: TEMPLATE_NAME per line
  --json: an array of the requested fields; default is true for the template
    render and build use from the current directory without -t
  On a terminal, aligned columns under a header. Piped, \, tab, CR and LF in a field are written as \\, \t, \r and \n.

Lookup:
  The directory is derived from the resolved default template path.
  Name-only -t/--template values are resolved in that same directory.

Usage:
  invox template list [flags]

Flags:
      --json fields   Output JSON with the specified fields
      --names         Print only template names

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

JSON fields:
  default, name, path

Examples:
  $ invox template list
  $ invox template list --names
  $ invox template list --json name,default
  $ invox build invoice.yaml -t multi_vat.tex
```

## See also

- [invox template](invox_template.md): Template-related commands
