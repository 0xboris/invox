# invox

Go CLI for YAML- and LaTeX-driven invoices: `new`, `validate`, `render`, `build` (via
`tectonic`), `email`, `archive`. One dependency (`gopkg.in/yaml.v3`).

## Commands

```sh
go build ./...                 # or: make build  (-> ./bin/invox)
go test -race ./...            # or: make test
go vet ./...
gofmt -l .                     # must print nothing
go mod tidy -diff              # must print nothing
```

CI (`.github/workflows/ci.yml`) runs the tests on linux/macos/windows with Go 1.22 (the
minimum in `go.mod`) and stable, and gofmt/vet/tidy on Linux. Keep all of it green.

## Layout

- `cmd/invox/main.go`: shim, `os.Exit(cli.Run(os.Args[1:]))`.
- `internal/cli`: argument parsing (`command_specs.go`, `parsing.go`), commands
  (`commands_*.go`), hand-written help (`help.go`), editor/opener/mail launchers (`editor.go`).
- `internal/invoice`: domain logic (loading and validation, money and VAT, numbering,
  rendering, EPC QR, email drafts, archive). `service.go` is large and being split up.
- `internal/invoice/starter`: files embedded for `invox init`.

## Quality roadmap

Work is tracked in the roadmap tracking issue #9 and its phase epics (#10–#16) and sub-issues.
The standards come from the `quality-cli` skill (`.claude/skills/quality-cli`). The
assessment behind the roadmap is in `docs/quality/ASSESSMENT.md`.

Settled decisions (#9):
- usage errors exit 2
- the CLI moves to cobra one noun at a time
- stdout becomes data-only (status goes to stderr)
- `fonts/` is removed

## Conventions

- One issue per PR. Reference it with `Fixes #N` and keep the diff to what the issue asks.
- Every bug fix comes with a regression test that fails without the fix.
- Treat stdout/stderr content, exit codes, flag names and file formats as a contract. Change
  them only when the issue calls for it, and update the tests that pin them.
- Domain code (`internal/invoice`) must not print, read env vars, call `os.Exit`, or mention
  CLI flags in new code. Return typed errors; the CLI layer words them.
- Match the surrounding style: early returns, `fmt.Errorf("...: %w", err)`, table tests where
  several cases share a shape.

## Testing

- CLI tests: `captureRun(t, args)` returns `(exitCode, stdout, stderr)`; assert all three.
- Never depend on the developer's real config: point `XDG_CONFIG_HOME` (and for paths derived
  from home, `setPlatformHome(t, dir)` in `internal/invoice`) at `t.TempDir()`.
- `build` tests use `installFakeTectonic(t, fakeTectonicWritePDF|fakeTectonicFail)`, which
  puts the test binary on PATH as `tectonic`. No shell scripts, so tests run on Windows.
- Use `chdirForTest` for working-directory changes. Swapped package-level hooks
  (`openTextFile`, `openDocument`, `currentDate`, ...) must be restored with `t.Cleanup`.
- Tests must pass on Windows and macOS too. Build paths with `filepath.Join`, and remember that
  `os.UserHomeDir` reads `USERPROFILE` on Windows.
