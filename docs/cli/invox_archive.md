# invox archive

```text
Archive invoices, and list or edit archived ones.

Default lookup:
  archive.dir: config.yaml, then $HOME/.local/share/invox/invoices

Usage:
  invox archive <subcommand> [flags]

Commands:
  add   Archive a built or edited invoice YAML file into the configured archive directory
  edit  Copy an archived invoice into the current directory and mark it as editing
  list  List archived invoices from the configured archive directory

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

Examples:
  $ invox archive add invoice.yaml
  $ invox archive list
  $ invox archive edit 2026-03-06.yaml
```

## See also

- [invox](invox.md): Generate LaTeX and PDF invoices from YAML data
- [invox archive add](invox_archive_add.md): Archive a built or edited invoice YAML file into the configured archive directory
- [invox archive edit](invox_archive_edit.md): Copy an archived invoice into the current directory and mark it as editing
- [invox archive list](invox_archive_list.md): List archived invoices from the configured archive directory
