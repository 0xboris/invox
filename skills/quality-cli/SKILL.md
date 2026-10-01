---
name: quality-cli
description: >
  Architecture, UX and engineering standards for production-quality command-line
  tools, distilled from the GitHub CLI (gh) and Docker CLI source code. Use this
  skill whenever you create, extend, refactor, review or test a CLI — adding a
  command, subcommand, flag or argument; writing help text, error messages or exit
  codes; adding prompts, confirmations, colors, spinners, pagers, tables or
  --json/--format output; handling config files, env vars, credentials, shell
  completion, man pages, plugins/extensions, versioning or releases. Trigger even
  when the user never says "CLI" but the code is a main package with subcommands or
  argument parsing (cobra, urfave/cli, flag, clap, click, typer, argparse,
  commander, oclif) or the change touches terminal output. Go/cobra-first; the
  principles apply to any language.
---

# Quality CLI

Build CLIs that behave like `gh` and `docker`: predictable for scripts, pleasant
for humans, safe for agents, and easy to test. Everything here was extracted from
their source code. **When gh and docker disagree, follow gh**
(see `references/gh-vs-docker.md`).

## How to use this skill

1. **Always** apply "The contract" below to any CLI change — it is short on purpose.
2. Find your task in "Task recipes" — each recipe names the one or two reference
   files worth opening. Do not read all references up front.
3. Starting a new Go CLI? Copy `assets/go-skeleton/` (compiles, tested, implements the
   whole contract) instead of writing scaffolding. Adding to an existing CLI? Match its
   conventions first, and use the skeleton's files as the model for anything missing.
4. Reviewing or auditing? Run `scripts/cli_audit.sh <repo>` and walk
   `references/review-checklist.md`.

## The contract

These hold for every command. Each line says why, so you can judge edge cases.

