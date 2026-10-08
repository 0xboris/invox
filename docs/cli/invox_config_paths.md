# invox config paths

```text
Show where each config and support file is read from.

Output:
  One row per path with the columns NAME, PATH and SOURCE, in this order:
  config-dir, config, customers, issuer, defaults, template, archive.
  The support files are looked up from the current directory, as a command
  run here would. PATH is empty when nothing is found. On a terminal the
  rows are aligned under a header; piped, they are tab-separated.

Sources:
  flag     the --config option
  env      INVOX_CONFIG_DIR
  default  the default config or archive directory
  legacy   the deprecated invoice-tool directory
  project  the upward search from the current directory
  config   a paths.* or archive.dir setting in config.yaml
  none     not found

Usage:
  invox config paths [flags]

Global flags:
      --config string   Read this config file instead of config.yaml
  -h, --help            Show help for a command
      --no-input        Never prompt or open an editor; fail with exit 2 instead

Examples:
  $ invox config paths
```

## See also

- [invox config](invox_config.md): Open config.yaml in your editor
