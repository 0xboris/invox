---
name: quality-cli
description: >
  Architecture, UX and engineering standards for building production-quality
  command-line tools, distilled from the GitHub CLI (gh) and Docker CLI source. Use
  when creating, extending, refactoring, reviewing or testing a CLI: adding commands,
  subcommands, flags or arguments; CLI help text, error messages and exit codes;
  prompts, confirmations, colors, spinners, pagers, tables or --json output in a
  terminal tool; a CLI's config files, env vars, credentials, shell completion, man
  pages, plugins or release packaging. Also use when the code is a main package with
  subcommands or argument parsing (cobra, urfave/cli, flag, clap, click, typer,
  argparse, commander, oclif) even if the user never says "CLI". Not for merely
  running existing CLIs (gh, docker, git), or for errors, config and releases of web
  services and libraries that have no command-line interface. Go/cobra-first; the
  principles apply to any language.
---

# Quality CLI

Build CLIs that are predictable for scripts, pleasant for humans, safe for agents,
and easy to test. **Where gh and docker disagree, follow gh** (`references/gh-vs-docker.md`).

## How to use this skill

1. Apply the **behavior rules** below to every CLI change. Scale the **structure
   rules** to the tool's size and existing code.
2. **Existing CLI?** Never change what scripts depend on (see "Script-facing contract").
   Apply the rules to new code; match existing conventions otherwise; don't refactor
   wholesale unless asked (`references/architecture.md` §Migrating has the safe order).
3. Find the task in "Task recipes" and open only the references it names.
   For a new command or a changed interface, sketch it before coding: 3–5 example
   invocations with their stdout, stderr and exit code, including the failure and
   non-interactive cases. For a new tool or noun, also sketch the packages: domain
   types and operations, adapters for external programs, and who imports whom
   (`references/layers.md`). Use the `create-cli` skill when a full spec is wanted.
4. **New Go CLI?** Start from the starter template, pinned to the version these rules
   describe: `git clone --depth 1 --branch v0.3.0 https://github.com/0xboris/go-cli-starter`
   (drop `--branch` if the tag is missing). It compiles, is tested, and its README
   explains renaming. If it can't be cloned, scaffold from `references/architecture.md`,
   reading "Cobra gotchas" first.
5. Code snippets in the references come from gh's own packages (`cmdutil`,
   `iostreams`, `httpmock`, `run`, ...). They are **not importable** (many are
   `internal/`). Copy the pattern, or use the starter's equivalents. The only
   supported reusable library from gh is `github.com/cli/go-gh/v2`.

Go names in parentheses are examples; `references/other-languages.md` maps them to
Python, Node and Rust.

## Behavior rules (always)

**Exit codes & errors**
- Commands return errors; one place maps them to exit codes and prints once, to
  stderr. Codes (gh's): `0` ok, `1` error (including usage errors), `2` cancelled,
  `4` auth required, `8` pending (optional), and `128+signal` (`130` Ctrl-C, `143`
  SIGTERM) when a handled signal ends the run. Document them in `help exit-codes`.
  (clig.dev and many tools use `2` for usage errors; we follow gh. Keep whatever an
  existing tool already documents.)
- Print usage **only** for usage errors (`FlagError`), never for runtime failures.
  Turn off the framework's own error/usage printing (cobra `SilenceErrors`,
  `SilenceUsage`).
- Not failures: an empty result (exit 0, message on TTY only), the user quitting the
  pager (EPIPE → exit 0). A declined prompt, or Ctrl-C/EOF *at a prompt* → exit 2,
  quiet. Ctrl-C while an operation runs → `130`.
- Errors say what to do next: the missing flag, valid values, or the command to run.

**Streams**
- stdout = the data asked for (rows, JSON, the URL/path of what was created).
  stderr = progress, warnings, hints, update notices **and prompts**.
- TTY and non-TTY are two contracts. TTY: color, header, truncation, relative time,
  spinner. Non-TTY: no color, no header, tab-separated, no truncation, RFC3339 time,
  state as words, and `\t` `\n` `\\` escaped inside fields so a record stays one line.
- `--json` output is one JSON value built completely before writing (a failure leaves
  stdout empty): `[]` for an empty list, never `null`; dedicated output types (no
  internal structs, no secrets); no `omitempty` on meaningful zero values. Check
  write/flush errors.
