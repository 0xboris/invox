# Applying the rules outside Go/cobra

The behavior rules in SKILL.md are language-neutral. Use the default listed for each
building block. If the project already uses something else, keep it.

| Concept | Go (gh) | Python | Node/TypeScript | Rust |
|---|---|---|---|---|
| Command framework | cobra | typer | commander | clap (derive) |
| Main returns code | `os.Exit(int(app.Main()))` | `sys.exit(main())` | `process.exitCode = await main()` (avoid `process.exit()`, which can truncate piped output) | `fn main() -> ExitCode` |
| IOStreams | `pkg/iostreams` | object holding `stdin/stdout/stderr` + `isatty()` per stream | object wrapping `process.stdout` + `isTTY` per stream | struct with `Box<dyn Write>` + `std::io::IsTerminal` |
| Factory / DI | struct of lazy funcs | dataclass of callables (`functools.cache` for laziness) | context object of lazy getters | struct of `OnceCell`s/closures |
| Options + run | `Options`, `NewCmdX(f, runF)`, `xRun` | command builds an `Options` dataclass, calls `run(opts, io)` | action builds `Options`, calls `runX(opts, io)` | clap `Args` struct → `fn run(opts, io)` |
| Domain layer | `api/`-style package: structs, `ctx` first, typed errors; no cobra/iostreams import | module of dataclasses/pydantic models and functions; never imports click/typer or `sys.stdout` | module with TS types and functions taking an `AbortSignal`; never imports commander or `process.stdout` | library crate or module: structs, `thiserror` errors; no `clap` or `println!` |
| External-program adapter | `git.Client` shape: injected streams, `exec.CommandContext`, typed error with stderr | class wrapping `subprocess.run` with an injectable runner | class wrapping `execa`/`spawn` with an injectable spawner | struct building `std::process::Command` behind a trait |
| Import rules | depguard + forbidigo | `import-linter` contracts | `eslint-plugin-boundaries` or `dependency-cruiser` | crate boundaries (domain crate has no `clap` dependency) |
| Typed errors | `FlagError`, `CancelError`, `SilentError` | exception classes mapped in `main()` | error classes mapped in `main()` | `enum CliError` (`thiserror`) mapped in `main` |
| TTY color | go-gh `term` + ColorScheme | `rich` (honors `NO_COLOR`) | `picocolors` | `anstream` + `anstyle` |
| Tables | go-gh `tableprinter` | `rich.table` on TTY, `\t`-joined lines otherwise | `cli-table3` on TTY, `\t`-joined lines otherwise | `comfy-table` on TTY, `\t`-joined lines otherwise |
| `--jq` filter | go-gh `jq` (gojq) | `jq` (PyPI binding) | `jq-wasm`, or skip `--jq` and document piping to `jq` | `jaq-core` |
| Prompts | Prompter interface | `questionary` behind an interface | `@inquirer/prompts` behind an interface | `inquire` behind a trait |
| HTTP stubs | `httpmock.Registry` + `Verify` | `responses` with `assert_all_requests_are_fired=True` | `nock` + `nock.isDone()` | `wiremock` with `.expect(n)` |
| Exec stubs | `run.Stub()` | inject a runner object | inject a runner object | inject a runner trait |
| Snapshot tests | gotest.tools golden | `syrupy` | vitest snapshots | `insta` |
| E2E scripts | testscript `.txtar` | `pytest` + `subprocess` in a temp `HOME` | vitest + `execa` in a temp `HOME` | `trycmd` |
| Docs generation | cobra `doc` → man/markdown | `typer` docs + `click-man` | commander help → hand-maintained docs | `clap_mangen` |
| Completion | cobra completion | typer `--install-completion` | `omelette`, or ship static scripts | `clap_complete` |
| Release | GoReleaser | `uv tool install` / `pipx` from PyPI | npm package (`bin` field) | `cargo-dist` |

Language-specific pitfalls:
- **Python**: import time is startup time, so import heavy modules inside commands. Map
  exceptions in `main()`; show tracebacks only with `TOOL_DEBUG`.
- **Node**: handle `EPIPE` on stdout (`head` closed the pipe) as success.
- **Rust**: `println!` panics on a broken pipe. Write through a locked handle and treat
  `BrokenPipe` as exit 0.
- **All**: check TTY per stream (stdin, stdout and stderr separately), never "the terminal".
