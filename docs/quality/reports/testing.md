# invox test suite audit (against quality-cli)

## (a) Verdict
The suite is large (140 tests, 76% combined statement coverage) and careful in places: it uses `t.Helper`, `t.TempDir`, `t.Cleanup` and `t.Setenv`, it passes `-race`, and it passes with shuffled test order. It still breaks the skill's main testing rules. It fails on Linux today, it reads the developer's real config, it captures output by swapping `os.Stdout`, and it checks output with `strings.Contains` rather than exact strings. It also has no e2e, fuzz, TTY or adapter tests. Every external seam is a package-global variable or a process-global setting (env, cwd), so nothing can run in parallel. The fix is structural: IOStreams + Factory with injected clock, env and runner. A testscript layer can come first, as a safety net, because it needs no refactor.

## (b) Test run + coverage

| Run | Result |
|---|---|
| `go test ./... -count=1` | cli ok · **invoice FAIL** (TestEditableConfigPathCreatesCommentedTemplate) |
| `go test -race ./... -count=1` | same single failure, no races (but nothing runs in parallel) |
| `-shuffle=on -count=3` | no order dependence |
| `GOOS=windows/darwin go vet` | clean |
| poisoned `$XDG_CONFIG_HOME` / `$HOME` | **4 extra failures** (see F2) |
| real editor/osascript/open replaced by `panic` | never reached; real tectonic never needed (4 tests use a fake `/bin/sh` tectonic) |

| Package | own pkg | `-coverpkg=./...` |
|---|---|---|
| cmd/invox | 0% (no tests) | 0% |
| internal/cli | 76.1% | — |
| internal/invoice | 69.7% | — |
| **total** | 71.9% | **75.9%** |

Notable gaps (own package / cross-package): all of `editor.go` exec code 0% (`defaultOpenTextFile`, `shellEditorCommand`, `defaultOpenNativeEmailDraft`, `defaultCleanupOpenedDocument`, `defaultOpenDocumentCommand`). `archiveDataBaseDir` 29%. `writeBase64MIME` 56% (write errors). `ArchiveInvoice` 59/70%. `writeFileAtomic` 59%. `coerceDecimal` 40%. `formatQuantity` 43%. `IncrementInvoiceNumber`, `formatInvoiceNumber` and `isArchivedInvoicePath` 67%. `BuildPDF` 0/89%. `runEmail` 59%, `runHelp` 38%, `lookupCommand` 12%, the `*UsageError` helpers 0%. Dead code shows up as 0% with no callers: `invoiceEmailBody` (email.go:339), `prependPath`, `firstPresentPath`, `firstNonEmptyPath`. EPC/IBAN is well covered (IBAN 90%, payload 81%).

## (c) Findings

**F1 [blocker] Red suite on Linux, and on Windows too (from reading the code); no CI.** *Rule:* "`go test -race ./...` in CI, on Linux, macOS and Windows."
- Root cause: `defaultConfigTemplate` derives `archive.dir` from `DefaultArchiveDir()`. That calls `archiveDataBaseDir()`, which switches on `runtime.GOOS` (service.go:983-1009). The test hardcodes the darwin value `'~/Library/Application Support/invox/invoices'` (service_test.go:2388). It also pins only `HOME`, not `XDG_DATA_HOME`/`APPDATA`.
- Fix the test: pin `XDG_DATA_HOME=""` and `APPDATA`, then expect `"#   dir: '"+configTemplatePath(DefaultArchiveDir())+"'"`, or use the per-OS switch that service_test.go:2240 already has.
- Fix properly: make the function pure, `archiveDataBaseDir(goos string, getenv func(string) string, home string) string`, and table-test all three OS branches on every host.
- Other Windows failures, found by reading the code:
  - `TestResolveArchiveDirUsesConfigOverride`: on Windows `os.UserHomeDir` reads `USERPROFILE`, not `HOME`.
  - 4 build tests write a `#!/bin/sh` fake tectonic (cli_test.go:1758/1826/1889/1934). `LookPath` won't find it without a PATHEXT extension.
  - Prod hint "`brew install tectonic`" (service.go:936) is macOS-only.
- No `.github/workflows`. The Makefile `test` target runs no `-race`.

**F2 [major] Not hermetic.** *Rule:* "Never touch the real home dir… point config dir env to `t.TempDir()`."
- `captureRun` sandboxes only `if os.Getenv("XDG_CONFIG_HOME") == ""` (cli_test.go:2431). A developer who exports it runs tests against their real config.
- Many invoice tests never set it.
- Proven with a poisoned config: `TestNewAcceptsInlineLongFlagsAfterCustomerID`, `TestEmailUsesEditableNativeComposeByDefault`, `TestCreateNewInvoicePrefillsVATFromCustomerDefaultWhenMissing` and `TestCreateNewInvoicePreservesSourceVATWhenCustomerHasDefault` fail.
- Fix: a `TestMain` that sets `HOME`, `USERPROFILE`, `APPDATA`, `XDG_CONFIG_HOME` and `XDG_DATA_HOME` to temp dirs, or better, inject a config dir via the Factory.

