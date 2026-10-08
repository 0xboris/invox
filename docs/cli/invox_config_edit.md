# invox config edit

```text
Open config.yaml in your editor.

Opens the resolved config.yaml, or the file given with --config. If it does
not exist yet, creates it from the template that `invox help config` shows.

Usage:
  invox config edit [flags]

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

Examples:
  $ invox config edit
  $ invox --config ./config.yaml config edit
```

## See also

- [invox config](invox_config.md): Open config.yaml in your editor