**Process & errors**
- `main` is a shim: `os.Exit(app.Main())`. `Main()` returns an exit code and never calls `os.Exit`, so deferred cleanup runs and tests can call it in-process.
- Commands **return errors**; they never print-and-exit or choose exit codes. One function maps typed errors to codes: `0` ok, `1` error, `2` cancelled, `4` auth required (gh's set; extend only with documented codes).
- Usage is printed **only** for usage errors (`FlagError`), never for runtime failures. Silence the framework's own error/usage printing (`SilenceErrors`, `SilenceUsage`) and print once, to stderr.
- Not every "error" is a failure: empty list → exit 0 (message on TTY only); user quit the pager (EPIPE) → exit 0; Ctrl-C → cancelled, quiet.
- Error messages state the problem, then the fix: name the missing flag, the command to run (`try: mycli auth login`), or the valid values.

**Dependencies**
- No command touches `os.Stdout`, `os.Stdin`, `os.Getenv`, the clock, `exec.Command` or a global HTTP client directly — all arrive via an injected Factory/IOStreams so tests can replace them.
- Expensive dependencies (config, HTTP client, git/repo resolution, network ping) are **lazy** `func() (T, error)`. `--help`, `version` and completion must work with broken config and no network.

**Command shape** (one package per leaf command)
- `Options` struct + `NewCmdX(f, runF)` + `xRun(opts)`. Flags bind to `Options`; validation happens in `RunE` **before** `runF`; `xRun` has no framework imports.
- Pass `cmd.Context()` into Options and down to every network/subprocess call so Ctrl-C cancels work.
- Grammar: `tool <noun> <verb> [args] [flags]`. Modifiers are flags, not new commands. Sibling commands share flag names and behavior (`list` always has `--limit`, `--json`, `--web` if applicable).

**Streams & output**
- stdout = the data the user asked for (rows, JSON, a created URL/path). stderr = everything else (progress, warnings, hints, prompts, update notices).
- TTY and non-TTY are two contracts. TTY: color, headers, truncation, relative time, spinners. Pipe: no color, no header, tab-separated, no truncation, RFC3339 time, explicit state words instead of color.
- Respect `NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE`, and offer `<TOOL>_FORCE_TTY`. Color enhances meaning, never carries it alone.
- Every list/view command offers `--json <fields>` (validated, lists fields when empty) plus `--jq`/`--template`. Requested fields should shape what you fetch.
- Sanitize ANSI escapes in remote/untrusted text before printing it.

**Interactivity**
- Prompt only if `CanPrompt()` = stdin **and** stdout are TTYs and prompts aren't disabled. Every prompt has a flag equivalent; without a TTY, fail with a `FlagError` naming those flags.
- Destructive actions require confirmation on a TTY and `--yes` otherwise; for high-impact deletes make the user type the target name. Never let an implicit target (current dir/repo) be deleted by `--yes` alone.
- Prompts/waits honor context cancellation; a declined prompt is a quiet cancel.

**Configuration**
- Precedence: flag > env var > config file > default — and write it down (`help environment`).
- Config dir: `<TOOL>_CONFIG_DIR` > `$XDG_CONFIG_HOME/<tool>` > OS default. Separate config, state and data. Writes are atomic (temp file + rename, preserve mode). Secrets go to the OS keyring (with timeouts), plaintext only by explicit opt-in.

**Docs & evolution**
- Help is data: `Short`, `Long`, `Example` (`$ tool …` lines), annotations for arguments/env/JSON fields. Generate man pages + markdown from the command tree; CI fails if they drift.
- Once shipped, these are public API: flag names, args, defaults, exit codes, error wording scripts grep for, JSON fields, non-TTY output, stdout/stderr routing. Change them only via deprecation (`MarkDeprecated`, hidden aliases, ≥1 release).

**Background work**
- Update checks, telemetry and extension checks never block: run async, cancel when the command ends, cache 24h, disable in CI, non-TTY, and via env var.

## Architecture at a glance (Go)

```
cmd/<tool>/main.go          func main() { os.Exit(int(app.Main())) }
internal/app/               Main(): build Factory, root cmd, signals, error→exit-code
internal/build/             Version/Date via -ldflags, ReadBuildInfo fallback
pkg/iostreams/              In/Out/ErrOut, TTY detection+overrides, color, pager, spinner
pkg/cmdutil/                Factory, typed errors, flag helpers, JSON exporter
pkg/cmd/root/               root command, groups, help topics, flag-error func
pkg/cmd/<noun>/<verb>/      one package per leaf command (+ _test.go beside it)
pkg/cmd/<noun>/shared/      helpers shared by a noun's verbs
internal/prompter/          Prompter interface (+ generated mock)
internal/run/               exec seam (PrepareCmd) for subprocesses
```

```go
type ListOptions struct {
    IO         *iostreams.IOStreams
    HttpClient func() (*http.Client, error) // lazy
    Exporter   cmdutil.Exporter              // nil unless --json
    Limit      int
    Now        func() time.Time              // injectable clock
}

func NewCmdList(f *cmdutil.Factory, runF func(*ListOptions) error) *cobra.Command {
    opts := &ListOptions{IO: f.IOStreams, HttpClient: f.HttpClient, Now: time.Now}
    cmd := &cobra.Command{
        Use: "list", Aliases: []string{"ls"}, Args: cobra.NoArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            if opts.Limit < 1 {
                return cmdutil.FlagErrorf("invalid value for --limit: %v", opts.Limit)
            }
            if runF != nil { return runF(opts) } // test seam: stop after parsing
            return listRun(opts)
        },
    }
    cmd.Flags().IntVarP(&opts.Limit, "limit", "L", 30, "Maximum number of items to fetch")
    cmdutil.AddJSONFlags(cmd, &opts.Exporter, itemFields)
    return cmd
}
```

## Task recipes

| Task | Do | Open |
|---|---|---|
| **New CLI** | Copy `assets/go-skeleton/`, rename module/binary, delete the sample `item` noun when you add real ones | `assets/go-skeleton/README.md`, `references/architecture.md` |
| **Add a command** | New package `pkg/cmd/<noun>/<verb>`; Options + `NewCmdX(f, runF)` + `xRun`; register in the noun; add `Short`, `Long`, `Example`; tests for flags (via `runF`) and run (TTY + non-TTY). Model: skeleton's `pkg/cmd/item/list` (read) and `item/delete` (mutating) with their tests | `references/architecture.md`, `references/testing.md` |
| **Add/change a flag** | Bind to Options; validate in `RunE`; use enum/tri-state/mutually-exclusive helpers; register completion; if renaming, keep old name hidden+deprecated | `references/ux-and-help.md` §Flags |
| **Output (tables, JSON, colors)** | Route through IOStreams + table printer + Exporter; check both TTY modes | `references/io-and-output.md` |
| **Prompts / editor / destructive ops** | `CanPrompt()` gate, flag fallback, `--yes`, Prompter interface | `references/ux-and-help.md` §Interactivity |
| **Errors & exit codes** | Typed errors, single mapping, actionable text | `references/architecture.md` §Errors, `references/ux-and-help.md` §Errors |
| **Help text, docs, completion** | Help sections, topics, annotations, generated docs | `references/ux-and-help.md` §Help, `references/build-and-release.md` §Docs |
| **Config, env vars, auth** | Precedence, dirs, atomic writes, keyring | `references/config-and-auth.md` |
| **Plugins, aliases, API escape hatch** | `tool-<name>` executables, no shadowing core | `references/extensibility.md` |
| **Signals, update checks, telemetry** | Context cancellation, async never-blocking work | `references/architecture.md` §Signals, `references/build-and-release.md` |
| **Tests** | `iostreams.Test()`, self-verifying HTTP/exec/prompt stubs, golden or exact output, testscript | `references/testing.md` |
| **Build, release, lint, CI** | ldflags version, GoReleaser, linters, docs drift check | `references/build-and-release.md` |
| **Review a CLI diff** | Run the audit script, walk the checklist | `references/review-checklist.md`, `scripts/cli_audit.sh` |
| **Refactor a hand-rolled CLI** | Migrate incrementally: IOStreams first, then typed errors, then one command at a time onto cobra | `references/architecture.md` §Migrating |
| **Non-Go CLI** | Same contract; map the concepts | `references/other-languages.md` |

## Before you finish any CLI change

- [ ] Ran the command with stdout piped (`| cat`) and on a TTY; both outputs make sense.
- [ ] Ran it with no TTY and missing required input → clear `FlagError`, no prompt hang.
- [ ] `--help` reads well, has an Example, and works with no config/network.
- [ ] Exit code is correct for success, failure, cancel (`echo $?`).
- [ ] Tests cover flag parsing (`runF`) and run output in TTY and non-TTY modes.
- [ ] No script-facing contract changed silently (flags, JSON fields, non-TTY output, exit codes).
- [ ] Generated docs/completions regenerated if commands or flags changed.
- [ ] Go: `scripts/cli_audit.sh .` shows no new warnings for the code you touched.
