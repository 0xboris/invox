# Stack

This file names the libraries invox uses, shows where each part of `docs/ARCHITECTURE.md` sits in invox, and how the reference slice uses them. `docs/architecture/go.md` has the rules that apply to any library. This file belongs to the project, so the change that adds or replaces a library also updates this file. Like everything in `docs/architecture/`, it is a gate file, and the owner reviews changes to it.

## Libraries

- **Commands.** `github.com/spf13/cobra` and its `github.com/spf13/pflag`, the `@cli-framework` set in `architecture.json`. Only UI, shared UI, and the composition root may import them.
- **YAML.** `gopkg.in/yaml.v3`, in the domain. It does no I/O: the domain reads the bytes through `ports.Files` and parses them.
- **PDFs.** The `tectonic` program, behind the `ports.PDFCompiler` port, in `internal/adapters/programs/tectonic`.
- **Tests.** `testing`, and `github.com/rogpeppe/go-internal/testscript` for the end-to-end scripts in `cmd/invox/testdata/script`.
- **Lint.** golangci-lint at the version in the `Makefile`, with the rules in `.golangci.yml`, and govulncheck in CI.

## Layout

```
cmd/invox/main.go                      composition: os.Exit(app.Main(os.Args[1:], app.Live()))
internal/app/                          composition root: Dependencies, the adapters, one wire_<slice>.go per slice
internal/features/<slice>/ui/          UI of a slice: one cobra command per package, NewCmdX(f, runF)
internal/features/<slice>/ui/shared/   the slice's Factory and what its commands share
internal/features/<slice>/usecases/    use cases of a slice: the only way from its commands to the ports
internal/features/cli/ui/              the cli slice: cli.Main, the cobra root, global flags, exit codes, help rendering, completion
internal/sharedui/cmdutil/             shared UI: error types, FlagErrorFunc, prompts, the editor gate
internal/sharedui/tableprinter/        shared UI: table output
internal/domain/invoice/               domain: loading, validation, VAT totals, numbering, the render, email and archive steps, HostSource
internal/domain/ports/                 domain: the ports (Files, PDFCompiler, Editor, Opener, MailComposer)
internal/domain/config/                domain: the config.yaml schema
internal/domain/archive/               domain: the archive directory: walk, list, name resolution, .history backups
internal/domain/money/                 domain: decimals, cents, 1.234,56 formatting
internal/domain/epc/                   domain: EPC QR code rules and payload
internal/domain/email/                 domain: the subject and body templates of the invoice email
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
internal/utilities/eml/                utilities: an unsent .eml message with one PDF attachment
internal/testsupport/                  test support: port implementations that fail
internal/docs/gen/                     tool: regenerates docs/cli and share/man; the release binary may not import it
```

Slices are the command nouns: `config`, `customer`, `init`, `invoice`, `template` and `version`. `cli` is the application shell, not a capability: it owns the root command, help rendering, exit codes and completion, and receives the other slices' commands and help topics. Help wording lives in the slice whose files it describes: each slice's `Topics` returns its `cmdutil.Topic` pages, and the domain's `schema` package holds only the field and placeholder data they print.

The use cases reach the files through the `invoice.Host` that `invoice.HostSource` resolves, which carries `ports.Files`. Commands still call `invoice.Host` methods that read the support files, such as `Host.CreateNewInvoice`, and those methods hold the render, email and archive steps. Moving those steps from the domain into use cases is open work.

A command package isn't named `build`, because the architecture check skips directories with that name. The `build` command lives in `internal/features/invoice/ui/buildcmd`.

`app.Dependencies` holds the streams, the `env.Env`, the `invoice.HostSource` and the adapters behind their ports. Each `wire_<slice>.go` builds that slice's use cases from them, puts them into the slice's `shared.Factory`, and returns the slice's commands. `config.go` is the one composition file with decisions in it: Apple Mail exists only on macOS, and the user directories are resolved from the environment there, on first use, before the domain gets them as `invoice.HostInputs`.

## How the reference slice uses them

- `customer list` parses its flags into `ListOptions`, resolves `customers.yaml` with `shared.CustomersPath`, and calls `usecases.ListCustomers`, which reads the file through `ports.Files`. It prints a `tableprinter.Table`, or the `--json` fields through `cmdutil.Exporter`.
- `customer edit` resolves the same file, and hands `usecases.EditCustomers.Run` to `cmdutil.OpenInEditor`, which refuses to start the editor without a terminal.
- `internal/app/wire_customer.go` builds both use cases from `Dependencies` and puts them into the slice's `Factory`.

## Tests

- Command tests in `internal/app` run `app.Main` through `captureRun` and `testFactory`, which return a `*app.Dependencies` with a `run.Stub` in place of the programs. `CLAUDE.md`'s Testing section has the details.
- Use case and command tests use the real file system in `t.TempDir()`, through `fsutil.Files`, and read back what was written. `run.Stub` stands in for the programs, and `testsupport` holds port implementations that fail, such as `FailingOpener`.
- The e2e suite in `cmd/invox/testdata/script` pins every command's stdout, stderr and exit code.
- `go test -race -shuffle=on ./...` runs in `./check`, so tests must pass in any order and without data races.

## Other checks of the same layers

`./check` runs `tools/archcheck.py`. Two older checks state parts of the same rules and stay in place: depguard and forbidigo in `.golangci.yml` check each file's imports and calls, and `internal/archtest` checks the transitive import graph. `internal/adapters/platform` holds the tests that only `platform` reads the process environment and streams. A change to the layering updates all of them in the same pull request.

## Commands

`./check` runs the architecture check, the module map, `gofmt`, `go mod tidy -diff`, `go vet`, `go build`, and `go test -race -shuffle=on`. CI also runs golangci-lint, govulncheck, fuzzing, and the tests on macOS and Windows.

`.claude/skills/verify-invox/scripts/app` builds invox, runs it against a temporary config directory, and keeps each command and its output as evidence. Its `SKILL.md` has the steps.

## Changing a library

1. Allow the new package for the kinds that use it in `architecture.json`, under `external.allow`. Frameworks belong in UI, adapters, and the composition root, never in the domain or use cases. A pure package that does no I/O, such as a decimal package, may go in the domain. Other packages then use only the domain's types and parse functions, and the domain maps the package's errors to its own errors.
2. Rewrite the reference slice and its tests with the new library, so that agents copy the new patterns. Keep what the slice does unchanged.
3. Update the commands in `./check` and the CI workflows.
4. Add any new compiler, linter, or test runner settings file to `.github/CODEOWNERS` if it does not match a listed pattern yet.
5. Update this file.
