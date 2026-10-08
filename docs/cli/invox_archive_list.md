# invox archive list

```text
List archived invoices from the configured archive directory.

Default lookup:
  archive.dir: config.yaml, then $HOME/.local/share/invox/invoices

Output:
  One archived invoice per line as FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS
  On a terminal, aligned columns under a header. Piped, \, tab, CR and LF in a field are written as \\, \t, \r and \n.

Usage:
  invox archive list [flags]

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

Examples:
  $ invox archive list
```

## See also

- [invox archive](invox_archive.md): Archive a built or edited invoice YAML file into the configured archive directory
