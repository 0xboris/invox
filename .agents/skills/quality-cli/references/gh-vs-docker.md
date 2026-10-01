# Where gh and docker disagree — decisions

Rule: **follow gh**. Use docker's approach only where gh has nothing, or for the
specific situations noted.

| Topic | gh | docker | Decision |
|---|---|---|---|
| Command grammar | strict `noun verb` | legacy flat verbs (`ps`, `rmi`) + `container ls`; shortcuts hideable | **gh**. Use docker's shared-constructor alias trick only when migrating an existing flat CLI. |
| Dependency injection | `Factory` struct of lazy funcs; commands copy what they need into Options | `command.Cli` interface + functional options + `sync.Once` client | **gh**. Borrow `sync.Once` for caching a lazy value used several times. |
| Run signature | `xRun(opts)`; context via cobra | `runX(ctx, cli, opts)` | **gh**, but pass `cmd.Context()` into Options (or as first arg) so cancellation works. |
| Exit codes | 0/1/2 cancel/4 auth/8 pending | 1, 125 usage, 128+signal (126/127 in `docker run`) | **gh** set, plus `128+signal` (`130`/`143`) when a handled signal ends the run. |
| Structured output | `--json fields` + `--jq` + `--template` | `--format table\|json\|<template>` | **gh**. Docker's per-command default format in config is an optional extra. |
| Piped output | TSV, no header, no truncation, RFC3339 | template-driven; table header kept | **gh**. |
| Skip confirmation | `--yes` (`--confirm` deprecated) | `-f/--force` | **gh** `--yes`; reserve `--force` for overriding safety checks. |
| Confirm default | type-the-name for high-impact deletes | `[y/N]` default No | **gh** for irreversible deletes; docker's `[y/N]` for lighter ops. |
| Help rendering | uppercase sections, groups, JSON FIELDS/ARGUMENTS/ENVIRONMENT annotations, LEARN MORE | `Usage:` / `Options:` / "Management Commands", aliases with full paths | **gh**. |
| Typo suggestions | yes, incl. nested commands | none | **gh**. |
| Help topics | `help environment/formatting/exit-codes/reference` | docs site only | **gh**. |
| Extensions | `gh-<name>` repos, script/binary, no shadowing, `GH_PATH` | `docker-<name>` + metadata handshake + socket lifecycle + hooks | **gh**. Use docker's protocol only if plugins must share host state/flags natively. |
| Global flags | almost none; per-command `-R`, env `GH_REPO` | `--context`, `--host`, `--config`, `--debug` before the subcommand | **gh**. A global `--config`/`TOOL_CONFIG_DIR` env is fine. |
| Config location | XDG split: config/state/data | single `~/.docker` | **gh**. |
| Config format | YAML with comments, options table | JSON | **gh**. |
| Docs generation | gen-docs → website markdown + man pages | marker blocks in hand-written md + CI drift check | **gh** generation **plus docker's drift check**. |
| Test assertions | inline expected strings, testify | golden files, gotest.tools | **gh** for short output; golden files for long output (help, big tables). |
| E2E | testscript `.txtar`, in-process `Main()` | `icmd` running the binary | **gh**. |
| Signals | plain `context.Background()`; SIGINT falls back to Go's default (process exits) | cause-carrying cancel, 3× force exit, terminal restore | Use `signal.NotifyContext` (the starter does) so Ctrl-C cancels in-flight work; **adopt docker's** 3× force-exit + terminal restore if you use raw mode or long streams. |
| Lint | `default: none` curated set | ~50 linters, depguard/forbidigo | gh set + selected docker linters (`forbidigo` to enforce I/O rules). |
| Feature gating | feature detection per host (introspection) | annotations for API version/OS/experimental, hide unsupported in help | gh's capability detection; **adopt docker's** annotations if you talk to servers with varying versions. |

## Decisions beyond gh and docker

Rules where this skill deliberately differs from gh, or settles a question gh and
docker leave open (from clig.dev and the former go-cli-development skill):

| Topic | gh | This skill | Why |
|---|---|---|---|
| Prompt channel | prompts on stdout; `CanPrompt` = stdin && stdout TTY | prompts on **stderr**; `CanPrompt` = stdin && **stderr** TTY | `tool x > file` can still ask, and the file only gets data |
| Disabling prompts | `GH_PROMPT_DISABLED` env / config | also a global `--no-input` flag | deterministic per-invocation switch for CI and agents (a PTY doesn't prove a human) |
| Telemetry | opt-out (`GH_TELEMETRY`, `DO_NOT_TRACK`) | **opt-in** only | clig.dev: never phone home without explicit consent |
| Secrets | `GH_TOKEN` env, `--with-token` via stdin | same, and **never** a secret-valued flag | argv leaks via `ps` and shell history |
| Exit code `2` | cancelled | cancelled (kept); clig.dev's `2` = usage error is **not** adopted | consistency with gh; existing tools keep their documented codes |
| Non-TTY fields | raw TSV | TSV with `\t` `\n` `\\` escaped | one record per line, always |
| Color off switches | `NO_COLOR`, `CLICOLOR` | also `TERM=dumb`, `--no-color` | clig.dev conventions |
| Dry run | per-command | `-n/--dry-run` = same selection + validation, zero mutations | preview must not drift from execution |

Things only docker does that are worth adopting anywhere:
- Atomic config writes that preserve symlinks/permissions.
- Multi-target commands that continue past failures and `errors.Join` at the end.
- Cost-aware fetching driven by what the output template uses (gh does the same via `--json` fields).
- `inspect` fallback to raw JSON for fields newer than the client.
- Plugin/extension failures reported as data, never crashes.

Things only gh does that are worth adopting anywhere:
- AI-agent detection → full help on flag errors, no spinner.
- `NoResultsError` and closed-pager-is-success semantics.
- `PreserveInput` + `--recover` for long typed input.
- Accessible prompter mode.
- Self-verifying HTTP/exec/prompter stubs.
