# Cleanup plan

Base: `236d558`. Every `file:line` and count below refers to that commit and was taken from the code
in this worktree. "Inferred" marks a claim reasoned from the code and not run.

Scope: everything except `internal/billing/ports.go` and the `Service` method set, which the
ports v2 design (another agent, in parallel) owns. Ports v2 lands first. Units that touch
`internal/billing`, a port implementation or a type in `ports.go` say so and rebase on it.

Binding decisions applied here:

- CLI output may change when every change is listed. No capability is removed without the
  maintainer, except where the maintainer decided it.
- Drop the legacy `invoice-tool` config directory.
- Drop the deprecated command forms of #106 (`d244b22`).
- Stop reading Markdown archived invoices, and warn once per run when some were skipped.
- Evaluate the remaining compatibility code (placeholder migration, removed YAML keys, others).

How it was measured: `go vet ./...` (clean), `deadcode ./...` and `deadcode -test ./...` from
`golang.org/x/tools@v0.30.0`, `staticcheck` 2025.1.1 with `-checks all`, the `quality-cli` audit
script, `go test ./...` (all pass), greps and per-function line counts, and a built binary run
against a scratch config (`invox init`, `new`, `render`, `email --dry-run`, `validate`).

## 1. Verdict

**Keep.** The ring structure works and is enforced twice: depguard rules per package row in
`.golangci.yml` and the transitive import check in `internal/archtest`. The entities import only
the standard library and `money`. Errors are typed and mapped to exit codes in one place
(`internal/cli/exit.go:21`). Commands follow gh's `NewCmdX(f, runF)` shape with injected streams,
env and runner, and `run.Stub` fails a test on any unregistered program. The 35 testscript files
pin stdout, stderr and exit codes end to end. Dead code is near zero. `deadcode ./...` reports only
the test helpers `run.Stub` and `factorytest`, `deadcode -test ./...` reports nothing, and
`staticcheck` reports only the gh-style names `SilentError` and `CancelError` (ST1012).

**Costs the most.**

1. Compatibility code in every ring. The legacy directory, the deprecated forms and the single-dash
   normaliser are about 330 lines of production Go, a 241-line `deprecated.txtar`, warnings in
   eight other scripts and about 25 Go tests. Markdown archives add about 80 lines and about 20
   tests.
2. Facts with several owners and no check between them. Template placeholders live in
   `render/latex` and `cli/helptext`, and a misspelled one renders literally with exit 0 (run:
   `@@CUSTOMR_NAME@@` stayed in `out.tex`). Email placeholders are listed three times, config keys
   four times, support-file flag names twice, and `AbsPath` is implemented three times.
3. Command boilerplate that carries strings. 37 hard-coded command names in error calls, 11
   hand-written `Args` funcs, two `runF` signatures.
4. Test architecture. `internal/cli` holds 7,943 test lines (58 tests in `cli_test.go` alone,
   mostly per-command), the context fixtures exist in three copies, and help text is pinned three
   times (help.txtar goldens, `docs/cli`, Go help tests).

Two packages are large but cohesive enough once the removals land: `store` (2,478 lines) and
`billing` (1,959, out of scope). The plan removes before it moves. It proposes one new package
(test fixtures) and no new runtime layer.

## 2. Findings

Earlier review items keep their ids (F1 to F8). C are the compatibility removals, N are new.
Behavior: "none" means testscript goldens, `docs/cli` and `share/man` stay byte-identical.

