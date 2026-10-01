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
4. **New Go CLI?** Clone the starter <https://github.com/0xboris/go-cli-starter>
   (compiles, tested, implements these rules; README explains renaming). If it can't
   be cloned, scaffold from `references/architecture.md`, reading "Cobra gotchas" first.
5. Code snippets in the references come from gh's own packages (`cmdutil`,
   `iostreams`, `httpmock`, `run`, ...). They are **not importable** (many are
   `internal/`). Copy the pattern, or use the starter's equivalents. The only
   supported reusable library from gh is `github.com/cli/go-gh/v2`.

Go names in parentheses are examples; `references/other-languages.md` maps them to
Python, Node and Rust.

## Behavior rules (always)

**Exit codes & errors**
- Commands return errors; one place maps them to exit codes and prints once, to
  stderr. Codes: `0` ok, `1` error, `2` cancelled, `4` auth required, `8` pending
  (optional); document any others in a `help exit-codes` topic.
- Print usage **only** for usage errors (`FlagError`), never for runtime failures.
  Turn off the framework's own error/usage printing (cobra `SilenceErrors`,
  `SilenceUsage`).
- Not failures: an empty result (exit 0, message on TTY only), the user quitting the
  pager (EPIPE → exit 0). Ctrl-C or a declined prompt → exit 2, quiet.
- Errors say what to do next: the missing flag, valid values, or the command to run.

**Streams**
- stdout = the data asked for (rows, JSON, the URL/path of what was created).
  stderr = progress, warnings, hints, update notices. Prompts render on the terminal
  and only run when `CanPrompt()` holds.
- TTY and non-TTY are two contracts. TTY: color, header, truncation, relative time,
  spinner. Non-TTY: no color, no header, tab-separated, no truncation, RFC3339 time,
  state as words.
- Honor `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE`; offer `TOOL_FORCE_TTY`. Color adds
  meaning, never carries it alone.
- Strip ANSI escapes from remote/untrusted text before printing it.

**Interactivity**
- Prompt only when stdin **and** stdout are TTYs and prompting isn't disabled
  (`TOOL_PROMPT_DISABLED`). Every prompt has a flag; without a TTY, fail with a
  `FlagError` naming those flags. Never hang waiting on stdin.
- Destructive actions: confirm on a TTY, require `--yes` otherwise; for high-impact
  deletes make the user type the target's name. `--yes` alone must not delete an
  implicit target (the current directory/repo).
- Pass the command's context down to network and subprocess calls so Ctrl-C cancels.

**Configuration**
- Precedence flag > env var > config file > default, documented in `help environment`.
- Writes are atomic (temp file + rename, keep mode). Secrets go to the OS keyring
  with timeouts; plaintext only by explicit opt-in.
- `--help`, `version` and completion work offline with a broken config.

## Structure rules (scale to the tool)

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
  in CI and non-TTY.

## Script-facing contract

Once shipped these are public API: flag names, arguments, defaults, exit codes, error
wording scripts match on, JSON fields, non-TTY output, stdout/stderr routing. Change
them only by deprecation (hidden alias + warning, kept ≥1 release).

## Task recipes

| Task | Do | Open |
|---|---|---|
| **New CLI** | Clone the starter and rename; otherwise scaffold: shim `main`, `app.Main`, IOStreams, Factory, typed errors, root with help topics, one noun with `list` + a mutating verb, tests from day one | `references/architecture.md`, `references/testing.md` |
| **Add a command** | New package `pkg/cmd/<noun>/<verb>`; Options + constructor + run; register it; `Short`, `Long`, `Example`; tests for parsing and for output in both TTY modes | `references/architecture.md`, `references/testing.md` |
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
| **Refactor a hand-rolled CLI** | Freeze output with tests, then IOStreams, typed errors, one command at a time | `references/architecture.md` §Migrating |
| **Non-Go CLI** | Same rules; map the concepts | `references/other-languages.md` |

## Audit script (Go only)

```sh
bash <skill-dir>/scripts/cli_audit.sh <repo-root> [--strict]
```

Grep-based heuristics for the rules above (raw stdio, `os.Exit` outside main,
scattered env reads, no TTY detection, no `--json`, untested packages, ...). WARN means
"look here", not "wrong"; gh itself gets several. Exit: 0, or 1 with `--strict` when
anything warned; 2 if no Go sources. To check only your change, run it before and
after and compare the WARN lines.

## Before you finish (loop until clean)

1. Non-TTY: `tool … | cat` and `tool … </dev/null`. Output has no color or header,
   and missing input gives a `FlagError` naming the flags, not a hang.
2. TTY: `script -qec 'tool …' /dev/null` (Linux) or `script -q /dev/null tool …`
   (macOS), or `TOOL_FORCE_TTY=1 tool …`.
3. Exit codes: `echo $?` after success, a usage error and a runtime error.
   Cancellation (2) is covered by a test returning the cancel error.
4. `tool <cmd> --help` has an Example and works with `TOOL_CONFIG_DIR=$(mktemp -d)`.
5. Tests cover parsing and output in both TTY modes; `go test ./...` passes.
6. No script-facing contract changed without deprecation; generated docs regenerated.
7. Go: the audit script shows no new WARN lines; fix and rerun until it doesn't.

Sources: cli/cli @ fc4b137 (2026-09-29), docker/cli @ 7fc2dff (2026-09-23),
go-gh v2.16.1. Re-verify specific file paths against upstream before citing them.
