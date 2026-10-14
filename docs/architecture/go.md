# The architecture in invox

This guide shows where each part of `docs/ARCHITECTURE.md` sits in invox today. invox adopted the architecture after it was built, so its packages do not follow the kit's template layout yet. `architecture.json` maps each package to a kind, and `architecture-baseline.json` accepts the violations the code had on adoption. New code follows the rules; old code is fixed as it is touched.

## Layout

```
cmd/invox/main.go                composition root: builds the Factory and runs cli.Main
internal/cli/                    composition root: the cobra root, global flags, exit codes, signals, help rendering
internal/cmd/<noun>/<verb>/      UI of the <noun> slice: one cobra command per package
internal/cmd/invoice/shared/     UI shared by the invoice slice's commands
internal/cli/cmdutil/            shared UI: the Factory, error types, prompts, completion helpers
internal/cli/helptext/           shared UI: help topics and reference tables
internal/tableprinter/           shared UI: table output
internal/invoice/                domain: loading, validation, VAT totals, numbering, render, email and archive steps
internal/money/                  domain: decimals, cents, 1.234,56 formatting
internal/epc/                    domain: EPC QR code rules and payload
internal/email/                  domain: subject and body templates, the .eml MIME layout
internal/render/latex/           domain: LaTeX escaping, rows, line item blocks, template checks
internal/adapters/<name>/        adapters: run (the only package that starts programs), tectonic, editor, opener, applemail
internal/archive/                adapter: the archive directory
internal/config/                 adapter: config file loading
internal/fsutil/                 adapter: file system helpers
internal/env/                    utilities: env.Env, the clock, GOOS, home and working directory
internal/iostreams/              utilities: stdin, stdout, stderr and TTY detection
internal/build/                  utilities: version information
internal/docs/gen/               tool: regenerates docs/cli and share/man; the release binary may not import it
```

Slices are the command nouns: `config`, `customer`, `init`, `invoice`, `template` and `version`. A slice's UI module is its whole `internal/cmd/<noun>` tree. invox has no use case modules yet. Commands call `internal/invoice` directly.

## Where new code goes today

`tools/new.py` generates the kit's template layout (`internal/features/<slice>`, `internal/app`), which invox does not have. Do not run its `slice`, `usecase` or `port` commands here. Build by hand and copy the closest existing code:

- **A new command.** A package under `internal/cmd/<noun>/<verb>` with an Options struct, `NewCmdX(f, runF)` and a run function, like `internal/cmd/invoice/validate`. Register it in its noun's parent command. A new noun is a new slice.
- **A business rule or value.** A function or type in the domain module that owns the concept, with its name in `GLOSSARY.md`. A rule that only one command uses still goes into the domain.
- **A program invox runs.** An adapter under `internal/adapters/<name>` that wraps one program on top of `run.Runner`, like `internal/adapters/tectonic`. The command gets it from the `cmdutil.Factory`.
- **Output.** Commands write to the `IOStreams` they are given. The domain returns data and typed errors and never prints.
- **The environment, the clock or the platform.** Read them from the `env.Env` that the caller passes in. Only `env.System()` and `iostreams.System()` touch the process.

## Moving toward the template layout

The baseline lists what does not fit yet. The biggest groups:

- `internal/invoice` performs file I/O and parses YAML, and it imports the `archive`, `config` and `fsutil` adapters. The kit wants that I/O behind ports with adapters implementing them.
- Commands and `cmdutil` import adapters directly. The kit wants a composition root that creates adapters and passes them in.
- `internal/cli` holds the composition root plus help rendering and signal handling, which the kit keeps out of the composition root.
- `iostreams` and `env` read the process, which utilities may not.

When a change removes violations, run `python3 tools/archcheck.py --write-baseline` in the same pull request so the baseline goes down. The baseline never goes up.

## Tests

- `go test -race -shuffle=on ./...` runs in `./check`, so tests must pass in any order and without data races.
- CLI tests use `captureRun` and the e2e suite in `cmd/invox/testdata/script` pins every command's stdout, stderr and exit code. CLAUDE.md's Testing section has the details.
- `./check` runs gofmt, `go mod tidy -diff`, vet, build and the tests. CI also runs golangci-lint, govulncheck, fuzzing and the tests on macOS and Windows.