**F3 [major] Output captured by swapping globals.** *Rule:* "Buffer-backed IOStreams… no direct stdout."
- `captureRun` (cli_test.go:2428-2467) replaces `os.Stdout`/`os.Stderr` with `os.Pipe` writers, calls `Run`, and restores them **without `defer`**. Three ways this goes wrong:
  - **Leaked state:** a stub that calls `t.Fatalf` inside `Run` (cli_test.go:1429), or any panic, leaves `os.Stdout` pointing at a closed pipe for the rest of the package.
  - **Deadlock:** the pipes are drained only after `Run` returns, so output larger than the pipe buffer blocks forever. Linux buffers 64 KiB. Windows' anonymous-pipe default is ~4 KiB, and `help template` emits 5.4 KB, so a Windows hang is likely (not verified).
  - **No parallelism:** process-global, so `t.Parallel()` is impossible (0 uses).
- Other global mutation:
  - `chdirForTest` does `os.Chdir` plus a manual `os.Setenv("PWD")`. It is used 16× and duplicated in both packages. `t.Chdir` needs `go 1.24` in go.mod, which currently says 1.22.
  - Hook variables are swapped by hand about 25 times: `openTextFile`, `openDocument`, `cleanupOpenedDocument`, `preferNativeMailCompose`, `openNativeEmailDraft` (editor.go:14-18) and `currentDate` (drafts.go:16, swapped 7×).
  - `t.Setenv` of `XDG_CONFIG_HOME` (22×), `PATH` (4), `HOME` (3), `VISUAL`/`EDITOR`, `APPDATA`, `XDG_DATA_HOME`. These are fine but forced, because domain code reads env, cwd and clock directly (service.go:248/350/983, templates.go:132).

**F4 [major] Weak assertions.** *Rule:* "exact stdout/stderr, TTY and non-TTY."
- There are 151 `strings.Contains` checks and exactly **one** exact stdout comparison (archive list, cli_test.go:1621).
- Help text has no golden files.
- Exit codes are checked (60×), and stderr is usually asserted empty. That part is good.
- Example of what this misses: `invox validate` prints `total 120,00 \euro`, raw LaTeX in terminal output, and no test notices.
- There are no TTY tests, because the product has no TTY detection.

**F5 [major] External programs untested.** *Rule:* "each external program behind one adapter… typed error… injected test seam; unstubbed commands panic."
- The editor goes through `$SHELL -lc 'eval "$INVOX_EDITOR"…'`. The osascript argv and the background `sleep; rm` cleanup are all 0% covered. Their quoting is exactly what breaks in practice.
- `BuildPDF` sends tectonic's stdout to `os.Stdout` (service.go:941). That pollutes the data stream, and the silent fake hides it.
- There is no exec recorder, so tests can't assert the argv that was run.

**F6 [major] No acceptance layer.** *Rule:* layer 4, testscript. Nothing runs the built binary; `main` is 0%. (Domain tests are correctly separate: `invoice` tests never import `cli`.)

**F7 [major] No fuzz tests.** A 30-second scratch fuzz found real bugs:
- `parseInvoiceCounter` **panics** in `regexp.MustCompile` on invalid UTF-8 in the pattern or customer id (numbering.go:339). It should use `regexp.Compile` and return the error.
- A leading space in the pattern breaks the format/parse round-trip: format trims (numbering.go:296), parse does not.
- `parseDecimal` (service.go:1728) accepts `1/3`, `0x10` and `1e30`. `quantizeMoney("1e30")` silently overflows to `-8814407033341083648` (`big.Int.Int64`, no bounds check).

**F8 [minor] Time.** `email.go:270/275` uses `time.Now()` for the MIME boundary and the `Date:` header, so the `.eml` can't be golden-tested. Tests check only the `--invox-boundary-` prefix (service_test.go:1479). The fixed-noon `currentDate` stub is safe, but CLI-level tests can't pin the clock.

**F9 [minor] Design.**
- No subtests in cli_test.go, and only 3 tables in total.
- Big copy-paste tests of 80-108 lines.
- Duplicated fixture helpers across packages (`writeContextFixtures` returns 6 positional strings; `writeConfigFile`, `quoteYAMLString`, `chdirForTest`).
- 54 inline YAML literals, no `testdata/`, no `-update` golden flow.
- Fix: one `testutil` package, a fixture struct, `testdata/fixtures/*.yaml`, and a `stub(t, &hook, fn)` helper.

## (d) Strategy

1. **IOStreams + Factory** (architecture.md). `cli.Main(args []string, f *Factory) int`, where the Factory provides:
   - `IO` (`In`/`Out`/`ErrOut` + `IsStdoutTTY`)
   - `Now func() time.Time`, `Getenv`, `Getwd`, `ConfigDir`
   - `Tectonic`, `Editor`, `Opener` and `Mailer` adapters
   - `main` stays `os.Exit(cli.Main(os.Args[1:], factory.New()))`.

   Tests then use `ios, _, out, errb := iostreams.Test()`, assert `out.String() == want` and `errb.String() == ""` in both TTY modes, and can call `t.Parallel()`. Domain functions take `now` and paths as parameters.
