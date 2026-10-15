# The architecture in invox

This guide shows where each part of `docs/ARCHITECTURE.md` sits in invox. `architecture.json` maps each directory to a module kind, and `docs/architecture/modules.md` lists the result. The code has no accepted violations, so there is no baseline file: `python3 tools/archcheck.py` fails on any violation.

## Layout

```
cmd/invox/main.go                      composition: os.Exit(app.Main(os.Args[1:], app.Live()))
internal/app/                          composition root: Dependencies, the adapters, one wire_<slice>.go per slice
internal/features/<slice>/ui/          UI of a slice: one cobra command per package, NewCmdX(f, runF)
internal/features/<slice>/ui/shared/   the slice's Factory and what its commands share
internal/features/cli/ui/              the cli slice: cli.Main, the cobra root, global flags, exit codes, help rendering, completion
internal/sharedui/cmdutil/             shared UI: error types, FlagErrorFunc, prompts, the editor gate
internal/sharedui/tableprinter/        shared UI: table output
internal/domain/invoice/               domain: loading, validation, VAT totals, numbering, the render, email and archive steps, HostSource
internal/domain/ports/                 domain: the ports (Files, PDFCompiler, Editor, Opener, MailComposer)
internal/domain/config/                domain: the config.yaml schema
internal/domain/archive/               domain: the archive directory: walk, list, name resolution, .history backups
internal/domain/money/                 domain: decimals, cents, 1.234,56 formatting
internal/domain/epc/                   domain: EPC QR code rules and payload
internal/domain/email/                 domain: subject and body templates, the .eml MIME layout
internal/domain/latex/                 domain: LaTeX escaping, rows, line item blocks, template checks
internal/domain/schema/                domain: the documented fields of the support files, their examples, the template placeholders
internal/adapters/fsutil/              adapter: ports.Files on the real file system
internal/adapters/platform/            adapter: the process environment, streams, terminals and signals
internal/adapters/programs/run/        adapter: run.Runner, the only package that starts programs
internal/adapters/programs/<name>/     adapters: tectonic, editor, opener and applemail on top of run.Runner
internal/adapters/starter/             adapter: the files invox init writes, embedded
internal/adapters/buildinfo/           adapter: the version stamped in at link time
internal/utilities/env/                utilities: env.Env (GOOS, Getenv, HomeDir, Getwd, Now)
internal/utilities/iostreams/          utilities: stdin, stdout, stderr and TTY flags
internal/utilities/proc/               utilities: ExitError, SignalError and the Ctrl-C hold
internal/docs/gen/                     tool: regenerates docs/cli and share/man; the release binary may not import it
```

Slices are the command nouns: `config`, `customer`, `init`, `invoice`, `template` and `version`. `cli` is the application shell, not a capability: it owns the root command, help rendering, exit codes and completion, and receives the other slices' commands and help topics. Help wording lives in the slice whose files it describes: each slice's `Topics` returns its `cmdutil.Topic` pages, and the domain's `schema` package holds only the field and placeholder data they print. A slice's UI module is its whole `internal/features/<slice>/ui` tree. invox has no use case modules yet: commands call `internal/domain/invoice` directly, which holds the render, email and archive steps.

`app.Dependencies` holds the streams, the `env.Env`, the `invoice.HostSource` and the adapters behind their ports. Each `wire_<slice>.go` copies what its slice needs into that slice's `shared.Factory` and returns the slice's commands. `config.go` is the one composition file with decisions in it: Apple Mail exists only on macOS, and the user directories are resolved from the environment there, on first use, before the domain gets them as `invoice.HostInputs`.

## Where new code goes

`tools/new.py` generates use cases and HTTP handlers, which invox does not use. Do not run it here. Build by hand and copy the closest existing code:

- **A new command.** A package under `internal/features/<noun>/ui/<verb>` with an Options struct, `NewCmdX(f, runF)` and a run function, like `internal/features/invoice/ui/validate`. Register it in its noun's parent command, or in the slice's `wire_<noun>.go` for a top-level command. A new noun is a new slice, with its own `ui/shared.Factory` and `internal/app/wire_<noun>.go`, and an entry in `commandGroups` in `internal/features/cli/ui/root.go`.
- **A business rule or value.** A function or type in the domain module that owns the concept, with its name in `GLOSSARY.md`. A rule that only one command uses still goes into the domain.
- **File access.** Through `ports.Files`, which the `invoice.Host` carries (`h.Files()`). Domain code never imports `os`. `Files` mirrors `os` and `filepath` because it replaced direct calls one for one; prefer adding a purpose-named port (an archive store, the support files) over widening it.
- **A program invox runs.** A port in `internal/domain/ports`, and an adapter under `internal/adapters/programs/<name>` that implements it on top of `run.Runner`, like `internal/adapters/programs/tectonic`. `internal/app` builds it into `Dependencies` and copies it into the slices' Factories.
- **Output.** Commands write to the `IOStreams` they are given. The domain returns data and typed errors and never prints.
- **The environment, the clock or the platform.** Read them from the `env.Env` that the caller passes in. Only `platform.Env()` and `platform.Streams()` touch the process.

## Other checks of the same layers

`./check` runs `tools/archcheck.py`. Two older checks state parts of the same rules and stay in place: depguard and forbidigo in `.golangci.yml` check each file's imports and calls, and `internal/archtest` checks the transitive import graph. `internal/adapters/platform` holds the tests that only `platform` reads the process environment and streams. A change to the layering updates all of them in the same pull request.

## Tests

- `go test -race -shuffle=on ./...` runs in `./check`, so tests must pass in any order and without data races.
- CLI tests in `internal/app` use `captureRun` and `testFactory`, which return a `*app.Dependencies`. The e2e suite in `cmd/invox/testdata/script` pins every command's stdout, stderr and exit code. CLAUDE.md's Testing section has the details.
- `./check` runs gofmt, `go mod tidy -diff`, vet, build and the tests. CI also runs golangci-lint, govulncheck, fuzzing and the tests on macOS and Windows.
