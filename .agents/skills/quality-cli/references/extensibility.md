# Extensibility: extensions, aliases, API escape hatch

Only add these when the tool has users who need to go beyond built-ins. Prefer gh's
model (extensions as `tool-<name>` executables) over docker's richer plugin protocol
unless you need host-provided lifecycle management.

## Contents
- [Extensions (gh model, preferred)](#extensions-gh-model-preferred)
- [Plugins (docker model)](#plugins-docker-model)
- [Aliases](#aliases)
- [Raw API escape hatch](#raw-api-escape-hatch)

---

## Extensions (gh model, preferred)

`pkg/cmd/extension`, `pkg/extensions`.

- **Naming**: repo/binary `tool-<name>` → invoked as `tool <name>`.
- **Kinds**: script (git clone, executable `tool-<name>` at repo root), precompiled (release assets named `tool-<name>-<GOOS>-<GOARCH>[.exe]`, recorded in `manifest.yml`), local (symlink, for development via `tool ext install .`).
- **Storage**: data dir, `extensions/tool-<name>/`.
- **Dispatch**: each discovered extension becomes a root subcommand in the `extension` group with `DisableFlagParsing: true`; args passed through verbatim; exit code passed through (`ExternalCommandExitError`).
- **Never shadow core commands** (`cmd.Find` first; installing a conflicting name fails). Provide `tool extension exec <name>` for conflicts.
- **Environment for the child**: `TOOL_EXTENSION=1`-style marker, `TOOL_PATH=<abs path of tool binary>` so the extension calls back into the *same* tool (and auth/config). Ship a small SDK (`go-gh`) that reuses the host's config, auth and IO conventions.
- **Install**: pick an asset matching `GOOS-GOARCH`; fall back to amd64 on Apple Silicon when Rosetta is present; if nothing matches, print a ready-to-run "request support" command. `--pin <tag|sha>`. On Windows run script extensions via `sh` so shebangs work.
- **Updates**: `tool ext upgrade [--all] [--dry-run]`; async update notice every 24h in `PostRun`, only if the check already finished — never blocks.
- **Discovery**: `tool ext search` / `browse` (repo topic `tool-extension`); `tool ext create [--precompiled=go]` scaffolds a repo with a release workflow.
- **Trust**: say clearly that extensions are not verified/endorsed. Track "official" extensions by owner, not name.
- **Telemetry/privacy**: don't record extension or alias names (they may be sensitive).

## Plugins (docker model)

`cli-plugins/`. Use when plugins must look *native* (share global flags, help, contexts).

- **Discovery**: `docker-<name>` executables in ordered dirs (config `cliPluginsExtraDirs`, `~/.docker/cli-plugins`, system dirs). Earlier wins; later ones reported as shadowed.
- **Handshake**: run `<plugin> docker-cli-plugin-metadata` → JSON `{SchemaVersion, Vendor, Version, ShortDescription, URL}`. Validate: name `^[a-z][a-z0-9]*$`, no clash with builtins, valid JSON, accepted schema major, non-empty vendor.
- **Failures are data**: invalid plugins get `Plugin.Err` and appear under "Invalid Plugins" in help — the CLI never crashes because of a bad plugin.
- **Performance**: only scan plugins for help and completion; for execution, look up the single requested name.
- **Lifecycle**: host opens a Unix socket and passes its address via env; closing it (EOF) means "terminate". No framing, backward compatible both ways.
- **SDK**: `plugin.Run(newCmd, meta)` gives plugins the full host CLI object (streams, config, context, global flags).
- **Completion**: delegate to `<plugin> __complete ...`.
- **Hooks**: opt-in in config; host calls the plugin after matching commands with a JSON request and prints its "What's next" template — TTY only, failures logged at debug level and ignored. Public request/response types: only add fields; consumers ignore unknown ones.

## Aliases

gh `pkg/cmd/alias`, stored in config `aliases:`.
- `tool alias set <name> <expansion>`; `list`, `delete`, `import <file|->` (YAML).
- Placeholders `$1`, `$2`; extra args appended; leftover placeholders → "not enough arguments for alias".
- Shell aliases: `!` prefix or `--shell` → `sh -c '<expr>' -- args` (Git-for-Windows sh on Windows); exit codes pass through.
- Validation: alias name must not shadow a command; expansion must resolve to a command/extension/alias unless shell; `--clobber` to overwrite.
- Expansion `-` reads from stdin to avoid quoting hell.
- Help for an alias shows the target command's help. Aliases appear in an "Alias commands" group.
- Ship one useful default alias to teach the feature (`co: pr checkout`).

## Raw API escape hatch

gh `pkg/cmd/api/api.go` — highest leverage feature for power users and agents:
- `tool api <endpoint>` with placeholders `{owner}`, `{repo}`, `{branch}` filled from context.
- `-f key=value` raw string; `-F key=value` typed (`true/false/null/ints`, `@file`, `@-` for stdin), nested `key[sub]=v`, arrays `key[]=v`.
- Method defaults to GET, switches to POST when fields are present; `-X` overrides.
- `--paginate` (REST `Link` header / GraphQL `endCursor`), `--slurp` to wrap pages in one array.
- `--jq`, `--template` built in; `-i` include headers; `--cache 1h`; `--verbose`; `-H` headers; `--input file`.
- Output escape sequences neutralized unless `--allow-escape-sequences`.
- Validate combinations (`--slurp` requires `--paginate`, excludes `--jq`).