| ID | Problem | Evidence | Change | Behavior | Size | Risk | Done when |
|---|---|---|---|---|---|---|---|
| C1 | Deprecated command forms kept for #106 | `archive FILE` via `add.Configure(..., deprecated bool)` (`cmd/invoice/archive/add/add.go:78-106`), hidden `customer config` (`cmd/customer/edit/edit.go:39-50`), `send` alias (`cmd/invoice/email/email.go:47,84`), `new -s/--source` (`cmd/invoice/new/new.go:84-100`), `normalizeLongFlags` (`cli/root.go:198-257`), `cmdutil/deprecated.go` | Delete them. Replace the normaliser with a 20-line pre-parse check that rejects, not rewrites, a single-dash word naming a long flag. Add `SuggestFor` on `email` ("send") and `customer edit` ("config") | Listed in U1 | M | Low | `grep -rnE 'WarnDeprecated\|DeprecateFlag\|normalizeLongFlags\|CalledAs\|"send"' internal` is empty; `deprecated.txtar` is gone; new `grammar.txtar` cases pass |
| C2 | Legacy `invoice-tool` config directory | Fallback lookup `store/host.go:122-192`, copy `store/init.go:63-134`, warning `cli/cli.go:49,62-84`, init prompt `cmd/init/init.go:63-121`, help `cli/helptext/topics.go:190-215`, `cmd/config/config.go:32`, `cmd/config/paths/paths.go:42,87` | Delete the CLI, store, help and docs side. Ports v2 deletes `Directory.LegacyFiles/CopyLegacy/LegacyFilesUsed`, `Locations.LegacyDir`, `SourceLegacy` | Listed in U2 | M | Low | `grep -rn 'invoice-tool\|LegacyDir\|LegacyFiles\|legacy fallback' internal cmd docs/cli share` is empty |
| C3 | Markdown archived invoices | `store/yamldoc.go:141-205`, `store/invoices.go:205-225`, `archive/archive.go:188-196,288-305`, `_invox.archive_replace_path` (`store/archive_metadata.go:13-34`, `invoice/models.go:118`) exists only for the Markdown-to-YAML swap | Stop reading `.md`/`.markdown`; drop `archive_replace_path`; print one stderr warning per run from the skip count ports v2 carries | Listed in U3 | M | Med | `grep -rniE 'markdown\|front ?matter\|archive_replace' internal --include=*.go` hits only the skip count and the warning; `TestMarkdownArchiveWarning` passes |
| C4 | Removed YAML keys table | `store/schema.go:114-122`, branch `store/yaml.go:268-272`, help group "Unsupported legacy keys" (`cli/helptext/reference.go:227`) | Delete. The keys fall through to the unknown-key error and its existing help-topic hint | `unsupported key; use positions` becomes `unknown key "line_items"` plus the hint; exit 1 unchanged | S | Low | `grep -rn removedKeys internal` is empty; a test proves each path that hit it decodes strictly |
| C5 | Silent template migration and removed-placeholder map | `latex.MigrateLegacyPlaceholders` rewrites the old starter VAT row on every render (`render/latex/template.go:34-39`, `renderer.go:126`); a 4-entry replacement map (`template.go:15-24`); help line `cmd/template/template.go:27` | Delete both. F4's unknown-placeholder check reports old names like any other | An old starter VAT row fails with exit 1 instead of rendering | S | Low | `grep -rn MigrateLegacyPlaceholders internal` is empty |
| F1 | Store holds a second, wider API under its port adapter | `store` 2,478 lines. 22 exported `Host` methods (`store/host.go`, `sources.go`, `config_paths.go`, `templates.go`, `init.go`), of which only `ResolveArchiveDir` and `Settings` are used outside `store` (`factory/factory.go:54,58`). `Store` converts mirror types `PathReport`, `InitFileResult`, `TemplateSummary`, `SupportFile`, `store.Files` (`store/store.go:34-39,180-211`) | Unexport the `Host` methods, delete the mirror types, return `billing` types directly | none | M | Low | `go doc ./internal/store` lists only `Store`, `Host`, `NewHost`, `HostInputs`, `ReadArchived` and what `factory` calls |
| F2 | Status partly typed | No status literals remain outside `invoice/lifecycle.go`. `Header.Status` is `Text` (`invoice/models.go:100`), converted at `billing/email.go:59`, `store/invoices.go:84`, `cmd/invoice/build/build.go:166`. `ArchiveEntry.Status` is `string` (`billing/ports.go:224`, ports v2) | Make `Header.Status` an `invoice.Status` that `store` decodes trimmed | none | S | Low | `grep -rn 'invoice.Status(' internal --include=*.go` is empty outside tests |
| F3a | Command names as strings in errors | 37 literals in `FlagErrorf`, `UsageError` and `shared.*` calls across 14 files; `FlagError.Command` (`cmdutil/errors.go:14`); `Main` drops the command `ExecuteContextC` returns (`cli/cli.go:45`) | `Main` names the help from the executed command; `FlagError` keeps an override only for global flags | `unexpected arguments for version: x` becomes `unexpected arguments: x` | M | Low | `grep -rnE '(FlagErrorf\|UsageError\|shared\.[A-Za-z]+)\("[a-z ]+"' internal` is empty |
| F3b | Hand-written `Args` funcs | 11 `unexpected arguments` funcs plus `shared.TakeInput` (`cmdutil`, `cmd/*`) | `cmdutil.NoArgs` and `cmdutil.ExactArgs(names...)` | none | S | Low | `grep -rn 'unexpected arguments' internal --include=*.go` hits only `cmdutil` |
| F3c | Support-file flags registered by hand | `-c/--customers` 8 times, `-u/--issuer` 5, `--defaults`, `-t`; their names again in `supportFlags` (`cmdutil/support.go:11-16`) | Register them from the `supportFlags` table: name, shorthand, usage, completion | none | S | Low | `grep -rn '"customers", "c"' internal` hits only `cmdutil` |
| F3d | Two `runF` signatures | 16 `if runF != nil`; 9 take no `context.Context` | One signature with `ctx` | none | S | Low | `grep -rn 'runF func(\*' internal` is empty |
| F4 | Placeholders unchecked | Template values in `render/latex/renderer.go:15-49` and `line_item_blocks.go:11-20`, docs in `cli/helptext/reference.go:41-111`. Email placeholders in `billing/emailtext.go:73-89`, `cmd/config/config.go:56-67`, `store/config_paths.go:88-99`. Run: `@@CUSTOMR_NAME@@` rendered with exit 0; `--subject 'Invoice {invoice_numbr}'` drafted with exit 0 | `latex` rejects any `@@[A-Z0-9_]+@@` it does not know; `billing` rejects any `{[a-z_]+}` it does not know. One table per set; help and the starter config render from it; a cross-check test in `internal/factory` compares `latex.Placeholders()` with the helptext table | Unknown names fail with exit 1 (U4) | M | Med: a body with literal `{word}` text now fails | `TestUnknownPlaceholderFails` (render and email) and `TestPlaceholderDocsMatch` pass |
| F5 | Long archive function | `Archive.Add` is 51 lines (`archive/adapter.go:143`) after unit B | No action | none | - | - | - |
| F6 | Validation messages as strings | Resolved: `invoice.Problem` and `billing.DecodeError` are typed and `validate --json` reads fields (`cmd/invoice/validate/json.go:37`). Leftovers: `DecodeError` has both `Path` and `Field` (`billing/errors.go:49-63`); three hand-rolled error-tree walks (`cli/exit.go:100`, `validate/json.go:37`, `billing/errors.go:67`) | After ports v2, merge `Path` and `Field`. Leave the walks: two of them sit in different rings | none | S | Low | `DecodeError` has one field name |
| F7 | Temp-file policy in the email command | Resolved: `email.Mailer` owns the directory and pruning (`email/mailer.go:40,110`). Leftover: `Files.EmailOutput` and `EmailRequest.Keep` are the same boolean set twice (`cmd/invoice/email/email.go:144,154`) | Ports v2 decides; CLI drops one | none | S | Low | one boolean |
| F8a | Regexp compiled per call | `numbering.Parse` compiles a regexp per archived entry (`numbering/numbering.go:240`), and `Validate`, `Format`, `Parse` each switch over the same six tokens (`:58`, `:169`, `:218`) | One token table; `Parse` formats the text before and after `{counter}` and cuts them off | none | S | Low | `FuzzInvoiceNumberRoundTrip` passes; `grep -n regexp.Compile internal/numbering` is empty |
| F8b | Date layout repeated | `"2006-01-02"` 10 times in 6 files | `time.DateOnly` | none | S | Low | grep is empty |
| F8c | `latex.Item`/`VATRow` struct-conversion coupling | `render/latex/latex.go:16-30`, `renderer.go:53-67`; `latex` already imports `invoice` | Use `invoice.LineItem` and `invoice.VATBreakdown` | none | S | Low | grep `Item(item)` is empty |
| F8d | `run.Stub` in the release package | `adapters/run/stub.go`; the linker drops it (deadcode: unreachable) but it is compiled with `run` | Move to `adapters/run/runtest` | none | S | Low | `go list -deps ./cmd/invox` has no `runtest` |
| F8e | Three `edit` and three `list` packages; `cmd/invoice/` | gh lays out `pkg/cmd/<noun>/<verb>` the same way; `CLAUDE.md` documents `cmd/invoice/` | No action | none | - | - | - |
| F8f | `completion` outside `internal/cmd` | `cli/completion.go` | Move to `internal/cmd/completion` | none | S | Low | file gone from `cli` |
| F8g | `errorHint` grows per error type | 6 branches (`cli/exit.go:61-87`) plus the `configFlagError` wrap in `mainContext` (`cli/cli.go:50-53`) | Keep the switch; fold the wrap into it by passing `f.ConfigFile`. A hint registry would move the branches, not remove them | none | S | Low | `configFlagError` gone |
| N1 | Two YAML decoders | `config.Load` decodes with yaml.v3 `KnownFields` and rewrites its error strings with three regexps (`config/config.go:206-280`); `store` has its own strict node decoder with line numbers and alias limits. Only `store` imports `config` | Decode `config.yaml` with the `store` decoder; delete the regexp rewriting and `keysByLine` | Config type-error wording takes the support-file wording (list in U8) | M | Med | `grep -n regexp internal/config` is empty, or `internal/config` is gone |
| N2 | `AbsPath` three times | `cmdutil.AbsPath` (`cmdutil/paths.go:11`), `store.absPath` (`store/abs_path.go:10`), `factory.absAgainst` (`factory/factory.go:97`). Inferred bug: `absAgainst` lacks the rooted-path case, so on Windows `--config \x.yaml` resolves under the working directory while `-c \x.yaml` resolves at the drive root | `factory` and `store` share `fsutil.Abs`; `cmdutil` keeps its copy (the CLI may not import `fsutil`) | `--config \x` on Windows resolves like `-c \x` | S | Low | a Windows CI test for `--config \x.yaml` |
| N3 | Output-exists errors worded three ways | `new.go:142`, `email.go:198-207` (matches `fs.ErrExist` because `email.Mailer` does not return `*billing.OutputExistsError`), `archive/edit/edit.go:103` | `email.Mailer` returns the billing error; one `cmdutil` wording for `-o` | email: `already exists; pass --force or choose another -o path` becomes `already exists; pass --force to replace it or choose a different -o/--output path`; same for "is a directory" | S | Low | email.txtar golden updated |
| N4 | The build command knows the compiler is tectonic | `cmd/invoice/build/build.go:13,139-150` imports `adapters/run` and rewraps `*run.ExecError` as `cmdutil.ExecError{Program: "tectonic"}` | `tectonic` returns a `billing` tool-failure error next to `ToolMissingError`; `build` stops importing `run` | none | S | Low | `grep -rn adapters/run internal/cmd` is empty |
| N5 | Help built two ways | Command `Long` texts are templates (`helptext/render.go:43`); topics are 127 `Fprintf` calls (`helptext/topics.go`); six alias methods map old template names to `Locations` fields (`render.go:55-60`) | Topics become embedded `.tmpl` files rendered by `helptext.Render`; templates use the `Locations` field names | none | M | Low | `grep -c Fprintf internal/cli/helptext/topics.go` is 0 |
| N6 | Config keys documented four times | `config.Config`, `cmd/config/config.go:39-48`, the starter comment `store/config_paths.go:72-99`, `help environment`. The starter is a 76-line `Sprintf` (`config_paths.go:66`) while the other starters are embedded files (`store/init.go:14`) | Starter `config.yaml` becomes an embedded template; a test asserts every key in the help list decodes | none | S | Low | `TestConfigKeysDocumented` passes |
| N7 | Adapter surfaces wider than their use | `tectonic.Build` has one caller (`Compile`); `applemail.Compose`/`Message` are used by `Draft` and one test; `archive.Store` exports 7 methods only `Archive` calls; `CheckAttachment` is the same `os.Stat` in two mailers | Inline or unexport them; one `CheckAttachment` helper in `fsutil` if ports v2 keeps the method | none | S | Low | `go doc` of each package shows only what `factory` or a port needs |
| N8 | `printError` reads the process cwd | `cli/exit.go:53` calls `os.Getwd`, an entry on the ambient allowlist | Pass `f.Env.Getwd` | none | S | Low | the allowlist entry is gone from `ambient_test.go` |
| N9 | Test fixtures duplicated | `writeContextFixtures` in `billing/fixtures_test.go:11`, `render/latex/invoice_test.go:74`, `cli/cli_test.go:2157`; `writeDraftFixtures` twice; `writeArchivedInvoiceMarkdown` twice; `writeConfigFile` five times; inline issuer YAML in 7 test files | One test-only package `internal/testfixture` with the files under `testdata` | none | M | Low | each helper name is defined once |
| N10 | Command tests in the wrong package | `internal/cli` has 7,943 test lines; most of `cli_test.go`'s 58 tests drive one command | Move per-command tests to their command packages on `factorytest`; `internal/cli` keeps `Main`, exit codes, signals, help routing, global flags | none | L | Low | `internal/cli` test lines below 4,000; test names unchanged (`go test -list`) |
| N11 | Help pinned three times | `help.txtar` (1,548 lines of page goldens), `docs/cli/*.md` with CI drift check, Go help tests | `help.txtar` keeps routing checks (`help X` equals `X --help`) and compares pages to each other; `docs/cli` is the content golden | none | M | Low: content regressions are caught by `make docs` diff, not `go test` | `help.txtar` under 300 lines |