- No color when `NO_COLOR` is set, `TERM=dumb`, `CLICOLOR=0` or `--no-color`;
  `CLICOLOR_FORCE` forces it; offer `TOOL_FORCE_TTY`. Color adds meaning, never
  carries it alone. Measure column widths in terminal cells, not bytes or runes.
- Strip ANSI escapes from remote/untrusted text before printing it.

**Interactivity**
- Prompt only when stdin **and** stderr (the prompt's channel) are TTYs and prompting
  isn't disabled (`--no-input`, `TOOL_PROMPT_DISABLED`). Every prompt has a flag;
  otherwise fail promptly with a `FlagError` naming those flags. Never hang on stdin;
  EOF or Ctrl-C at a prompt is never consent.
- Destructive actions: confirm on a TTY, require `--yes` otherwise; for high-impact
  deletes make the user type the target's name. `--yes` alone must not delete an
  implicit target (the current directory/repo). `--yes` only answers the
  confirmation: it never supplies missing values or skips validation.
- `--yes` (skip confirmation), `--force` (override a documented safety check) and
  `-n/--dry-run` (preview: same selection and validation as the real run, no
  mutation) are three different flags.
- Pass the command's context down to network and subprocess calls so Ctrl-C cancels.
  A cancelled wait doesn't mean remote work stopped: report its state/ID.

**Configuration**
- Precedence flag > env var > config file > default, documented in `help environment`.
  An explicitly set `false`, `0` or `""` beats lower layers (track presence, not
  truthiness). An explicit `--config` path replaces discovery; if it's missing or
  fails to parse, error out instead of falling back.
- Writes are atomic (temp file + rename, keep mode). Secrets go to the OS keyring
  with timeouts; plaintext only by explicit opt-in. Never accept a secret as a flag
  value (use stdin, a file or the keyring); token env vars (`TOOL_TOKEN`) are fine.
- `--help`, `version` and completion work offline with a broken config.

## Structure rules (scale to the tool)

- **Commands are thin; business logic lives below them.** Domain/client packages
  (gh `api/`, docker's moby client) hold typed models, operations and typed errors,
  take `ctx` first, and never import cobra/IOStreams, print, read env/config/clock or
  exit. Imports point down only; enforce it with depguard/forbidigo
  (`references/layers.md`).
- Each external program sits behind one adapter type (gh `git.Client`): injected
  streams (child stdout → stderr unless it is the data), `exec.CommandContext`, a
  typed error carrying exit code and stderr, an injected test seam.
- Verbs of one noun share code through `pkg/cmd/<noun>/shared` (finders, listers,
  display helpers); a leaf command never imports another leaf. Code two nouns share
  belongs in the domain.
- Typed data end to end: parse files/responses/flags into structs once (no
  `map[string]any` models), params structs instead of runs of positional strings,
  value types for identities (gh `ghrepo`), `pflag.Value` types that validate in `Set`
  (docker `opts/`).
- `main` is a shim (`os.Exit(int(app.Main()))`); `Main()` returns the code so
  deferred cleanup runs and tests can call it in-process.
- Commands get dependencies injected (Factory + IOStreams): no direct stdout/stdin,
  env, clock, subprocess or global HTTP client. Expensive ones are lazy
  (`func() (T, error)`).
- Each command = Options struct + constructor that parses/validates flags + run
  function with no framework imports (`NewCmdX(f, runF)` + `xRun(opts)`; validate in
  `RunE` before `runF` so tests can stop after parsing).
- Grammar `tool <noun> <verb> [args] [flags]`; modifiers are flags; siblings share flag
  names (`--limit`, `--json`, `--web`).
- List/view commands offer `--json <fields>` (validated; bare `--json` lists fields);
  add `--jq`/`--template` when users script against the tool.
- Help is data (`Short`, `Long`, `Example` with `$ tool …` lines); generate man pages
  and markdown from the command tree once the tool is distributed.
- Background work (update checks, telemetry) never blocks, is cancellable, and is off
  in CI and non-TTY. Telemetry is opt-in only (deliberately stricter than gh).

## Script-facing contract

Once shipped these are public API: flag names, arguments, defaults, exit codes, error
wording scripts match on, JSON fields, non-TTY output, stdout/stderr routing. Change
them only by deprecation (hidden alias + warning, kept ≥1 release).

## Task recipes

| Task | Do | Open |
|---|---|---|
| **New CLI** | Clone the starter (`v0.3.0`) and rename; otherwise scaffold: shim `main`, `app.Main`, IOStreams, Factory, typed errors, root with help topics, one noun with `list` + a mutating verb, a domain package behind it, tests from day one | `references/architecture.md`, `references/layers.md`, `references/testing.md` |
| **Add a command** | New package `pkg/cmd/<noun>/<verb>`; Options + constructor + run; register it; `Short`, `Long`, `Example`; tests for parsing and for output in both TTY modes; domain logic in the domain layer, not the command | `references/architecture.md`, `references/testing.md` |
| **Design packages / add business logic** | Layer stack, domain client with typed models and errors, adapters for external programs, `shared/` per noun, presentation helpers, lint-enforced import rules | `references/layers.md` |
| **Add/change a flag** | Bind to Options; validate before run; enum/tri-state/mutually-exclusive helpers; completion; renames keep the old name hidden+deprecated | `references/ux-and-help.md` §Flags |
| **Output (tables, JSON, colors)** | IOStreams + table printer + exporter; check both TTY modes | `references/io-and-output.md` |
| **Prompts / editor / destructive ops** | `CanPrompt()` gate, flag fallback, `--yes`, Prompter interface | `references/ux-and-help.md` §Interactivity |
| **Errors & exit codes** | Typed errors, single mapping, actionable text | `references/architecture.md` §Errors, `references/ux-and-help.md` §Errors |
| **Help, docs, completion** | Help sections, topics, annotations, generated docs | `references/ux-and-help.md` §Help, `references/build-and-release.md` §Docs |
| **Config, env vars, auth** | Precedence, dirs, atomic writes, keyring | `references/config-and-auth.md` |
| **Plugins, aliases, API escape hatch** | `tool-<name>` executables, never shadow core commands | `references/extensibility.md` |
| **Signals, update checks, telemetry** | Context cancellation; async, never-blocking work | `references/architecture.md` §Signals, `references/build-and-release.md` |
| **Tests** | Buffer-backed IOStreams, self-verifying HTTP/exec/prompt stubs, exact output, testscript | `references/testing.md` |
| **Build, release, lint, CI** | ldflags version, GoReleaser, linters, docs drift check | `references/build-and-release.md` |
| **Review a CLI diff** | Run the audit script, walk the checklist, use its report format | `references/review-checklist.md` |
| **Refactor a hand-rolled CLI** | Freeze output with tests, then IOStreams, typed errors, domain extraction, one command at a time | `references/architecture.md` §Migrating, `references/layers.md` §Anti-patterns |
| **Non-Go CLI** | Same rules; map the concepts | `references/other-languages.md` |

## Audit script (Go only)

```sh
bash <skill-dir>/scripts/cli_audit.sh <repo-root> [--strict]
```

Grep-based heuristics for the rules above (raw stdio, `os.Exit` outside main,
scattered env reads, no TTY detection, no `--json`, untested packages, layer leaks
such as cobra/streams imported below the commands or one command importing another,
...). WARN means "look here", not "wrong"; gh itself gets several. Exit: 0, or 1 with `--strict` when
anything warned; 2 if no Go sources. To check only your change, run it before and
after and compare the WARN lines.

## Before you finish (loop until clean)

1. Non-TTY: `tool … | cat`, `tool … </dev/null` and `tool … --no-input`. Output has no
   color or header, and missing input gives a `FlagError` naming the flags, not a hang.
2. TTY: `script -qec 'tool …' /dev/null` (Linux) or `script -q /dev/null tool …`
   (macOS), or `TOOL_FORCE_TTY=1 tool …`.
3. Exit codes: build the binary (not `go run`, which hides the real code) and check
   `echo $?` after success, a usage error and a runtime error. Cancellation (2) and
   signals (130) are covered by tests. For prompts and streams, the full
   stdin/stdout/stderr matrix is in `references/testing.md`.
4. `tool <cmd> --help` has an Example and works with `TOOL_CONFIG_DIR=$(mktemp -d)`.
5. Tests cover parsing and output in both TTY modes, and domain packages have their
   own tests that build no command; `go test ./...` passes.
6. No script-facing contract changed without deprecation; generated docs regenerated.
7. Go: the audit script shows no new WARN lines; fix and rerun until it doesn't.

Sources: cli/cli @ fc4b137 (2026-09-29), docker/cli @ 7fc2dff (2026-09-23),
go-gh v2.16.1, clig.dev (items adopted from the former go-cli-development skill). Re-verify specific file paths against upstream before citing them.
