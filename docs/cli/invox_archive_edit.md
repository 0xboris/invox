# invox archive edit

```text
Copy an archived invoice into the current directory and mark it as editing.

Required inputs:
  FILENAME                  Required positional argument

Default lookup:
  archive.dir: config.yaml, then $HOME/.local/share/invox/invoices

Behavior:
  Copies the archived invoice from archive.dir into the current directory.
  The working copy is written as YAML with invoice.status set to editing.
  Re-running invox archive on that working copy replaces the archived invoice.
  It asks first, or needs --yes without a terminal, and keeps the previous version in archive.dir/.history.

Usage:
  invox archive edit FILENAME [flags]

Flags:
      --json fields   Output JSON with the specified fields

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

JSON fields:
  archivedPath, path

Examples:
  $ invox archive edit 2026-03-06.yaml
  $ invox archive edit customer-a/2026-03-06.yaml
  $ invox archive edit 2026-03-06.yaml --json path
```

## See also

- [invox archive](invox_archive.md): Archive a built or edited invoice YAML file into the configured archive directory