## 3. Units in recommended order

Each unit is one worker and one to three commits. Every unit ends with `go test -race ./...`,
`make lint`, `go run ./internal/docs/gen` with no diff left uncommitted, and its "Done when"
checks. Units that change output rewrite the goldens with
`go test ./cmd/invox -run TestScript -update`, and the reviewer reads the golden diff against the
behavior list. Each unit adds a CHANGELOG line for every listed behavior change.

Removals come first, so the later units move less code.

### U1. Drop the deprecated command forms (C1). Now, no ports v2 dependency.

Delete `archive FILE` (fold `add.Configure` back into `NewCmdAdd`; give `archive` the `customer`
noun's `Args`/`RunE`), `customer config` (`NewCmdConfig`, the `name` parameter and
`EditOptions.Command`), the `send` alias, `new -s/--source`, `cmdutil/deprecated.go`, the
hidden-flag pass in `closestFlag` (`cmdutil/cobra.go:79-100`), the hidden `--json` check in
`usage.go:51`, and the hidden-command exception in `helpTopic` (`usage.go:132`). Replace
`normalizeLongFlags` with a check that rewrites nothing. A single-dash word of three or more
characters fails when its name is a long flag of the command, or when its first letter is not a
shorthand (so `-ofile.yaml` still works). The check exists because pflag would otherwise read
`-output=x.pdf` as `-o utput=x.pdf` and write a file named `utput=x.pdf` (inferred from pflag's shorthand parsing and `RequireExtension`, which accepts it).

Behavior changes:

- `invox archive FILE` exits 2: `error: unknown archive subcommand "FILE"` and
  `Run 'invox archive --help' for usage.`
- `invox send ...` exits 2: `error: unknown subcommand "send"; did you mean "email"?`
- `invox customer config` exits 2: `error: unknown customer subcommand "config"; did you mean "edit"?`
- `invox new C -s F` exits 2 with `unknown shorthand flag: -s`; `--source` with `unknown flag: --source`.
- `-names`, `-input`, `-help`, `-version` and the like exit 2 with
  `error: -names is not a flag; use --names`. No more `warning: -X is deprecated` lines.
- `invox help customer config` and `invox help send` exit 2 with `unknown help topic`.
- Help pages, `docs/cli` and `share/man` should not change. The hidden forms have no pages and
  their flags are hidden (checked: no `customer config`, `send` or `--source` in `docs/cli`).
  Any diff `make docs` shows is a surprise to review.

Tests: see Appendix A1.

### U2. Drop the legacy config directory (C2, CLI, store, help and docs side). After ports v2.

Delete `Host.LegacyConfigDir`, `legacyUse`, the `SourceLegacy` row in `findInConfigDir`,
`LegacyFilesToCopy`, `CopyLegacyFiles`, `legacyFilePerm`, the three `Store` pass-throughs,
`warnLegacyFiles` and its call in `mainContext`, `copyLegacyFiles` in `init`, the `legacy` source
label in `config paths`, `data.LegacyConfigFile`, and the help text. If ports v2 has not yet
removed the port methods, this unit removes them in the same commit and tells that agent.

`init --force` had one job, copying legacy files without asking. The plan removes it with the
directory. See "Needs the maintainer" item 1 for the alternative.

Behavior changes:

- Files only in `$XDG_CONFIG_HOME/invoice-tool` are no longer found. A missing support file is the
  existing usage error `customers.yaml file not found; pass -c/--customers, set paths.customers in
  config.yaml, or place customers.yaml at <path>` (exit 2). A missing config.yaml means defaults.
- The `warning: using ... from deprecated config directory ...` line is gone.
- `invox init` never asks to copy and never fails for lack of a terminal. `invox init --force`
  exits 2 with `unknown flag: --force`.
- `invox config paths` never prints the `legacy` source.
- Help: `config` loses `legacy fallback:`; `init` loses the copy paragraph and the `--force` flag;
  `config paths` loses the `legacy` row; the `environment` topic loses "Legacy config directory
  (deprecated)" and the "legacy directory" step in both precedence lists. `docs/cli` and
  `share/man` pages for these four regenerate.

Tests: Appendix A2.

### U3. Stop reading Markdown archives (C3). After ports v2.

Ports v2 carries the count of skipped Markdown files from `archive` through `billing`. This unit:

- `archive.isInvoiceFile` accepts only `.yaml`/`.yml`; `Target.Edit` loses its Markdown branch and
  `Edit.Replace`. The walk counts `.md`/`.markdown` files outside `.history` and reports them.
- `store` deletes `markdownFrontMatter`, the Markdown case of `loadArchivedInvoiceDocument`,
  `loadSourceDocument`'s Markdown path and `isMarkdown`, and stops reading or writing
  `_invox.archive_replace_path` (`archive_metadata.go`, `schema.go:110`, `invoice.ArchiveLink`).
- `cli.Main` prints the warning after the command, where `warnLegacyFiles` runs today, once per
  run, never for `__complete` requests.

Behavior changes:

- New stderr line when the archive holds Markdown invoices:
  `warning: 3 Markdown invoices in <archive dir> are no longer read; convert them to .yaml to include them`
  (`1 Markdown invoice ... is no longer read` for one). Exit code, stdout and `--json` unchanged.
- Markdown invoices no longer count for numbering, the duplicate-number check, `archive list`,
  `archive edit`, `email`'s PDF-to-YAML lookup or `new --from-last`. `archive edit x.md` fails
  as not found (inferred from `isInvoiceFile`; the unit pins the exact message).
- A working copy made before this change that still has `_invox.archive_replace_path` fails strict
  decoding with `unknown key "archive_replace_path" in _invox` (exit 1). See maintainer item 6.
- No help or README text mentions Markdown archives today (checked with grep); only the CHANGELOG
  gains a "Removed" entry.

Tests: Appendix A3, plus a new `TestMarkdownArchiveWarning` (count, singular form, once per run,
stdout and `--json` unchanged, no line for `__complete`).

### U4. Check placeholders; drop the template migration and the removed-key table (F4, C4, C5). Now for `latex` and `store`; the email half after ports v2.

`latex.ValidateTemplate` collects every `@@[A-Z0-9_]+@@` outside the known set and reports
`<name>: unknown placeholder`, one per line, as it does for the four removed names today. Export
`latex.Placeholders()` from the same table `buildTemplateValues` fills, and add
`TestPlaceholderDocsMatch` in `internal/factory` (that ring may import both `render/latex` and
`cli/helptext`). Delete `MigrateLegacyPlaceholders`, the replacement map and the help line about
`@@VAT_RATE@@`. Delete `removedKeys` and its branch. Check every decode path that matched a removed
key is strict, or the key would now be ignored silently: `--from-last` reads an archived invoice,
and `TestCreateNewInvoiceFromLastRejectsLegacyArchivedInvoiceKeys` must still fail on `line_items`.

Email half: one placeholder table in `billing/emailtext.go` (name and description) drives
`render`, the `config` help list and the starter config comment. `subject` and `body` fail on an
unknown `{[a-z_]+}` token.

Behavior changes:

- `render`/`build` with an unknown placeholder exit 1:
  `error: <template>: @@CUSTOMR_NAME@@: unknown placeholder`.
- The four removed names give the same message instead of `unsupported placeholder; use ...`.
- A template with the old starter row `VAT (@@VAT_RATE@@\%): & @@VAT_AMOUNT@@\\` fails with that
  message instead of rendering.
- `email` with an unknown subject or body placeholder exits 1:
  `error: email.subject: unknown placeholder {invoice_numbr}` (`email.body` likewise).
- `line_items`, `invoice.period_label`, `invoice.vat_rate_percent` give
  `unknown key "line_items"` and the hint `Remove the unknown keys or fix their spelling; 'invox
  help defaults' lists the supported fields.` instead of `unsupported key; use positions`.
- Help: `invox help defaults` loses "Unsupported legacy keys"; `invox template` loses the
  `@@VAT_RATE@@` line.

Tests: Appendix A4.

### U5. Command boilerplate (F3a to F3d). After U1.

`Main` keeps the command `ExecuteContextC` returns and names it in `Run 'invox X --help'`.
`FlagError.Command` stays only as an override for global-flag errors (`cmdutil/cobra.go:34`).
Add `cmdutil.NoArgs`, `cmdutil.ExactArgs`, and register the support-file flags from the
`supportFlags` table. All `runF` take `ctx`. Drop `UsageError`'s command parameter.

Behavior: `invox version x` says `unexpected arguments: x` (was `unexpected arguments for
version: x`). Everything else is byte-identical.

### U6. Error wording and adapter errors (N3, N4, N8, F8g). After ports v2.

`email.Mailer` returns `*billing.OutputExistsError` and `*billing.OutputIsDirError`; one
`cmdutil` helper words them for `-o`. `tectonic` returns a `billing` tool-failure error, and
`cmdutil.ExecError` remains for the editor only if the editor still needs it. `printError` gets
the working directory from `f.Env`. Fold `configFlagError` into `errorHint`.

Behavior: the `email -o` messages listed under N3.

### U7. Entity and library tidy (F2, F8a to F8d, N2). F8a to F8d now; F2 after ports v2.

`time.DateOnly`; the numbering token table and prefix/suffix `Parse`; delete `latex.Item` and
`VATRow`; move `run.Stub` to `run/runtest` (and add the package to the archtest rules);
`Header.Status` as `invoice.Status`; `fsutil.Abs` shared by `store` and `factory`.

Behavior: on Windows only, `--config \x.yaml` resolves on the working directory's drive root
(inferred; add the test that proves it before the fix).

### U8. One YAML decoder (N1). After U2.

Decode `config.yaml` with the `store` decoder: a schema struct for `Config` in `schemaKeys`, the
indentation rule moved into the decoder's top-level check, `config.Path` and `~/` expansion kept.
Delete the yaml.v3 error rewriting. If nothing is left in `internal/config` but the types, move
them into `store` and delete the package and its depguard row.

Behavior: syntax errors keep their wording, because both paths wrap the yaml.v3 parser error
with the file name (`broken_config.txtar:50`). Unknown-key messages keep their wording too:
`config_paths.txtar:91` pins `bad.yaml:2: unknown key "patern" in numbering`, which is the store
decoder's format. Wrong-type messages for `config.yaml` take the store decoder's wording. No
testscript pins one (checked with grep), so the change shows in `internal/config` tests, and the
worker lists each changed message in the commit message. `config.yaml` gains the alias limits the
support files have.

### U9. Store surface (F1, N6). After ports v2, U2 and U8.

Unexport the 20 `Host` methods nobody outside `store` calls; delete `PathReport`,
`InitFileResult`, `TemplateSummary`, `SupportFile` and `store.Files` in favor of the `billing`
types; make the starter `config.yaml` an embedded template next to the other starters; add
`TestConfigKeysDocumented`. Consider splitting the YAML codec into `store/yamlx` (allowed by the
target spec) only if `store` is still above about 2,000 lines after U2, U3 and U8.

Behavior: none.

### U10. Adapter surfaces (N7, F8f). After ports v2.

Inline `tectonic.Build` into `Compile`; collapse `applemail.Compose`/`Message` into `Draft`;
unexport `archive.Store` and `ExistingFiles`; share `CheckAttachment`; move `completion` to
`internal/cmd/completion`.

Behavior: none.

### U11. Help as templates (N5). After U1, U2 and U4.

Topics become embedded `.tmpl` files under `cli/helptext/topics/`, rendered by `helptext.Render`
with functions for the field tables. Rename template actions to the `Locations` fields and delete
the alias methods. Behavior: none.

### U12. Test architecture (N9, N10, N11). Last.

First `internal/testfixture` (fixture files under `testdata`, writers that take `t`), then move
per-command tests out of `internal/cli` one noun per commit, then shrink `help.txtar` to routing
checks. Test and fuzz names stay the same (`go test -list . ./...` before and after).

### Order and dependencies

| Order | Unit | Waits for | Output changes |
|---|---|---|---|
| 1 | U1 deprecated forms | nothing | yes |
| 2 | U4 placeholders, removed keys (latex, store half) | nothing | yes |
| 3 | U7 F8a to F8d | nothing | no |
| 4 | U5 boilerplate | U1 | one line |
| 5 | U2 legacy directory | ports v2 | yes |
| 6 | U3 Markdown archives | ports v2 | yes |
| 7 | U4 email half, U6, U7 F2 and N2 | ports v2 | yes |
| 8 | U8 one decoder | U2 | yes |
| 9 | U9 store surface | ports v2, U2, U3, U8 | no |
| 10 | U10 adapter surfaces | ports v2 | no |
| 11 | U11 help templates | U1, U2, U4 | no |
| 12 | U12 tests | U1 to U11 | no |

Units 1 to 4 can run in parallel with ports v2. They change no production file in
`internal/billing`; U4 edits four `billing` tests (Appendix A4), which may need a rebase.

### What adding things costs, before and after

| Change | Places today | After the units |
|---|---|---|
| Command | package, `root.go` `AddCommand` and `commandGroups`, `Args` func, command-name strings in each error, help golden, `make docs` | package, `root.go`, `make docs` |
| Support-file flag | `StringVarP`, `MarkFlagFilename`, `supportFlags` entry, per command | one table entry, one call per command |
| Template placeholder | `buildTemplateValues`, helptext table, nothing checks they agree | both, and `TestPlaceholderDocsMatch` fails until they agree |
| Email placeholder | `emailtext.go`, `config` help, starter comment | one table |
| Config setting | `config.Config`, `settingsOf`, `billing.Settings`, starter comment, `config` help | schema struct, `settingsOf`, `billing.Settings`, starter template; `TestConfigKeysDocumented` checks help |
| Output format | `Renderer` is format-neutral in shape, but placeholder values are LaTeX-escaped inside `latex` | unchanged; see maintainer item 4 |

## 4. Needs the maintainer

1. **`init --force`.** Its only job was copying legacy files (`cmd/init/init.go:63`). The plan
   removes it with the directory, so `invox init --force` exits 2. The alternative is a hidden
   no-op flag for one release.
2. **Alternate customer keys.** `legal_company_name`, `billing.email`, `billing.contact_person`,
   `billing.email_greeting` and top-level `currency` are documented "Alternate supported paths"
   (`cli/helptext/reference.go:141-148`). They cost five schema keys and five `firstText`
   fallbacks (`invoice/models.go:129-153`). Removing them breaks existing `customers.yaml` files.
   Recommendation: keep.
3. **Strict email placeholders.** U4 fails an email whose subject or body has an unknown
   `{lower_snake}` token. A body that quotes such text literally would stop working. The
   alternative is a warning on stderr instead of an error.
4. **A second output format.** If one is planned (Typst, HTML), placeholder values should move to a
   format-neutral table with escaping done by each renderer. If not, leave them in `latex`.
5. **`--json` gaps.** `config paths` is a list command without `--json`, and `email` has none.
   quality-cli asks for `--json` on list commands. Adding them is new surface, not cleanup.
6. **In-flight working copies with `archive_replace_path`.** After U3 they fail strict decoding.
   The alternative is to ignore that one key for a release.
7. **Help goldens.** U12 makes `docs/cli` the only content golden for help pages, so a help
   regression shows up in `make docs`, not in `go test`. Keep the full `help.txtar` if local
   `go test` must catch it.
8. **Repo tooling.** `.agents/skills/quality-cli` is a second copy of `.claude/skills/quality-cli`
   (identical by `diff -rq`), tracked by `skills-lock.json`. Not code; delete one if only one
   harness reads them.

## Appendix A. Tests and goldens per removal

"Delete" means the test pins only removed behavior. "Edit" means it keeps a purpose and drops a
case or a fixture.

### A1. Deprecated forms (U1)

Testscript:

- `deprecated.txtar`: delete the file. Every case pins a deprecated form, including "A file named
  list or edit is archived with archive add", which only exists because `archive FILE` could
  shadow a subcommand.
- `grammar.txtar`: no change. It pins the current forms only (checked).
- Single-dash warnings, edit to the new error: `cobra_archive.txtar:26-28`,
  `cobra_build_email.txtar:24-26`, `cobra_customer_edit.txtar:14,44`, `cobra_flags.txtar:26,67`,
  `cobra_init.txtar:9-12`, `cobra_invoice.txtar:29-34`, `cobra_root.txtar:28,45`,
  `help.txtar:17-20` and `want-help-warning.txt`.
- `help.txtar:77` (customer config page) and `:156` (send page): delete those steps.

Go:

- Delete `TestNormalizeLongFlags` (`cli/root_test.go:10`), `TestSendAliasUsesEmailCommand`
  (`cli/cli_test.go`).
- Edit `TestCobraCommandUsageErrors` (the `-nmaes` case), `TestNewCmdNewParsing` (the `-s` and
  `--source` cases), `TestNewCmdAddParsing` (the `archive ARGS` run of each case),
  `TestNewCmdEditParsing` (the `config` case), `TestCustomerAndConfigHelpMatchesHelpCommand` (the
  `customer config` row), `TestBuildAndEmailHelpMatchesHelpCommand` (the `send` request).
- Add cases for each new error in `grammar.txtar`.

### A2. Legacy directory (U2)

Testscript: `config_paths.txtar` (the legacy fixture `home/.config/invoice-tool/`, the
`want-legacy-warning.txt` step, the `legacy` row), `help.txtar` (config, init, config paths
pages), `help_topics.txtar` (environment page lines 45, 80-81, 96, 109), `cobra_init.txtar` (the
`--force` steps).

Go, delete: `TestResolveDefaultPathsFallbackToLegacyConfigFiles`,
`TestResolveArchiveDirUsesLegacyConfigOverride`, `TestLegacyFallbackIsPerFileAndRecorded`,
`TestCopyLegacyFilesNeverReplacesAndIsIdempotent`, `TestCopyLegacyFilesFollowsSymlinks`,
`TestCopyLegacyFilesUsesPrivateModes` (with `store/legacy_copy_mode_unix_test.go`),
`TestLegacyFileWarningComesBeforeTheError`, `TestInitCopiesLegacyFiles`,
`TestLegacyWarningStopsAfterInit`, `TestSignalAtInitLegacyPrompt` (with
`cli/init_signal_test.go`; `init` no longer prompts), `TestHelpConfigNamesLegacyPathUnderConfigHome`.
Each pins only the legacy directory.

Go, edit: `TestConfigFilePrecedence`, `TestPathsReportsEachSource`, `TestConfigIsReadOncePerHost`,
`TestNewHostResolvesUserDirectories` (store); `TestConfigPathsReportsEachSource`,
`TestConfigPathsWithInvoxConfigDir`, `TestInitWritesToInvoxConfigDir` (cli). Each drops its legacy
layer or expectation.

### A3. Markdown archives (U3)

Testscript: `archive.txtar:46-47` (the `legacy.md` copy), the `-- legacy.md --` fixture at `:174`
and its `archive list` row at `:196`.

Go, delete: `TestEditArchivedMarkdownInvoiceAndRearchiveAsYAML`,
`TestArchiveInvoiceReplaceBacksUpMarkdownOriginalInSubdirectory`,
`TestCreateNewInvoiceFailsWhenArchiveContainsInvalidFrontMatter` (billing);
`TestArchiveEditMarkdownThenRearchiveReplacesOriginal` (cli);
`TestArchivedInvoiceIdentityReportsMarkdownFileLines`,
`TestArchivedMarkdownInvoiceUsesTheInvoiceDecoder` (store). Each pins front matter decoding or the
Markdown-to-YAML swap. If no YAML twin of the invalid-archive test exists, rewrite that one with a
broken `.yaml` instead of deleting it.

Go, edit (switch the fixture from `writeArchivedInvoiceMarkdown` to a YAML writer, or drop a
case): `TestCreateNewInvoicePrefillsDatesAndNumber`, `TestIncrementInvoiceNumberAdvancesCurrentInvoice`,
`TestArchiveInvoiceReturnsDuplicateInvoiceNumberError`, `TestNextInvoiceNumberReportsSkippedArchiveFiles`
(the "markdown archive in an old format is skipped" case) (billing);
`TestArchiveListPrintsArchivedInvoices`, `TestIncrementUpdatesInvoiceNumber`, `TestPortOrder` (the
`old.md` replace scenario) (cli); `TestYAMLLoadersRejectRecursiveAndExplosiveAliases` (the two
Markdown loaders) (store); `TestWalkVisitsInvoiceFilesBelowTheRootInOrder`, `TestTargetEdit`,
`TestBackupKeepsEveryVersion`, `TestIsInvoiceFile` (archive). Then delete both
`writeArchivedInvoiceMarkdown` helpers. `internal/archtest/ports_test.go:81` may keep `md` in its
extension ban.

### A4. Placeholders and removed keys (U4)

Go, delete: `TestRenderInvoiceMigratesLegacyStarterVATRow` (pins the migration).

Go, edit: `TestRenderInvoiceRejectsLegacyVATPlaceholders`,
`TestRenderInvoiceRejectsLegacyCityAndPostalCodePlaceholders` (expect `unknown placeholder`);
`TestLoadContextRejectsLegacyInvoiceAliases`, `TestCreateNewInvoiceRejectsLegacyDefaultKeys`,
`TestCreateNewInvoiceFromLastRejectsLegacyArchivedInvoiceKeys`,
`TestEditArchivedInvoiceRejectsLegacyKeys` (expect the unknown-key error);
`TestHelpDefaultsShowsInvoiceDefaultsDocumentation` (drop "Unsupported legacy keys:").

Testscript: `help.txtar` defaults page and `template` page; `render.txtar` gains the unknown
placeholder case; `email.txtar` gains the unknown subject placeholder case.