2. **Runner seam:** `type Runner interface{ Run(ctx context.Context, dir string, stdout, stderr io.Writer, name string, args ...string) error }`. The production version wraps `exec.CommandContext` and returns `*ExecError{Code int; Stderr string}`. The test `StubRunner` registers `"tectonic invoice.tex" → (0, writes pdf)`, **panics on unregistered commands**, and `t.Cleanup` fails on stubs that were never used. Tectonic's stdout goes to `ErrOut`.
3. **Fuzz targets** (`internal/invoice/fuzz_test.go`):
   - `FuzzInvoiceNumberRoundTrip(f, pattern, customerID, issueDate string, counter int64)`: format→parse must equal counter and never panic
   - `FuzzParseDecimal(f, s string)`: bounded, no overflow in `quantizeMoney`
   - `FuzzIsValidIBAN(f, s string)`: valid ⇒ mod-97 holds; never panics
   - `FuzzLatexEscape(f, s string)`: no unescaped `\&%$#_{}~^` remain
   - `FuzzValidateTemplatePlaceholders(f, tpl string)`
   - `FuzzRenderEmailTemplate(f, tpl string)`
   - `FuzzBuildEPCPayload(f, name, iban, bic, ref string, cents int64)`: ≤331 bytes, valid UTF-8
4. **testscript suite** `cmd/invox/script_test.go` + `cmd/invox/testdata/script/*.txtar`:
   - `invox` runs in-process via `testscript.RunMain`.
   - **Fake `tectonic`/`osascript`/editor are also registered as in-process commands**, so they work on Windows, unlike `#!/bin/sh` stubs.
   - `Setup` sandboxes `HOME`, `USERPROFILE`, `APPDATA` and the `XDG_*` dirs, and trims `PATH` to testscript's bin plus `$WORK/bin`.
   - A custom `exits CODE cmd…` command asserts exact exit codes.
   - Run it in normal CI on 3 OSes with `-race`.

   I built this in scratch and all 4 scripts pass, including a build+archive script with a fake tectonic and a forced failure.

`init_new_validate.txtar`
```
exec invox init
stdout '^Initialized .*[/\\]home[/\\]\.config[/\\]invox$'
stdout '^created customers\.yaml$'
! stderr .
exists home/.config/invox/template.tex
exec invox init
stdout '^exists customers\.yaml$'
exec invox new CUST-001
cmp stdout want-new.txt
! stderr .
grep '^  status: draft$' CUST-001-001.yaml
exec invox validate -i CUST-001-001.yaml
stdout '^Validation OK: CUST-001-001 for CUST-001, 1 line item\(s\), total 120,00'
! stderr .
# drafts do not consume numbers; only archived invoices advance the counter
exec invox new CUST-001 -o second.yaml
stdout '\(CUST-001-001\)$'
-- want-new.txt --
Created CUST-001-001.yaml for CUST-001 (CUST-001-001)
```

`usage_errors.txtar`
```
exits 2 invox
! stdout .
stderr '^error: missing subcommand$'
exits 2 invox frobnicate
stderr '^error: unknown subcommand "frobnicate"$'
exits 2 invox validate
! stdout .
stderr '^error: missing required flags: -i, --input$'
stderr '^Use ''invox validate --help'' for more information\.$'
exits 2 invox new --bogus
stderr '^error: flag provided but not defined: -bogus$'
# runtime failure: exit 1, no usage block
exec invox init
exits 1 invox validate -i missing.yaml
! stdout .
stderr 'missing.yaml'
! stderr 'Usage:'
exits 0 invox validate --help
stdout '^Usage:'
! stderr .
```

`archive_list_nontty.txtar`: the archive dir is set in config, relative to config.yaml, so the script works on any OS
```
# archive dir set via config (relative to config.yaml) so this is OS-independent
mkdir home/.config/invox/archive
cp config.yaml home/.config/invox/config.yaml
exec invox archive list
! stdout .
! stderr .
cp a.yaml home/.config/invox/archive/2026-03-06.yaml
cp b.md home/.config/invox/archive/2026-03-05.md
exec invox archive list
cmp stdout want.tsv
! stderr .
exec invox init
exec invox new CUST-001 -o next.yaml
stdout '\(CUST-001-008\)$'
-- config.yaml --
archive:
  dir: archive
-- a.yaml --
customer_id: CUST-001
invoice:
  number: CUST-001-007
  issue_date: 2026-03-06
  status: archived
-- b.md --
---
customer_id: CUST-MD
invoice:
  number: CUST-MD-001
  issue_date: 2026-03-05
---
# Archived invoice
-- want.tsv --
2026-03-05.md	CUST-MD	2026-03-05	archived
2026-03-06.yaml	CUST-001	2026-03-06	archived
```

The harness was built and run in a scratch copy (go-internal v1.14.1; needs `go 1.23` in go.mod).
