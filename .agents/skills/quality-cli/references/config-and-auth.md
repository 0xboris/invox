# Config, environment and credentials

## Contents
- [Precedence](#precedence)
- [Locations](#locations)
- [Config file design](#config-file-design)
- [`tool config` command](#tool-config-command)
- [Atomic, safe writes](#atomic-safe-writes)
- [Schema evolution](#schema-evolution)
- [Credentials](#credentials)
- [Auth UX](#auth-ux)
- [Environment variable hygiene](#environment-variable-hygiene)

---

## Precedence

**flag > environment variable > config file (per-host/per-context > global) > built-in default.**
Write it down in `tool help environment` and in the doc comment of the code that resolves it.
`GetOrDefault` should also return the **source** (`ConfigUserProvided` / `ConfigDefaultProvided`,
or "keyring"/"TOOL_TOKEN") so `tool auth status` / `tool config list` can explain where a value came from.

**Presence, not truthiness.** An explicitly set `false`, `0` or `""` must beat a
lower layer's non-empty value:
- flags: `cmd.Flags().Changed("enabled")` (or `*bool` via `NilBoolFlag`);
- env: `v, ok := os.LookupEnv("TOOL_ENABLED")`, never `os.Getenv(...) != ""`;
- config: pointer fields (``Enabled *bool `yaml:"enabled"` ``), and no `omitempty`
  when writing values whose zero is meaningful.
Resolve every layer first, then validate the resolved value; marking a flag
"required" doesn't model a value that config or env could supply. Reject an
explicitly empty value that isn't allowed instead of treating it as unset.

## Locations

gh (`go-gh/pkg/config`):
- Config dir: `TOOL_CONFIG_DIR` > `$XDG_CONFIG_HOME/tool` > `%AppData%/Tool` (Windows) > `~/.config/tool`.
- **Separate by purpose**: config (`config.yml`, user-edited), auth (`hosts.yml`), state (`$XDG_STATE_HOME`: update-check timestamps), data (`$XDG_DATA_HOME`: extensions), cache (`$XDG_CACHE_HOME`).
- Docker uses one dir (`DOCKER_CONFIG` > `~/.docker`) with a `--config` flag — prefer gh's XDG split, but a `--config`/`TOOL_CONFIG_DIR` override is essential for tests and multiple setups.
- Resolve once (`sync.Once`); reject paths that escape the root (`filepath.IsLocal`).
- `os.UserConfigDir()` gives the platform-native base (`~/Library/Application Support` on
  macOS); gh uses `~/.config` everywhere via its own XDG logic. Pick one, document it,
  and never hard-code a Unix path for all platforms.
- **Explicit config file** (`--config path` or `TOOL_CONFIG`): it replaces automatic
  discovery, it doesn't outrank explicit flags or env values, a missing explicit file
  is an error, and a parse error is fatal. Never silently fall back to defaults. An
  absent *discovered* file is fine.
- Don't auto-load `.env` files or walk up project directories unless there's a
  real use case. Never edit another program's config without asking.

## Config file design

- Human-editable YAML, shipped as a **commented default** where each key documents allowed values:
  ```yaml
  # What protocol to use when performing git operations. Supported values: ssh, https
  git_protocol: https
  # What editor to use for authoring text. Leave blank to use $EDITOR.
  editor:
  # When to interactively prompt. Supported values: enabled, disabled
  prompt: enabled
  # A pager program to send command output to, e.g. "less". Set to "cat" to disable.
  pager:
  # Aliases allow you to create nicknames for commands.
  aliases:
    co: pr checkout
  ```
- Keys are declared in a table (gh `config.Options`): `Key, Description, DefaultValue, AllowedValues, Scope (global/host)`. That table drives validation, `config list`, completion **and** the `config --help` text.
- Per-host (or per-context) overrides: `hosts.<host>.<key>` checked before the global key.
- Unknown keys are preserved, not deleted. Malformed config: report clearly; never silently overwrite it.
- Per-command defaults (docker `psFormat`, `imagesFormat`) are a valid pattern for heavily-customized output.
- Plugins/extensions get a namespace (`plugins.<name>.<key>`) instead of polluting the top level.

## `tool config` command

`config get <key> [-h host]`, `config set <key> <value> [-h host]` (validates against allowed values),
`config list`, `config clear-cache`. Long help generated from the options table. `config set`
of an unknown key: warn or error with the known keys listed.

## Atomic, safe writes

Docker `configfile.Save` is the model:
1. `os.CreateTemp` in the **same directory** as the target (same filesystem → atomic rename).
2. Write, `Close`, check errors.
3. Resolve symlinks (including dangling) so you replace the target, not the link.
4. Copy permissions/ownership from the existing file (default `0600` for files holding secrets).
5. `os.Rename(tmp, target)`; remove the temp file on any error.

Also: skip the write when nothing changed (docker `context use`), create dirs with `0700`/`0755`
as appropriate, and never write in read-only commands. Use `github.com/google/renameio` or
`moby/sys/atomicwriter` if you don't want to hand-roll it.

## Schema evolution

- Store a `version` key. Migrations are explicit, ordered, idempotent functions.
- If state is ambiguous, **refuse** with a clear message (gh `CowardlyRefusalError`) rather than guess.
- Back up before migrating when the file contains credentials.
- Tolerate legacy keys for at least one release (docker still parses `"experimental": "enabled"`).

## Credentials

- **OS keyring first** (`github.com/zalando/go-keyring`): service `"tool:" + host`, key = username (supports multiple accounts per host and `auth switch`).
- **Timeouts on every keyring call** (gh: 60s) — a locked/hung keychain must not hang the CLI.
- Never accept a secret as a flag value (visible in `ps` and shell history). Read it from
  stdin (`tool auth login --with-token < token.txt`), a `--password-file`, the keyring,
  or a token env var. Env tokens (`TOOL_TOKEN`) are allowed (CI needs them; gh does
  this), but never log them or include them in `--json`/debug output.
- Plaintext fallback only by explicit opt-in (`--insecure-storage`) and say so (`insecureStorageUsed`). Docker prints a one-time "unencrypted" warning per process.
- External credential helpers (docker `docker-credential-<name>` binaries chosen per registry) are a good extension point for enterprise setups.
- Env tokens win over stored ones: `TOOL_TOKEN` > `TOOL_API_TOKEN`-style legacy name > keyring > config. Separate env vars for self-hosted/enterprise hosts (`GH_ENTERPRISE_TOKEN`).
- Never log tokens; classify by prefix (`gho_`, `ghp_`) without exposing them.
- `auth token` prints the token to stdout (for piping into other tools) — nothing else.
- Use a separate unauthenticated HTTP client for anything that isn't your API host so tokens never leak to third parties.

## Auth UX

- `auth login` (interactive: host → protocol → web flow or paste token; non-interactive: `--with-token < file`), `auth logout`, `auth status` (shows each host, account, token source, scopes, and problems), `auth refresh` (add scopes), `auth switch`, `auth token`.
- Root `PersistentPreRunE` blocks commands when not authenticated, except annotated opt-outs; exit code `4`.
- Hints adapt to environment:
  - GitHub Actions: show the YAML snippet `env:\n  GH_TOKEN: ${{ github.token }}`.
  - Other CI: "set the TOOL_TOKEN environment variable".
  - Interactive: "run `tool auth login`".
- HTTP 401 → suggest `auth refresh` if the token can be refreshed, else `auth login`. Missing scopes → name them and the refresh command.

## Environment variable hygiene

- Prefix everything with your tool name (`TOOL_*`); honor the cross-tool standards (`NO_COLOR`, `CLICOLOR*`, `PAGER`, `EDITOR`/`VISUAL`, `BROWSER`, `XDG_*`, `HTTPS_PROXY`, `DO_NOT_TRACK`).
- Declare names as constants with doc comments stating precedence.
- Parse booleans with `strconv.ParseBool` and fail with a descriptive message: `TOOL_FOO environment variable expects boolean value: %w`.
- Deprecate env vars like flags: keep reading the old one, document it as "(deprecated)" (gh `DEBUG` → `GH_DEBUG`).
- Commonly needed: `TOOL_TOKEN`, `TOOL_HOST`, `TOOL_REPO`/context override, `TOOL_CONFIG_DIR`, `TOOL_DEBUG`, `TOOL_PAGER`, `TOOL_EDITOR`, `TOOL_BROWSER`, `TOOL_PROMPT_DISABLED`, `TOOL_NO_UPDATE_NOTIFIER`, `TOOL_FORCE_TTY`, `TOOL_SPINNER_DISABLED`, `TOOL_TELEMETRY`.
