# Applying the contract outside Go/cobra

The contract in SKILL.md is language-neutral. Map the building blocks:

| Concept | Go (gh) | Python | Node/TypeScript | Rust |
|---|---|---|---|---|
| Command framework | cobra | typer / click | oclif / commander | clap (derive) |
| Main returns code | `os.Exit(int(app.Main()))` | `sys.exit(main())` | `process.exitCode = await main()` (avoid `process.exit()` so streams flush) | `fn main() -> ExitCode` |
| IOStreams | `pkg/iostreams` | object holding `stdin/stdout/stderr` + `isatty()` | object wrapping `process.stdout` + `isTTY` | struct with `Box<dyn Write>` + `std::io::IsTerminal` |
| Factory / DI | struct of lazy funcs | dataclass of callables / `functools.cached_property` | context object of lazy getters | struct of `OnceCell`/closures |
| Options + run | `Options`, `NewCmdX(f, runF)`, `xRun` | click command builds `Options` dataclass, calls `run(opts)` | command class `run()` → `runX(opts)` | clap `Args` struct → `fn run(opts, io)` |
| Typed errors | `FlagError`, `CancelError`, `SilentError` | exception classes mapped in `main()` | error classes mapped in `main()` | enum `CliError` + `thiserror`, mapped in `main` |
| TTY color | go-gh `term`, ColorScheme | `rich` (honors `NO_COLOR`) | `chalk`/`picocolors` with `supports-color` | `anstream`/`owo-colors` |
| Tables | go-gh `tableprinter` | `rich.table` on TTY, `csv` writer with `\t` when piped | `cli-table3` on TTY, TSV when piped | `comfy-table` on TTY, TSV when piped |
| JSON filter | go-gh `jq` | `jq` binding or JMESPath | `node-jq`/JSONata | `jaq` |
| Prompts | Prompter interface | `questionary` behind an interface | `@inquirer/prompts` behind an interface | `dialoguer`/`inquire` behind a trait |
| HTTP stubs | `httpmock.Registry` + `Verify` | `responses` (assert_all_requests_are_fired=True) | `nock` (`nock.isDone()`) | `wiremock`/`mockito` with `expect(n)` |
| Exec stubs | `run.Stub()` | inject a runner; `pytest-subprocess` | inject a runner | inject a trait |
| Golden files | gotest.tools golden | `syrupy`/`pytest-snapshot` | vitest/jest snapshots | `insta` |
| E2E scripts | testscript `.txtar` | `pytest` + `subprocess` / `scripttest` | `execa` tests | `trycmd`/`snapbox` (same idea as testscript) |
| Docs gen | cobra doc → man/markdown | `click-man`, `typer` docs | oclif readme generation | `clap_mangen`, `clap_complete` |
| Completion | cobra completion | `click`/`typer` completion | oclif autocomplete | `clap_complete` |
| Release | GoReleaser | build wheels + `pipx`/`uv tool` install, PyInstaller if single binary | `pkg`/`bun build --compile`, npm | `cargo-dist` |

Language-specific pitfalls:
- **Python**: import time is startup time — lazy-import heavy modules inside commands. Don't print tracebacks to users; map exceptions in `main()` and show traceback only with `TOOL_DEBUG`.
- **Node**: `process.exit()` can truncate piped stdout; set `process.exitCode`. Handle `EPIPE` on stdout (`head` closing the pipe) as success.
- **Rust**: `println!` panics on broken pipe — write through a handle and treat `BrokenPipe` as exit 0.
- **All**: detect TTY on the specific stream (stdout vs stderr vs stdin), not "the terminal".
