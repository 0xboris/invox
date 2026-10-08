# invox quality assessment (against `quality-cli`)

> **Historical record.** This assessment predates the quality roadmap
> ([#9](https://github.com/0xboris/invox/issues/9)). The roadmap's phase epics addressed its findings:
> [#10](https://github.com/0xboris/invox/issues/10), [#11](https://github.com/0xboris/invox/issues/11),
> [#12](https://github.com/0xboris/invox/issues/12), [#13](https://github.com/0xboris/invox/issues/13),
> [#14](https://github.com/0xboris/invox/issues/14), [#15](https://github.com/0xboris/invox/issues/15) and
> [#16](https://github.com/0xboris/invox/issues/16). The report below describes `c345065`, not the current code.

Assessed: 2026-10-02 on `c345065` (main). Method: the `quality-cli` audit script, then six independent
reviews (architecture and layering; errors, help and flags; streams, interactivity and config; testing;
build, release and docs; domain correctness). Each finding below was either reproduced by a reviewer
running the built binary or a proof test, or re-checked by reading the code. The full reviewer reports,
with file:line evidence and ready-to-paste snippets, are in [`reports/`](reports/).

## Verdict

The **domain core is good**:
- Money math uses `big.Rat` with half-up rounding and per-rate VAT.
- LaTeX escaping covers all ten special characters.
- The EPC QR payload follows the spec; a reviewer decoded it from a compiled PDF to confirm.
- osascript receives values as arguments, so nothing can be injected.
- Writes use temp file + rename. `new` and `archive` refuse to overwrite, and YAML comments survive.

The **CLI shell around it falls short of `quality-cli`**:
- Nothing is injected: commands and the domain reach for `os.Stdout`, env, clock and `exec` directly.
- Every command prints prose to stdout. Child processes write into stdout too.
- There is no TTY contract and no `--json`.
- Exit codes are ~45 hand-returned ints with no single mapping.
- Help is 850 hand-written lines that have drifted from the flags.
- There is no context or signal handling.

The **repo is not release-ready**:
- The test suite fails on Linux and there is no CI.
- There is no version command, no LICENSE and no lint config.
- The module path cannot be `go install`ed.

**Several real bugs need fixing before anything else.** Two delete or corrupt user data. Others change
billed amounts, or allow duplicate invoice numbers, which is a legal problem in most EU jurisdictions.

The domain logic is worth keeping. The answer is a **staged extraction, not a rewrite**: freeze behaviour
with tests, then add IOStreams, typed errors, adapters, typed config and models, and finally cobra.

## Scorecard

| Area (`quality-cli` section) | Status | Headline |
|---|---|---|
| Exit codes & errors | ❌ | Scattered `return 1/2`. Usage exit code changes when config is broken. Exit 2 is undocumented. Noisy 80-line usage dumps. |
| Streams | ❌ | Status sentences, tectonic, editor and `$SHELL -l` profile output all go to stdout. |
| TTY / non-TTY contract | ❌ | No detection. Lists print identical bytes either way. Tabs and newlines in fields break records. ANSI escapes from YAML pass through. |
| `--json` | ❌ | None, on any list or result. |
| Interactivity & safety | ❌ | Editor starts without a TTY check and can hang. No `--yes`, `--force` or `--dry-run`. `email` deletes user files. |
| Configuration | ⚠️ | Flag > upward search > config > default works, but config is an untyped map re-read 8–12× per run. A broken config bricks `init`. No `INVOX_CONFIG_DIR`/`--config`. No `help environment`. |
| Layering (commands thin, domain pure) | ⚠️ | Domain doesn't import the CLI (good), but holds the CLI flag bag (`invoice.Options`), prints LaTeX `\euro` to users, names flags in errors, and runs tectonic onto `os.Stdout`. `service.go` is 2,026 lines covering ~9 concepts. |
| External-program adapters | ❌ | 10 raw `exec.Command` calls, no context, no typed errors. Test seams are package globals. |
| Signals / cancellation | ❌ | Ctrl-C during `build` leaves tectonic running and leaks the temp dir. |
| Typed data end to end | ❌ | Customer, issuer, invoice and config are `map[string]any`. This is the root cause of the octal, merge-key and wrong-type bugs. |
| Help, docs, completion | ⚠️ | Every command has help and `-h` works everywhere, but it is hand-written and drifted. Completion is zsh-only and hand-maintained. No `version`. |
| Tests | ⚠️ | 140 tests, 76% coverage, passes `-race`; domain tests build no command (good). But red on Linux, not hermetic, captures by swapping `os.Stdout`, 151 `Contains` checks vs 1 exact check, no e2e, fuzz or TTY tests. |
| Build, release, lint, CI | ❌ | No CI, LICENSE, lint config, GoReleaser or version wiring. Module path `invox`. Makefile can run a stale binary. Builds are reproducible with `-trimpath` (good). |

## Fix first: bugs that lose data or money

All verified. "P" means proved by running the binary or a proof test. "R" means confirmed by reading the code.

| # | Bug | Evidence | Fix |
|---|---|---|---|
| 1 | **`invox email -o <path>` overwrites an existing file without checking, then deletes it 5 s later** with a detached `sh -c '(sleep 5; rm -f "$1") &'`. The default `<input>.eml` is also overwritten. `-o invoice.pdf` would destroy the PDF. | P, R: `internal/cli/commands_invoice.go:212`, `internal/invoice/email.go:201` | Refuse existing targets unless `--force`. Never schedule deletion of an explicit `-o`. Use a temp dir for the implicit draft. |
| 2 | **yaml.v3 reads numbers with a leading zero as octal**: `quantity: 010` is 8, `unit_price: 0100` is 64, `postal_code: 01067` becomes `"567"`, `number: 0042` becomes `"34"`. `parseDecimal` also accepts `1/3` and `0x10`. | P: `internal/invoice/yaml.go:62-71` | Keep the scalar's source text. Parse money with a strict decimal grammar. Decode into typed structs with string fields for codes. |
| 3 | **Duplicate invoice numbers.** Numbering only scans the archive, so two drafts get the same number, and `archive` accepts a second `CUST-001-001`. | P: `numbering.go:72`, `drafts.go:300-314` | Reject archiving a number that already exists. Reserve numbers, by scanning drafts or keeping a counter file. |
| 4 | **The starter template drops or mangles non-ASCII text.** It loads `T1`+`inputenc` under XeTeX, so "Čapek … Łódź" becomes "apek … ód", "Straße" becomes "StraSSe", and the build still exits 0. | P (with XeLaTeX, the engine tectonic wraps): `starter/template.tex:2-3` | Use `fontspec` and add `\tracinglostchars=3` so a missing glyph fails the build. |
| 5 | **A negative `paid_amount` is accepted**: total 120, paid −500 gives outstanding 620, and the QR code asks for €620. | P: `service.go:793` | Reject values below 0. |
| 6 | **A broken `config.yaml` blocks nearly every command**, including `init`, `customer list -c file` and usage errors (which then exit 1, not 2). Config is parsed before flags. | P: `internal/cli/parsing.go:57` | Parse flags first and load config lazily. `init`, `help` and explicit-path commands must not need it. |
| 7 | **YAML merge keys (`<<:`) are ignored and duplicate keys silently win.** A line item that inherits 10% VAT is billed at 20%. | P: `yaml.go:39-46` | Typed decode via yaml.v3 `Decode`. |
| 8 | **Re-archiving silently replaces the archived invoice**, and `build` on an archived invoice flips its status to `built`. | P / R: `drafts.go:340` | Confirm on a TTY, require `--yes` otherwise, keep a backup, preserve `archived`. |
| 9 | **The test suite is red on Linux**: `TestEditableConfigPathCreatesCommentedTemplate` hard-codes the macOS archive path, and there is no CI. | P: `service_test.go:2388` vs `service.go:983` | Compute the expectation from `DefaultArchiveDir()` with `XDG_DATA_HOME` pinned. Better: make `archiveDataBaseDir(goos, getenv, home)` pure. Add CI. |
| 10 | **Placeholder replacement is nondeterministic.** It iterates a map, and values can contain `@@X@@`; one invoice gave 4 distinct outputs in 200 renders. | P: `service.go:917-923` | One pass with a single `strings.NewReplacer`. |

Smaller correctness items (details in `reports/correctness.md` and `reports/testing.md`):
- **Money and amounts**
  - int64 cents overflow wraps silently.
  - Displayed unit price `0,13 × 8 = 1,00` (exact 0.125).
- **TeX output**
  - A value starting with `[` or `*` after `\\` breaks the TeX.
- **Email draft**
  - The `.eml` declares 7bit but contains UTF-8.
  - The attachment filename isn't RFC 2231-encoded.
- **Validation and parsing**
  - No YAML alias-expansion limit: a 330-byte file takes 16 s.
  - The IBAN check accepts check digits 00, 01 and 99.
  - A numeric `customer_id` is reported as "missing".
  - Wrong-type values are stringified, e.g. `map[first:X]`.
  - `regexp.MustCompile` in `numbering.go:339`. It is reachable only with invalid UTF-8, which yaml.v3 already rejects, so it is low risk, but use `Compile`.
- **Numbering**
  - Patterns that format but never parse back, e.g. `{customer_code}{counter:03}` with codes A and A1, make the counter restart silently: `numbering.go:204` skips files that don't parse back without warning.

## `quality-cli` conformance: what to change

### 1. Errors and exit codes
- `run*` functions return `error`. Add `FlagError`, `SilentError` and `CancelError` types. **One** `Main()` maps errors to codes and prints once, to stderr.
- **Keep exit 2 for usage errors** (14 tests assert it; the skill says to keep what an existing tool does) and document it in a new `help exit-codes`. Add 130 for SIGINT.
- Usage errors print the error plus `Run 'invox <cmd> --help'`, not the 80–100-line root, customer or template help.
- Runtime errors never print usage: `email` currently returns 2 and prints usage for a path-resolution failure (`commands_invoice.go:166`). Use a consistent `error:` prefix, relative paths, and a next step. For example, an unknown customer should suggest `invox customer list`, and the tectonic hint should match the platform, not always say `brew install`.

### 2. Parsing, flags and grammar
- `reorderArgs` causes wrong errors:
  - `new CUST-001 -o` reports "missing CUSTOMER_ID" because the dangling flag swallowed the argument.
  - An unknown flag after a positional is reported as "unexpected arguments".
  - `--` isn't honoured.
  - It is applied to only 3 commands, so the same typo gets different errors.
  - Replace it with pflag/cobra, which also gives "did you mean" suggestions.
- Accept the `INVOICE` positional on `validate`, `render` and `increment` too; it's additive.
- Grammar: keep invoice verbs at the top level, since invoice is the primary noun and moving them would break scripts. Split the overloaded `archive` (verb and noun) into `archive add|list|edit`. Rename `customer config` to `customer edit`, and `-s/--source` to `--defaults`. Old forms stay as hidden aliases with a one-line stderr deprecation warning for at least one release.
- Add `--yes`, `--force` and `-n/--dry-run` where they apply: `new` (force), `increment` (dry-run), `archive add` (yes/dry-run), `email -o` (force). Three different flags with three meanings.

### 3. Streams, TTY and machine output
- Introduce IOStreams (`In`, `Out`, `ErrOut`, `IsStdoutTTY`, `CanPrompt`), passed through `Run`.
- stdout carries **data only**:
  - `new`, `build`, `render` and `archive` print the path they created.
  - "Created/Opened/Initialized/Archived …" sentences go to stderr (or show only on a TTY).
  - tectonic, editor and opener stdout go to stderr.
  - This is a script-facing change; note it in the changelog.
- Two output modes:
  - TTY: header and aligned columns.
  - Non-TTY: tab-separated rows with no header, `\t` and `\n` escaped, ANSI stripped from YAML-sourced text, and an empty list message only on a TTY.
- Add `--json <fields>` to `customer list`, `archive list`, `template list` and `validate`, plus result objects for `new`, `build` and `archive`. Build the whole value before writing, and print `[]` for an empty list, never `null`. Field proposals are in `reports/io-config.md`.
- Format money for humans in the presentation layer. `validate` currently prints `total 120,00 \euro`.

### 4. Interactivity
- Launch an editor only if stdin and stderr are TTYs and `--no-input`/`INVOX_PROMPT_DISABLED` is unset. Otherwise fail fast with a `FlagError` (`new -e`: "stdin is not a terminal; edit the file then run …").
- Don't overwrite the user's `INVOX_EDITOR`. Drop `-l` (no login profile sourcing). Report editor exit failures with the editor's name.

### 5. Configuration
- Load config **once**, into a typed struct, strictly: unknown keys and wrong types are errors with file:line. Load it lazily through the Factory, after flags.
- Add `INVOX_CONFIG_DIR` and `--config`. An explicit path must never fall back. Add `help environment` listing `INVOX_*`, `XDG_*`, `APPDATA`, `VISUAL`/`EDITOR` and the precedence order.
- Stop the upward support-file search at `$HOME` or a project marker; a stray `~/customers.yaml` currently overrides config. Add a way to show which files were resolved (`invox config paths` or `--verbose`).
- `init` must detect the legacy `invoice-tool` dir and offer migration rather than silently shadowing it. Plan the legacy fallback's removal with a deprecation warning.
- macOS and Windows use different data-dir schemes than Linux; document them or unify on XDG-style overrides.
- Atomic writes should keep the existing file's mode (it is currently forced to 0644, so a 0600 invoice becomes 0644) and resolve symlinks before renaming. Create `issuer.yaml` (IBAN/BIC) and the archive with 0600/0700.

### 6. Architecture and layering
Target layout (detail and import diagram in `reports/architecture.md`):

```
cmd/invox/main.go            os.Exit(app.Main())
internal/app/                signal ctx, IOStreams, Factory, error → exit code
internal/iostreams/          streams, TTY detection
internal/cmdutil/            Factory, FlagError/SilentError, presentation helpers
internal/cmd/<noun>/<verb>/  Options + NewCmdX(f, runF) + xRun(opts)
internal/config/             typed Config, Dirs(env), strict Load, path resolution
internal/invoice/            typed Customer/Issuer/Invoice/LineItem, validation, totals, numbering
internal/invoice/yamldoc/    comment-preserving node edits + atomic write
internal/archive/            one Store (List/Latest/Resolve/Put): replaces 3 directory walkers
internal/money/  internal/epc/  internal/render/latex/  internal/email/
internal/adapters/{tectonic,editor,opener,applemail}/   CommandContext, typed ExecError
```
Rules, enforced with depguard and forbidigo:
- Nothing below `internal/cmd` imports cmd, cmdutil or iostreams.
- No `fmt.Print*`, `os.Std*`, `os.Getenv` or `os.Exit` outside `main` and the IOStreams constructor.

Also:
- Delete the dead code: `renameMappingKey`, `nodeIsEmpty`, `invoiceEmailBody`, `firstPresentPath`, `firstNonEmptyPath`, `prependPath`.
- Merge the three path-extension helpers.
- Move CLI wording out of domain errors: `-o/--output`, `invox template list`, `brew install tectonic`.

**cobra:** recommended, migrated one noun at a time behind the old dispatcher.
- **For:**
  - It deletes `reorderArgs`.
  - It generates help, bash/zsh/fish/pwsh completion, man pages and markdown, replacing ~950 hand-written lines.
  - It gives typo suggestions.
- **Cost:**
  - Roughly +1–2 MB of binary (estimated, not measured).
  - Two dependencies.
- **Risk:**
  - Stdlib `flag` accepts `-input` and `-customers`; pflag does not.
  - Ship an argument normaliser that rewrites known single-dash long flags with a warning, for at least one release.
- **Zero-dependency alternative:** one constructor per command on stdlib `flag`, with help generated from `flag.VisitAll`.

### 7. Tests
- **Now:**
  - Fix the Linux failure.
  - Make `captureRun` always sandbox the config dir; it currently uses the developer's config if `XDG_CONFIG_HOME` is set.
  - Restore swapped globals with `defer`/`t.Cleanup`.
- **Safety net before refactoring:** a `testscript` suite (`cmd/invox/testdata/script/*.txtar`) driving the real binary in process, with fake `tectonic`, `osascript` and editor registered as in-process commands so it works on Windows, sandboxed `HOME`/`XDG_*`/`APPDATA`, and an `exits CODE` command. A reviewer built a 4-script prototype that passes. The scripts are in `reports/testing.md`.
- **After IOStreams:**
  - Buffer-backed streams, exact stdout and stderr assertions in **both** TTY modes, and `t.Parallel()`.
  - A `Runner` stub that panics on unregistered commands.
  - An injected `Now`, so the `.eml` `Date:` and boundary can be golden-tested.
- **Fuzz targets** (a 30 s run already found the overflow and a round-trip bug): invoice-number format→parse, `parseDecimal`, IBAN, `latexEscape`, placeholder validation, EPC payload.

### 8. Build, release, docs
- `module github.com/0xboris/invox`, so `go install …@latest` works. Rewrite 10 imports.
- `invox version` / `--version`, set via ldflags with a `debug.ReadBuildInfo` fallback. The binary already embeds a pseudo-version.
- Add a LICENSE. Also ship the Ubuntu Font Licence with `fonts/`, or remove `fonts/`: nothing in the code uses it, it isn't embedded, and the starter template doesn't reference it.
- Add `.golangci.yml`, CI (linux/macos/windows × Go min/stable, `-race`, `tidy -diff`, vet, lint, govulncheck) and GoReleaser. Snippets are in `reports/build-docs.md`.
- **Makefile:**
  - `make build` can run a stale binary because the file target has no source deps; make `build` always run.
  - Drop the workflow wrappers, which duplicate CLI flags and diverge: `make email` always passes `-o`, which disables Apple Mail compose.
- `go mod tidy` (one missing go.sum line). Bump `go` to 1.24, keeping a CI job on the minimum version.
- **README:** add `go install` instructions, tectonic for Linux/Windows, exit codes, env vars, licence and an invoice YAML reference, and refresh "Last reviewed". Generate the command reference from the command tree and drift-check it in CI.
- **Repo hygiene:**
  - Extend `.gitignore`: `*.pdf`, `*.tex` at the root, `*.eml`, `*.aux`/`*.log`/`*.xdv`, `coverage.out`, `dist/`.
  - Remove the TypeScript `coding-standards` skill.
  - Pin or remove the unpinned skills.
  - Move `features/multi_vat/prd.md` (already shipped) to `docs/design/`.

## Roadmap

Each phase ships on its own and keeps `go test ./...` green.

| Phase | Scope | Why this order |
|---|---|---|
| **0. Stop the bleeding** (≈1–2 days) | Bugs 1–10 above, with a regression test each. Linux test fix. CI. LICENSE. Module path. `version`. `.gitignore`. Makefile `build`. | Data loss and wrong amounts first. CI keeps every later step honest. |
| **1. Freeze behaviour** | testscript suite covering every command's stdout, stderr and exit code, plus help pages. Hermetic `captureRun`. | The skill's §Migrating step 1: nothing changes without a test noticing. |
| **2. IOStreams + errors** | Streams through `Run`. Typed errors and a single exit mapping. Parse flags before config. Status to stderr. TTY/non-TTY list output. `help exit-codes` and `help environment`. | Highest rule coverage per line changed. Unblocks exact-output tests. |
| **3. Adapters + context** | Tectonic, editor, opener and mailer adapters on a Factory. `signal.NotifyContext`, exit 130. Injected clock and env. TTY gate for the editor. Remove package-global seams. | Makes tests parallel and Ctrl-C safe. |
| **4. Typed data + package split** | Typed config, customers, issuer and invoice. Split `service.go` into the layout above. Dead-code removal. depguard and forbidigo on. | Fixes the octal, merge-key and wrong-type bug class at the root. |
| **5. Command layer** | Per-command Options and constructors. cobra, one noun at a time. Generated help, completion and docs. `--json`. `--yes`/`--force`/`--dry-run`. Grammar cleanup with deprecations. | Biggest surface change, done last, on a frozen and typed base. |
| **6. Release** | GoReleaser, Homebrew tap (depends on tectonic), CHANGELOG, generated man pages. | Once the contract is stable. |

## Decisions for you

1. **Exit code for usage errors**: keep 2 (recommended, already tested) or move to gh's 1?
2. **cobra or stay zero-dependency**: cobra is recommended; the trade-offs are above.
3. **The stdout contract change** (status sentences to stderr, data-only stdout): acceptable as a breaking change with a changelog note, given no known scripts depend on it?
4. **The `fonts/` directory**: ship it with its licence, or delete it?

## Calibration notes

How the reviewer findings were weighed:
- **Re-rated**:
  - "No version command" and "module path" were raised as blockers. They are rated major here: they matter for distribution, but nothing breaks for current users.
  - The lint finding `nilerr` at `numbering.go:204` is intentional, since it skips non-matching archive files. Its real risk is silent counter resets, listed above.
  - The `regexp.MustCompile` panic is real but practically unreachable through YAML input.
- **Evidence limits**:
  - The starter-template Unicode finding was proved with XeLaTeX, not tectonic, because tectonic's bundle host was unreachable from the review environment.
  - Windows behaviour was assessed by reading the code and by cross-compiling and vetting, not by running on Windows.
  - The cobra binary-size cost is an estimate.
