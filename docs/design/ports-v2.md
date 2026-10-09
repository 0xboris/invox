# Billing ports, version 2

Design for the second narrowing of `internal/billing`'s ports and `Service`. Base: `236d558`.
Every `file:line` without another commit named refers to that commit. Round 1 is
`docs/design/ports-narrowing.md`. It kept every error order, and that kept the ports wide.

## Summary

The maintainer lifted the byte-identical constraint, dropped the legacy `invoice-tool`
directory, and stopped reading Markdown archived invoices. With those three changes billing
can hold only use cases, and the ports can carry only domain data.

- The six ports go from 36 methods to 24. `Invoices` has 4 (was 7), `Directory` 10 (14),
  `Archive` 5 (8), `Renderer` 3 (3), `Compiler` 1 (1) and `Mailer` 1 (3).
- `*billing.Service` goes from 21 exported methods to the 14 use cases of the spec.
- Three kinds of adapter data stop passing through billing. `Placement` left `Archive.Place`
  and went back into `Archive.Add`. `Template.FindAsset` is a store closure that billing
  carries to the renderer. `Draft.Discard` is a mailer closure that billing carries to the CLI.
  A new acceptance test fails on each of the three at `236d558` and passes on the prototype.
- Production code shrinks by 521 lines (540 added, 1,061 removed). `internal/billing` goes from
  1,959 to 1,747 lines and `internal/store` from 2,478 to 2,192.

Users see 20 output changes, listed under [Output changes](#output-changes). 19 were observed
in 28 differing scenarios, and 1 is inferred from the code. Five
testscript files change: `archive.txtar`, `cobra_init.txtar`, `config_paths.txtar`, `help.txtar` and
`help_topics.txtar`. Four `docs/cli` pages and four man pages change with them. Exit codes,
stdout and `--json` output change only where an invoice is now numbered, listed or matched
differently, or where a removed flag or directory was in use.

I applied the whole design to a scratch copy of `236d558` and compared it with the `236d558` binary.
`go build`, `go vet`, `gofmt -l`, `go mod tidy -diff`, golangci-lint v2.5.0 (0 issues), the
frozen target test (`-tags target -run TestTarget`), the new acceptance test and `TestScript`
pass on the prototype. The full race suite is not green there. The tests that fail are the ones
this document lists as removed or rewritten. A differential run of 82 scenarios gave 29
differences: the 28 scenarios behind the listed changes, plus the `.eml` timestamp in
`email-write`, which two runs of the base binary also differ on.

## The rule

Round 1's rule stays. Billing treats a path as an opaque reference, never builds or compares
one, and never touches the disk. Version 2 adds two rules:

1. **No round trips.** Billing may hand a value from one port to another, as `Checkout.Path`
   goes from `Archive` to `Invoices.Create`. It may not hand a port's own result back to that port.
   `Placement` (`ports.go:246`) did exactly that. `Archive.Place` built it from the archive's
   layout, and `Archive.Add` consumed it. Billing never read it.
2. **Port results are data.** A struct a port returns has no func fields. `Template.FindAsset`
   (`ports.go:124`, set at `store/store.go:154`, called at `render/latex/renderer.go:163,172`)
   and `Draft.Discard` (`ports.go:361`, called at `cmd/invoice/email/email.go:181-184`) were
   adapter behavior riding through billing to another layer.

Billing keeps the business rules: statuses and their transitions, numbering, the duplicate-number
rule, due dates, recipients, subjects and EPC. Adapters keep files, names, directories, temporary
directories, opening programs and how a document is copied.

## Maintainer decisions applied

| Decision | What it removes from the ports |
|---|---|
| Output may change, every change listed | The order-preserving methods: `Destination`, `Protects`, `Dir`, `Place`, `Head`, `Check`, `CheckAttachment`. |
| Drop the legacy `invoice-tool` directory | `Directory.LegacyFiles`, `CopyLegacy`, `LegacyFilesUsed`, `Service.LegacyFiles`, `CopyLegacyFiles`, `LegacyFilesUsed`, `Source` value `SourceLegacy`, `Locations.LegacyDir`. |
| Drop deprecated command forms (#106) | Nothing. `archive FILE`, hidden `customer config`, the `send` alias, `new -s/--source` and the single-dash normaliser call the same `Service` methods as their current forms (`new.go:99` binds `--source` to the same `DefaultsPath`). The cleanup audit removes them from the CLI. |
| Optional: legacy placeholders, removed-key messages | Nothing. They live in `render/latex/template.go:36` and `store/schema.go:116`, behind no port. Not taken. |
| Stop reading Markdown archived invoices, report skipped ones | `ArchivedHead`, the Markdown branch of `Create`'s source, `Target.Edit`'s `.md`-to-`.yaml` working copy, and `_invox.archive_replace_path`, which only that working copy wrote. Adds `Unread` to the archive walk results. |

## Disposition of every method

### Invoices: 7 to 4

| Method | Decision | Reason |
|---|---|---|
| `Load` | Keep | Spec name. Reads YAML only. |
| `ArchivedHead` | Merge into `Load` | Its only difference was Markdown front matter (`store/invoices.go:27-42`), which is gone. `EditArchived` and `New --from-last` call `Load` and drop unknown-key errors with `lenient`, as before. |
| `Head` | Merge into `Load` | `Head` (`ports.go:28`) restates `invoice.Invoice` fields as strings, plus the `_invox` link (already `invoice.ArchiveLink`) and a `HeaderShape`. Callers read `inv.Header` instead. `Increment` keeps only decode errors of the three fields it reads (`customer_id`, `invoice.number`, `invoice.issue_date`), so an unrelated bad value still does not block it. The `invoice` shape checks move to the writers. `Create` refuses a non-mapping `invoice` and `Archive.Add`'s rewrite already did (`store/yamldoc.go:75-84`). |
| `Drafts` | Keep, returns `[]invoice.Invoice` | Numbering must see unarchived drafts' numbers before `Create` runs. No other operation returns them. |
| `Destination` | Merge into `Create` | `Create` names `<number>.yaml` in `CreateOptions.Dir` when the path is `""` and runs the dir, exists and archived-file checks itself. |
| `Create` | Keep, returns the path written | Spec name. Gains the checks above. |
| `Update` | Keep | Spec name. |

### Directory: 14 to 10

| Method | Decision | Reason |
|---|---|---|
| `Customer`, `Customers`, `Issuer`, `Defaults`, `Template`, `Templates`, `EditablePath`, `Init` | Keep | Spec names. `Template` loses `FindAsset`. |
| `Paths(start)` | Keep as `Paths()` | The spec's signature. `store` already has `Getwd`, and the CLI passed the same value (`cmd/config/paths/paths.go:69`). |
| `Locate` | Keep | It is the one operation that answers "which file is f for this run". `Validate` and `New` name issuer messages after it (`validate.go:91`, `new.go:68`). `ListCustomers` reports it, and `ListTemplates` marks the default with it. Folding it into each read would add a path result to four methods. |
| `LegacyFiles`, `CopyLegacy`, `LegacyFilesUsed` | Delete | Legacy directory dropped. |
| `Locations` | Move to `factory`, as `cmdutil.Factory.Locations` | Help texts are the only reader after the legacy warning goes (`cli/root.go:90,106`). The values come from the environment, not from a use case. The type moves to `helptext.Locations`, next to its reader. |

### Archive: 8 to 5

| Method | Decision | Reason |
|---|---|---|
| `Entries` | Keep, sorted by `Filename`, returns `Unread` | Spec name. One order for listing and numbering removes billing's sort (`billing/archive.go:155-163`). |
| `Duplicate` | Keep, takes `invoice.Invoice`, returns `Unread` | `Validate` needs the check without archiving, and excluding a working copy's own file needs the archive's name resolution. The rule and its error stay in billing (`numberUnique`). |
| `Add` | Keep, takes the invoice instead of a `Placement` | Spec name. It absorbs `Place` and the "archive directory is unavailable" check, gets its clock from `factory`, and rewrites the file before the dry-run return, so a dry run checks the `invoice` shape too. |
| `Checkout` | Keep | Spec name. A Markdown ref is an error. The working copy keeps its archived name. |
| `Source` | Keep, returns `Unread` | `email` from a PDF. Finding the YAML next to the PDF or by base name in the archive is path work. |
| `Dir` | Delete | `ListArchive` gets the directory from `Unread.Dir`. `Archive` gets "unavailable" from `Add`. |
| `Place` | Merge into `Add` | Rule 1. |
| `Protects` | Move to `store.Store.Protected`, wired by `factory` | Refusing to overwrite an archived file is a check on the file `Create` writes. `archive.Archive.Protects` stays as the implementation. `factory` hands it to `store`, as it hands `store.Rewrite` to `archive` (`factory/factory.go:54`). |

### Renderer 3, Compiler 1: unchanged names

`Render`, `Write`, `Build` and `Compile` stay. `Render` is the check that dry runs need, and `Write`
and `Build` are the two outputs. The one change is that `Template` carries no closure.
`latex.Renderer{FindAsset: host.FindAsset}` gets asset lookup from `factory`. `Build` still
takes `s.Compiler` from billing. See [Needs the maintainer](#needs-the-maintainer) item 4.

### Mailer: 3 to 1

| Method | Decision | Reason |
|---|---|---|
| `Draft` | Keep as `Draft(ctx, m, dryRun) (string, error)` | The spec's shape. It checks the attachment and, for a kept draft, the output, then writes and opens the draft unless `dryRun`. |
| `Check`, `CheckAttachment` | Merge into `Draft` | They were `Draft`'s own checks, run early (`billing/email.go:66,109`). |
| (`Draft.Discard`) | Delete | Opening the `.eml` moves into `email.Mailer`, which `factory` gives the opener. The CLI no longer branches on the draft kind (`cmd/invoice/email/email.go:179-187`). `applemail` already opened its own draft. |

### Service: 21 to 14

| Method | Decision | Reason |
|---|---|---|
| The 14 use cases | Keep | `ListTemplates` becomes `ListTemplates(withDefault bool)`, and `Paths(start)` becomes `Paths()`. |
| `CheckNumberUnique` | Unexport into `Validate` | It had one production caller (`validate.go:26`). |
| `NextNumber` | Unexport | Shared by `New` and `Increment` only. A number preview is `new -n` or `increment -n`. |
| `DefaultTemplate` | Merge into `ListTemplates(true)` | Only `template list --json` reads it (`cmd/template/list/list.go:113`). The flag keeps the plain listing free of the working-directory search, which `TestTemplateListDoesNotNeedTheWorkingDirectory` pins. |
| `Locations` | Move to `cmdutil.Factory.Locations` | Not a use case. |
| `LegacyFiles`, `CopyLegacyFiles`, `LegacyFilesUsed` | Delete | Legacy directory dropped. |

## Where things belong

- **Legacy directory.** Gone. `store.Host.findInConfigDir` reads the config directory only.
  `LegacyConfigDir`, `legacyUse`, `LegacyFilesToCopy` and `CopyLegacyFiles` are deleted
  (`store/host.go:122-191`, `store/init.go:63-134`), along with `cli.warnLegacyFiles`
  (`cli/cli.go:62-84`) and the copy step and `--force` flag of `init` (`cmd/init/init.go:63,90-122`).
- **Help-text locations.** `factory` builds `helptext.Locations` from `store.Host` and puts it
  in `cmdutil.Factory.Locations`. `helptext` no longer imports `billing`.
- **Next-number preview.** Unexported `billing.nextNumber`. Users preview with `new -n` and
  `increment -n`. Nothing else asks for it.
- **Duplicate-number check.** The rule and `DuplicateInvoiceNumberError` stay in billing
  (`numberUnique`, called by `Validate` and `Archive`). Which archived file counts as "the same
  file" is `Archive.Duplicate`'s answer.
- **Markdown invoices.** `archive.Store.walk` collects `.md` and `.markdown` files instead of
  visiting them. `Entries`, `Duplicate` and `Source` return them as `Unread{Dir, Markdown}`.
  Billing puts `Unread` into `NewResult`, `IncrementResult`, `ValidateResult`, `ArchiveResult`
  (also `BuildResult.Archived`), `ArchiveList` and `EmailResult`. The CLI words it once per
  command in `shared.WarnUnread`. Billing does not word it. The existing skipped-files warning
  (`shared.WarnSkippedArchiveFiles`) is about numbering patterns and stays separate. Its data
  is per customer, and `Unread` is per archive.

## Final interfaces

```go
type Invoices interface {
	Load(path string) (invoice.Invoice, error)
	Drafts(workDir, output string) []invoice.Invoice
	Create(path, from string, inv invoice.Invoice, opts CreateOptions) (string, error)
	Update(path string, change func(*invoice.Invoice) error) error
}

type CreateOptions struct {
	Dir       string // holds the new invoice, named after its number, when path is ""
	Overwrite bool   // never an archived file: *ArchivedOutputError
	DryRun    bool
	Check     Check
}

type Directory interface {
	Locate(f File) (string, error)
	Customer(id string) (invoice.Customer, error)
	Customers() (CustomerTable, error)
	Issuer() (invoice.Issuer, error)
	Defaults() (string, error)
	Template(ref string) (Template, error)
	Templates() ([]Template, string, error)
	Paths() ([]PathReport, error)
	EditablePath(f File) (string, error)
	Init() (string, []InitFile, error)
}

type Template struct{ Name, Path string }

type Unread struct {
	Dir      string   // the archive directory, "" when there is none
	Markdown []string // archived invoices stored as Markdown, no longer read
}

type Archive interface {
	Entries() ([]ArchiveEntry, Unread, error)
	Duplicate(src string, inv invoice.Invoice) (string, Unread, error)
	Add(src string, inv invoice.Invoice, opts AddOptions) (ArchiveResult, error)
	Checkout(ref, workDir string) (Checkout, error)
	Source(pdf string) (string, Unread, error)
}

type AddOptions struct {
	Replace bool
	DryRun  bool
	Change  func(*invoice.Invoice) error // billing's rule: status archived, link cleared
}

type Renderer interface {
	Render(t Template, inv *invoice.Context, epc EPC) (string, error)
	Write(t Template, source, path string) error
	Build(ctx context.Context, c Compiler, t Template, source, output string) error
}

type Compiler interface {
	Compile(ctx context.Context, sourcePath string) (string, error)
}

type Mailer interface {
	Draft(ctx context.Context, m Message, dryRun bool) (string, error)
}
```

Removed types: `Head`, `HeaderShape`, `Placement`, `Draft`, `Locations` (moved to `helptext`),
`SourceLegacy`. `ArchiveResult` gains `Unread`, and `invoice.ArchiveLink` loses
`ArchiveReplacePath`. The `Service` struct is unchanged: `Invoices`, `Directory`, `Archives`,
`Renderer`, `Compiler`, `Mailer`, `Settings func() (Settings, error)`, `Now func() time.Time`.

### Method counts

| Port | Spec | 236d558 | v2 | Above the spec |
|---|---|---|---|---|
| `Invoices` | 3 | 7 | 4: `Create Drafts Load Update` | `Drafts` |
| `Directory` | 9 | 14 | 10: `Customer Customers Defaults EditablePath Init Issuer Locate Paths Template Templates` | `Locate` |
| `Archive` | 3 | 8 | 5: `Add Checkout Duplicate Entries Source` | `Duplicate`, `Source` |
| `Renderer` | 1 | 3 | 3: `Build Render Write` | `Write`, `Build` |
| `Compiler` | 1 | 1 | 1: `Compile` | |
| `Mailer` | 1 | 3 | 1: `Draft` | |
| total | 18 | 36 | 24 | 6 |

`*billing.Service`: `Archive Build DraftEmail EditArchived EditablePath Increment Init
ListArchive ListCustomers ListTemplates New Paths Render Validate`. Every name that
`target_test.go:122-131` requires stays, and `TestTarget*` passes on the prototype.

## Output changes

Grouped by cause. Items 1 to 19 account for the 28 differing scenarios of
[Appendix A](#appendix-a-differential-results), and item 20 is inferred. "Golden" names the testscript file whose expectation changes. "Go test" names the
test whose expectation changes. A change with neither was untested at `236d558` and gets a
pinning test in the commit that makes it.

### Legacy directory (commit 1)

1. **Support files and `config.yaml` in `invoice-tool` are no longer read.** `validate` with
   only `invoice-tool/issuer.yaml` now fails with "issuer file not found …" (exit 2), and
   `config paths` reports `config  none` instead of `…/invoice-tool/config.yaml  legacy`. Golden:
   `config_paths.txtar` (`want-mixed.tsv` row `issuer`). Go: `TestConfigPathsReportsEachSource`,
   `TestPathsReportsEachSource`, `TestConfigFilePrecedence`.
2. **The warning `using X from deprecated config directory …; run 'invox init' to copy …` is
   gone.** Golden: `config_paths.txtar` (`want-legacy-warning.txt` deleted).
3. **`init` no longer asks to copy legacy files**, and no longer exits 2 without a terminal when
   they exist. It writes the starter files. No golden. Go: `TestInitCopiesLegacyFiles` deleted.
4. **`init --force` is an unknown flag** (exit 2), and `init --forse` no longer suggests
   `--force`. Golden: `cobra_init.txtar` (`want-misspelt.txt`, and the `-force` block becomes a
   plain `init`). Go: `TestNewCmdInitParsing`.
5. **Help.** `config --help` prints `default: …/config.yaml` instead of `preferred:` plus
   `legacy fallback:`. `init --help` loses three Behavior lines, the Flags block and
   `$ invox init --force`. `help environment` loses the "Legacy config directory" section, item 4
   of "Config file" and "then in the legacy directory". The `INVOX_CONFIG_DIR` text is
   rewrapped, and `config paths --help` loses the `legacy` source row. Golden: `help.txtar`,
   `help_topics.txtar`. Docs: `invox_config.md`, `invox_config_paths.md`,
   `invox_help_environment.md`, `invox_init.md` and their four man pages. Go:
   `TestHelpConfigNamesLegacyPathUnderConfigHome`, `TestInitHelpMatchesHelpCommand`.

### Markdown archived invoices (commit 3)

6. **`archive list` leaves out `.md` invoices** and prints `warning: N Markdown invoice(s) in
   <archive dir> is/are no longer read; convert it/them to .yaml to include it/them` on stderr.
   `--json` gets the same warning on stderr and no `.md` entries on stdout. Golden:
   `archive.txtar` (`want-list.tsv` loses `legacy.md`, new `want-markdown-warning.txt`). Go:
   `TestArchiveListPrintsArchivedInvoices`.
7. **Numbering ignores `.md` invoices.** `new` and `increment` can give a lower number than
   before (scenario `md-new`: 016 becomes 010). They print the warning first. Go:
   `TestCreateNewInvoicePrefillsDatesAndNumber`, `TestIncrementInvoiceNumberAdvancesCurrentInvoice`,
   `TestIncrementUpdatesInvoiceNumber`, `TestNewDoesNotWarnWhenNoArchivedInvoiceOfTheCustomerIsSkipped`.
8. **`new --from-last` no longer starts from a `.md` invoice.** With only `.md` invoices it
   fails with "no archived invoice found for customer_id …".
9. **The duplicate check ignores `.md` invoices.** `validate` drops the duplicate warning, and
   `archive add` archives where it refused before. Both print the Markdown warning. Go:
   `TestArchiveInvoiceReturnsDuplicateInvoiceNumberError` (fixture to YAML).
10. **`archive add`, `build --archive` and `email` from a PDF print the Markdown warning** when
    their archive walk skipped any.
11. **`archive edit X.md` fails** with "… is a Markdown invoice, which invox no longer reads;
    convert it to .yaml" (exit 1). Go: `TestEditArchivedMarkdownInvoiceAndRearchiveAsYAML`,
    `TestArchiveEditMarkdownThenRearchiveReplacesOriginal` deleted.
12. **Re-archiving a working copy with `_invox.archive_replace_path` no longer replaces or backs
    up that file.** Only `archive edit` of a `.md` invoice wrote the key. `archive add`
    ignores it, and `validate` reports it as an unknown key in `_invox`. Scenarios
    `rearchive-dry-run`, `rearchive-yes`, `replace-resolve-no-number` and
    `replace-resolve-with-number` (the last two archive instead of failing with "must stay
    within"). Go: `TestPortOrder` cases `replace-resolve-*` and `rearchive-yes`,
    `TestForgedReplacePathDoesNotExemptDuplicateNumber`,
    `TestArchiveInvoiceReplaceBacksUpMarkdownOriginalInSubdirectory`.

### Reading the invoice through `Load` (commit 4)

13. **`archive add` and `archive edit` with `invoice: null` or `invoice: ~`** say "missing
    `invoice` mapping" instead of "`invoice` must be a mapping". Go:
    `TestArchiveRefusesInvoiceKeyThatIsNotAMapping` subtests `add_null`, `add_tilde`,
    `edit_null`.
14. **`archive add` with a scalar `invoice:`** says "inv.yaml:2: invoice must be a mapping, got
    an integer", the decoder's message with a line number. Go: same test, `add_scalar`.
15. **`increment` with a malformed `issue_date`** gains the line number:
    "invoice.yaml:4: invoice.issue_date: expected YYYY-MM-DD, got `2026-13-45`".
16. **`new` checks the output file last.** An existing or directory `-o`, or default
    `<number>.yaml`, now loses to an unknown customer, a missing `payment` mapping, a bad
    `due_days` and the numbering errors. Scenarios `new-default-exists-and-bad-due`,
    `new-output-exists-and-bad-due`, `new-output-exists-unknown-customer`. Go: `TestPortOrder`
    case `new-default-exists-and-bad-due`. The messages are unchanged.

### Archive placement inside `Add` (commit 5)

17. **The archive directory is read after the status check.** A broken `config.yaml` or an
    unusable `archive.dir` now loses to "invoice.status must be `built` …". Scenario
    `dir-error-before-status`.
18. **The duplicate number wins over "already exists" and "is already in the archive
    directory".** Scenario `exists-vs-duplicate`. Go: `TestPortOrder` case `exists-vs-duplicate`.
19. **The duplicate number wins over an aliased `invoice` mapping.** Scenario
    `archive-aliased-header-and-duplicate`. `archive add -n` of an aliased mapping still fails
    with "`invoice` must be a mapping", because `Add` rewrites before the dry-run return.

### Mailer (commit 7)

20. **`email` checks the PDF inside `Draft`**, after the recipient and the subject. The
    message is unchanged ("read X: stat X: …"). No scenario reached a case where a recipient or
    subject error now wins, because validation catches a missing customer email first.
    Inferred from the code, not observed.

### Error orders that round 1 protected

Of the 14 orders in `internal/cli/port_order_test.go`, five change: `replace-resolve-no-number`,
`replace-resolve-with-number` and `rearchive-yes` (item 12), `exists-vs-duplicate` (18) and
`new-default-exists-and-bad-due` (16). `status-before-resolve`, `target-dir-vs-duplicate`,
`already-in-archive`, `archive-comments-and-empty-link`, `new-output-in-other-dir-drafts` and
the four `email-*` cases pass unchanged on the prototype.

## Acceptance checks

`internal/archtest/ports_test.go` keeps its two checks with new lists. It also gains one test,
shown in full in [Appendix C](#appendix-c-acceptance-test):

1. **`TestPortMethodSets`.** The exact lists in [Method counts](#method-counts) and the 14 use
   cases. Each commit below updates the lists it changes.
2. **`TestBillingHandlesNoPaths`.** Unchanged: no `path` or `path/filepath` import and no
   file-extension literal in non-test billing files.
3. **`TestPortsCarryNoAdapterData`** (new). For each port, it collects the billing struct types
   in its method results and in its parameters, through pointers and slices. A type in both
   sets of the same port fails. So does a func-typed field in any struct reachable from a port
   result. On `236d558` it reports exactly the three leaks: `Archive both returns and takes
   billing.Placement`, `Directory.Template(s) returns Template.FindAsset`, and `Mailer.Draft
   returns Draft.Discard`. It passes on the prototype.

The check is cheap (go/types, under 2 s) and hard to game: carrying a closure through a result
needs a func field, and a round trip needs the same type on both sides. It does not catch an
adapter value encoded as a `string` that comes back to its port. `Renderer.Render`'s source,
which goes back into `Write` and `Build`, is that case, and it is accepted under
[Tradeoffs](#tradeoffs-accepted).

The depguard rule `use-cases-no-paths` and the frozen target test stay as they are. depguard
needs no new rule: `cmdutil` and `factory` importing `helptext` are allowed edges, and
golangci-lint reports 0 issues on the prototype.

## Implementation plan

Nine commits. Each passes `go build ./...`, `go vet ./...`, `gofmt -l .`, `go mod tidy -diff`,
`go test -race ./...`, `make lint` and `go test -tags target ./internal/archtest`. Goldens change
only in the commit that causes the change, regenerated with `go test ./cmd/invox -run TestScript
-update` and reviewed. Docs change with `go run ./internal/docs/gen`. Every deleted test gets a
line in `docs/design/target-removed-tests.md` in the same commit. The prototype patch
([Appendix E](#appendix-e-prototype-patch)) is the end state, and each commit is a slice of it.
Subtract first, then reshape.

1. **Drop the legacy directory.** Store: `host.go` legacy lookup and record, `init.go` copy,
   `store.go` three methods, `SourceLegacy`. Billing: three `Directory` and three `Service`
   methods, `Locations.LegacyDir`. CLI: `warnLegacyFiles`, `init` copy and `--force`, help lines,
   `LegacyConfigFile`. Goldens: `config_paths.txtar`, `cobra_init.txtar` (script edits from
   [Appendix D](#appendix-d-testscript-changes)), `help.txtar`, `help_topics.txtar`. Docs: 4 pages
   and 4 man pages. Output items 1 to 5. Counts: `Directory` 14 to 11, `Service` 21 to 18.
2. **Move `Locations` out of billing.** `helptext.Locations`, `cmdutil.Factory.Locations` and the
   `factory` builder. No output change. `Directory` 11 to 10, `Service` 18 to 17.
3. **Stop reading Markdown, report it.** `archive.Store.walk` returns the Markdown files.
   `List`, `Entries`, `Duplicate` and `Source` return `Unread`. `Find` refuses `.md`, and
   `Target.Edit` goes. `store`: `markdownFrontMatter`, `loadArchivedInvoiceDocument` and the
   Markdown branch of `loadSourceDocument` go. `invoice.ArchiveLink.ArchiveReplacePath` and its
   schema key go. Results gain `Unread`, and `shared.WarnUnread` prints it. Golden:
   `archive.txtar`. Output items 6 to 12. Billing fixtures written as Markdown become YAML.
4. **Read invoices with `Load`.** Delete `Head`, `ArchivedHead`, `HeaderShape` and `Destination`.
   `Drafts` returns `[]invoice.Invoice`. `Create` gets `Dir`, returns the path, runs the
   destination checks, refuses a non-mapping `invoice` and asks `Store.Protected`.
   `Archive.Place` and `Duplicate` take `invoice.Invoice`. Billing loses
   `refuseArchivedOverwrite` and `requireHeader`. `Increment` keeps the decode errors of its
   three fields (`keepDecodeErrors` in `errors.go`). Output items 13 to 16. `Invoices` 7 to 4.
5. **Placement inside `Add` (riskiest).** `Add(src, inv, opts)` does placement, the
   unavailable check, the rewrite before the dry run, and backups stamped by
   `archive.Archive.Now`. Delete `Place`, `Dir`, `Placement`, `AddOptions.Now`, and `Protects`
   from the port. Output items 17 to 19. `Archive` 8 to 5.
6. **`Template` without `FindAsset`.** `latex.Renderer{FindAsset}`, wired from
   `store.Host.FindAsset`. No output change.
7. **One `Mailer.Draft`.** `Draft(ctx, m, dryRun) (string, error)`. `email.Mailer{Open}` is wired
   by `factory` with a closure over `f.Opener`. Delete `Check`, `CheckAttachment` and `Draft`, and
   delete the open-and-discard branch in `emailRun`. Output item 20. `Mailer` 3 to 1.
8. **Service holds only use cases.** Unexport `NextNumber` and `CheckNumberUnique`, fold
   `DefaultTemplate` into `ListTemplates(withDefault)`, and change `Paths()`. No output change.
   `Service` 17 to 14.
9. **Lock it in.** Add `TestPortsCarryNoAdapterData`. Add a progress row to
   `docs/design/target-progress.md`. Update `CLAUDE.md`'s Layout: `store` no longer owns the
   legacy directory or Markdown front matter, and `archive` reports Markdown files.

**Riskiest step: commit 5, with commit 4.** Moving the `invoice` shape checks from billing to the
writers opened a data-loss path in the prototype. With `Head` gone, `archive edit` of an
archived invoice whose `invoice:` is an alias reached `Create`. There `getOrCreateMappingNode`
(`store/yamldoc.go:86-102`) turned the alias into an empty mapping, and the edit succeeded with
the header erased. `TestArchiveRefusesInvoiceKeyThatIsNotAMapping/edit_alias` caught it, and the
shape guard in `Create` fixed it. Keep that test, and add `archive add -n` with an aliased
mapping to it (output item 19).

**`verify-target.sh`.** Its step "goldens, docs/cli or share/man differ from the base"
(`scripts/verify-target.sh:44-46`) fails from commit 1 on, by design. The script is frozen.
See [Needs the maintainer](#needs-the-maintainer) item 1.

## Tests

### Deleted: they pin only removed behavior

| Test | Reason |
|---|---|
| `TestLegacyFallbackIsPerFileAndRecorded` (store) | Legacy lookup and its record. |
| `TestCopyLegacyFilesNeverReplacesAndIsIdempotent` (store) | Legacy copy. |
| `TestCopyLegacyFilesFollowsSymlinks` (store) | Legacy copy. |
| `TestCopyLegacyFilesUsesPrivateModes` (store, `legacy_copy_mode_unix_test.go`) | Legacy copy. |
| `TestResolveArchiveDirUsesLegacyConfigOverride` (store) | Legacy `config.yaml`. |
| `TestResolveDefaultPathsFallbackToLegacyConfigFiles` (store) | Legacy support files. |
| `TestLegacyFileWarningComesBeforeTheError` (cli) | Legacy warning. |
| `TestInitCopiesLegacyFiles` (cli) | `init` legacy copy. |
| `TestLegacyWarningStopsAfterInit` (cli) | Legacy warning. |
| `TestHelpConfigNamesLegacyPathUnderConfigHome` (cli) | `legacy fallback:` help line. |
| `TestSignalAtInitLegacyPrompt` (cli) | The legacy copy prompt. |
| `TestArchivedMarkdownInvoiceUsesTheInvoiceDecoder` (store) | Markdown front matter. |
| `TestArchivedInvoiceIdentityReportsMarkdownFileLines` (store) | Markdown front matter. |
| `TestCreateNewInvoiceFailsWhenArchiveContainsInvalidFrontMatter` (billing) | Markdown front matter. A new test pins that `new` warns instead. |
| `TestEditArchivedMarkdownInvoiceAndRearchiveAsYAML` (billing) | Markdown working copy. |
| `TestArchiveInvoiceReplaceBacksUpMarkdownOriginalInSubdirectory` (billing) | `archive_replace_path`. |
| `TestArchiveEditMarkdownThenRearchiveReplacesOriginal` (cli) | Markdown working copy. |
| `TestForgedReplacePathDoesNotExemptDuplicateNumber` (cli) | `archive_replace_path`. |
| `TestTargetEdit` (archive) | `Target.Edit` existed for `.md` to `.yaml`. |

Subtests deleted: `TestConfigFilePrecedence/legacy_dir_when_default_has_no_config.yaml`, the
`loadArchivedInvoiceDocument` and `archivedInvoiceIdentity` Markdown rows of
`TestYAMLLoadersRejectRecursiveAndExplosiveAliases`, and the `LegacyConfigDir()` row of
`TestNewHostResolvesUserDirectories`.

### Rewritten under the same name

- Port signatures: `TestAddWritesTheChangedInvoiceOnce`, `TestAddMovesNothingWhenTheRewriteFails`
  (archive). `TestArchiveHistoryIsIgnoredByNumberingAndDuplicateCheck`,
  `TestCheckArchivedNumberUniqueIgnoresTheArchivedOriginal` and
  `TestNextInvoiceNumberReportsSkippedArchiveFiles` (billing) call `New`, `Increment` or
  `Validate` instead of the unexported methods. The `Source` tests in `archive` take the extra
  result.
- Opening moves into the mailer, so the six `TestCreateInvoiceEmailDraft*` tests (billing) and
  `TestEmailRunResolvesPathsAgainstGetwd` (cmd/invoice/email) register the opener. The
  `draftRecorder` in `paths_test.go` takes the new signature, and
  `TestBuildInvoicePDFCompilesTheRenderedTeXAndCopiesThePDF` (latex) sets `FindAsset`.
- Expectations listed under [Output changes](#output-changes): `TestPortOrder` (5 cases),
  `TestArchiveRefusesInvoiceKeyThatIsNotAMapping` (4 subtests), `TestConfigPathsReportsEachSource`,
  `TestPathsReportsEachSource`, `TestNewCmdInitParsing`, `TestInitHelpMatchesHelpCommand`,
  `TestIsInvoiceFile`, `TestWalkVisitsInvoiceFilesBelowTheRootInOrder`, and the four numbering
  tests of item 7.
- New pinning tests: the Markdown warning for `new`, `increment`, `validate`, `archive add` and
  `email` from a PDF. Also `archive edit X.md`, `increment` with an unrelated bad value (still
  succeeds), and the order changes 17 and 19.

## Needs the maintainer

1. **`verify-target.sh`'s "output must not change" step.** It is frozen and fails on every
   commit here. For: retire the step, or move its base to the commit that lands this design, so
   the script keeps checking the rest (tests, skips, fixtures). Against: the frozen spec says
   to record a blocker and work around it. Designed version: commit 1 records it under Blockers
   in `target-progress.md`, and CI does not run the script.
2. **`init --force`.** Removing the flag removes a capability by the brief's own definition, but
   its only job was the legacy copy (`cmd/init/init.go:63`). For: dead flag, one less usage line.
   Against: scripts that pass `--force` break with exit 2. Designed version: removed. A hidden
   no-op flag would keep old scripts working at the cost of one line.
3. **`_invox.archive_replace_path`.** Working copies made from `.md` invoices before the upgrade
   carry it. Designed version: `archive add` ignores it and the old `.md` stays in the archive,
   where the Markdown warning names it, and `validate` reports the key as unknown. The
   alternative, removing the `.md` on re-archive, keeps Markdown code alive.
4. **`Service.Compiler`.** Billing only passes it to `Renderer.Build`. Injecting the compiler into
   `latex.Renderer` in `factory` would be cleaner, but it leaves `Service` without a `Compiler`
   field, which `TestTargetService` (`target_test.go:196-216`) requires. Designed version: keep
   the field. Billing choosing the compiler is a real decision, and the cost is one argument.
5. **Help locations no longer come from `Service`.** The spec says `helptext` fills paths from
   `Service.Paths()` (`docs/design/target/README.md:48`). The help pages show default
   locations, which `Paths` does not report, and need no config read. Designed version:
   `cmdutil.Factory.Locations`.

Dropping Markdown removed about 110 lines of production code, counted by function at
`236d558`: front matter parsing 41 (`store/yamldoc.go:141-155,180-205`), the Markdown source
branch 22 (`store/invoices.go`), `Target.Edit` 24 (`archive/archive.go:176-197`), `ArchivedHead` 16
and the replace-path reader 8. It also removed 8 tests.

## Tradeoffs accepted

- We accept `Renderer.Render`'s source string going back into `Write` and `Build` in exchange
  for keeping the dry-run check (`Render`) separate from the two outputs. Folding would add a
  `dryRun` flag to both output methods. The string is the use case's product, not the
  adapter's layout.
- We accept the `store` to `archive` wiring in both directions (`Rewrite` one way, `Protected`
  the other) in exchange for keeping the archived-file policy next to the write it guards.
  Both are plain funcs set in one place, `factory.New`.
- We accept `Increment` naming the three fields it reads in exchange for one `Load` instead of a
  second lenient reader.
- We accept a boolean on `ListTemplates` in exchange for one use case instead of two. Only the
  `--json` caller sets it.
- We accept `Locate` beside `EditablePath`, which overlap for support files, because `Locate`
  is a read and `EditablePath(ConfigFile)` creates a file.

## Alternatives considered

- **`Unread` as a per-run record on the archive adapter**, read once by `cli.Main`, as
  `LegacyFilesUsed` was. Smaller signatures, but hidden mutable state shared between calls,
  and the maintainer asked for data in results.
- **Billing computes duplicates from `Entries`.** One fewer port method, but excluding a working
  copy's own file means comparing cleaned relative names in billing, which the path rule forbids.
- **`Head` kept as a lenient identity read.** One more method and a type that restates
  `invoice.Invoice` as strings. `Load` with filtered decode errors gives the same messages,
  except the line number in item 15.
- **`Protects` kept on `Archive` and called by billing before `Create`.** Billing would need
  the final path before `Create` names it, so the destination check would have to come back to
  billing.

## How this design was checked

- **Prototype.** `git archive 236d558` into a scratch directory. The design was applied there:
  45 production files, +540 and −1,061 lines ([Appendix E](#appendix-e-prototype-patch)). Checks
  passed: build, vet, gofmt, `go mod tidy -diff`, golangci-lint v2.5.0 (CI pins v2.14.0),
  `go test -tags target -run TestTarget ./internal/archtest`, the full `internal/archtest`
  package with the v2 `ports_test.go`, `go run ./internal/docs/gen`, and `TestScript` after
  `-update` and the script edits in [Appendix D](#appendix-d-testscript-changes). The full suite
  fails only in the tests listed under [Tests](#tests). The deleted ones were removed in the
  prototype to reach the rest, and the rewrites are not done there.
- **Differential.** Round 1's script, extended with 32 scenarios for Markdown, legacy, header
  shapes, increment, templates and help ([Appendix B](#appendix-b-differential-script)). It
  ran 82 scenarios against `236d558` and the prototype and compared exit code, stdout, stderr,
  and every file's path, mode and SHA-256. 53 matched, and the 29 that differ are in
  [Appendix A](#appendix-a-differential-results). `email-write` also differs between two
  base runs.
- **Acceptance test.** It fails on `236d558` with the three leaks and the old method sets, and
  passes on the prototype.

## Open questions

1. Should the Markdown warning name the files, as the numbering warning does (first five, then
   "and N more")? The data has the paths. The designed version prints only the count and the
   directory, per the maintainer's example.
2. `archive list` prints the archive directory absolute in the warning because the command has
   no working directory (`list.go` passes `""`), the same as its empty-list hint. Should it gain
   `Getwd` to shorten the path?
3. `Renderer` stays at three methods. Should a later round fold `Render` into the outputs with a
   `dryRun` flag, to remove the source round trip?

## Appendix A: differential results

The 29 scenarios of [Appendix B](#appendix-b-differential-script) whose output differs between
`236d558` and the prototype. The other 53 match in exit code, stdout, stderr and every file's
path, mode and SHA-256. Paths are relative to the scenario directory, `$W` where the program
printed it absolute. Lines are joined with ` / ` and cut at about 170 characters.
`email-write` differs in the `.eml` bytes only, and two runs of the base binary differ the same
way (the `Date` header and the MIME boundary). `help-environment` is item 5.

| Scenario | 236d558 | v2 prototype |
|---|---|---|
| `dir-error-before-status` | exit 1; stderr `error: home/.config/invox/config.yaml: yaml: line 1: did not find expected node content / Run 'invox config' to open and fix the config file.` | exit 1; stderr `error: invoice.yaml: invoice.status must be \`built\` before archiving, got \`draft\`` |
| `replace-resolve-no-number` | exit 1; stderr `error: ../escape.md must stay within home/.config/invox/archive` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml` |
| `replace-resolve-with-number` | exit 1; stderr `error: ../escape.md must stay within home/.config/invox/archive` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml` |
| `exists-vs-duplicate` | exit 1; stderr `error: home/.config/invox/archive/invoice.yaml already exists` | exit 1; stderr `error: invoice.yaml: invoice number CUST-001-001 is already used by archived invoice home/.config/invox/archive/other.yaml / Run 'invox increment -i invoice.yaml' to give` |
| `rearchive-dry-run` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `Would replace archived invoice home/.config/invox/archive/invoice.yaml; the previous version would be kept in home/.config/invox/archive/.history / Would replace archived` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it / Would replace archived invoice home/.config/invox/archive` |
| `rearchive-yes` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `Replaced archived invoice home/.config/invox/archive/invoice.yaml; previous version kept at home/.config/invox/archive/.history/invoice.20261009T115929Z.yaml / Replaced a` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it / Replaced archived invoice home/.config/invox/archive/invo` |
| `email-write` | exit 1; stderr `error: created d.eml but failed to open it: exec: "xdg-open": executable file not found in $PATH` | exit 1; stderr `error: created d.eml but failed to open it: exec: "xdg-open": executable file not found in $PATH` |
| `new-default-exists-and-bad-due` | exit 1; stderr `error: CUST-001-010.yaml already exists; pass --force to replace it or choose a different -o/--output path` | exit 1; stderr `error: home/.config/invox/issuer.yaml: payment.due_days: must be >= 0` |
| `new-output-exists-and-bad-due` | exit 1; stderr `error: o.yaml already exists; pass --force to replace it or choose a different -o/--output path` | exit 1; stderr `error: home/.config/invox/issuer.yaml: payment.due_days: must be >= 0` |
| `md-list` | exit 0; stdout `old.md	CUST-001	2026-03-07	archived / x.yaml	CUST-001	2026-03-06	archived` | exit 0; stdout `x.yaml	CUST-001	2026-03-06	archived`; stderr `warning: 1 Markdown invoice in $W/home/.config/invox/archive is no longer read; convert it to .yaml to include it` |
| `md-list-json` | exit 0; stdout `[{"file":"old.md","number":"CUST-001-015"}]` | exit 0; stdout `[]`; stderr `warning: 1 Markdown invoice in $W/home/.config/invox/archive is no longer read; convert it to .yaml to include it` |
| `md-new` | exit 0; stdout `CUST-001-016.yaml`; stderr `Created CUST-001-016.yaml for CUST-001 (CUST-001-016)` | exit 0; stdout `CUST-001-010.yaml`; stderr `warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it / Created CUST-001-010.yaml for CUST-001 (CUST-001-010)` |
| `md-new-from-last` | exit 0; stdout `CUST-001-016.yaml`; stderr `Would create CUST-001-016.yaml for CUST-001 (CUST-001-016)` | exit 1; stderr `error: no archived invoice found for customer_id `CUST-001`` |
| `md-increment` | exit 0; stdout `invoice.yaml`; stderr `Would increment invoice.yaml for CUST-001: CUST-001-001 -> CUST-001-016` | exit 0; stdout `invoice.yaml`; stderr `warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it / Would increment invoice.yaml for CUST-001: CUST-001-001 -` |
| `md-validate-duplicate` | exit 0; stderr `warning: invoice number CUST-001-001 is already used by archived invoice home/.config/invox/archive/old.md; run 'invox increment -i invoice.yaml' before archiving / Valid` | exit 0; stderr `warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it / Validation OK: CUST-001-001 for CUST-001, 2 line item(s),` |
| `md-archive-duplicate` | exit 1; stderr `error: invoice.yaml: invoice number CUST-001-001 is already used by archived invoice home/.config/invox/archive/old.md / Run 'invox increment -i invoice.yaml' to give it ` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it / Archived invoice.yaml -> home/.config/invox/archive/invoi` |
| `md-archive-ok` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml` | exit 0; stdout `home/.config/invox/archive/invoice.yaml`; stderr `warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it / Archived invoice.yaml -> home/.config/invox/archive/invoi` |
| `md-edit` | exit 0; stdout `old.yaml`; stderr `Would copy home/.config/invox/archive/old.md -> old.yaml` | exit 1; stderr `error: home/.config/invox/archive/old.md is a Markdown invoice, which invox no longer reads; convert it to .yaml` |
| `md-build-archive-dry-run` | exit 0; stdout `invoice.pdf`; stderr `Would build invoice.pdf for CUST-001 (CUST-001-001) / Would set invoice.status to built in invoice.yaml / Would archive invoice.yaml -> home/.config/invox/archive/invoice` | exit 0; stdout `invoice.pdf`; stderr `warning: 1 Markdown invoice in home/.config/invox/archive is no longer read; convert it to .yaml to include it / Would build invoice.pdf for CUST-001 (CUST-001-001) / Wou` |
| `legacy-issuer-validate` | exit 0; stderr `Validation OK: CUST-001-001 for CUST-001, 2 line item(s), total 252,00 € / warning: using issuer.yaml from deprecated config directory $W/home/.config/invoice-tool; run '` | exit 2; stderr `error: issuer file not found; pass -u/--issuer, set paths.issuer in config.yaml, or place issuer.yaml at home/.config/invox/issuer.yaml / Run 'invox validate --help' for ` |
| `legacy-config` | exit 0; stdout `config-dir	$W/home/.config/invox	default / config	$W/home/.config/invoice-tool/config.yaml	legacy / customers	`; stderr `warning: using config.yaml from deprecated config directory $W/home/.config/invoice-tool; run 'invox init' to copy it to $W/home/.config/invox` | exit 0; stdout `config-dir	$W/home/.config/invox	default / config		none / customers	$W/home/.config/invox/customers.yaml	defau` |
| `legacy-init` | exit 2; stderr `error: the deprecated config directory home/.config/invoice-tool has files that home/.config/invox lacks; pass --force to copy them (no terminal to ask on) / Run 'invox i` | exit 0; stderr `Initialized $W/home/.config/invox / exists config.yaml / exists customers.yaml / exists issuer.yaml / exists invoice_defaults.yaml / exists template.tex` |
| `legacy-init-force` | exit 0; stderr `copied notes.txt from $W/home/.config/invoice-tool / invox no longer reads $W/home/.config/invoice-tool for these files; remove it once you are happy with $W/home/.config` | exit 2; stderr `error: unknown flag: --force / Run 'invox init --help' for usage.` |
| `increment-bad-date` | exit 1; stderr `error: invoice.yaml: invoice.issue_date: expected YYYY-MM-DD, got `2026-13-45`` | exit 1; stderr `error: invoice.yaml:4: invoice.issue_date: expected YYYY-MM-DD, got `2026-13-45`` |
| `archive-null-header` | exit 1; stderr `error: invoice.yaml: `invoice` must be a mapping` | exit 1; stderr `error: invoice.yaml: missing `invoice` mapping` |
| `archive-aliased-header-and-duplicate` | exit 1; stderr `error: invoice.yaml: `invoice` must be a mapping` | exit 1; stderr `error: invoice.yaml: invoice number CUST-001-001 is already used by archived invoice home/.config/invox/archive/other.yaml / Run 'invox increment -i invoice.yaml' to give` |
| `new-output-exists-unknown-customer` | exit 1; stderr `error: o.yaml already exists; pass --force to replace it or choose a different -o/--output path` | exit 1; stderr `error: home/.config/invox/customers.yaml: unknown customer_id `CUST-404` / Run 'invox customer list' to see the customer IDs.` |
| `help-environment` | exit 0; stdout `Environment variables and default directories. /  / Usage: /   invox help environment /  / Environment variabl` | exit 0; stdout `Environment variables and default directories. /  / Usage: /   invox help environment /  / Environment variabl` |
| `archive-scalar-header` | exit 1; stderr `error: invoice.yaml: `invoice` must be a mapping` | exit 1; stderr `error: invoice.yaml:2: invoice must be a mapping, got an integer` |

## Appendix B: differential script

Round 1's script (`docs/design/ports-narrowing.md`, Appendix B) with 32 scenarios added, a way to
delete a fixture file (`None` for a path without a trailing slash), and scenario names as
optional filters. Run it as `python3 difftest.py <base>/cmd/invox/testdata/script/archive.txtar
<base-binary> <new-binary> [scenario ...]`, with both binaries built by `go build -o <file>
./cmd/invox`. It takes the fixtures from the base commit's `archive.txtar`.

```python
"""Run order-sensitive scenarios against two invox binaries and diff them.

usage: python3 difftest.py <archive.txtar> <base-binary> <new-binary>
"""
import os
import shutil
import subprocess
import sys
import tempfile

txtar, base_bin, new_bin = sys.argv[1:4]
only = sys.argv[4:]


def parse_txtar(path):
    files, name, buf = {}, None, []
    for line in open(path, encoding='utf-8').read().splitlines(keepends=True):
        if line.startswith('-- ') and line.rstrip().endswith(' --'):
            if name:
                files[name] = ''.join(buf)
            name, buf = line[3:].rstrip()[:-3].strip(), []
        elif name:
            buf.append(line)
    if name:
        files[name] = ''.join(buf)
    return files


FIX = {k: v for k, v in parse_txtar(txtar).items() if not k.startswith('want')}
CFG = 'home/.config/invox/'
ARCH = CFG + 'archive/'
NO_EMAIL_CUSTOMER = 'CUST-004:\n  name: Silent GmbH\n  status: active\n  address:\n    street: A 1\n    postal_code: "1"\n    city: X\n    country: Austria\n'


def working_copy(number='CUST-001-001', status='editing', archive_path='invoice.yaml', replace_path=None):
    link = '_invox:\n  archive_path: %s\n' % archive_path
    if replace_path:
        link += '  archive_replace_path: %s\n' % replace_path
    num = '  number: %s\n' % number if number else ''
    return ('customer_id: CUST-001\ninvoice:\n%s  issue_date: 2026-03-06\n  due_date: 2026-04-05\n'
            '  status: %s\n  period: March\n  vat_percent: 20\n  paid_amount: 0\npositions:\n  - name: Dev\n'
            '    description: Work\n    unit_price: 100\n    quantity: 1\n%s') % (num, status, link)


def built(customer='CUST-001', number='CUST-001-001', status='built'):
    return FIX['invoice.yaml'].replace('status: draft', 'status: ' + status).replace('CUST-001', customer).replace('CUST-001-001', number)


SCENARIOS = [
    # (name, extra files {path: content or None for dir}, cwd relative, args)
    ('dir-error-before-status', {CFG + 'config.yaml': 'archive: [\n'}, '', ['archive', 'add', 'invoice.yaml']),
    ('unavailable-dir-vs-status', {CFG + 'config.yaml': 'archive:\n  dir: not-a-dir\n', CFG + 'not-a-dir': 'x\n'}, '', ['archive', 'add', 'invoice.yaml']),
    ('status-before-resolve', {'invoice.yaml': working_copy(status='draft', archive_path='/abs/x.yaml')}, '', ['archive', 'add', 'invoice.yaml']),
    ('resolve-archive-path', {'invoice.yaml': working_copy(archive_path='../escape.yaml')}, '', ['archive', 'add', 'invoice.yaml']),
    ('replace-resolve-no-number', {'invoice.yaml': working_copy(number=None, replace_path='../escape.md')}, '', ['archive', 'add', 'invoice.yaml']),
    ('replace-resolve-with-number', {'invoice.yaml': working_copy(replace_path='../escape.md')}, '', ['archive', 'add', 'invoice.yaml']),
    ('target-dir-vs-duplicate', {'invoice.yaml': working_copy(), ARCH + 'other.yaml': built(), ARCH + 'invoice.yaml/': None}, '', ['archive', 'add', 'invoice.yaml', '--yes']),
    ('exists-vs-duplicate', {'invoice.yaml': built(), ARCH + 'invoice.yaml': built(number='X-1'), ARCH + 'other.yaml': built()}, '', ['archive', 'add', 'invoice.yaml']),
    ('duplicate', {'invoice.yaml': built(), ARCH + 'other.yaml': built(status='archived')}, '', ['archive', 'add', 'invoice.yaml']),
    ('already-in-archive', {ARCH + 'invoice.yaml': working_copy()}, ARCH, ['archive', 'add', 'invoice.yaml', '--yes']),
    ('rearchive-needs-yes', {'invoice.yaml': working_copy(), ARCH + 'invoice.yaml': built(status='archived')}, '', ['archive', 'add', 'invoice.yaml']),
    ('rearchive-dry-run', {'invoice.yaml': working_copy(replace_path='old.md'), ARCH + 'invoice.yaml': built(status='archived'), ARCH + 'old.md': '---\ncustomer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n---\n'}, '', ['archive', 'add', 'invoice.yaml', '-n']),
    ('rearchive-yes', {'invoice.yaml': working_copy(replace_path='old.md'), ARCH + 'invoice.yaml': built(status='archived'), ARCH + 'old.md': '---\ncustomer_id: CUST-001\ninvoice:\n  number: CUST-001-001\n---\n'}, '', ['archive', 'add', 'invoice.yaml', '--yes']),
    ('archive-new-ok', {'invoice.yaml': built()}, '', ['archive', 'add', 'invoice.yaml']),
    ('build-archive-dry-run', {}, '', ['build', 'invoice.yaml', '--archive', '--dry-run']),
    ('build-no-tectonic', {}, '', ['build', 'invoice.yaml']),
    ('validate-duplicate', {ARCH + 'other.yaml': built(status='archived')}, '', ['validate', 'invoice.yaml']),
    ('validate-working-copy-own-number', {'invoice.yaml': working_copy(), ARCH + 'invoice.yaml': built(status='archived')}, '', ['validate', 'invoice.yaml']),
    ('validate-bad-link', {'invoice.yaml': working_copy(archive_path='/abs.yaml')}, '', ['validate', 'invoice.yaml']),
    ('edit-force-into-archive', {ARCH + 'invoice.yaml': built(status='archived')}, ARCH, ['archive', 'edit', 'invoice.yaml', '--force']),
    ('edit-exists', {ARCH + 'invoice.yaml': built(status='archived')}, '', ['archive', 'edit', 'invoice.yaml']),
    ('edit-ok-dry-run', {ARCH + 'other.yaml': built(status='archived')}, '', ['archive', 'edit', 'other.yaml', '-n']),
    ('email-missing-pdf-and-recipient', {CFG + 'customers.yaml': FIX[CFG + 'customers.yaml'] + NO_EMAIL_CUSTOMER, 'invoice.yaml': built(customer='CUST-004', number='CUST-004-001')}, '', ['email', 'invoice.yaml', '-o', 'd.eml']),
    ('email-missing-pdf', {'invoice.yaml': built()}, '', ['email', 'invoice.yaml', '-o', 'd.eml']),
    ('email-orphan-pdf', {'orphan.pdf': '%PDF\n'}, '', ['email', 'orphan.pdf', '-o', 'd.eml', '-n']),
    ('email-pdf-next-to-draft', {'invoice.pdf': '%PDF\n'}, '', ['email', 'invoice.pdf', '-o', 'd.eml', '-n']),
    ('email-pdf-from-archive', {'out/inv.pdf': '%PDF\n', ARCH + 'inv.yaml': built(status='archived')}, '', ['email', 'out/inv.pdf', '-o', 'd.eml', '-n']),
    ('email-pdf-ambiguous-archive', {'out/inv.pdf': '%PDF\n', ARCH + 'a/inv.yaml': built(status='archived'), ARCH + 'b/inv.yaml': built(status='archived')}, '', ['email', 'out/inv.pdf', '-o', 'd.eml', '-n']),
    ('email-upper-PDF', {'INV.PDF': '%PDF\n', 'INV.yaml': built()}, '', ['email', 'INV.PDF', '-o', 'd.eml', '-n']),
    ('email-pdf-with-p', {'invoice.pdf': '%PDF\n', 'other.pdf': '%PDF\n', 'invoice.yaml': built()}, '', ['email', 'invoice.pdf', '-p', 'other.pdf', '-o', 'd.eml', '-n']),
    ('email-dry-run-output-exists-no-recipient', {CFG + 'customers.yaml': FIX[CFG + 'customers.yaml'] + NO_EMAIL_CUSTOMER, 'invoice.yaml': built(customer='CUST-004', number='CUST-004-001'), 'invoice.pdf': '%PDF\n', 'd.eml': 'x'}, '', ['email', 'invoice.yaml', '-o', 'd.eml', '-n']),
    ('email-dry-run-output-exists', {'invoice.yaml': built(), 'invoice.pdf': '%PDF\n', 'd.eml': 'x'}, '', ['email', 'invoice.yaml', '-o', 'd.eml', '-n']),
    ('email-write', {'invoice.yaml': built(), 'invoice.pdf': '%PDF\n'}, '', ['email', 'invoice.yaml', '-o', 'd.eml']),
    ('new-default-exists-and-bad-due', {CFG + 'issuer.yaml': FIX[CFG + 'issuer.yaml'].replace('due_days: 30', 'due_days: -1'), 'CUST-001-010.yaml': 'x: 1\n'}, '', ['new', 'CUST-001']),
    ('new-output-exists-and-bad-due', {CFG + 'issuer.yaml': FIX[CFG + 'issuer.yaml'].replace('due_days: 30', 'due_days: -1'), 'o.yaml': 'x: 1\n'}, '', ['new', 'CUST-001', '-o', 'o.yaml']),
    ('new-output-dir', {'o.yaml/': None}, '', ['new', 'CUST-001', '-o', 'o.yaml']),
    ('new-default-dir', {'CUST-001-010.yaml/': None}, '', ['new', 'CUST-001']),
    ('new-ok', {}, '', ['new', 'CUST-001']),
    ('new-output-in-other-dir-drafts', {'sub/d.yaml': built(number='CUST-001-007', status='draft')}, '', ['new', 'CUST-001', '-o', 'sub/n.yaml']),
    ('new-force-into-archive', {ARCH + 'x.yaml': built(status='archived')}, '', ['new', 'CUST-001', '-o', CFG + 'archive/x.yaml', '--force']),
    ('new-from-last', {ARCH + 'x.yaml': built(status='archived')}, '', ['new', 'CUST-001', '--from-last', '-n']),
    ('email-missing-pdf-and-bad-subject', {'invoice.yaml': built()}, '', ['email', 'invoice.yaml', '-o', 'd.eml', '--subject', '{nope}']),
    ('email-dry-run-output-exists-bad-subject', {'invoice.yaml': built(), 'invoice.pdf': '%PDF\n', 'd.eml': 'x'}, '', ['email', 'invoice.yaml', '-o', 'd.eml', '-n', '--subject', '{nope}']),
    ('email-temp-dry-run', {'invoice.yaml': built(), 'invoice.pdf': '%PDF\n'}, '', ['email', 'invoice.yaml', '-n']),
    ('archive-comments-and-empty-link', {'invoice.yaml': '# keep me\n' + built() + '_invox: {}\n'}, '', ['archive', 'add', 'invoice.yaml']),
    ('rearchive-comments-mode', {'invoice.yaml': '# working copy\n' + working_copy(), ARCH + 'invoice.yaml': built(status='archived')}, '', ['archive', 'add', 'invoice.yaml', '--yes']),
    ('archive-list-root', {'invoice.yaml': '- a\n- b\n'}, '', ['archive', 'add', 'invoice.yaml']),
    ('archive-scalar-root', {'invoice.yaml': 'hello\n'}, '', ['archive', 'add', 'invoice.yaml']),
    ('archive-aliased-header', {'invoice.yaml': 'x: &h {number: A-1, status: built}\ninvoice: *h\ncustomer_id: CUST-001\n'}, '', ['archive', 'add', 'invoice.yaml']),
    ('render-ok', {}, '', ['render', 'invoice.yaml']),
]

LEGACY = 'home/.config/invoice-tool/'
MD = '---\ncustomer_id: CUST-001\ninvoice:\n  number: CUST-001-015\n  issue_date: 2026-03-07\n  status: archived\n---\n# Archived invoice\n'
MD_SAME = MD.replace('CUST-001-015', 'CUST-001-001')
BAD_POSITION = FIX['invoice.yaml'].replace('unit_price: 10\n', 'unit_price: abc\n')

EXTRA = [
    ('md-list', {ARCH + 'old.md': MD, ARCH + 'x.yaml': built(status='archived', number='CUST-001-002')}, '', ['archive', 'list']),
    ('md-list-json', {ARCH + 'old.md': MD}, '', ['archive', 'list', '--json', 'file,number']),
    ('md-new', {ARCH + 'old.md': MD}, '', ['new', 'CUST-001']),
    ('md-new-from-last', {ARCH + 'old.md': MD}, '', ['new', 'CUST-001', '--from-last', '-n']),
    ('md-increment', {ARCH + 'old.md': MD}, '', ['increment', 'invoice.yaml', '-n']),
    ('md-validate-duplicate', {ARCH + 'old.md': MD_SAME}, '', ['validate', 'invoice.yaml']),
    ('md-archive-duplicate', {'invoice.yaml': built(), ARCH + 'old.md': MD_SAME}, '', ['archive', 'add', 'invoice.yaml']),
    ('md-archive-ok', {'invoice.yaml': built(), ARCH + 'old.md': MD}, '', ['archive', 'add', 'invoice.yaml']),
    ('md-edit', {ARCH + 'old.md': MD}, '', ['archive', 'edit', 'old.md', '-n']),
    ('md-email-pdf', {'out/inv.pdf': '%PDF\n', ARCH + 'inv.yaml': built(status='archived'), ARCH + 'old.md': MD}, '', ['email', 'out/inv.pdf', '-o', 'd.eml', '-n']),
    ('md-build-archive-dry-run', {ARCH + 'old.md': MD}, '', ['build', 'invoice.yaml', '--archive', '--dry-run']),
    ('legacy-issuer-validate', {CFG + 'issuer.yaml': None, LEGACY + 'issuer.yaml': FIX[CFG + 'issuer.yaml']}, '', ['validate', 'invoice.yaml']),
    ('legacy-config', {CFG + 'config.yaml': None, LEGACY + 'config.yaml': FIX[CFG + 'config.yaml']}, '', ['config', 'paths']),
    ('legacy-init', {LEGACY + 'notes.txt': 'x\n'}, '', ['init']),
    ('legacy-init-force', {LEGACY + 'notes.txt': 'x\n'}, '', ['init', '--force']),
    ('increment-bad-date', {'invoice.yaml': FIX['invoice.yaml'].replace('issue_date: 2026-03-06', 'issue_date: 2026-13-45')}, '', ['increment', 'invoice.yaml', '-n']),
    ('increment-bad-position', {'invoice.yaml': BAD_POSITION}, '', ['increment', 'invoice.yaml', '-n']),
    ('increment-unknown-key', {'invoice.yaml': FIX['invoice.yaml'] + 'extra: 1\n'}, '', ['increment', 'invoice.yaml', '-n']),
    ('increment-no-number', {'invoice.yaml': FIX['invoice.yaml'].replace('  number: CUST-001-001\n', '')}, '', ['increment', 'invoice.yaml', '-n']),
    ('archive-null-header', {'invoice.yaml': 'customer_id: CUST-001\ninvoice:\n'}, '', ['archive', 'add', 'invoice.yaml']),
    ('archive-aliased-header-dry-run', {'invoice.yaml': 'x: &h {number: A-1, status: built}\ninvoice: *h\ncustomer_id: CUST-001\n'}, '', ['archive', 'add', 'invoice.yaml', '-n']),
    ('archive-aliased-header-and-duplicate', {'invoice.yaml': 'x: &h {number: CUST-001-001, status: built}\ninvoice: *h\ncustomer_id: CUST-001\n', ARCH + 'other.yaml': built(status='archived')}, '', ['archive', 'add', 'invoice.yaml']),
    ('edit-dir-then-exists', {ARCH + 'other.yaml': built(status='archived'), 'other.yaml/': None}, '', ['archive', 'edit', 'other.yaml']),
    ('email-temp-open-fails', {'invoice.yaml': built(), 'invoice.pdf': '%PDF\n'}, '', ['email', 'invoice.yaml']),
    ('template-list-json', {}, '', ['template', 'list', '--json', 'name,default']),
    ('template-list', {}, '', ['template', 'list']),
    ('new-output-exists-unknown-customer', {'o.yaml': 'x: 1\n'}, '', ['new', 'CUST-404', '-o', 'o.yaml']),
    ('build-archive-dry-run-exists', {ARCH + 'invoice.yaml': built(status='archived', number='X-9')}, '', ['build', 'invoice.yaml', '--archive', '--dry-run']),
    ('help-environment', {}, '', ['help', 'environment']),
    ('increment-no-customers-bad-invoice', {CFG + 'customers.yaml': None, 'invoice.yaml': 'invoice:\n  number: X-1\n'}, '', ['increment', 'invoice.yaml', '-n']),
    ('archive-scalar-header', {'invoice.yaml': 'customer_id: CUST-001\ninvoice: 5\n'}, '', ['archive', 'add', 'invoice.yaml']),
    ('edit-aliased-header', {ARCH + 'al.yaml': 'x: &h {number: A-1, status: archived}\ninvoice: *h\ncustomer_id: CUST-001\n'}, '', ['archive', 'edit', 'al.yaml']),
]

SCENARIOS += EXTRA


def run(binary, scenario):
    name, extra, cwd, args = scenario
    work = tempfile.mkdtemp(prefix='invox-diff-')
    try:
        for path, content in list(FIX.items()) + list(extra.items()):
            full = os.path.join(work, path)
            if content is None and not path.endswith('/'):
                if os.path.exists(full):
                    os.remove(full)
                continue
            if content is None:
                os.makedirs(full, exist_ok=True)
                continue
            os.makedirs(os.path.dirname(full), exist_ok=True)
            with open(full, 'w', encoding='utf-8') as f:
                f.write(content)
        home = os.path.join(work, 'home')
        env = {'PATH': '/usr/bin:/bin', 'HOME': home, 'XDG_CONFIG_HOME': os.path.join(home, '.config'),
               'XDG_DATA_HOME': os.path.join(home, '.local', 'share'), 'TZ': 'UTC'}
        p = subprocess.run([binary] + args, cwd=os.path.join(work, cwd), env=env, capture_output=True, text=True, stdin=subprocess.DEVNULL)
        tree = []
        for dirpath, dirs, files in os.walk(work):
            dirs.sort()
            for fn in sorted(files):
                rel = os.path.relpath(os.path.join(dirpath, fn), work)
                if '.history' in rel:
                    rel = rel[:rel.index('.history')] + '.history/<backup>'
                full = os.path.join(dirpath, fn)
                import hashlib
                digest = hashlib.sha256(open(full, 'rb').read()).hexdigest()[:10]
                if '.history' in rel:
                    digest = 'x'
                tree.append('%s:%o:%s' % (rel, os.stat(full).st_mode & 0o777, digest))
        out = 'exit %d\nstdout:\n%sstderr:\n%sfiles: %s\n' % (p.returncode, p.stdout, p.stderr, ' '.join(tree))
        return out.replace(work, '$W')
    finally:
        shutil.rmtree(work, ignore_errors=True)


diffs = 0
for scenario in SCENARIOS:
    if only and scenario[0] not in only:
        continue
    a, b = run(base_bin, scenario), run(new_bin, scenario)
    first = a.splitlines()[0] + ' | ' + next((l for l in a.splitlines()[3:] if l and not l.startswith('files')), '')
    if a == b:
        print('same  %-42s %s' % (scenario[0], first[:110]))
    else:
        diffs += 1
        print('DIFF  %s\n--- base\n%s--- new\n%s' % (scenario[0], a, b))
print('%d scenarios, %d differ' % (len(SCENARIOS), diffs))
```

## Appendix C: acceptance test

`internal/archtest/ports_test.go` as the prototype has it. `TestBillingHandlesNoPaths` is
unchanged from `236d558`.

```go
package archtest

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// narrowPorts is the exact method set of each port billing owns, and
// serviceMethods that of *billing.Service. docs/design/ports-v2.md
// explains each method; a new one belongs there first.
var (
	narrowPorts = map[string][]string{
		"Invoices":  {"Create", "Drafts", "Load", "Update"},
		"Directory": {"Customer", "Customers", "Defaults", "EditablePath", "Init", "Issuer", "Locate", "Paths", "Template", "Templates"},
		"Archive":   {"Add", "Checkout", "Duplicate", "Entries", "Source"},
		"Renderer":  {"Build", "Render", "Write"},
		"Compiler":  {"Compile"},
		"Mailer":    {"Draft"},
	}
	serviceMethods = []string{
		"Archive", "Build", "DraftEmail", "EditArchived", "EditablePath", "Increment", "Init",
		"ListArchive", "ListCustomers", "ListTemplates", "New", "Paths", "Render", "Validate",
	}
)

func TestPortMethodSets(t *testing.T) {
	billing, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(mod + "internal/billing")
	if err != nil {
		t.Fatalf("type-check billing: %v", err)
	}
	for name, want := range narrowPorts {
		tn, _ := billing.Scope().Lookup(name).(*types.TypeName)
		if tn == nil {
			t.Errorf("billing.%s is not declared", name)
			continue
		}
		iface, ok := tn.Type().Underlying().(*types.Interface)
		if !ok {
			t.Errorf("billing.%s is not an interface", name)
			continue
		}
		var have []string
		for i := range iface.NumMethods() {
			have = append(have, iface.Method(i).Name())
		}
		if !slices.Equal(have, want) {
			t.Errorf("billing.%s methods = %v, want %v", name, have, want)
		}
	}
	tn, _ := billing.Scope().Lookup("Service").(*types.TypeName)
	if tn == nil {
		t.Fatal("billing.Service is not declared")
	}
	ms := types.NewMethodSet(types.NewPointer(tn.Type()))
	var have []string
	for i := range ms.Len() {
		if ms.At(i).Obj().Exported() {
			have = append(have, ms.At(i).Obj().Name())
		}
	}
	slices.Sort(have)
	if !slices.Equal(have, serviceMethods) {
		t.Errorf("*billing.Service methods = %v, want %v", have, serviceMethods)
	}
}

// fileExtension matches a string literal that names a file extension, the
// sign of a use case deriving a path by hand.
var fileExtension = regexp.MustCompile(`\.(?i:ya?ml|pdf|eml|tex|md|markdown)\b`)

func TestBillingHandlesNoPaths(t *testing.T) {
	dir := filepath.Join("..", "billing")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			if path, _ := strconv.Unquote(imp.Path.Value); path == "path" || path == "path/filepath" {
				t.Errorf("%s imports %s; a driven adapter builds paths", name, path)
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && fileExtension.MatchString(lit.Value) {
				t.Errorf("%s: string %s names a file extension; a driven adapter names files", fset.Position(lit.Pos()), lit.Value)
			}
			return true
		})
	}
}

// TestPortsCarryNoAdapterData checks the two ways adapter data has passed
// through billing. A port result that billing hands back to the same port
// (Placement from Archive.Place to Archive.Add) is the adapter's own state
// taking a detour. A func in a port result (Template.FindAsset,
// Draft.Discard) is adapter behavior that billing carries for someone else.
func TestPortsCarryNoAdapterData(t *testing.T) {
	billing, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(mod + "internal/billing")
	if err != nil {
		t.Fatalf("type-check billing: %v", err)
	}
	for name := range narrowPorts {
		iface := billing.Scope().Lookup(name).Type().Underlying().(*types.Interface)
		results, params := map[string]bool{}, map[string]bool{}
		for i := range iface.NumMethods() {
			sig := iface.Method(i).Type().(*types.Signature)
			for j := range sig.Results().Len() {
				collect(sig.Results().At(j).Type(), billing, results)
				if path := funcField(sig.Results().At(j).Type(), map[types.Type]bool{}); path != "" {
					t.Errorf("billing.%s.%s returns %s, a func: adapter behavior passing through billing", name, iface.Method(i).Name(), path)
				}
			}
			for j := range sig.Params().Len() {
				collect(sig.Params().At(j).Type(), billing, params)
			}
		}
		for typ := range results {
			if params[typ] {
				t.Errorf("billing.%s both returns and takes billing.%s: the adapter's own data takes a detour through billing", name, typ)
			}
		}
	}
}

// collect adds the billing struct types t names, through pointers and
// slices, to into.
func collect(t types.Type, billing *types.Package, into map[string]bool) {
	switch t := t.(type) {
	case *types.Pointer:
		collect(t.Elem(), billing, into)
	case *types.Slice:
		collect(t.Elem(), billing, into)
	case *types.Named:
		if _, ok := t.Underlying().(*types.Struct); ok && t.Obj().Pkg() == billing {
			into[t.Obj().Name()] = true
		}
	}
}

// funcField returns the field path of the first func-typed field in t,
// following structs, pointers and slices, or "".
func funcField(t types.Type, seen map[types.Type]bool) string {
	if seen[t] {
		return ""
	}
	seen[t] = true
	switch u := t.(type) {
	case *types.Pointer:
		return funcField(u.Elem(), seen)
	case *types.Slice:
		return funcField(u.Elem(), seen)
	case *types.Named:
		st, ok := u.Underlying().(*types.Struct)
		if !ok {
			return ""
		}
		for i := range st.NumFields() {
			f := st.Field(i)
			if _, isFunc := f.Type().Underlying().(*types.Signature); isFunc {
				return u.Obj().Name() + "." + f.Name()
			}
			if path := funcField(f.Type(), seen); path != "" {
				return path
			}
		}
	}
	return ""
}
```

## Appendix D: testscript changes

`diff -ruN` of `cmd/invox/testdata` between `236d558` and the prototype. The goldens
(`want-*` sections) come from `go test ./cmd/invox -run TestScript -update`. The script
commands in `archive.txtar`, `config_paths.txtar` and `cobra_init.txtar` were edited by hand.

```diff
diff -ruN -x .git base/cmd/invox/testdata/script/archive.txtar proto/cmd/invox/testdata/script/archive.txtar
--- base/cmd/invox/testdata/script/archive.txtar	2026-10-09 09:01:29.000000000 +0000
+++ proto/cmd/invox/testdata/script/archive.txtar	2026-10-09 11:48:56.681926273 +0000
@@ -42,12 +42,14 @@
 cmp got.txt want-archive.txt
 ! exists second.yaml
 
-# archive list prints FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS,
-# including Markdown invoices with YAML front matter.
+# archive list prints FILENAME<TAB>CUSTOMER_ID<TAB>ISSUE_DATE<TAB>STATUS.
+# A Markdown invoice is no longer read: one warning line names the archive.
 cp legacy.md home/.config/invox/archive/legacy.md
 exec invox archive list
 cmp stdout want-list.tsv
-! stderr .
+scrubpaths stderr got.txt
+cmp got.txt want-markdown-warning.txt
+rm home/.config/invox/archive/legacy.md
 
 # archive edit copies an archived invoice back for editing; archiving the
 # working copy replaces the archived one. stdin is not a terminal here, so
@@ -179,6 +181,8 @@
   issue_date: 2026-03-05
 ---
 # Archived invoice
+-- want-markdown-warning.txt --
+warning: 1 Markdown invoice in $WORK/home/.config/invox/archive is no longer read; convert it to .yaml to include it
 -- want-build-archive.txt --
 fake tectonic: compiling invoice.tex
 Built invoice.pdf for CUST-001 (CUST-001-001)
@@ -193,7 +197,6 @@
 Archived second.yaml -> home/.config/invox/archive/second.yaml
 -- want-list.tsv --
 invoice.yaml	CUST-001	2026-03-06	archived
-legacy.md	CUST-MD	2026-03-05	archived
 second.yaml	CUST-003	2026-03-07	archived
 -- want-edit.txt --
 Editing home/.config/invox/archive/invoice.yaml -> invoice.yaml
diff -ruN -x .git base/cmd/invox/testdata/script/cobra_init.txtar proto/cmd/invox/testdata/script/cobra_init.txtar
--- base/cmd/invox/testdata/script/cobra_init.txtar	2026-10-09 09:01:29.000000000 +0000
+++ proto/cmd/invox/testdata/script/cobra_init.txtar	2026-10-09 11:48:48.999346903 +0000
@@ -1,17 +1,15 @@
 # init parses its flags with pflag.
 
-# A misspelt flag suggests the right one, and nothing is written.
+# An unknown flag is a usage error, and nothing is written.
 exits 2 invox init --forse
 ! stdout .
 cmp stderr want-misspelt.txt
 ! exists home/.config/invox/config.yaml
 
-# A single-dash long flag works and warns.
-exec invox init -force
+exec invox init
 ! stdout .
-stderr '^warning: -force is deprecated; use --force$'
 stderr '^created config.yaml$'
 
 -- want-misspelt.txt --
-error: unknown flag: --forse; did you mean --force?
+error: unknown flag: --forse
 Run 'invox init --help' for usage.
diff -ruN -x .git base/cmd/invox/testdata/script/config_paths.txtar proto/cmd/invox/testdata/script/config_paths.txtar
--- base/cmd/invox/testdata/script/config_paths.txtar	2026-10-09 09:01:29.000000000 +0000
+++ proto/cmd/invox/testdata/script/config_paths.txtar	2026-10-09 11:48:48.998984520 +0000
@@ -7,18 +7,16 @@
 scrubpaths -tsv stdout got.tsv
 cmp got.tsv want-defaults.tsv
 
-# A project file in the working directory, a file only in the legacy
-# directory, and --config with paths.* and archive.dir. The legacy file gets
-# one warning line on stderr.
+# A project file in the working directory, and --config with paths.* and
+# archive.dir. The deprecated invoice-tool directory is not read.
 cp legacy-issuer.yaml home/.config/invoice-tool/issuer.yaml
 cp project-customers.yaml customers.yaml
 exec invox config paths --config acme.yaml
 scrubpaths -tsv stdout got.tsv
 cmp got.tsv want-mixed.tsv
-scrubpaths stderr got.txt
-cmp got.txt want-legacy-warning.txt
+! stderr .
 
-# INVOX_CONFIG_DIR replaces the config directory and turns off the legacy one.
+# INVOX_CONFIG_DIR replaces the config directory.
 env INVOX_CONFIG_DIR=$WORK/env-config
 rm customers.yaml
 exec invox config paths
@@ -73,12 +71,10 @@
 config-dir	$WORK/home/.config/invox	default
 config	$WORK/acme.yaml	flag
 customers	$WORK/customers.yaml	project
-issuer	$WORK/home/.config/invoice-tool/issuer.yaml	legacy
+issuer		none
 defaults		none
 template	$WORK/templates/acme.tex	config
 archive	$WORK/archive	config
--- want-legacy-warning.txt --
-warning: using issuer.yaml from deprecated config directory $WORK/home/.config/invoice-tool; run 'invox init' to copy it to $WORK/home/.config/invox
 -- want-env.tsv --
 config-dir	$WORK/env-config	env
 config	$WORK/env-config/config.yaml	env
diff -ruN -x .git base/cmd/invox/testdata/script/help.txtar proto/cmd/invox/testdata/script/help.txtar
--- base/cmd/invox/testdata/script/help.txtar	2026-10-09 09:01:29.000000000 +0000
+++ proto/cmd/invox/testdata/script/help.txtar	2026-10-09 11:48:17.922722891 +0000
@@ -626,8 +626,7 @@
   Existing config.yaml files are left unchanged.
 
 Config paths:
-  preferred: $WORK/home/.config/invox/config.yaml
-  legacy fallback: $WORK/home/.config/invoice-tool/config.yaml
+  default: $WORK/home/.config/invox/config.yaml
   --config PATH and INVOX_CONFIG_DIR change them; see `invox help environment`.
   `invox config paths` shows the config file and the support files in use.
 
@@ -950,9 +949,6 @@
   Writes starter versions of config.yaml, customers.yaml, issuer.yaml,
   invoice_defaults.yaml, and template.tex.
   Existing non-empty files are left unchanged.
-  When the deprecated invoice-tool directory has files the config directory
-  lacks, asks first, then copies them in before writing the starter files.
-  Nothing is replaced, and the invoice-tool directory is left in place.
 
 Config directory:
   $WORK/home/.config/invox
@@ -960,9 +956,6 @@
 Usage:
   invox init [flags]
 
-Flags:
-      --force   Copy files from the deprecated config directory without asking (required without a terminal)
-
 Global flags:
       --config string   Read this config file instead of config.yaml
   -h, --help            Show help for a command
@@ -970,7 +963,6 @@
 
 Examples:
   $ invox init
-  $ invox init --force
 -- want-template-list.txt --
 List available LaTeX invoice templates from the same directory as the resolved default template.
 
diff -ruN -x .git base/cmd/invox/testdata/script/help_topics.txtar proto/cmd/invox/testdata/script/help_topics.txtar
--- base/cmd/invox/testdata/script/help_topics.txtar	2026-10-09 09:01:29.000000000 +0000
+++ proto/cmd/invox/testdata/script/help_topics.txtar	2026-10-09 11:48:17.354722904 +0000
@@ -41,9 +41,8 @@
 Environment variables:
   INVOX_CONFIG_DIR
       Config directory to use in place of the default one, on every OS. invox
-      reads config.yaml and the global support files there, `init` writes there,
-      and the legacy directory is not read. It must exist. --config still wins
-      for config.yaml.
+      reads config.yaml and the global support files there, and `init` writes
+      there. It must exist. --config still wins for config.yaml.
   XDG_CONFIG_HOME
       Base directory for the config directory, on every OS.
       Default: $HOME/.config. A relative value is ignored.
@@ -77,12 +76,6 @@
   INVOX_CONFIG_DIR replaces it on every OS.
   here:      $WORK/home/.config/invox
 
-Legacy config directory (deprecated):
-  invoice-tool next to the invox directory, such as $HOME/.config/invoice-tool.
-  A file missing from the invox directory is still read from here, and invox
-  prints a warning. `invox init` copies the files into the invox directory.
-  Not read when INVOX_CONFIG_DIR is set.
-
 Default archive directory (when config.yaml sets no archive.dir):
   Linux:     $XDG_DATA_HOME/invox/invoices, else $HOME/.local/share/invox/invoices
   macOS:     $XDG_DATA_HOME/invox/invoices, else $HOME/Library/Application Support/invox/invoices
@@ -93,7 +86,6 @@
   1. --config PATH
   2. config.yaml in INVOX_CONFIG_DIR
   3. config.yaml in the config directory
-  4. config.yaml in the legacy directory
   A --config file that is missing or broken is an error. invox never falls
   back to another config file.
 
@@ -106,7 +98,7 @@
   1. explicit flag (-c, -u, --defaults, -t)
   2. upward search from the current directory
   3. paths.* in config.yaml, relative to config.yaml
-  4. the file in the config directory, then in the legacy directory
+  4. the file in the config directory
 
 Upward search:
   It always searches the current directory. It goes up to the nearest directory
```

## Appendix E: prototype patch

`diff -ruN` of the production code (`internal`, without `_test.go` files and `testdata`)
between `236d558` and the prototype: 45 files. Apply it with `patch -p1` in a checkout of
`236d558`. Test changes are not included, because the prototype only made the tests compile
(see [Tests](#tests)).

```diff
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/adapters/applemail/applemail.go proto/internal/adapters/applemail/applemail.go
--- base/internal/adapters/applemail/applemail.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/adapters/applemail/applemail.go	2026-10-09 11:46:05.464835485 +0000
@@ -75,7 +75,13 @@
 
 // Draft opens m in an Apple Mail compose window with the PDF attached. It
 // implements billing.Mailer; the draft has no file.
-func (c *Composer) Draft(ctx context.Context, m billing.Message) (billing.Draft, error) {
+func (c *Composer) Draft(ctx context.Context, m billing.Message, dryRun bool) (string, error) {
+	if _, err := os.Stat(m.Attachment); err != nil {
+		return "", fmt.Errorf("read %s: %w", m.Attachment, err)
+	}
+	if dryRun {
+		return "", nil
+	}
 	err := c.Compose(ctx, Message{
 		To:         m.To,
 		Subject:    m.Subject,
@@ -84,16 +90,7 @@
 		Sender:     m.FromAddress,
 	})
 	if err != nil {
-		return billing.Draft{}, fmt.Errorf("failed to open editable email draft: %w", err)
+		return "", fmt.Errorf("failed to open editable email draft: %w", err)
 	}
-	return billing.Draft{}, nil
+	return "", nil
 }
-
-// CheckAttachment returns why the file at path cannot be attached.
-func (c *Composer) CheckAttachment(path string) error {
-	_, err := os.Stat(path)
-	return err
-}
-
-// Check has nothing to check: Apple Mail writes no file.
-func (c *Composer) Check(billing.Message) error { return nil }
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/archive/adapter.go proto/internal/archive/adapter.go
--- base/internal/archive/adapter.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/archive/adapter.go	2026-10-09 11:44:44.042727783 +0000
@@ -7,6 +7,7 @@
 	"path/filepath"
 	"sort"
 	"strings"
+	"time"
 
 	"github.com/0xboris/invox/internal/billing"
 	"github.com/0xboris/invox/internal/fsutil"
@@ -22,6 +23,8 @@
 	// Rewrite returns the invoice at path with change applied, keeping its
 	// comments and layout.
 	Rewrite func(path string, change func(*invoice.Invoice) error) ([]byte, error)
+	// Now stamps the backups of replaced files.
+	Now func() time.Time
 }
 
 var _ billing.Archive = Archive{}
@@ -34,87 +37,57 @@
 	return Store{Dir: dir}, nil
 }
 
-// Entries reads every archived invoice in the order of Walk.
-func (a Archive) Entries() ([]billing.ArchiveEntry, error) {
+// Entries reads every archived invoice, sorted by Filename.
+func (a Archive) Entries() ([]billing.ArchiveEntry, billing.Unread, error) {
 	s, err := a.store()
 	if err != nil {
-		return nil, err
+		return nil, billing.Unread{}, err
 	}
-	return s.entries(a.Read)
+	return s.List(a.Read)
 }
 
-// Dir returns the archive directory.
-func (a Archive) Dir() (string, error) {
-	return a.Locate()
-}
-
-// Place says where archiving the invoice at src writes it.
-func (a Archive) Place(src string, head billing.Head) (billing.Placement, error) {
-	s, err := a.store()
-	if err != nil {
-		return billing.Placement{}, err
+// link returns the archived file the working copy inv goes back to, ""
+// for an invoice that is not a working copy.
+func link(inv invoice.Invoice) string {
+	if inv.Archive == nil {
+		return ""
 	}
-	p := billing.Placement{Path: filepath.Join(s.Dir, filepath.Base(src)), HistoryDir: s.HistoryDir()}
-	if head.WorkingCopy() {
-		target, err := s.Resolve(head.ArchivePath)
-		if err != nil {
-			return billing.Placement{}, err
-		}
-		p.Path, p.Overwrite = target.Path, true
-	} else if isFile(p.Path) {
-		return billing.Placement{}, fmt.Errorf("%s already exists", p.Path)
-	}
-	if filepath.Clean(src) == p.Path {
-		return billing.Placement{}, fmt.Errorf("%s is already in the archive directory", src)
-	}
-	if head.WorkingCopy() && head.ReplacePath != "" && head.ReplacePath != head.ArchivePath {
-		target, err := s.Resolve(head.ReplacePath)
-		if err != nil {
-			return billing.Placement{}, err
-		}
-		if target.Path != p.Path {
-			p.Remove = target.Path
-		}
-	}
-	return p, nil
+	return filepath.FromSlash(inv.Archive.ArchivePath.Trim())
 }
 
 // Duplicate returns the archived invoice, in file name order, that has
-// head's number, other than src and, for a working copy, the archived
-// files it replaces.
-func (a Archive) Duplicate(src string, head billing.Head) (string, error) {
+// inv's number, other than src and the archived file a working copy
+// replaces.
+func (a Archive) Duplicate(src string, inv invoice.Invoice) (string, billing.Unread, error) {
 	s, err := a.store()
 	if err != nil {
-		return "", err
+		return "", billing.Unread{}, err
+	}
+	var number string
+	if inv.Header != nil {
+		number = inv.Header.Number.Trim()
 	}
-	if strings.TrimSpace(s.Dir) == "" || head.Number == "" {
-		return "", nil
+	if strings.TrimSpace(s.Dir) == "" || number == "" {
+		return "", billing.Unread{Dir: s.Dir}, nil
 	}
 	excluded := map[string]bool{filepath.Clean(src): true}
-	// Only a working copy from `archive edit` may reuse the number of the
-	// archived file it replaces.
-	if head.WorkingCopy() {
-		for _, name := range []string{head.ArchivePath, head.ReplacePath} {
-			if name == "" {
-				continue
-			}
-			target, err := s.Resolve(name)
-			if err != nil {
-				return "", err
-			}
-			excluded[target.Path] = true
+	if name := link(inv); name != "" {
+		target, err := s.Resolve(name)
+		if err != nil {
+			return "", billing.Unread{Dir: s.Dir}, err
 		}
+		excluded[target.Path] = true
 	}
-	entries, err := s.List(a.Read)
+	entries, unread, err := s.List(a.Read)
 	if err != nil {
-		return "", err
+		return "", unread, err
 	}
 	for _, entry := range entries {
-		if entry.Number == head.Number && !excluded[filepath.Clean(entry.Path)] {
-			return filepath.Clean(entry.Path), nil
+		if entry.Number == number && !excluded[filepath.Clean(entry.Path)] {
+			return filepath.Clean(entry.Path), unread, nil
 		}
 	}
-	return "", nil
+	return "", unread, nil
 }
 
 // Checkout resolves ref and says where its working copy in workDir goes.
@@ -130,66 +103,77 @@
 	if err != nil {
 		return billing.Checkout{}, err
 	}
-	edit := target.Edit()
+	rel := filepath.Clean(target.Rel)
 	return billing.Checkout{
 		Archived: target.Path,
-		Path:     filepath.Join(workDir, edit.Filename),
-		Link:     billingLink(edit),
+		Path:     filepath.Join(workDir, filepath.Base(rel)),
+		Link:     invoice.ArchiveLink{ArchivePath: invoice.Text(filepath.ToSlash(rel))},
 	}, nil
 }
 
-// Add writes the invoice at src to p.Path with opts.Change applied, after
-// backing up the archived files it replaces, then removes p.Remove and src.
-func (a Archive) Add(src string, p billing.Placement, opts billing.AddOptions) (billing.ArchiveResult, error) {
-	var replaced []string
-	if p.Overwrite {
-		var err error
-		if replaced, err = ExistingFiles(p.Path, p.Remove); err != nil {
+// Add writes the invoice at src into the archive with opts.Change applied:
+// over the archived file a working copy names, else under src's name. It
+// backs up the archived file it replaces, then removes src.
+func (a Archive) Add(src string, inv invoice.Invoice, opts billing.AddOptions) (billing.ArchiveResult, error) {
+	s, err := a.store()
+	if err != nil {
+		return billing.ArchiveResult{}, err
+	}
+	if strings.TrimSpace(s.Dir) == "" {
+		return billing.ArchiveResult{}, errors.New("archive directory is unavailable")
+	}
+	path, overwrite := filepath.Join(s.Dir, filepath.Base(src)), false
+	if name := link(inv); name != "" {
+		target, err := s.Resolve(name)
+		if err != nil {
 			return billing.ArchiveResult{}, err
 		}
+		path, overwrite = target.Path, true
+	} else if isFile(path) {
+		return billing.ArchiveResult{}, fmt.Errorf("%s already exists", path)
 	}
-	if len(replaced) > 0 && !opts.Replace {
-		return billing.ArchiveResult{}, &billing.ArchiveReplaceError{InvoicePath: src, Paths: replaced, HistoryDir: p.HistoryDir}
+	if filepath.Clean(src) == path {
+		return billing.ArchiveResult{}, fmt.Errorf("%s is already in the archive directory", src)
 	}
-	if opts.DryRun {
-		result := billing.ArchiveResult{Path: p.Path, HistoryDir: p.HistoryDir}
-		for _, path := range replaced {
-			result.Replaced = append(result.Replaced, billing.Backup{Path: path})
+	var replaced []string
+	if overwrite {
+		if replaced, err = ExistingFiles(path); err != nil {
+			return billing.ArchiveResult{}, err
 		}
-		return result, nil
+	}
+	if len(replaced) > 0 && !opts.Replace {
+		return billing.ArchiveResult{}, &billing.ArchiveReplaceError{InvoicePath: src, Paths: replaced, HistoryDir: s.HistoryDir()}
 	}
 	data, err := a.Rewrite(src, opts.Change)
 	if err != nil {
 		return billing.ArchiveResult{}, err
 	}
-	s, err := a.store()
-	if err != nil {
-		return billing.ArchiveResult{}, err
+	if opts.DryRun {
+		result := billing.ArchiveResult{Path: path, HistoryDir: s.HistoryDir()}
+		for _, p := range replaced {
+			result.Replaced = append(result.Replaced, billing.Backup{Path: p})
+		}
+		return result, nil
 	}
 	if err := fsutil.MkdirAll(s.Dir, fsutil.Private); err != nil {
 		return billing.ArchiveResult{}, err
 	}
-	backups, err := s.Backup(replaced, opts.Now)
+	backups, err := s.Backup(replaced, a.Now())
 	if err != nil {
 		return billing.ArchiveResult{}, err
 	}
-	if p.Overwrite {
-		err = fsutil.WriteFile(p.Path, data, fsutil.Private)
-	} else if err = fsutil.WriteNewFile(p.Path, data, fsutil.Private); errors.Is(err, os.ErrExist) {
-		return billing.ArchiveResult{}, fmt.Errorf("%s already exists", p.Path)
+	if overwrite {
+		err = fsutil.WriteFile(path, data, fsutil.Private)
+	} else if err = fsutil.WriteNewFile(path, data, fsutil.Private); errors.Is(err, os.ErrExist) {
+		return billing.ArchiveResult{}, fmt.Errorf("%s already exists", path)
 	}
 	if err != nil {
 		return billing.ArchiveResult{}, err
 	}
-	if p.Remove != "" {
-		if err := os.Remove(p.Remove); err != nil && !os.IsNotExist(err) {
-			return billing.ArchiveResult{}, fmt.Errorf("remove %s: %w", p.Remove, err)
-		}
-	}
 	if err := os.Remove(filepath.Clean(src)); err != nil {
 		return billing.ArchiveResult{}, fmt.Errorf("remove %s: %w", filepath.Clean(src), err)
 	}
-	return billing.ArchiveResult{Path: p.Path, Replaced: backups, HistoryDir: p.HistoryDir}, nil
+	return billing.ArchiveResult{Path: path, Replaced: backups, HistoryDir: s.HistoryDir()}, nil
 }
 
 // Protects reports whether path is an existing file inside the archive
@@ -234,12 +218,12 @@
 
 // Source returns the invoice YAML file the PDF at pdf was built from: next
 // to it, else in the archive.
-func (a Archive) Source(pdf string) (string, error) {
+func (a Archive) Source(pdf string) (string, billing.Unread, error) {
 	base := strings.TrimSuffix(pdf, filepath.Ext(pdf))
 	candidates := []string{base + ".yaml", base + ".yml"}
 	for _, candidate := range candidates {
 		if isFile(candidate) {
-			return candidate, nil
+			return candidate, billing.Unread{}, nil
 		}
 	}
 	return a.findFile(filepath.Base(candidates[0]), filepath.Base(candidates[1]))
@@ -254,21 +238,21 @@
 // findFile returns the archived file named one of names: one directly in
 // the archive directory, else the only one below it. It returns "" when
 // there is none or no archive directory, and an error when several match.
-func (a Archive) findFile(names ...string) (string, error) {
+func (a Archive) findFile(names ...string) (string, billing.Unread, error) {
 	s, err := a.store()
 	if err != nil {
-		return "", err
+		return "", billing.Unread{}, err
 	}
 	if strings.TrimSpace(s.Dir) == "" {
-		return "", nil
+		return "", billing.Unread{}, nil
 	}
 	for _, name := range names {
 		if info, err := os.Stat(filepath.Join(s.Dir, name)); err == nil && !info.IsDir() {
-			return filepath.Join(s.Dir, name), nil
+			return filepath.Join(s.Dir, name), billing.Unread{Dir: s.Dir}, nil
 		}
 	}
 	var matches []string
-	err = s.Walk(func(path string) error {
+	markdown, err := s.Walk(func(path string) error {
 		for _, name := range names {
 			if filepath.Base(path) == name {
 				matches = append(matches, path)
@@ -277,19 +261,16 @@
 		}
 		return nil
 	})
+	unread := billing.Unread{Dir: s.Dir, Markdown: markdown}
 	if err != nil {
-		return "", err
+		return "", unread, err
 	}
 	switch len(matches) {
 	case 0:
-		return "", nil
+		return "", unread, nil
 	case 1:
-		return matches[0], nil
+		return matches[0], unread, nil
 	}
 	sort.Strings(matches)
-	return "", fmt.Errorf("%s: multiple archived invoice YAML files match %s; pass the YAML path explicitly: %s", s.Dir, names[0], strings.Join(matches, ", "))
-}
-
-func billingLink(e Edit) invoice.ArchiveLink {
-	return invoice.ArchiveLink{ArchivePath: invoice.Text(e.Target), ArchiveReplacePath: invoice.Text(e.Replace)}
+	return "", unread, fmt.Errorf("%s: multiple archived invoice YAML files match %s; pass the YAML path explicitly: %s", s.Dir, names[0], strings.Join(matches, ", "))
 }
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/archive/archive.go proto/internal/archive/archive.go
--- base/internal/archive/archive.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/archive/archive.go	2026-10-09 11:48:02.138723252 +0000
@@ -41,48 +41,59 @@
 }
 
 // Walk calls visit for every archived invoice file below the root, in the
-// lexical order of filepath.WalkDir. A missing or empty Dir has no files. A
-// Dir that is not a directory is an error. Backups in the history directory
-// are not archived invoices and are skipped.
-func (s Store) Walk(visit func(path string) error) error {
+// lexical order of filepath.WalkDir, and returns the Markdown files it
+// skipped. A missing or empty Dir has no files. A Dir that is not a
+// directory is an error. Backups in the history directory are not archived
+// invoices and are skipped.
+func (s Store) Walk(visit func(path string) error) ([]string, error) {
 	return s.walk(func(path, _ string) error { return visit(path) })
 }
 
-func (s Store) walk(visit func(path, rel string) error) error {
+func (s Store) walk(visit func(path, rel string) error) ([]string, error) {
 	if strings.TrimSpace(s.Dir) == "" {
-		return nil
+		return nil, nil
 	}
 	info, err := os.Stat(s.Dir)
 	if errors.Is(err, os.ErrNotExist) {
-		return nil
+		return nil, nil
 	}
 	if err != nil {
-		return err
+		return nil, err
 	}
 	if !info.IsDir() {
-		return fmt.Errorf("%s: archive.dir must point to a directory", s.Dir)
+		return nil, fmt.Errorf("%s: archive.dir must point to a directory", s.Dir)
 	}
+	var markdown []string
 
 	root, err := filepath.EvalSymlinks(s.Dir)
 	if err != nil {
 		root = s.Dir
 	}
-	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
+	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
 		if walkErr != nil {
 			return walkErr
 		}
 		if entry.IsDir() && isHistoryDir(root, path) {
 			return filepath.SkipDir
 		}
-		if entry.IsDir() || !isInvoiceFile(path) {
+		if entry.IsDir() {
 			return nil
 		}
 		rel, err := filepath.Rel(root, path)
 		if err != nil {
 			return err
 		}
+		switch {
+		case isMarkdown(path):
+			markdown = append(markdown, filepath.Join(s.Dir, rel))
+			return nil
+		case !isInvoiceFile(path):
+			return nil
+		}
 		return visit(filepath.Join(s.Dir, rel), rel)
 	})
+	sort.Strings(markdown)
+	return markdown, err
 }
 
 // Reader reads what the archived invoice at path says about itself. ok is
@@ -91,22 +102,10 @@
 type Reader func(path string) (entry billing.ArchiveEntry, ok bool, err error)
 
 // List reads every archived invoice with read and returns them sorted by
-// Filename.
-func (s Store) List(read Reader) ([]billing.ArchiveEntry, error) {
-	entries, err := s.entries(read)
-	if err != nil {
-		return nil, err
-	}
-	sort.Slice(entries, func(i, j int) bool {
-		return entries[i].Filename < entries[j].Filename
-	})
-	return entries, nil
-}
-
-// entries reads every archived invoice with read, in the order of Walk.
-func (s Store) entries(read Reader) ([]billing.ArchiveEntry, error) {
+// Filename, and what the archive holds that invox no longer reads.
+func (s Store) List(read Reader) ([]billing.ArchiveEntry, billing.Unread, error) {
 	entries := make([]billing.ArchiveEntry, 0)
-	err := s.walk(func(path, rel string) error {
+	markdown, err := s.walk(func(path, rel string) error {
 		entry, ok, err := read(path)
 		if err != nil || !ok {
 			return err
@@ -115,10 +114,14 @@
 		entries = append(entries, entry)
 		return nil
 	})
+	unread := billing.Unread{Dir: s.Dir, Markdown: markdown}
 	if err != nil {
-		return nil, err
+		return nil, unread, err
 	}
-	return entries, nil
+	sort.Slice(entries, func(i, j int) bool {
+		return entries[i].Filename < entries[j].Filename
+	})
+	return entries, unread, nil
 }
 
 // Target is a file inside the archive, named relative to the root.
@@ -170,30 +173,10 @@
 	if info.IsDir() {
 		return Target{}, fmt.Errorf("%s: archived invoice must be a file", target.Path)
 	}
-	return target, nil
-}
-
-// Edit describes the working copy `archive edit` makes of a Target.
-type Edit struct {
-	// Filename is the working copy's file name.
-	Filename string
-	// Target is where re-archiving the working copy writes it, relative to
-	// the root.
-	Target string
-	// Replace is the archived file the working copy supersedes, relative to
-	// the root, or "" when that is Target itself.
-	Replace string
-}
-
-// Edit returns the working copy of t. A Markdown archived invoice is edited
-// as YAML and re-archived as YAML, replacing the Markdown original.
-func (t Target) Edit() Edit {
-	rel := filepath.Clean(t.Rel)
-	if isMarkdown(rel) {
-		yamlRel := strings.TrimSuffix(rel, filepath.Ext(rel)) + ".yaml"
-		return Edit{Filename: filepath.Base(yamlRel), Target: yamlRel, Replace: rel}
+	if isMarkdown(target.Path) {
+		return Target{}, fmt.Errorf("%s is a Markdown invoice, which invox no longer reads; convert it to .yaml", target.Path)
 	}
-	return Edit{Filename: filepath.Base(rel), Target: rel}
+	return target, nil
 }
 
 // Backup copies each archived file to
@@ -285,16 +268,18 @@
 }
 
 // isInvoiceFile reports whether path has the extension of an archived
-// invoice: YAML, or Markdown with the invoice as front matter.
+// invoice.
 func isInvoiceFile(path string) bool {
 	switch strings.ToLower(filepath.Ext(path)) {
 	case ".yaml", ".yml":
 		return true
 	default:
-		return isMarkdown(path)
+		return false
 	}
 }
 
+// isMarkdown reports whether path is a Markdown file, which older versions
+// of invox read as an archived invoice with its front matter.
 func isMarkdown(path string) bool {
 	switch strings.ToLower(filepath.Ext(path)) {
 	case ".md", ".markdown":
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/archive.go proto/internal/billing/archive.go
--- base/internal/billing/archive.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/archive.go	2026-10-09 11:55:14.154942791 +0000
@@ -3,7 +3,6 @@
 import (
 	"errors"
 	"fmt"
-	"sort"
 	"strings"
 	"time"
 
@@ -47,51 +46,47 @@
 }
 
 func (s *Service) archive(path string, opts ArchiveOptions) (ArchiveResult, error) {
-	head, err := s.headWithInvoice(path)
-	if err != nil {
+	inv, err := s.Invoices.Load(path)
+	// Values that do not decode read as unset, except the header itself.
+	if err := keepDecodeErrors(err, func(e *DecodeError) bool { return e.Field == "invoice" }); err != nil {
 		return ArchiveResult{}, err
 	}
-	status := head.Status
+	if inv.Header == nil {
+		return ArchiveResult{}, fmt.Errorf("%s: missing `invoice` mapping", path)
+	}
+	status := invoice.Status(inv.Header.Status.Trim())
 	if opts.AssumeBuilt {
 		status, _ = status.Apply(invoice.Building)
 	}
-	dir, err := s.Archives.Dir()
-	if err != nil {
-		return ArchiveResult{}, err
-	}
-	if strings.TrimSpace(dir) == "" {
-		return ArchiveResult{}, errors.New("archive directory is unavailable")
-	}
-	if err := archivable(path, head, status); err != nil {
+	if err := archivable(path, inv, status); err != nil {
 		return ArchiveResult{}, err
 	}
-	place, err := s.Archives.Place(path, head)
+	unread, err := s.numberUnique(path, inv)
 	if err != nil {
 		return ArchiveResult{}, err
 	}
-	if err := s.numberUnique(path, head); err != nil {
-		return ArchiveResult{}, err
-	}
-	return s.Archives.Add(path, place, AddOptions{
+	result, err := s.Archives.Add(path, inv, AddOptions{
 		Replace: opts.Replace,
 		DryRun:  opts.DryRun,
-		Now:     s.Now(),
 		Change: func(inv *invoice.Invoice) error {
 			inv.Header.Status = invoice.Text(invoice.Archived)
 			inv.Archive = nil
 			return nil
 		},
 	})
+	result.Unread = unread
+	return result, err
 }
 
-// archivable returns why the invoice at path, whose head is head, cannot be
-// archived with status: a working copy is re-archived when editing or
-// built, any other invoice archived when built.
-func archivable(path string, head Head, status invoice.Status) error {
+// archivable returns why the invoice at path cannot be archived with
+// status: a working copy is re-archived when editing or built, any other
+// invoice archived when built.
+func archivable(path string, inv invoice.Invoice, status invoice.Status) error {
+	workingCopy := inv.Archive != nil && inv.Archive.ArchivePath.IsSet()
 	switch {
-	case head.WorkingCopy() && !status.Allows(invoice.Rearchiving):
+	case workingCopy && !status.Allows(invoice.Rearchiving):
 		return fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", path, status)
-	case head.WorkingCopy():
+	case workingCopy:
 		return nil
 	case status == "":
 		return fmt.Errorf("%s: invoice.status: missing value", path)
@@ -101,70 +96,19 @@
 	return nil
 }
 
-// headWithInvoice reads the head of the invoice at path, which must have an
-// `invoice` mapping. Values that do not decode read as unset.
-func (s *Service) headWithInvoice(path string) (Head, error) {
-	head, err := s.Invoices.Head(path)
-	if err != nil && !isDecodeError(err) {
-		return Head{}, err
-	}
-	if err := requireHeader(path, head); err != nil {
-		return Head{}, err
-	}
-	return head, nil
-}
-
-// requireHeader returns why the `invoice` key of the invoice at path is not
-// a mapping, or nil.
-func requireHeader(path string, head Head) error {
-	switch head.Header {
-	case HeaderMissing:
-		return fmt.Errorf("%s: missing `invoice` mapping", path)
-	case HeaderOther:
-		return fmt.Errorf("%s: `invoice` must be a mapping", path)
-	}
-	return nil
-}
-
-// CheckNumberUnique returns a *invoice.DuplicateInvoiceNumberError when
-// the invoice at path uses a number that an archived invoice already has.
-// The archived file the invoice was opened from (`archive edit`) does not
-// count as a duplicate.
-func (s *Service) CheckNumberUnique(path string) error {
-	head, err := s.Invoices.Head(path)
-	if err != nil && !isDecodeError(err) {
-		return err
-	}
-	if !head.HasHeader {
-		return nil
-	}
-	return s.numberUnique(path, head)
-}
-
 // numberUnique returns a *invoice.DuplicateInvoiceNumberError when an
-// archived invoice other than the one the invoice at path replaces has its
-// number.
-func (s *Service) numberUnique(path string, head Head) error {
-	archived, err := s.Archives.Duplicate(path, head)
+// archived invoice other than the one inv replaces has its number.
+func (s *Service) numberUnique(path string, inv invoice.Invoice) (Unread, error) {
+	archived, unread, err := s.Archives.Duplicate(path, inv)
 	if err != nil || archived == "" {
-		return err
+		return unread, err
 	}
-	return &invoice.DuplicateInvoiceNumberError{InvoicePath: path, InvoiceNumber: head.Number, ArchivedPath: archived}
-}
-
-// archiveEntries returns the archived invoices sorted by Filename.
-func (s *Service) archiveEntries() ([]ArchiveEntry, error) {
-	entries, err := s.Archives.Entries()
-	if err != nil {
-		return nil, err
-	}
-	sort.Slice(entries, func(i, j int) bool { return entries[i].Filename < entries[j].Filename })
-	return entries, nil
+	return unread, &invoice.DuplicateInvoiceNumberError{InvoicePath: path, InvoiceNumber: inv.Header.Number.Trim(), ArchivedPath: archived}
 }
 
 // latestArchived returns the archived invoice of customerID issued last.
 func (s *Service) latestArchived(customerID string) (string, bool, error) {
-	entries, err := s.archiveEntries()
+	entries, _, err := s.Archives.Entries()
 	if err != nil {
 		return "", false, err
 	}
@@ -210,22 +154,18 @@
 
 // ArchiveList is the archive's content.
 type ArchiveList struct {
-	// Dir is the archive directory, "" when there is none.
-	Dir     string
 	Entries []ArchiveEntry
+	// Unread names the archive directory and what the walk could not read.
+	Unread Unread
 }
 
 // ListArchive lists the archived invoices sorted by file name.
 func (s *Service) ListArchive() (ArchiveList, error) {
-	entries, err := s.archiveEntries()
+	entries, unread, err := s.Archives.Entries()
 	if err != nil {
 		return ArchiveList{}, err
 	}
-	dir, err := s.Archives.Dir()
-	if err != nil {
-		return ArchiveList{}, err
-	}
-	return ArchiveList{Dir: dir, Entries: entries}, nil
+	return ArchiveList{Entries: entries, Unread: unread}, nil
 }
 
 // EditOptions control EditArchived.
@@ -252,23 +192,14 @@
 	}
 	// The working copy keeps keys invox does not know, so opening an
 	// archived invoice never fails over them; validate reports them.
-	archived, err := s.Invoices.ArchivedHead(checkout.Archived)
-	if err != nil && !isDecodeError(err) {
-		return Edited{}, err
-	}
+	archived, err := s.Invoices.Load(checkout.Archived)
 	if err := lenient(err); err != nil {
 		return Edited{}, err
 	}
-	if err := requireHeader(checkout.Archived, archived); err != nil {
-		return Edited{}, err
+	if archived.Header == nil {
+		return Edited{}, fmt.Errorf("%s: missing `invoice` mapping", checkout.Archived)
 	}
-	if _, err := s.Invoices.Destination(checkout.Path, workDir, "", opts.Overwrite); err != nil {
-		return Edited{}, err
-	}
-	if err := s.refuseArchivedOverwrite(checkout.Path, opts.Overwrite); err != nil {
-		return Edited{}, err
-	}
-	err = s.Invoices.Create(checkout.Path, checkout.Archived, invoice.Invoice{
+	_, err = s.Invoices.Create(checkout.Path, checkout.Archived, invoice.Invoice{
 		Header:  &invoice.Header{Status: invoice.Text(invoice.Editing)},
 		Archive: &checkout.Link,
 	}, CreateOptions{Overwrite: opts.Overwrite, DryRun: opts.DryRun, Check: CheckNone})
@@ -277,20 +208,3 @@
 	}
 	return Edited{Path: checkout.Path, Archived: checkout.Archived}, nil
 }
-
-// refuseArchivedOverwrite returns an *ArchivedOutputError when overwrite
-// would replace an existing file inside the archive directory: an archived
-// invoice is replaced only by re-archiving, which keeps a backup.
-func (s *Service) refuseArchivedOverwrite(path string, overwrite bool) error {
-	if !overwrite {
-		return nil
-	}
-	protected, err := s.Archives.Protects(path)
-	if err != nil {
-		return err
-	}
-	if protected {
-		return &ArchivedOutputError{Path: path}
-	}
-	return nil
-}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/directory.go proto/internal/billing/directory.go
--- base/internal/billing/directory.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/directory.go	2026-10-09 11:52:32.216552483 +0000
@@ -53,32 +53,34 @@
 	// Dir is the directory `template list` reads.
 	Dir       string
 	Templates []Template
+	// Default is the template a command uses when none is named, "" when
+	// there is none.
+	Default string
 }
 
-// ListTemplates lists the .tex files of the template catalog.
-func (s *Service) ListTemplates() (TemplateList, error) {
+// ListTemplates lists the .tex files of the template catalog and, with
+// withDefault, the default template, which takes a search from the working
+// directory.
+func (s *Service) ListTemplates(withDefault bool) (TemplateList, error) {
 	templates, dir, err := s.Directory.Templates()
-	if err != nil {
-		return TemplateList{}, err
+	if err != nil || !withDefault {
+		return TemplateList{Dir: dir, Templates: templates}, err
 	}
-	return TemplateList{Dir: dir, Templates: templates}, nil
-}
-
-// DefaultTemplate returns the template a command uses when none is named,
-// "" when there is none.
-func (s *Service) DefaultTemplate() (string, error) {
-	path, err := s.Directory.Locate(TemplateFile)
+	def, err := s.Directory.Locate(TemplateFile)
 	var notFound *FileNotFoundError
 	if errors.As(err, &notFound) {
-		return "", nil
+		def, err = "", nil
 	}
-	return path, err
+	if err != nil {
+		return TemplateList{}, err
+	}
+	return TemplateList{Dir: dir, Templates: templates, Default: def}, nil
 }
 
-// Paths reports where each file comes from for a command run in start, in a
-// fixed order: config-dir, config, the support files, then archive.
-func (s *Service) Paths(start string) ([]PathReport, error) {
-	return s.Directory.Paths(start)
+// Paths reports where each file comes from for this run, in a fixed order:
+// config-dir, config, the support files, then archive.
+func (s *Service) Paths() ([]PathReport, error) {
+	return s.Directory.Paths()
 }
 
 // InitResult is what Init set up.
@@ -97,18 +99,6 @@
 	return InitResult{ConfigDir: dir, Files: files}, nil
 }
 
-// LegacyFiles returns the files of the deprecated config directory that the
-// config directory lacks.
-func (s *Service) LegacyFiles() ([]string, error) {
-	return s.Directory.LegacyFiles()
-}
-
-// CopyLegacyFiles copies LegacyFiles into the config directory and returns
-// them. It never replaces a file and leaves the legacy directory as it is.
-func (s *Service) CopyLegacyFiles() ([]string, error) {
-	return s.Directory.CopyLegacy()
-}
-
 // EditablePath returns the file f for an editor: customers.yaml where the
 // commands find it, or config.yaml, created from its template when missing.
 func (s *Service) EditablePath(f File) (string, error) {
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/email.go proto/internal/billing/email.go
--- base/internal/billing/email.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/email.go	2026-10-09 11:43:50.734321601 +0000
@@ -36,8 +36,12 @@
 	CustomerID string
 	Number     string
 	Message    Message
-	// Draft is where the draft went; unset in a dry run.
-	Draft Draft
+	// Draft is the draft's file, "" when a mail app holds it or in a dry
+	// run.
+	Draft string
+	// Unread is what the PDF's invoice lookup in the archive could not
+	// read.
+	Unread Unread
 }
 
 // DraftEmail drafts an email with the invoice's PDF attached, for an
@@ -47,7 +51,7 @@
 	if err != nil {
 		return EmailResult{}, err
 	}
-	invoicePath, err := s.emailInvoice(req)
+	invoicePath, unread, err := s.emailInvoice(req)
 	if err != nil {
 		return EmailResult{}, err
 	}
@@ -63,9 +67,6 @@
 		}
 		return EmailResult{}, fmt.Errorf("%s: invoice.status must be `built` or `archived` before creating an email draft, got `%s`", invoicePath, status)
 	}
-	if err := s.Mailer.CheckAttachment(pdfPath); err != nil {
-		return EmailResult{}, fmt.Errorf("read %s: %w", pdfPath, err)
-	}
 
 	recipient := strings.TrimSpace(req.To)
 	if recipient == "" {
@@ -89,6 +90,7 @@
 	}
 
 	result := EmailResult{
+		Unread:     unread,
 		CustomerID: inv.CustomerID,
 		Number:     inv.InvoiceNumber,
 		Message: Message{
@@ -104,15 +106,7 @@
 			Date:        s.Now(),
 		},
 	}
-	if req.DryRun {
-		if req.Keep {
-			if err := s.Mailer.Check(result.Message); err != nil {
-				return EmailResult{}, err
-			}
-		}
-		return result, nil
-	}
-	result.Draft, err = s.Mailer.Draft(ctx, result.Message)
+	result.Draft, err = s.Mailer.Draft(ctx, result.Message, req.DryRun)
 	if err != nil {
 		return EmailResult{}, err
 	}
@@ -121,18 +115,18 @@
 
 // emailInvoice returns the invoice of req: the one it names, else the one
 // its PDF was built from.
-func (s *Service) emailInvoice(req EmailRequest) (string, error) {
+func (s *Service) emailInvoice(req EmailRequest) (string, Unread, error) {
 	if req.Invoice != "" {
-		return req.Invoice, nil
+		return req.Invoice, Unread{}, nil
 	}
-	invoicePath, err := s.Archives.Source(req.FromPDF)
+	invoicePath, unread, err := s.Archives.Source(req.FromPDF)
 	if err != nil {
-		return "", err
+		return "", unread, err
 	}
 	if invoicePath == "" {
-		return "", fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", req.FromPDF)
+		return "", unread, fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", req.FromPDF)
 	}
-	return invoicePath, nil
+	return invoicePath, unread, nil
 }
 
 // emailFields is what the email placeholders stand for in ctx.
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/errors.go proto/internal/billing/errors.go
--- base/internal/billing/errors.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/errors.go	2026-10-09 11:50:27.005136287 +0000
@@ -171,13 +171,19 @@
 // lenient drops the unknown-key problems from err, the joined *DecodeError
 // values of a strict decode, and returns what is left, or nil.
 func lenient(err error) error {
+	return keepDecodeErrors(err, func(e *DecodeError) bool { return !e.UnknownKey })
+}
+
+// keepDecodeErrors returns err without the *DecodeError values keep
+// rejects, or nil when nothing is left.
+func keepDecodeErrors(err error, keep func(*DecodeError) bool) error {
 	var kept []error
 	var walk func(error)
 	walk = func(err error) {
 		switch e := err.(type) {
 		case nil:
 		case *DecodeError:
-			if !e.UnknownKey {
+			if keep(e) {
 				kept = append(kept, e)
 			}
 		case interface{ Unwrap() []error }:
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/new.go proto/internal/billing/new.go
--- base/internal/billing/new.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/new.go	2026-10-09 11:42:44.880184046 +0000
@@ -32,6 +32,8 @@
 	// Skipped are archived invoices of the customer whose numbers do not
 	// match numbering.pattern, so they did not count towards Number.
 	Skipped []string
+	// Unread is what the archive walk could not read.
+	Unread Unread
 }
 
 // New drafts the next invoice of a customer from invoice_defaults.yaml or
@@ -50,12 +52,6 @@
 	if err != nil {
 		return NewResult{}, err
 	}
-
-	if req.Output != "" {
-		if _, err := s.Invoices.Destination(req.Output, req.WorkDir, "", req.Overwrite); err != nil {
-			return NewResult{}, err
-		}
-	}
 	customer, err := s.Directory.Customer(req.CustomerID)
 	if err != nil {
 		return NewResult{}, err
@@ -78,11 +74,7 @@
 	if err != nil {
 		return NewResult{}, err
 	}
-	number, skipped, err := s.NextNumber(req.CustomerID, issueDate, customer, draftCounter)
-	if err != nil {
-		return NewResult{}, err
-	}
-	output, err := s.Invoices.Destination(req.Output, req.WorkDir, number, req.Overwrite)
+	number, skipped, unread, err := s.nextNumber(req.CustomerID, issueDate, customer, draftCounter)
 	if err != nil {
 		return NewResult{}, err
 	}
@@ -108,20 +100,17 @@
 		},
 		Positions: []invoice.Position{},
 	}
-	if err := s.refuseArchivedOverwrite(output, req.Overwrite); err != nil {
-		return NewResult{}, err
-	}
 	// An archived invoice is a record: keys it has that invox does not
 	// know are copied over as they are, for validate to report.
 	check := CheckStrict
 	if req.FromLast {
 		check = CheckLenient
 	}
-	err = s.Invoices.Create(output, from, draft, CreateOptions{Overwrite: req.Overwrite, DryRun: req.DryRun, Check: check})
+	output, err := s.Invoices.Create(req.Output, from, draft, CreateOptions{Dir: req.WorkDir, Overwrite: req.Overwrite, DryRun: req.DryRun, Check: check})
 	if err != nil {
 		return NewResult{}, err
 	}
-	return NewResult{Number: number, Path: output, Skipped: skipped}, nil
+	return NewResult{Number: number, Path: output, Skipped: skipped, Unread: unread}, nil
 }
 
 // newSource returns the document a new invoice starts from, after checking
@@ -140,7 +129,7 @@
 	if !ok {
 		return "", fmt.Errorf("no archived invoice found for customer_id `%s`", customerID)
 	}
-	if _, err := s.Invoices.ArchivedHead(archivePath); err != nil && !isDecodeError(err) {
+	if _, err := s.Invoices.Load(archivePath); err != nil && !isDecodeError(err) {
 		return "", err
 	}
 	return archivePath, nil
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/numbering.go proto/internal/billing/numbering.go
--- base/internal/billing/numbering.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/numbering.go	2026-10-09 11:59:25.671733507 +0000
@@ -20,80 +20,73 @@
 	return settings.Numbering, nil
 }
 
-// NextNumber returns the next invoice number of the customer on issueDate:
+// nextNumber returns the next invoice number of the customer on issueDate:
 // above every archived invoice's counter, minimumCounter and the start
 // before it. skipped are the customer's archived invoices from the same
 // period whose numbers do not match the pattern.
-func (s *Service) NextNumber(customerID, issueDate string, customer invoice.Customer, minimumCounter int64) (string, []string, error) {
+func (s *Service) nextNumber(customerID, issueDate string, customer invoice.Customer, minimumCounter int64) (string, []string, Unread, error) {
 	settings, err := s.numberingSettings()
 	if err != nil {
-		return "", nil, err
+		return "", nil, Unread{}, err
 	}
 	start, err := numberingStart(customerID, customer, settings.Start)
 	if err != nil {
-		return "", nil, err
+		return "", nil, Unread{}, err
 	}
-	highest, skipped, err := s.highestArchivedCounter(settings.Pattern, customerID, issueDate, customer)
+	entries, unread, err := s.Archives.Entries()
 	if err != nil {
-		return "", nil, err
-	}
-	next := numbering.Next(start, max(highest, minimumCounter))
-	number, err := numbering.Format(settings.Pattern, customerID, customer.Numbering.Code.Trim(), issueDate, next)
-	if err != nil {
-		return "", nil, err
-	}
-	return number, skipped, nil
-}
-
-func numberingStart(customerID string, customer invoice.Customer, globalStart int64) (int64, error) {
-	if !customer.Numbering.Start.IsSet() {
-		return globalStart, nil
-	}
-	if start := customer.Numbering.Start.Int(); start > 0 {
-		return start, nil
-	}
-	return 0, fmt.Errorf("customers.%s.numbering.start: must be >= 1", customerID)
-}
-
-func (s *Service) highestArchivedCounter(pattern, customerID, issueDate string, customer invoice.Customer) (int64, []string, error) {
-	entries, err := s.Archives.Entries()
-	if err != nil {
-		return 0, nil, err
+		return "", nil, Unread{}, err
 	}
+	code := customer.Numbering.Code.Trim()
 	var highest int64
 	var skipped []string
 	for _, entry := range entries {
 		if entry.Number == "" {
 			continue
 		}
-		counter, err := numbering.Parse(pattern, entry.Number, customerID, customer.Numbering.Code.Trim(), issueDate)
+		counter, err := numbering.Parse(settings.Pattern, entry.Number, customerID, code, issueDate)
 		if err != nil {
-			if entry.CustomerID == customerID && numbering.InPeriod(pattern, entry.IssueDate, issueDate) {
+			if entry.CustomerID == customerID && numbering.InPeriod(settings.Pattern, entry.IssueDate, issueDate) {
 				skipped = append(skipped, entry.Path)
 			}
 			continue
 		}
 		highest = max(highest, counter)
 	}
-	return highest, skipped, nil
+	number, err := numbering.Format(settings.Pattern, customerID, code, issueDate, numbering.Next(start, max(highest, minimumCounter)))
+	if err != nil {
+		return "", nil, Unread{}, err
+	}
+	return number, skipped, unread, nil
+}
+
+func numberingStart(customerID string, customer invoice.Customer, globalStart int64) (int64, error) {
+	if !customer.Numbering.Start.IsSet() {
+		return globalStart, nil
+	}
+	if start := customer.Numbering.Start.Int(); start > 0 {
+		return start, nil
+	}
+	return 0, fmt.Errorf("customers.%s.numbering.start: must be >= 1", customerID)
 }
 
 // highestDraftCounter returns the highest counter used by unarchived
 // invoices (status draft or built) where a new invoice goes, workDir and
-// the directory of output, so that two drafts created before either is
-// archived do not get the same number. Files that cannot be read or do not
-// match the numbering pattern are ignored.
+// next to output, so that two drafts created before either is archived do
+// not get the same number. Files that cannot be read or do not match the
+// numbering pattern are ignored.
 func (s *Service) highestDraftCounter(workDir, output, customerID, issueDate string, customer invoice.Customer) (int64, error) {
 	settings, err := s.numberingSettings()
 	if err != nil {
 		return 0, err
 	}
 	var highest int64
-	for _, head := range s.Invoices.Drafts(workDir, output) {
-		if !head.Status.Allows(invoice.Numbering) || head.Number == "" {
+	for _, draft := range s.Invoices.Drafts(workDir, output) {
+		number := draft.Header.Number.Trim()
+		if !invoice.Status(draft.Header.Status.Trim()).Allows(invoice.Numbering) || number == "" {
 			continue
 		}
-		counter, err := numbering.Parse(settings.Pattern, head.Number, customerID, customer.Numbering.Code.Trim(), issueDate)
+		counter, err := numbering.Parse(settings.Pattern, number, customerID, customer.Numbering.Code.Trim(), issueDate)
 		if err != nil {
 			continue
 		}
@@ -110,6 +103,8 @@
 	// Skipped are archived invoices of the customer whose numbers do not
 	// match numbering.pattern, so they did not count towards NewNumber.
 	Skipped []string
+	// Unread is what the archive walk could not read.
+	Unread Unread
 }
 
 // Increment writes the next invoice number into the invoice at path. With
@@ -118,26 +113,27 @@
 	if _, err := s.Directory.Locate(CustomersFile); err != nil {
 		return IncrementResult{}, err
 	}
-	head, err := s.Invoices.Head(path)
-	if err != nil {
+	inv, err := s.Invoices.Load(path)
+	// Only the fields numbering reads have to decode.
+	read := map[string]bool{"customer_id": true, "invoice.number": true, "invoice.issue_date": true}
+	if err := keepDecodeErrors(err, func(e *DecodeError) bool { return read[e.Field] }); err != nil {
 		return IncrementResult{}, err
 	}
+	customerID := inv.CustomerID.Trim()
 	switch {
-	case head.CustomerID == "":
+	case customerID == "":
 		return IncrementResult{}, fmt.Errorf("%s: missing `customer_id`", path)
-	case !head.HasHeader:
+	case inv.Header == nil:
 		return IncrementResult{}, fmt.Errorf("%s: missing `invoice` mapping", path)
-	case head.IssueDate == "":
+	case !inv.Header.IssueDate.IsSet():
 		return IncrementResult{}, fmt.Errorf("%s: invoice.issue_date: missing value", path)
 	}
-	if _, err := invoice.ParseDate(head.IssueDate); err != nil {
-		return IncrementResult{}, fmt.Errorf("%s: invoice.issue_date: expected YYYY-MM-DD, got `%s`", path, head.IssueDate)
-	}
-	if head.Number == "" {
+	issueDate, number := inv.Header.IssueDate.String(), inv.Header.Number.Trim()
+	if number == "" {
 		return IncrementResult{}, fmt.Errorf("%s: invoice.number: missing value", path)
 	}
 
-	customer, err := s.Directory.Customer(head.CustomerID)
+	customer, err := s.Directory.Customer(customerID)
 	if err != nil {
 		return IncrementResult{}, err
 	}
@@ -145,15 +141,15 @@
 	if err != nil {
 		return IncrementResult{}, err
 	}
-	current, err := numbering.Parse(settings.Pattern, head.Number, head.CustomerID, customer.Numbering.Code.Trim(), head.IssueDate)
+	current, err := numbering.Parse(settings.Pattern, number, customerID, customer.Numbering.Code.Trim(), issueDate)
 	if err != nil {
 		return IncrementResult{}, err
 	}
-	next, skipped, err := s.NextNumber(head.CustomerID, head.IssueDate, customer, current)
+	next, skipped, unread, err := s.nextNumber(customerID, issueDate, customer, current)
 	if err != nil {
 		return IncrementResult{}, err
 	}
-	result := IncrementResult{CustomerID: head.CustomerID, OldNumber: head.Number, NewNumber: next, Skipped: skipped}
+	result := IncrementResult{CustomerID: customerID, OldNumber: number, NewNumber: next, Skipped: skipped, Unread: unread}
 	if dryRun {
 		return result, nil
 	}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/ports.go proto/internal/billing/ports.go
--- base/internal/billing/ports.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/ports.go	2026-10-09 11:42:05.978731398 +0000
@@ -22,38 +22,6 @@
 	return [...]string{"customers", "issuer", "defaults", "template", "config"}[f]
 }
 
-// Head is what numbering and the archive read of an invoice, as written. It
-// is decoded leniently, so a file with keys invox does not know, or with
-// values it cannot read, still counts.
-type Head struct {
-	CustomerID string
-	Number     string
-	IssueDate  string
-	Status     invoice.Status
-	// HasHeader is false when the `invoice` key is missing or null.
-	HasHeader bool
-	// Header is what the `invoice` key holds as written. Archiving rewrites
-	// that mapping in place, so an alias to a mapping is HeaderOther.
-	Header HeaderShape
-	// ArchivePath and ReplacePath are the `_invox` link of a working copy
-	// from `archive edit`, relative to the archive, "" when it has none.
-	ArchivePath string
-	ReplacePath string
-}
-
-// WorkingCopy reports whether h is a working copy from `archive edit`,
-// which re-archiving writes back over the archived files it names.
-func (h Head) WorkingCopy() bool { return h.ArchivePath != "" }
-
-// HeaderShape is what the `invoice` key of an invoice file holds.
-type HeaderShape int
-
-const (
-	HeaderMissing HeaderShape = iota // no `invoice` key
-	HeaderMapping                    // a mapping
-	HeaderOther                      // null, a scalar, a list or an alias
-)
-
 // Check says how Create checks the invoice it writes.
 type Check int
 
@@ -68,42 +36,34 @@
 
 // CreateOptions control Invoices.Create.
 type CreateOptions struct {
-	// Overwrite replaces an existing file.
+	// Dir holds the new invoice when Create is given no path: it is named
+	// after the invoice's number there.
+	Dir string
+	// Overwrite replaces an existing file, unless it is an archived
+	// invoice: those change only by re-archiving.
 	Overwrite bool
 	// DryRun runs every check and writes nothing.
 	DryRun bool
 	Check  Check
 }
 
-// Invoices reads and writes invoice files. Update keeps the file's comments
-// and layout; TestInvoiceWritesKeepComments pins that.
+// Invoices reads and writes invoice files. Create and Update keep the
+// file's comments and layout; TestInvoiceWritesKeepComments pins that.
 type Invoices interface {
 	// Load decodes the invoice at path strictly. Values that do not fit
 	// come back as joined *DecodeError values, with the rest decoded.
 	Load(path string) (invoice.Invoice, error)
-	// ArchivedHead is Head for an archived invoice, which may be the front
-	// matter of a Markdown file. It decodes the whole invoice strictly, as
-	// Load does, and returns those errors.
-	ArchivedHead(path string) (Head, error)
-	// Head reads what numbering and the archive need of the invoice at
-	// path. Values that do not decode are left unset and reported as
-	// *DecodeError values.
-	Head(path string) (Head, error)
-	// Drafts returns the invoices directly in workDir and, when output is
-	// set, in the directory output is in, best effort: files that cannot
-	// be read are left out.
-	Drafts(workDir, output string) []Head
-	// Destination returns the file a new invoice is written to: path, or
-	// when path is "", <number>.yaml in workDir. It returns an
-	// *OutputIsDirError when that is a directory and, unless overwrite is
-	// set, an *OutputExistsError when it exists.
-	Destination(path, workDir, number string, overwrite bool) (string, error)
-	// Create writes a new invoice to path from the document at from, which
-	// may be an archived invoice, keeping its comments and keys. Every
-	// customer and header field set in inv is written, as text. Positions
-	// is added when inv's is not nil and from has none. The `_invox` link
-	// is replaced by inv's.
-	Create(path, from string, inv invoice.Invoice, opts CreateOptions) error
+	// Drafts returns, best effort, the invoices next to where a new one
+	// goes: in workDir and, when output is set, next to output. Only their
+	// customer and header are read.
+	Drafts(workDir, output string) []invoice.Invoice
+	// Create writes inv as a new invoice to path, or when path is "" to
+	// opts.Dir, starting from the document at from and keeping its
+	// comments and keys, and returns the file. Every customer and header
+	// field set in inv is written. It returns an *OutputIsDirError for a
+	// directory, an *OutputExistsError for an existing file without
+	// opts.Overwrite, and an *ArchivedOutputError for an archived one.
+	Create(path, from string, inv invoice.Invoice, opts CreateOptions) (string, error)
 	// Update rewrites the invoice at path with change applied, writing
 	// back only the fields that changed.
 	Update(path string, change func(*invoice.Invoice) error) error
@@ -120,14 +80,11 @@
 	Lookup(id string, strict bool) (c invoice.Customer, ok bool, err error)
 }
 
-// Template is a LaTeX template.
+// Template is a LaTeX template: a name for listings and a reference the
+// Renderer reads.
 type Template struct {
 	Name string
 	Path string
-	// FindAsset returns the file or directory rel that the template uses:
-	// next to the template, else in the config directories. It returns ""
-	// when there is none.
-	FindAsset func(rel string, dir bool) string
 }
 
 // Source says where a resolved path came from.
@@ -138,7 +95,6 @@
 	SourceExplicit               // the config file the user named
 	SourceEnvDir                 // the config directory the user chose, or a file in it
 	SourceDefault                // the OS default directory, or a file in it
-	SourceLegacy                 // the deprecated invoice-tool directory, or a file in it
 	SourceProject                // found by the upward search from the working directory
 	SourceConfig                 // a paths.* or archive.dir setting in the config file
 )
@@ -150,23 +106,6 @@
 	Source Source
 }
 
-// Locations are where invox keeps its files by default, for help texts.
-// They come from the environment only; nothing is read.
-type Locations struct {
-	ConfigDir  string
-	ConfigFile string
-	Customers  string
-	Issuer     string
-	Defaults   string
-	Template   string
-	ArchiveDir string
-	// LegacyDir is the deprecated invoice-tool directory, "" when it is
-	// not read.
-	LegacyDir string
-	// ConfigTemplate is the text a new config.yaml starts with.
-	ConfigTemplate string
-}
-
 // InitFile is a file Init made sure exists.
 type InitFile struct {
 	Path    string
@@ -192,23 +131,15 @@
 	Template(ref string) (Template, error)
 	// Templates lists the template catalog and returns its directory.
 	Templates() ([]Template, string, error)
-	Paths(start string) ([]PathReport, error)
+	// Paths reports where each file comes from for this run, in a fixed
+	// order: config-dir, config, the support files, then archive.
+	Paths() ([]PathReport, error)
 	// EditablePath returns f for an editor, creating config.yaml from its
 	// template when f is ConfigFile and it does not exist.
 	EditablePath(f File) (string, error)
 	// Init creates the config directory and the starter files it lacks,
 	// and returns the directory.
 	Init() (string, []InitFile, error)
-	// LegacyFiles returns the files of the legacy directory, relative to
-	// it, that the config directory lacks.
-	LegacyFiles() ([]string, error)
-	// CopyLegacy copies LegacyFiles into the config directory and returns
-	// them.
-	CopyLegacy() ([]string, error)
-	// LegacyFilesUsed returns the files read from the legacy directory so
-	// far.
-	LegacyFilesUsed() []string
-	Locations() Locations
 }
 
 // ArchiveEntry is an archived invoice: where it is and what it says about
@@ -225,6 +156,15 @@
 	Number string
 }
 
+// Unread is what a walk of the archive left out: archived invoices stored
+// as Markdown, which invox no longer reads.
+type Unread struct {
+	// Dir is the archive directory, "" when there is none.
+	Dir string
+	// Markdown are the Markdown files, in file name order.
+	Markdown []string
+}
+
 // Backup is an archived file that re-archiving replaced, and where its
 // previous version was kept.
 type Backup struct {
@@ -240,19 +180,8 @@
 	Replaced []Backup
 	// HistoryDir is where the replaced files' previous versions are kept.
 	HistoryDir string
-}
-
-// Placement is where Archive.Add puts an invoice. Archive.Place makes it.
-type Placement struct {
-	// Path is the archived file to write.
-	Path string
-	// Overwrite allows Path to exist, when a working copy is re-archived.
-	Overwrite bool
-	// Remove is an archived file the invoice supersedes, removed after it
-	// is written, or "".
-	Remove string
-	// HistoryDir is where replaced files' previous versions are kept.
-	HistoryDir string
+	// Unread is what the duplicate number check could not read.
+	Unread Unread
 }
 
 // AddOptions control Archive.Add.
@@ -263,7 +192,6 @@
 	// DryRun runs every check and returns the result without writing. The
 	// result's backups have no BackupPath.
 	DryRun bool
-	Now    time.Time
 	// Change is applied to the invoice as it is archived, keeping its
 	// comments and layout, so the archived file is written once.
 	Change func(*invoice.Invoice) error
@@ -281,36 +209,27 @@
 
 // Archive holds finished invoices.
 type Archive interface {
-	// Entries reads every archived invoice, in the lexical order of a
-	// directory walk.
-	Entries() ([]ArchiveEntry, error)
-	// Dir returns the archive directory, "" when there is none.
-	Dir() (string, error)
-	// Place says where archiving the invoice at src, whose head is head,
-	// writes it: over the archived file a working copy names, else under
-	// src's name in the archive directory, which must not exist yet. It
-	// refuses src when it is that file already.
-	Place(src string, head Head) (Placement, error)
+	// Entries reads every archived invoice, sorted by Filename.
+	Entries() ([]ArchiveEntry, Unread, error)
 	// Duplicate returns the archived invoice, in file name order, that has
-	// head's number, other than src and, for a working copy, the archived
-	// files it replaces. It returns "" when there is none, when head has
-	// no number, or when there is no archive directory.
-	Duplicate(src string, head Head) (string, error)
-	// Add moves the invoice at src into the archive at p with opts.Change
-	// applied, in one write, after backing up the archived files it
-	// replaces. Without opts.Replace it refuses to replace any.
-	Add(src string, p Placement, opts AddOptions) (ArchiveResult, error)
+	// inv's number, other than src and the archived file a working copy
+	// replaces. It returns "" when there is none, when inv has no number,
+	// or when there is no archive directory.
+	Duplicate(src string, inv invoice.Invoice) (string, Unread, error)
+	// Add moves the invoice at src, whose content is inv, into the archive
+	// with opts.Change applied, in one write: over the archived file a
+	// working copy names, else under src's name, which must not exist yet.
+	// It backs up the archived file it replaces first, and refuses to
+	// replace one without opts.Replace.
+	Add(src string, inv invoice.Invoice, opts AddOptions) (ArchiveResult, error)
 	// Checkout resolves ref, an archived invoice relative to the archive
 	// directory, and says where its working copy in workDir goes.
 	Checkout(ref, workDir string) (Checkout, error)
-	// Protects reports whether path is an existing file inside the archive
-	// directory, which nothing but re-archiving overwrites.
-	Protects(path string) (bool, error)
-	// Source returns the invoice YAML file the PDF at pdf was built from:
-	// the one with its name next to it, else the one in the archive. It
+	// Source returns the invoice file the PDF at pdf was built from: the
+	// one with its name next to it, else the one in the archive. It
 	// returns "" when there is none, and an error when the archive has
 	// several.
-	Source(pdf string) (string, error)
+	Source(pdf string) (string, Unread, error)
 }
 
 // EPC is what a template's EPC QR code placeholders need.
@@ -328,10 +247,10 @@
 type Renderer interface {
 	// Render checks template t and fills it in. It writes nothing.
 	Render(t Template, inv *invoice.Context, epc EPC) (string, error)
-	// Write writes source to path and copies t's assets next to it.
+	// Write writes source to path with the assets t uses next to it.
 	Write(t Template, source, path string) error
-	// Build writes source with t's assets to a scratch directory, compiles
-	// it there with c, and copies the PDF to output.
+	// Build compiles source with c, in a scratch directory with t's
+	// assets, and writes the PDF to output.
 	Build(ctx context.Context, c Compiler, t Template, source, output string) error
 }
 
@@ -357,20 +276,10 @@
 	Date      time.Time
 }
 
-// Draft is where Mailer.Draft put the draft.
-type Draft struct {
-	// Path is the .eml file, "" when a mail app opened the draft.
-	Path string
-	// Discard removes a temporary draft; nil for one that is kept.
-	Discard func()
-}
-
-// Mailer drafts an email with the PDF attached.
+// Mailer drafts an email with the PDF attached and opens it for the user.
 type Mailer interface {
-	Draft(ctx context.Context, m Message) (Draft, error)
-	// Check runs the checks Draft runs on m.Output without writing.
-	Check(m Message) error
-	// CheckAttachment returns why the file at path cannot be attached, or
-	// nil.
-	CheckAttachment(path string) error
+	// Draft checks that m can be drafted and, unless dryRun is set, drafts
+	// and opens it. It returns the draft's file, "" when a mail app holds
+	// the draft or in a dry run.
+	Draft(ctx context.Context, m Message, dryRun bool) (string, error)
 }
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/render.go proto/internal/billing/render.go
--- base/internal/billing/render.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/render.go	2026-10-09 11:43:33.875623186 +0000
@@ -130,11 +130,15 @@
 // archived invoice keeps `archived`: rebuilding its PDF does not take it
 // out of the archive.
 func (s *Service) markBuilt(path string) error {
-	head, err := s.Invoices.Head(path)
+	inv, err := s.Invoices.Load(path)
 	if err != nil && !isDecodeError(err) {
 		return err
 	}
-	if next, _ := head.Status.Apply(invoice.Building); next != invoice.Built {
+	var status invoice.Status
+	if inv.Header != nil {
+		status = invoice.Status(inv.Header.Status.Trim())
+	}
+	if next, _ := status.Apply(invoice.Building); next != invoice.Built {
 		return nil
 	}
 	return s.Invoices.Update(path, func(inv *invoice.Invoice) error {
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/service.go proto/internal/billing/service.go
--- base/internal/billing/service.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/service.go	2026-10-09 11:48:02.202723250 +0000
@@ -34,14 +34,3 @@
 	Settings func() (Settings, error)
 	Now      func() time.Time
 }
-
-// Locations are where invox keeps its files by default.
-func (s *Service) Locations() Locations {
-	return s.Directory.Locations()
-}
-
-// LegacyFilesUsed returns the files read from the deprecated config
-// directory so far.
-func (s *Service) LegacyFilesUsed() []string {
-	return s.Directory.LegacyFilesUsed()
-}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/billing/validate.go proto/internal/billing/validate.go
--- base/internal/billing/validate.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/billing/validate.go	2026-10-09 11:43:33.876107976 +0000
@@ -14,26 +14,26 @@
 	// kept the archive from being checked. It never makes the invoice
 	// invalid.
 	Duplicate error
+	// Unread is what the duplicate number check could not read.
+	Unread Unread
 }
 
 // Validate loads the invoice at path with its customer and issuer and
 // checks them together.
 func (s *Service) Validate(path string) (ValidateResult, error) {
-	ctx, err := s.load(path)
+	customersPath, issuerPath, err := s.locateParties()
 	if err != nil {
 		return ValidateResult{}, err
 	}
-	return ValidateResult{Context: ctx, Duplicate: s.CheckNumberUnique(path)}, nil
-}
-
-// load locates customers.yaml and issuer.yaml and loads the invoice at path
-// with them.
-func (s *Service) load(path string) (*invoice.Context, error) {
-	customersPath, issuerPath, err := s.locateParties()
+	ctx, err := s.loadContext(customersPath, issuerPath, path)
 	if err != nil {
-		return nil, err
+		return ValidateResult{}, err
 	}
-	return s.loadContext(customersPath, issuerPath, path)
+	// loadContext has read the invoice already, so only the archive can
+	// fail here.
+	inv, _ := s.Invoices.Load(path)
+	unread, duplicate := s.numberUnique(path, inv)
+	return ValidateResult{Context: ctx, Duplicate: duplicate, Unread: unread}, nil
 }
 
 func (s *Service) locateParties() (string, string, error) {
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cli/cli.go proto/internal/cli/cli.go
--- base/internal/cli/cli.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cli/cli.go	2026-10-09 11:48:02.222723250 +0000
@@ -3,9 +3,6 @@
 import (
 	"context"
 	"errors"
-	"fmt"
-	"path/filepath"
-	"strings"
 	"syscall"
 
 	"github.com/spf13/cobra"
@@ -46,7 +43,6 @@
 	if err == nil {
 		err = helpErr()
 	}
-	warnLegacyFiles(f)
 	var configErr *billing.ConfigError
 	if f.ConfigFile != "" && errors.As(err, &configErr) {
 		err = &configFlagError{err: err, path: f.ConfigFile}
@@ -58,27 +54,3 @@
 	}
 	return exitCode(f.IOStreams, err)
 }
-
-// warnLegacyFiles prints one line when the command read files from the
-// deprecated config directory.
-func warnLegacyFiles(f *cmdutil.Factory) {
-	svc := f.Service(cmdutil.Files{})
-	used := svc.LegacyFilesUsed()
-	if len(used) == 0 {
-		return
-	}
-	locations := svc.Locations()
-	legacyDir := locations.LegacyDir
-	names := make([]string, len(used))
-	for i, path := range used {
-		names[i] = path
-		if rel, err := filepath.Rel(legacyDir, path); err == nil {
-			names[i] = rel
-		}
-	}
-	list, pronoun := names[0], "it"
-	if len(names) > 1 {
-		list, pronoun = strings.Join(names[:len(names)-1], ", ")+" and "+names[len(names)-1], "them"
-	}
-	fmt.Fprintf(f.IOStreams.ErrOut, "warning: using %s from deprecated config directory %s; run '%s init' to copy %s to %s\n", list, legacyDir, commandName, pronoun, locations.ConfigDir)
-}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cli/cmdutil/completion.go proto/internal/cli/cmdutil/completion.go
--- base/internal/cli/cmdutil/completion.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cli/cmdutil/completion.go	2026-10-09 11:52:32.217543723 +0000
@@ -3,8 +3,9 @@
 import (
 	"strings"
 
-	"github.com/0xboris/invox/internal/billing"
 	"github.com/spf13/cobra"
+
+	"github.com/0xboris/invox/internal/billing"
 )
 
 // The completion funcs below read the same files as the commands. Any error,
@@ -39,7 +40,7 @@
 // shows, and with file names when no name matches.
 func CompleteTemplates(f *Factory) cobra.CompletionFunc {
 	return func(cmd *cobra.Command, args []string, toComplete string) ([]cobra.Completion, cobra.ShellCompDirective) {
-		list, err := completionService(f, cmd, Files{}).ListTemplates()
+		list, err := completionService(f, cmd, Files{}).ListTemplates(false)
 		if err != nil {
 			return nil, cobra.ShellCompDirectiveDefault
 		}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cli/cmdutil/factory.go proto/internal/cli/cmdutil/factory.go
--- base/internal/cli/cmdutil/factory.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cli/cmdutil/factory.go	2026-10-09 11:46:36.595606243 +0000
@@ -4,6 +4,7 @@
 	"github.com/0xboris/invox/internal/adapters/editor"
 	"github.com/0xboris/invox/internal/adapters/opener"
 	"github.com/0xboris/invox/internal/billing"
+	"github.com/0xboris/invox/internal/cli/helptext"
 	"github.com/0xboris/invox/internal/env"
 	"github.com/0xboris/invox/internal/iostreams"
 )
@@ -33,4 +34,7 @@
 	ConfigFile string
 	// Service returns the use cases for a command that names files.
 	Service func(Files) *billing.Service
+	// Locations are where invox keeps its files by default, for help
+	// texts.
+	Locations func() helptext.Locations
 }
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cli/helptext/render.go proto/internal/cli/helptext/render.go
--- base/internal/cli/helptext/render.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cli/helptext/render.go	2026-10-09 11:48:02.290723248 +0000
@@ -2,10 +2,7 @@
 
 import (
 	"io"
-	"path/filepath"
 	"text/template"
-
-	"github.com/0xboris/invox/internal/billing"
 )
 
 // The lines of a command's "Default lookup:" section. Like every Long, they
@@ -37,10 +34,24 @@
 		"  --yes only answers the question; every other check still applies.\n"
 }
 
+// Locations are where invox keeps its files by default, for help texts.
+// They come from the environment only; nothing is read.
+type Locations struct {
+	ConfigDir  string
+	ConfigFile string
+	Customers  string
+	Issuer     string
+	Defaults   string
+	Template   string
+	ArchiveDir string
+	// ConfigTemplate is the text a new config.yaml starts with.
+	ConfigTemplate string
+}
+
 // Render writes text, a command's Long, with its {{...}} actions filled in
-// from l. They can read the fields of billing.Locations and call the
-// methods of data.
-func Render(w io.Writer, text string, l billing.Locations) error {
+// from l. They can read the fields of Locations and call the methods of
+// data.
+func Render(w io.Writer, text string, l Locations) error {
 	tmpl, err := template.New("").Option("missingkey=error").Parse(text)
 	if err != nil {
 		return err
@@ -49,7 +60,7 @@
 }
 
 type data struct {
-	billing.Locations
+	Locations
 }
 
 func (d data) GlobalConfigPath() string          { return d.ConfigFile }
@@ -58,11 +69,3 @@
 func (d data) GlobalInvoiceDefaultsPath() string { return d.Defaults }
 func (d data) GlobalTemplatePath() string        { return d.Template }
 func (d data) DefaultArchiveDir() string         { return d.ArchiveDir }
-
-// LegacyConfigFile is config.yaml in the legacy directory, or "none".
-func (d data) LegacyConfigFile() string {
-	if dir := d.LegacyDir; dir != "" {
-		return filepath.Join(dir, "config.yaml")
-	}
-	return "none"
-}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cli/helptext/topics.go proto/internal/cli/helptext/topics.go
--- base/internal/cli/helptext/topics.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cli/helptext/topics.go	2026-10-09 11:48:02.290723248 +0000
@@ -3,8 +3,6 @@
 import (
 	"fmt"
 	"io"
-
-	"github.com/0xboris/invox/internal/billing"
 )
 
 // Topic is a page of `invox help NAME`. Print is nil for a topic that is the
@@ -13,7 +11,7 @@
 	Name    string
 	Aliases []string
 	Short   string
-	Print   func(w io.Writer, l billing.Locations)
+	Print   func(w io.Writer, l Locations)
 }
 
 // Topics lists the help topics in the order the root help shows them.
@@ -24,7 +22,7 @@
 	{Name: "defaults", Aliases: []string{"invoice-defaults", "invoice_defaults"}, Short: "invoice_defaults.yaml shape and new-command behavior", Print: printDefaultsHelp},
 	{Name: "template", Short: "template placeholders and authoring rules"},
 	{Name: "environment", Short: "environment variables, default directories, and precedence", Print: printEnvironmentHelp},
-	{Name: "exit-codes", Short: "what each exit status means", Print: func(w io.Writer, _ billing.Locations) { printExitCodesHelp(w) }},
+	{Name: "exit-codes", Short: "what each exit status means", Print: func(w io.Writer, _ Locations) { printExitCodesHelp(w) }},
 }
 
 // LookupTopic returns the topic called name or one of its aliases.
@@ -46,7 +44,7 @@
 	return commandName + " " + args
 }
 
-func printCustomersHelp(w io.Writer, l billing.Locations) {
+func printCustomersHelp(w io.Writer, l Locations) {
 	fmt.Fprintf(w, "customers.yaml reference.\n\n")
 	fmt.Fprintf(w, "Usage:\n")
 	fmt.Fprintf(w, "  %s help customers\n\n", commandName)
@@ -71,7 +69,7 @@
 	printCustomerYAMLExample(w)
 }
 
-func printIssuerHelp(w io.Writer, l billing.Locations) {
+func printIssuerHelp(w io.Writer, l Locations) {
 	fmt.Fprintf(w, "issuer.yaml reference.\n\n")
 	fmt.Fprintf(w, "Usage:\n")
 	fmt.Fprintf(w, "  %s help issuer\n\n", commandName)
@@ -96,7 +94,7 @@
 	printIssuerYAMLExample(w)
 }
 
-func printDefaultsHelp(w io.Writer, l billing.Locations) {
+func printDefaultsHelp(w io.Writer, l Locations) {
 	fmt.Fprintf(w, "invoice_defaults.yaml reference.\n\n")
 	fmt.Fprintf(w, "Usage:\n")
 	fmt.Fprintf(w, "  %s help defaults\n", commandName)
@@ -132,9 +130,8 @@
 var environmentVariables = []environmentVariable{
 	{"INVOX_CONFIG_DIR", []string{
 		"Config directory to use in place of the default one, on every OS. invox",
-		"reads config.yaml and the global support files there, `init` writes there,",
-		"and the legacy directory is not read. It must exist. --config still wins",
-		"for config.yaml.",
+		"reads config.yaml and the global support files there, and `init` writes",
+		"there. It must exist. --config still wins for config.yaml.",
 	}},
 	{"XDG_CONFIG_HOME", []string{
 		"Base directory for the config directory, on every OS.",
@@ -170,7 +167,7 @@
 	}},
 }
 
-func printEnvironmentHelp(w io.Writer, l billing.Locations) {
+func printEnvironmentHelp(w io.Writer, l Locations) {
 	fmt.Fprintf(w, "Environment variables and default directories.\n\n")
 	fmt.Fprintf(w, "Usage:\n")
 	fmt.Fprintf(w, "  %s help environment\n\n", commandName)
@@ -187,11 +184,6 @@
 	fmt.Fprintf(w, "  Windows:   %%XDG_CONFIG_HOME%%\\invox, else %%USERPROFILE%%\\.config\\invox\n")
 	fmt.Fprintf(w, "  INVOX_CONFIG_DIR replaces it on every OS.\n")
 	fmt.Fprintf(w, "  here:      %s\n\n", l.ConfigDir)
-	fmt.Fprintf(w, "Legacy config directory (deprecated):\n")
-	fmt.Fprintf(w, "  invoice-tool next to the invox directory, such as $HOME/.config/invoice-tool.\n")
-	fmt.Fprintf(w, "  A file missing from the invox directory is still read from here, and invox\n")
-	fmt.Fprintf(w, "  prints a warning. `%s init` copies the files into the invox directory.\n", commandName)
-	fmt.Fprintf(w, "  Not read when INVOX_CONFIG_DIR is set.\n\n")
 	fmt.Fprintf(w, "Default archive directory (when config.yaml sets no archive.dir):\n")
 	fmt.Fprintf(w, "  Linux:     $XDG_DATA_HOME/invox/invoices, else $HOME/.local/share/invox/invoices\n")
 	fmt.Fprintf(w, "  macOS:     $XDG_DATA_HOME/invox/invoices, else $HOME/Library/Application Support/invox/invoices\n")
@@ -201,7 +193,6 @@
 	fmt.Fprintf(w, "  1. --config PATH\n")
 	fmt.Fprintf(w, "  2. config.yaml in INVOX_CONFIG_DIR\n")
 	fmt.Fprintf(w, "  3. config.yaml in the config directory\n")
-	fmt.Fprintf(w, "  4. config.yaml in the legacy directory\n")
 	fmt.Fprintf(w, "  A --config file that is missing or broken is an error. invox never falls\n")
 	fmt.Fprintf(w, "  back to another config file.\n\n")
 	fmt.Fprintf(w, "Precedence:\n")
@@ -212,7 +203,7 @@
 	fmt.Fprintf(w, "  1. explicit flag (-c, -u, --defaults, -t)\n")
 	fmt.Fprintf(w, "  2. upward search from the current directory\n")
 	fmt.Fprintf(w, "  3. paths.* in config.yaml, relative to config.yaml\n")
-	fmt.Fprintf(w, "  4. the file in the config directory, then in the legacy directory\n\n")
+	fmt.Fprintf(w, "  4. the file in the config directory\n\n")
 	fmt.Fprintf(w, "Upward search:\n")
 	fmt.Fprintf(w, "  It always searches the current directory. It goes up to the nearest directory\n")
 	fmt.Fprintf(w, "  that holds .git, invox.yaml or invoice_defaults.yaml, and stops below your\n")
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cli/root.go proto/internal/cli/root.go
--- base/internal/cli/root.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cli/root.go	2026-10-09 11:46:36.594928646 +0000
@@ -87,7 +87,7 @@
 			return helpCompletions(root, args), cobra.ShellCompDirectiveNoFileComp
 		},
 		RunE: func(cmd *cobra.Command, args []string) error {
-			return helpTopic(cmd.OutOrStdout(), root, f.Service(cmdutil.Files{}).Locations(), args)
+			return helpTopic(cmd.OutOrStdout(), root, f.Locations(), args)
 		},
 	}
 	root.SetHelpCommand(help)
@@ -103,7 +103,7 @@
 		if cmd == help {
 			cmd = root
 		}
-		if err := writeHelp(cmd.OutOrStdout(), cmd, f.Service(cmdutil.Files{}).Locations()); err != nil {
+		if err := writeHelp(cmd.OutOrStdout(), cmd, f.Locations()); err != nil {
 			helpErr = err
 		}
 	})
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cli/usage.go proto/internal/cli/usage.go
--- base/internal/cli/usage.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cli/usage.go	2026-10-09 11:48:02.338723247 +0000
@@ -7,7 +7,6 @@
 
 	"github.com/spf13/cobra"
 
-	"github.com/0xboris/invox/internal/billing"
 	"github.com/0xboris/invox/internal/cli/cmdutil"
 	"github.com/0xboris/invox/internal/cli/helptext"
 )
@@ -16,7 +15,7 @@
 // command tree: the Long text (or the Short one), then usage, subcommands,
 // flags and examples. Command aliases are deprecated names, so it leaves
 // them out. The root page also lists the help topics.
-func writeHelp(w io.Writer, cmd *cobra.Command, l billing.Locations) error {
+func writeHelp(w io.Writer, cmd *cobra.Command, l helptext.Locations) error {
 	description := cmd.Long
 	if description == "" {
 		description = cmd.Short + "."
@@ -120,7 +119,7 @@
 // helpTopic writes the page `invox help ARGS` names: a topic, or the help of
 // a command, including a hidden deprecated name such as `customer config`.
 // Anything else, including `help` itself, is an unknown topic.
-func helpTopic(w io.Writer, root *cobra.Command, l billing.Locations, args []string) error {
+func helpTopic(w io.Writer, root *cobra.Command, l helptext.Locations, args []string) error {
 	if len(args) == 0 {
 		return writeHelp(w, root, l)
 	}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/config/config.go proto/internal/cmd/config/config.go
--- base/internal/cmd/config/config.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/config/config.go	2026-10-09 11:46:58.174327809 +0000
@@ -28,8 +28,7 @@
   Existing config.yaml files are left unchanged.
 
 Config paths:
-  preferred: {{.GlobalConfigPath}}
-  legacy fallback: {{.LegacyConfigFile}}
+  default: {{.GlobalConfigPath}}
   --config PATH and INVOX_CONFIG_DIR change them; see ` + "`invox help environment`" + `.
   ` + "`invox config paths`" + ` shows the config file and the support files in use.
 
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/config/paths/paths.go proto/internal/cmd/config/paths/paths.go
--- base/internal/cmd/config/paths/paths.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/config/paths/paths.go	2026-10-09 11:46:50.300382814 +0000
@@ -39,7 +39,6 @@
   flag     the --config option
   env      INVOX_CONFIG_DIR
   default  the default config or archive directory
-  legacy   the deprecated invoice-tool directory
   project  the upward search from the current directory
   config   a paths.* or archive.dir setting in config.yaml
   none     not found
@@ -62,11 +61,7 @@
 }
 
 func pathsRun(opts *PathsOptions) error {
-	cwd, err := opts.Getwd()
-	if err != nil {
-		return err
-	}
-	reports, err := opts.Service(cmdutil.Files{}).Paths(cwd)
+	reports, err := opts.Service(cmdutil.Files{}).Paths()
 	if err != nil {
 		return err
 	}
@@ -84,7 +79,6 @@
 	billing.SourceExplicit: "flag",
 	billing.SourceEnvDir:   "env",
 	billing.SourceDefault:  "default",
-	billing.SourceLegacy:   "legacy",
 	billing.SourceProject:  "project",
 	billing.SourceConfig:   "config",
 }
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/init/init.go proto/internal/cmd/init/init.go
--- base/internal/cmd/init/init.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/init/init.go	2026-10-09 11:48:02.350723247 +0000
@@ -19,8 +19,6 @@
 type InitOptions struct {
 	IO      *iostreams.IOStreams
 	Service func(cmdutil.Files) *billing.Service
-
-	Force bool
 }
 
 // NewCmdInit returns the init command. runF replaces initRun in tests.
@@ -37,15 +35,11 @@
   Writes starter versions of config.yaml, customers.yaml, issuer.yaml,
   invoice_defaults.yaml, and template.tex.
   Existing non-empty files are left unchanged.
-  When the deprecated invoice-tool directory has files the config directory
-  lacks, asks first, then copies them in before writing the starter files.
-  Nothing is replaced, and the invoice-tool directory is left in place.
 
 Config directory:
   {{.ConfigDir}}
 `,
 		Example: `$ invox init
-$ invox init --force
 `,
 		Args: func(cmd *cobra.Command, args []string) error {
 			if len(args) > 0 {
@@ -60,17 +54,11 @@
 			return initRun(cmd.Context(), opts)
 		},
 	}
-	cmd.Flags().BoolVar(&opts.Force, "force", false, "Copy files from the deprecated config directory without asking (required without a terminal)")
 	return cmd
 }
 
 func initRun(ctx context.Context, opts *InitOptions) error {
-	svc := opts.Service(cmdutil.Files{})
-	if err := copyLegacyFiles(ctx, opts.IO, svc, opts.Force); err != nil {
-		return err
-	}
-
-	initialized, err := svc.Init()
+	initialized, err := opts.Service(cmdutil.Files{}).Init()
 	if err != nil {
 		return err
 	}
@@ -86,37 +74,3 @@
 	}
 	return nil
 }
-
-// copyLegacyFiles copies the files of the deprecated config directory that
-// the config directory lacks, after asking, unless force is set.
-func copyLegacyFiles(ctx context.Context, ios *iostreams.IOStreams, svc *billing.Service, force bool) error {
-	missing, err := svc.LegacyFiles()
-	if err != nil || len(missing) == 0 {
-		return err
-	}
-	locations := svc.Locations()
-	legacyDir, configDir := locations.LegacyDir, locations.ConfigDir
-	if !force {
-		if !ios.CanPrompt() {
-			return cmdutil.FlagErrorf("init", "the deprecated config directory %s has files that %s lacks; pass --force to copy them (no terminal to ask on)", legacyDir, configDir)
-		}
-		confirmed, err := cmdutil.Confirm(ctx, ios, fmt.Sprintf("Copy %s from %s to %s?", strings.Join(missing, ", "), legacyDir, configDir))
-		if err != nil {
-			return err
-		}
-		if !confirmed {
-			fmt.Fprintf(ios.ErrOut, "not initialized; nothing was changed\n")
-			return cmdutil.CancelError
-		}
-	}
-
-	copied, err := svc.CopyLegacyFiles()
-	for _, rel := range copied {
-		fmt.Fprintf(ios.ErrOut, "copied %s from %s\n", rel, legacyDir)
-	}
-	if err != nil {
-		return err
-	}
-	fmt.Fprintf(ios.ErrOut, "invox no longer reads %s for these files; remove it once you are happy with %s\n", legacyDir, configDir)
-	return nil
-}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/invoice/archive/add/add.go proto/internal/cmd/invoice/archive/add/add.go
--- base/internal/cmd/invoice/archive/add/add.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/invoice/archive/add/add.go	2026-10-09 11:47:46.252221681 +0000
@@ -120,12 +120,14 @@
 		if err != nil {
 			return err
 		}
+		shared.WarnUnread(opts.IO, result.Unread, baseDir)
 		shared.PrintArchivePreview(opts.IO, result, invoicePath, baseDir)
 	} else {
 		result, err = svc.Archive(invoicePath, billing.ArchiveOptions{Replace: opts.Yes, Confirm: shared.ConfirmReplace(ctx, opts.IO, opts.Command, invoicePath, baseDir, "")})
 		if err != nil {
 			return err
 		}
+		shared.WarnUnread(opts.IO, result.Unread, baseDir)
 		shared.PrintArchiveReplacements(opts.IO, result, baseDir)
 		fmt.Fprintf(opts.IO.ErrOut, "Archived %s -> %s\n", cmdutil.DisplayPath(invoicePath, baseDir), cmdutil.DisplayPath(result.Path, baseDir))
 	}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/invoice/archive/list/list.go proto/internal/cmd/invoice/archive/list/list.go
--- base/internal/cmd/invoice/archive/list/list.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/invoice/archive/list/list.go	2026-10-09 11:48:02.358723247 +0000
@@ -9,6 +9,7 @@
 	"github.com/0xboris/invox/internal/billing"
 	"github.com/0xboris/invox/internal/cli/cmdutil"
 	"github.com/0xboris/invox/internal/cli/helptext"
+	"github.com/0xboris/invox/internal/cmd/invoice/shared"
 	"github.com/0xboris/invox/internal/iostreams"
 	"github.com/0xboris/invox/internal/tableprinter"
 )
@@ -76,7 +77,8 @@
 	if err != nil {
 		return err
 	}
-	archivedInvoices, archiveDir := list.Entries, list.Dir
+	archivedInvoices, archiveDir := list.Entries, list.Unread.Dir
+	shared.WarnUnread(opts.IO, list.Unread, "")
 
 	if opts.Exporter != nil {
 		items := make([]archivedJSON, 0, len(archivedInvoices))
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/invoice/build/build.go proto/internal/cmd/invoice/build/build.go
--- base/internal/cmd/invoice/build/build.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/invoice/build/build.go	2026-10-09 11:47:46.252529422 +0000
@@ -152,6 +152,9 @@
 		return cmdutil.UsageError("build", err)
 	}
 	inv := result.Context
+	if result.Archived != nil {
+		shared.WarnUnread(opts.IO, result.Archived.Unread, baseDir)
+	}
 
 	// printResult prints the PDF's path, or with --json the build's result.
 	printResult := func(archivedPath *string) error {
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/invoice/email/email.go proto/internal/cmd/invoice/email/email.go
--- base/internal/cmd/invoice/email/email.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/invoice/email/email.go	2026-10-09 11:47:46.253256836 +0000
@@ -165,6 +165,7 @@
 		return outputExists(cmdutil.UsageError("email", err), baseDir)
 	}
 	message := result.Message
+	shared.WarnUnread(opts.IO, result.Unread, baseDir)
 
 	if opts.DryRun {
 		fmt.Fprintf(opts.IO.ErrOut, "Would open email draft for %s (%s) to %s\n", result.CustomerID, result.Number, message.To)
@@ -176,16 +177,6 @@
 		return nil
 	}
 
-	if draft := result.Draft; draft.Path != "" {
-		if err := opts.Opener.Open(ctx, draft.Path); err != nil {
-			if draft.Discard == nil {
-				return fmt.Errorf("created %s but failed to open it: %w", cmdutil.DisplayPath(draft.Path, baseDir), err)
-			}
-			draft.Discard()
-			return fmt.Errorf("failed to open email draft: %w", err)
-		}
-	}
-
 	fmt.Fprintf(opts.IO.ErrOut, "Opened email draft for %s (%s) to %s\n", result.CustomerID, result.Number, message.To)
 	if explicitOutput {
 		fmt.Fprintln(opts.IO.Out, cmdutil.DisplayPath(message.Output, baseDir))
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/invoice/increment/increment.go proto/internal/cmd/invoice/increment/increment.go
--- base/internal/cmd/invoice/increment/increment.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/invoice/increment/increment.go	2026-10-09 11:47:46.249793357 +0000
@@ -90,6 +90,7 @@
 	if err != nil {
 		return cmdutil.UsageError("increment", err)
 	}
+	shared.WarnUnread(opts.IO, incremented.Unread, baseDir)
 	shared.WarnSkippedArchiveFiles(opts.IO, incremented.CustomerID, incremented.Skipped, baseDir)
 
 	displayPath := cmdutil.DisplayPath(invoicePath, baseDir)
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/invoice/new/new.go proto/internal/cmd/invoice/new/new.go
--- base/internal/cmd/invoice/new/new.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/invoice/new/new.go	2026-10-09 11:47:46.251545444 +0000
@@ -148,6 +148,7 @@
 	if err != nil {
 		return cmdutil.UsageError("new", err)
 	}
+	shared.WarnUnread(opts.IO, created.Unread, baseDir)
 	shared.WarnSkippedArchiveFiles(opts.IO, opts.CustomerID, created.Skipped, baseDir)
 	displayPath := cmdutil.DisplayPath(created.Path, baseDir)
 	if opts.Edit && !opts.DryRun {
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/invoice/shared/shared.go proto/internal/cmd/invoice/shared/shared.go
--- base/internal/cmd/invoice/shared/shared.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/invoice/shared/shared.go	2026-10-09 11:47:46.249429199 +0000
@@ -8,6 +8,7 @@
 	"path/filepath"
 	"strings"
 
+	"github.com/0xboris/invox/internal/billing"
 	"github.com/0xboris/invox/internal/cli/cmdutil"
 	"github.com/0xboris/invox/internal/invoice"
 	"github.com/0xboris/invox/internal/iostreams"
@@ -31,6 +32,18 @@
 	return nil
 }
 
+// WarnUnread warns on stderr about the archived invoices stored as
+// Markdown, which invox no longer reads.
+func WarnUnread(ios *iostreams.IOStreams, unread billing.Unread, baseDir string) {
+	switch n := len(unread.Markdown); n {
+	case 0:
+	case 1:
+		fmt.Fprintf(ios.ErrOut, "warning: 1 Markdown invoice in %s is no longer read; convert it to .yaml to include it\n", cmdutil.DisplayPath(unread.Dir, baseDir))
+	default:
+		fmt.Fprintf(ios.ErrOut, "warning: %d Markdown invoices in %s are no longer read; convert them to .yaml to include them\n", n, cmdutil.DisplayPath(unread.Dir, baseDir))
+	}
+}
+
 // maxListedSkippedArchiveFiles caps the files named in the skipped archive
 // warning.
 const maxListedSkippedArchiveFiles = 5
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/invoice/validate/validate.go proto/internal/cmd/invoice/validate/validate.go
--- base/internal/cmd/invoice/validate/validate.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/invoice/validate/validate.go	2026-10-09 11:47:46.251905225 +0000
@@ -100,6 +100,7 @@
 	}
 	ctx := result.Context
 
+	shared.WarnUnread(opts.IO, result.Unread, baseDir)
 	shared.WarnArchivedDuplicate(opts.IO, result.Duplicate, invoicePath, baseDir)
 
 	fmt.Fprintf(
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/cmd/template/list/list.go proto/internal/cmd/template/list/list.go
--- base/internal/cmd/template/list/list.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/cmd/template/list/list.go	2026-10-09 11:52:32.217047469 +0000
@@ -77,15 +77,14 @@
 }
 
 func listRun(opts *ListOptions) error {
-	svc := opts.Service(cmdutil.Files{})
-	catalog, err := svc.ListTemplates()
+	catalog, err := opts.Service(cmdutil.Files{}).ListTemplates(opts.Exporter != nil)
 	if err != nil {
 		return err
 	}
 	templates, templateDir := catalog.Templates, catalog.Dir
 
 	if opts.Exporter != nil {
-		return exportTemplates(opts, svc, templates)
+		return exportTemplates(opts, catalog)
 	}
 
 	list := tableprinter.Table{
@@ -106,16 +105,10 @@
 	return nil
 }
 
-func exportTemplates(opts *ListOptions, svc *billing.Service, templates []billing.Template) error {
-	if _, err := opts.Getwd(); err != nil {
-		return err
-	}
-	defaultTemplate, err := svc.DefaultTemplate()
-	if err != nil {
-		return err
-	}
-	items := make([]templateJSON, 0, len(templates))
-	for _, template := range templates {
+func exportTemplates(opts *ListOptions, catalog billing.TemplateList) error {
+	defaultTemplate := catalog.Default
+	items := make([]templateJSON, 0, len(catalog.Templates))
+	for _, template := range catalog.Templates {
 		items = append(items, templateJSON{
 			Name:    template.Name,
 			Path:    template.Path,
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/email/mailer.go proto/internal/email/mailer.go
--- base/internal/email/mailer.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/email/mailer.go	2026-10-09 11:46:05.464449274 +0000
@@ -21,40 +21,56 @@
 	draftMaxAge = 24 * time.Hour
 )
 
-// Mailer writes drafts as .eml files. It implements billing.Mailer.
-type Mailer struct{}
+// Mailer writes drafts as .eml files and opens them. It implements
+// billing.Mailer.
+type Mailer struct {
+	// Open opens a draft in the user's mail app.
+	Open func(ctx context.Context, path string) error
+}
 
 var _ billing.Mailer = Mailer{}
 
-// Check returns an *billing.OutputIsDirError when m.Output is a directory,
-// and an error matching fs.ErrExist when the draft would replace an
-// existing file and m.Overwrite is not set.
-func (Mailer) Check(m billing.Message) error {
-	return checkOutput(m.Output, m.Overwrite)
-}
-
-// Draft writes m as an .eml file: to m.Output, or, for a temporary draft,
-// to a new temporary directory that a draft a day later removes. Unless
-// m.Overwrite is set, an existing file is left untouched and the error
-// matches fs.ErrExist.
-func (Mailer) Draft(_ context.Context, m billing.Message) (billing.Draft, error) {
+// Draft writes m as an .eml file and opens it: to m.Output, or, for a
+// temporary draft, to a new temporary directory that a draft a day later
+// removes. Unless m.Overwrite is set, an existing file is left untouched
+// and the error matches fs.ErrExist. With dryRun it runs the same checks
+// and writes nothing.
+func (mailer Mailer) Draft(ctx context.Context, m billing.Message, dryRun bool) (string, error) {
+	if _, err := os.Stat(m.Attachment); err != nil {
+		return "", fmt.Errorf("read %s: %w", m.Attachment, err)
+	}
+	if dryRun {
+		if m.Temporary {
+			return "", nil
+		}
+		return "", checkOutput(m.Output, m.Overwrite)
+	}
 	if !m.Temporary {
-		return billing.Draft{Path: m.Output}, write(m, m.Output, m.Overwrite)
+		if err := write(m, m.Output, m.Overwrite); err != nil {
+			return "", err
+		}
+		if err := mailer.Open(ctx, m.Output); err != nil {
+			return "", fmt.Errorf("created %s but failed to open it: %w", m.Output, err)
+		}
+		return m.Output, nil
 	}
 	pruneDrafts(os.TempDir(), m.Date.Add(-draftMaxAge))
 	dir, err := os.MkdirTemp("", draftDirPrefix+"*")
 	if err != nil {
-		return billing.Draft{}, fmt.Errorf("create temporary draft directory: %w", err)
+		return "", fmt.Errorf("create temporary draft directory: %w", err)
 	}
-	discard := func() { _ = os.RemoveAll(dir) }
 	path := filepath.Join(dir, filepath.Base(m.Output))
 	if err := write(m, path, false); err != nil {
-		discard()
-		return billing.Draft{}, err
+		_ = os.RemoveAll(dir)
+		return "", err
 	}
 	// The draft stays in its temporary directory: the mail app can read it
 	// after the opener returns, so invox cannot know when to delete it.
-	return billing.Draft{Path: path, Discard: discard}, nil
+	if err := mailer.Open(ctx, path); err != nil {
+		_ = os.RemoveAll(dir)
+		return "", fmt.Errorf("failed to open email draft: %w", err)
+	}
+	return path, nil
 }
 
 func write(m billing.Message, path string, overwrite bool) error {
@@ -83,12 +99,6 @@
 	return writeFile(path, message, fsutil.Public)
 }
 
-// CheckAttachment returns why the file at path cannot be attached.
-func (Mailer) CheckAttachment(path string) error {
-	_, err := os.Stat(path)
-	return err
-}
-
 func checkOutput(path string, overwrite bool) error {
 	if info, err := os.Stat(path); err == nil && info.IsDir() {
 		return &billing.OutputIsDirError{Path: path}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/factory/factory.go proto/internal/factory/factory.go
--- base/internal/factory/factory.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/factory/factory.go	2026-10-09 11:47:17.853832635 +0000
@@ -3,6 +3,7 @@
 package factory
 
 import (
+	"context"
 	"path/filepath"
 	"strings"
 	"sync"
@@ -15,6 +16,7 @@
 	"github.com/0xboris/invox/internal/archive"
 	"github.com/0xboris/invox/internal/billing"
 	"github.com/0xboris/invox/internal/cli/cmdutil"
+	"github.com/0xboris/invox/internal/cli/helptext"
 	"github.com/0xboris/invox/internal/email"
 	"github.com/0xboris/invox/internal/env"
 	"github.com/0xboris/invox/internal/iostreams"
@@ -43,22 +45,37 @@
 	}
 	f.Service = func(files cmdutil.Files) *billing.Service {
 		h := host()
-		var mailer billing.Mailer = email.Mailer{}
+		var mailer billing.Mailer = email.Mailer{Open: func(ctx context.Context, path string) error { return f.Opener.Open(ctx, path) }}
 		if mailApp != nil && !files.EmailOutput {
 			mailer = mailApp
 		}
 		st := &store.Store{Host: h, Getwd: e.Getwd, Files: store.Files{Customers: files.Customers, Issuer: files.Issuer, Defaults: files.Defaults}}
+		archived := archive.Archive{Locate: h.ResolveArchiveDir, Read: store.ReadArchived, Rewrite: st.Rewrite, Now: e.Now}
+		st.Protected = archived.Protects
 		return &billing.Service{
 			Invoices:  st,
 			Directory: st,
-			Archives:  archive.Archive{Locate: h.ResolveArchiveDir, Read: store.ReadArchived, Rewrite: st.Rewrite},
-			Renderer:  latex.Renderer{},
+			Archives:  archived,
+			Renderer:  latex.Renderer{FindAsset: h.FindAsset},
 			Compiler:  compiler,
 			Mailer:    mailer,
 			Settings:  h.Settings,
 			Now:       e.Now,
 		}
 	}
+	f.Locations = func() helptext.Locations {
+		h := host()
+		return helptext.Locations{
+			ConfigDir:      h.ConfigDir(),
+			ConfigFile:     h.GlobalConfigPath(),
+			Customers:      h.GlobalCustomersPath(),
+			Issuer:         h.GlobalIssuerPath(),
+			Defaults:       h.GlobalInvoiceDefaultsPath(),
+			Template:       h.GlobalTemplatePath(),
+			ArchiveDir:     h.DefaultArchiveDir(),
+			ConfigTemplate: h.ConfigTemplate(),
+		}
+	}
 	return f
 }
 
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/invoice/models.go proto/internal/invoice/models.go
--- base/internal/invoice/models.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/invoice/models.go	2026-10-09 11:45:32.101546787 +0000
@@ -114,8 +114,7 @@
 // ArchiveLink is the `_invox` mapping of a working copy made by `archive
 // edit`: the archived file it replaces when it is archived again.
 type ArchiveLink struct {
-	ArchivePath        Text
-	ArchiveReplacePath Text
+	ArchivePath Text
 }
 
 const (
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/render/latex/renderer.go proto/internal/render/latex/renderer.go
--- base/internal/render/latex/renderer.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/render/latex/renderer.go	2026-10-09 11:46:05.463843512 +0000
@@ -113,7 +113,12 @@
 )
 
 // Renderer fills LaTeX templates in. It implements billing.Renderer.
-type Renderer struct{}
+type Renderer struct {
+	// FindAsset returns the file or directory rel that template uses: next
+	// to it, else in the config directories. It returns "" when there is
+	// none.
+	FindAsset func(template, rel string, dir bool) string
+}
 
 var _ billing.Renderer = Renderer{}
 
@@ -151,7 +156,7 @@
 
 // Write writes source to path and copies the assets it uses from next to
 // the template, or from the config directories, next to path.
-func (Renderer) Write(t billing.Template, source, path string) error {
+func (r Renderer) Write(t billing.Template, source, path string) error {
 	if err := fsutil.WriteFile(path, []byte(source), fsutil.Public); err != nil {
 		return err
 	}
@@ -160,7 +165,7 @@
 		return nil
 	}
 	for _, relDir := range AssetDirs(source) {
-		sourceDir := t.FindAsset(relDir, true)
+		sourceDir := r.FindAsset(t.Path, relDir, true)
 		if sourceDir == "" {
 			continue
 		}
@@ -169,7 +174,7 @@
 		}
 	}
 	for _, relFile := range AssetFiles(source) {
-		sourceFile := t.FindAsset(relFile, false)
+		sourceFile := r.FindAsset(t.Path, relFile, false)
 		if sourceFile == "" {
 			continue
 		}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/archive.go proto/internal/store/archive.go
--- base/internal/store/archive.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/archive.go	2026-10-09 11:45:27.674190791 +0000
@@ -1,11 +1,10 @@
 package store
 
 // archivedInvoiceIdentity reads what the archive lists of the invoice at
-// path. ok is false for a file that is not an invoice: Markdown without
-// front matter, or a document whose invoice fields cannot be read.
+// path. ok is false for a document whose invoice fields cannot be read.
 func archivedInvoiceIdentity(path string) (invoiceIdentity, bool, error) {
-	document, ok, err := loadArchivedInvoiceDocument(path)
-	if err != nil || !ok {
+	document, err := loadYAMLDocument(path)
+	if err != nil {
 		return invoiceIdentity{}, false, err
 	}
 	root, err := documentRootMapping(document, path)
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/archive_metadata.go proto/internal/store/archive_metadata.go
--- base/internal/store/archive_metadata.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/archive_metadata.go	2026-10-09 11:45:27.674494119 +0000
@@ -2,34 +2,18 @@
 
 import (
 	"path/filepath"
-	"strings"
 
 	yaml "gopkg.in/yaml.v3"
 )
 
 const (
-	internalMetadataKey       = "_invox"
-	internalArchivePathKey    = "archive_path"
-	internalArchiveReplaceKey = "archive_replace_path"
+	internalMetadataKey    = "_invox"
+	internalArchivePathKey = "archive_path"
 )
 
-func archiveMetadata(root *yaml.Node) (string, string) {
-	internalNode := findMappingValue(root, internalMetadataKey)
-	if internalNode == nil || internalNode.Kind != yaml.MappingNode {
-		return "", ""
-	}
-	return filepath.FromSlash(strings.TrimSpace(nodeText(findMappingValue(internalNode, internalArchivePathKey)))),
-		filepath.FromSlash(strings.TrimSpace(nodeText(findMappingValue(internalNode, internalArchiveReplaceKey))))
-}
-
-// setArchiveMetadata records archive-relative paths with forward slashes so
-// a working copy stays portable between operating systems.
-func setArchiveMetadata(root *yaml.Node, archivePath, archiveReplacePath string) {
+// setArchiveMetadata records the archive-relative path with forward slashes
+// so a working copy stays portable between operating systems.
+func setArchiveMetadata(root *yaml.Node, archivePath string) {
 	internalNode := getOrCreateMappingNode(root, internalMetadataKey)
 	setMappingString(internalNode, internalArchivePathKey, filepath.ToSlash(archivePath))
-	if strings.TrimSpace(archiveReplacePath) == "" || archiveReplacePath == archivePath {
-		deleteMappingKey(internalNode, internalArchiveReplaceKey)
-	} else {
-		setMappingString(internalNode, internalArchiveReplaceKey, filepath.ToSlash(archiveReplacePath))
-	}
 }
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/assets.go proto/internal/store/assets.go
--- base/internal/store/assets.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/assets.go	2026-10-09 11:45:27.673202911 +0000
@@ -2,9 +2,9 @@
 
 import "path/filepath"
 
-// findAsset looks for relPath next to the template, then in the config
+// FindAsset looks for relPath next to the template, then in the config
 // directories.
-func (h Host) findAsset(templatePath, relPath string, isDir bool) string {
+func (h Host) FindAsset(templatePath, relPath string, isDir bool) string {
 	if candidate := filepath.Join(filepath.Dir(templatePath), relPath); pathExists(candidate, isDir) {
 		return candidate
 	}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/host.go proto/internal/store/host.go
--- base/internal/store/host.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/host.go	2026-10-09 11:45:27.671385746 +0000
@@ -29,7 +29,7 @@
 
 // Host holds the user directories invox reads and writes, resolved once, and
 // the config file, read at most once. Copies of a Host share the loaded
-// config and the record of legacy files used.
+// config.
 type Host struct {
 	goos       string
 	home       string
@@ -37,12 +37,11 @@
 	dataBase   string
 	configDir  Resolved
 	configFile string
-	legacy     *legacyUse
 	config     func() (*config.Config, error)
 }
 
 func NewHost(in HostInputs) Host {
-	h := Host{goos: in.GOOS, home: in.Home, configFile: in.ConfigFile, legacy: &legacyUse{}}
+	h := Host{goos: in.GOOS, home: in.Home, configFile: in.ConfigFile}
 	homeKnown := strings.TrimSpace(in.Home) != ""
 
 	if xdg := strings.TrimSpace(in.XDGConfigHome); xdg != "" {
@@ -119,20 +118,9 @@
 	return h.configDir.Path
 }
 
-// LegacyConfigDir returns the deprecated invoice-tool directory, or "" when
-// it is not read: the home directory is unknown or the config directory was
-// chosen explicitly.
-func (h Host) LegacyConfigDir() string {
-	if h.configDir.Source != SourceDefault {
-		return ""
-	}
-	return filepath.Join(h.configBase, legacyConfigDirName)
-}
-
 // findInConfigDir returns the first of names, as a file or as a directory,
-// in the config directory. Each name missing there is looked up in the
-// legacy directory next, and a hit is recorded for LegacyFilesUsed. An
-// explicitly chosen config directory that does not exist is an error.
+// in the config directory. An explicitly chosen config directory that does
+// not exist is an error.
 func (h Host) findInConfigDir(isDir bool, names ...string) (Resolved, error) {
 	dir := h.configDir
 	if dir.Source == SourceEnvDir {
@@ -140,54 +128,16 @@
 			return Resolved{}, &billing.ConfigDirNotFoundError{Dir: dir.Path}
 		}
 	}
-	dirs := []Resolved{dir}
-	if legacy := h.LegacyConfigDir(); legacy != "" {
-		dirs = append(dirs, Resolved{Path: legacy, Source: SourceLegacy})
-	}
-	for _, d := range dirs {
-		if d.Path == "" {
-			continue
-		}
-		for _, name := range names {
-			candidate := filepath.Join(d.Path, name)
-			if !pathExists(candidate, isDir) {
-				continue
-			}
-			if d.Source == SourceLegacy {
-				h.legacy.record(candidate)
-			}
-			return Resolved{Path: candidate, Source: d.Source}, nil
-		}
+	if dir.Path == "" {
+		return Resolved{}, nil
 	}
-	return Resolved{}, nil
-}
-
-// LegacyFilesUsed returns the files read from the legacy directory so far,
-// in the order first used.
-func (h Host) LegacyFilesUsed() []string {
-	return h.legacy.files()
-}
-
-type legacyUse struct {
-	mu    sync.Mutex
-	paths []string
-}
-
-func (l *legacyUse) record(path string) {
-	l.mu.Lock()
-	defer l.mu.Unlock()
-	for _, p := range l.paths {
-		if p == path {
-			return
+	for _, name := range names {
+		candidate := filepath.Join(dir.Path, name)
+		if pathExists(candidate, isDir) {
+			return Resolved{Path: candidate, Source: dir.Source}, nil
 		}
 	}
-	l.paths = append(l.paths, path)
-}
-
-func (l *legacyUse) files() []string {
-	l.mu.Lock()
-	defer l.mu.Unlock()
-	return append([]string(nil), l.paths...)
+	return Resolved{}, nil
 }
 
 // Source says where a resolved path came from.
@@ -198,7 +148,6 @@
 	SourceExplicit = billing.SourceExplicit
 	SourceEnvDir   = billing.SourceEnvDir
 	SourceDefault  = billing.SourceDefault
-	SourceLegacy   = billing.SourceLegacy
 	SourceProject  = billing.SourceProject
 	SourceConfig   = billing.SourceConfig
 )
@@ -209,6 +158,5 @@
 }
 
 const (
-	configDirName       = "invox"
-	legacyConfigDirName = "invoice-tool"
+	configDirName = "invox"
 )
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/init.go proto/internal/store/init.go
--- base/internal/store/init.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/init.go	2026-10-09 11:48:02.510723243 +0000
@@ -3,9 +3,6 @@
 import (
 	"embed"
 	"errors"
-	"io/fs"
-	"os"
-	"path/filepath"
 	"strings"
 
 	"github.com/0xboris/invox/internal/fsutil"
@@ -59,76 +56,3 @@
 
 	return configDir, results, nil
 }
-
-// LegacyFilesToCopy returns the files in the legacy config directory, relative
-// to it, that the config directory does not have yet.
-func (h Host) LegacyFilesToCopy() ([]string, error) {
-	legacyDir := h.LegacyConfigDir()
-	if legacyDir == "" {
-		return nil, nil
-	}
-	var missing []string
-	err := filepath.WalkDir(legacyDir, func(path string, entry fs.DirEntry, err error) error {
-		if errors.Is(err, fs.ErrNotExist) && path == legacyDir {
-			return filepath.SkipDir
-		}
-		if err != nil || entry.IsDir() {
-			return err
-		}
-		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
-			return nil //nolint:nilerr // only regular files are copied; skip a link that does not resolve or cannot be read
-		}
-		rel, err := filepath.Rel(legacyDir, path)
-		if err != nil {
-			return err
-		}
-		if _, err := os.Lstat(filepath.Join(h.ConfigDir(), rel)); errors.Is(err, fs.ErrNotExist) {
-			missing = append(missing, rel)
-		}
-		return nil
-	})
-	return missing, err
-}
-
-// CopyLegacyFiles copies the files LegacyFilesToCopy reports into the config
-// directory and returns them. It never replaces a file, and it leaves the
-// legacy directory as it is, so running it again copies only what is still
-// missing.
-func (h Host) CopyLegacyFiles() ([]string, error) {
-	missing, err := h.LegacyFilesToCopy()
-	if err != nil {
-		return nil, err
-	}
-	if len(missing) == 0 {
-		return nil, nil
-	}
-	if err := fsutil.MkdirAll(h.ConfigDir(), fsutil.Private); err != nil {
-		return nil, err
-	}
-	copied := make([]string, 0, len(missing))
-	for _, rel := range missing {
-		content, err := os.ReadFile(filepath.Join(h.LegacyConfigDir(), rel))
-		if err != nil {
-			return copied, err
-		}
-		err = fsutil.WriteNewFile(filepath.Join(h.ConfigDir(), rel), content, legacyFilePerm(rel))
-		if errors.Is(err, fs.ErrExist) {
-			continue
-		}
-		if err != nil {
-			return copied, err
-		}
-		copied = append(copied, rel)
-	}
-	return copied, nil
-}
-
-// legacyFilePerm gives customers.yaml and issuer.yaml the same private mode
-// that init gives their starter files.
-func legacyFilePerm(rel string) fsutil.Perm {
-	switch rel {
-	case "customers.yaml", "issuer.yaml":
-		return fsutil.Private
-	}
-	return fsutil.Public
-}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/invoices.go proto/internal/store/invoices.go
--- base/internal/store/invoices.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/invoices.go	2026-10-09 11:55:14.174713370 +0000
@@ -22,68 +22,15 @@
 	return inv, err
 }
 
-// ArchivedHead reads the head of the archived invoice at path, a YAML file
-// or the front matter of a Markdown file, and decodes it strictly.
-func (s *Store) ArchivedHead(path string) (billing.Head, error) {
-	document, ok, err := loadArchivedInvoiceDocument(path)
-	if err != nil {
-		return billing.Head{}, err
-	}
-	if !ok {
-		return billing.Head{}, fmt.Errorf("%s: archived invoice could not be loaded", path)
-	}
-	err = decodeYAMLDocument(document, path, &invoice.Invoice{}, true)
-	if err != nil && !isDecodeError(err) {
-		return billing.Head{}, err
-	}
-	// The lenient decode's errors are among the strict decode's.
-	head, _ := documentHead(document, path)
-	return head, err
-}
-
-// Head reads the identity and the archive link of the invoice at path.
-func (s *Store) Head(path string) (billing.Head, error) {
-	document, err := loadYAMLDocument(path)
-	if err != nil {
-		return billing.Head{}, err
-	}
-	return documentHead(document, path)
-}
-
-func documentHead(document *yaml.Node, path string) (billing.Head, error) {
-	var identity invoiceIdentity
-	err := decodeYAMLDocument(document, path, &identity, false)
-	if err != nil && !isDecodeError(err) {
-		return billing.Head{}, err
-	}
-	head := identity.head()
-	root := document.Content[0]
-	head.Header = headerShape(root)
-	head.ArchivePath, head.ReplacePath = archiveMetadata(root)
-	return head, err
-}
-
-// headerShape is what the `invoice` key of root holds, the node itself
-// rather than what an alias points to, as invoiceMapping reads it.
-func headerShape(root *yaml.Node) billing.HeaderShape {
-	switch node := findMappingValue(root, "invoice"); {
-	case node == nil:
-		return billing.HeaderMissing
-	case node.Kind == yaml.MappingNode:
-		return billing.HeaderMapping
-	}
-	return billing.HeaderOther
-}
-
-func (identity invoiceIdentity) head() billing.Head {
-	head := billing.Head{CustomerID: identity.CustomerID.Trim()}
+// draft is the invoice identity names, with only its customer and the
+// header fields it reads.
+func (identity invoiceIdentity) draft() invoice.Invoice {
+	draft := invoice.Invoice{CustomerID: identity.CustomerID, Header: &invoice.Header{}}
 	if identity.Invoice != nil {
-		head.HasHeader = true
-		head.Number = identity.Invoice.Number.Trim()
-		head.IssueDate = identity.Invoice.IssueDate.Trim()
-		head.Status = invoice.Status(identity.Invoice.Status.Trim())
+		draft.Header.Number = identity.Invoice.Number
+		draft.Header.Status = identity.Invoice.Status
 	}
-	return head
+	return draft
 }
 
 // maxDraftScanSize skips large YAML files in the draft scan; invoices are
@@ -93,13 +40,13 @@
 // Drafts returns the invoices directly inside workDir and the directory of
 // output. The scan is best effort: the archive check still guarantees
 // unique numbers, so an unreadable directory or file is skipped.
-func (s *Store) Drafts(workDir, output string) []billing.Head {
+func (s *Store) Drafts(workDir, output string) []invoice.Invoice {
 	dirs := []string{workDir}
 	if output != "" {
 		dirs = append(dirs, filepath.Dir(output))
 	}
 	seen := make(map[string]bool, len(dirs))
-	var heads []billing.Head
+	var drafts []invoice.Invoice
 	for _, dir := range dirs {
 		if strings.TrimSpace(dir) == "" || seen[dir] {
 			continue
@@ -127,43 +74,51 @@
 			if err := decodeYAMLFile(filepath.Join(dir, entry.Name()), &identity, false); err != nil || identity.Invoice == nil {
 				continue
 			}
-			heads = append(heads, identity.head())
+			drafts = append(drafts, identity.draft())
 		}
 	}
-	return heads
+	return drafts
 }
 
-// Destination returns path, or <number>.yaml in workDir when path is "",
-// after refusing a directory there, and an existing file unless overwrite
-// is set.
-func (s *Store) Destination(path, workDir, number string, overwrite bool) (string, error) {
+// Create writes a new invoice to path, or to <number>.yaml in opts.Dir,
+// from the document at from, and returns the file.
+func (s *Store) Create(path, from string, inv invoice.Invoice, opts billing.CreateOptions) (string, error) {
 	if path == "" {
-		path = filepath.Join(workDir, number+".yaml")
+		path = filepath.Join(opts.Dir, inv.Header.Number.Trim()+".yaml")
 	}
 	if err := refuseDirOutput(path); err != nil {
 		return "", err
 	}
-	if !overwrite && fileExists(path) {
+	if !opts.Overwrite && fileExists(path) {
 		return "", &billing.OutputExistsError{Path: path}
 	}
-	return path, nil
-}
-
-// Create writes a new invoice to path from the document at from.
-func (s *Store) Create(path, from string, inv invoice.Invoice, opts billing.CreateOptions) error {
-	document, err := loadSourceDocument(from)
+	if opts.Overwrite && s.Protected != nil {
+		protected, err := s.Protected(path)
+		if err != nil {
+			return "", err
+		}
+		if protected {
+			return "", &billing.ArchivedOutputError{Path: path}
+		}
+	}
+	document, err := loadYAMLDocument(from)
 	if err != nil {
-		return err
+		return "", err
 	}
 	root, err := documentRootMapping(document, from)
 	if err != nil {
-		return err
+		return "", err
 	}
 	setArchiveLink(root, inv.Archive)
 	if inv.CustomerID.IsSet() {
 		setMappingString(root, "customer_id", string(inv.CustomerID))
 	}
 	if h := inv.Header; h != nil {
+		// The header is written in place, so it must be a mapping, not an
+		// alias to one.
+		if node := findMappingValue(root, "invoice"); node != nil && node.Kind != yaml.MappingNode && node.Tag != "!!null" {
+			return "", fmt.Errorf("%s: `invoice` must be a mapping", from)
+		}
 		invoiceNode := getOrCreateMappingNode(root, "invoice")
 		for _, field := range headerFields(h) {
 			if field.value != "" && field.key != "vat_percent" {
@@ -180,15 +135,15 @@
 	}
 	if opts.Check != billing.CheckNone {
 		if err := decodeYAMLDocument(document, from, &invoice.Invoice{}, opts.Check == billing.CheckStrict); err != nil {
-			return err
+			return "", err
 		}
 	}
 	data, err := encodeYAMLDocument(document)
 	if err != nil {
-		return err
+		return "", err
 	}
 	if opts.DryRun {
-		return nil
+		return path, nil
 	}
 	write := fsutil.WriteNewFile
 	if opts.Overwrite {
@@ -196,32 +151,11 @@
 	}
 	if err := write(path, data, fsutil.Public); err != nil {
 		if errors.Is(err, fs.ErrExist) {
-			return &billing.OutputExistsError{Path: path}
+			return "", &billing.OutputExistsError{Path: path}
 		}
-		return err
-	}
-	return nil
-}
-
-// loadSourceDocument reads the document a new invoice starts from: the
-// front matter of a Markdown file, else YAML.
-func loadSourceDocument(path string) (*yaml.Node, error) {
-	if !isMarkdown(path) {
-		return loadYAMLDocument(path)
-	}
-	document, ok, err := loadArchivedInvoiceDocument(path)
-	if err == nil && !ok {
-		err = fmt.Errorf("%s: archived invoice could not be loaded", path)
-	}
-	return document, err
-}
-
-func isMarkdown(path string) bool {
-	switch strings.ToLower(filepath.Ext(path)) {
-	case ".md", ".markdown":
-		return true
+		return "", err
 	}
-	return false
+	return path, nil
 }
 
 type headerField struct {
@@ -319,12 +253,11 @@
 		deleteMappingKey(root, internalMetadataKey)
 		return
 	}
-	setArchiveMetadata(root, string(link.ArchivePath), string(link.ArchiveReplacePath))
+	setArchiveMetadata(root, string(link.ArchivePath))
 }
 
 // ReadArchived reads what the archive lists of the invoice at path. ok is
-// false for a file that is not an invoice: Markdown without front matter,
-// or a document whose invoice fields cannot be read. An archived invoice
+// false for a document whose invoice fields cannot be read. An archived invoice
 // without a status is listed as `archived`.
 func ReadArchived(path string) (billing.ArchiveEntry, bool, error) {
 	identity, ok, err := archivedInvoiceIdentity(path)
@@ -352,8 +285,3 @@
 	}
 	return nil
 }
-
-func isDecodeError(err error) bool {
-	var decodeErr *billing.DecodeError
-	return errors.As(err, &decodeErr)
-}
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/schema.go proto/internal/store/schema.go
--- base/internal/store/schema.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/schema.go	2026-10-09 11:45:27.674732368 +0000
@@ -106,8 +106,7 @@
 		"vat_percent": "VATPercent",
 	},
 	reflect.TypeFor[invoice.ArchiveLink](): {
-		"archive_path":         "ArchivePath",
-		"archive_replace_path": "ArchiveReplacePath",
+		"archive_path": "ArchivePath",
 	},
 }
 
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/store.go proto/internal/store/store.go
--- base/internal/store/store.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/store.go	2026-10-09 11:45:27.672426731 +0000
@@ -24,6 +24,9 @@
 	Host  Host
 	Getwd func() (string, error)
 	Files Files
+	// Protected reports whether path is an archived invoice, which Create
+	// never overwrites.
+	Protected func(path string) (bool, error)
 }
 
 var (
@@ -148,13 +151,7 @@
 }
 
 func (s *Store) template(path string) billing.Template {
-	return billing.Template{
-		Name: filepath.Base(path),
-		Path: path,
-		FindAsset: func(rel string, dir bool) string {
-			return s.Host.findAsset(path, rel, dir)
-		},
-	}
+	return billing.Template{Name: filepath.Base(path), Path: path}
 }
 
 // Templates lists the template catalog and returns its directory.
@@ -176,8 +173,12 @@
 	return templates, dir, nil
 }
 
-// Paths reports where each file comes from for a command run in start.
-func (s *Store) Paths(start string) ([]billing.PathReport, error) {
+// Paths reports where each file comes from for this run.
+func (s *Store) Paths() ([]billing.PathReport, error) {
+	start, err := s.workDir()
+	if err != nil {
+		return nil, err
+	}
 	reports, err := s.Host.Paths(start)
 	if err != nil {
 		return nil, err
@@ -210,31 +211,6 @@
 	return dir, files, nil
 }
 
-// LegacyFiles returns the legacy files the config directory lacks.
-func (s *Store) LegacyFiles() ([]string, error) { return s.Host.LegacyFilesToCopy() }
-
-// CopyLegacy copies LegacyFiles into the config directory.
-func (s *Store) CopyLegacy() ([]string, error) { return s.Host.CopyLegacyFiles() }
-
-// LegacyFilesUsed returns the files read from the legacy directory so far.
-func (s *Store) LegacyFilesUsed() []string { return s.Host.LegacyFilesUsed() }
-
-// Locations are where invox keeps its files by default.
-func (s *Store) Locations() billing.Locations {
-	h := s.Host
-	return billing.Locations{
-		ConfigDir:      h.ConfigDir(),
-		ConfigFile:     h.GlobalConfigPath(),
-		Customers:      h.GlobalCustomersPath(),
-		Issuer:         h.GlobalIssuerPath(),
-		Defaults:       h.GlobalInvoiceDefaultsPath(),
-		Template:       h.GlobalTemplatePath(),
-		ArchiveDir:     h.DefaultArchiveDir(),
-		LegacyDir:      h.LegacyConfigDir(),
-		ConfigTemplate: h.ConfigTemplate(),
-	}
-}
-
 // Settings reads the parts of config.yaml the use cases need.
 func (h Host) Settings() (billing.Settings, error) {
 	cfg, err := h.Config()
diff -ruN -x .git -x '*_test.go' -x testdata base/internal/store/yamldoc.go proto/internal/store/yamldoc.go
--- base/internal/store/yamldoc.go	2026-10-09 09:01:29.000000000 +0000
+++ proto/internal/store/yamldoc.go	2026-10-09 11:45:42.837109625 +0000
@@ -5,9 +5,7 @@
 	"errors"
 	"fmt"
 	"os"
-	"path/filepath"
 	"reflect"
-	"strings"
 
 	yaml "gopkg.in/yaml.v3"
 )
@@ -138,21 +136,6 @@
 	}
 }
 
-func markdownFrontMatter(source []byte) ([]byte, bool) {
-	text := strings.ReplaceAll(string(source), "\r\n", "\n")
-	if !strings.HasPrefix(text, "---\n") {
-		return nil, false
-	}
-	remainder := text[len("---\n"):]
-	end := strings.Index(remainder, "\n---\n")
-	if end < 0 {
-		return nil, false
-	}
-	// The leading newline stands in for the opening `---`, so YAML line
-	// numbers in errors match the lines of the Markdown file.
-	return []byte("\n" + remainder[:end]), true
-}
-
 func findMappingValue(node *yaml.Node, key string) *yaml.Node {
 	if node == nil || node.Kind != yaml.MappingNode {
 		return nil
@@ -177,33 +160,6 @@
 	}
 }
 
-func loadArchivedInvoiceDocument(path string) (*yaml.Node, bool, error) {
-	switch strings.ToLower(filepath.Ext(path)) {
-	case ".yaml", ".yml":
-		document, err := loadYAMLDocument(path)
-		if err != nil {
-			return nil, true, err
-		}
-		return document, true, nil
-	case ".md", ".markdown":
-		source, err := os.ReadFile(path)
-		if err != nil {
-			return nil, false, err
-		}
-		frontMatter, ok := markdownFrontMatter(source)
-		if !ok {
-			return nil, false, nil
-		}
-		document, err := parseYAMLDocumentSource(frontMatter, "front matter in "+path)
-		if err != nil {
-			return nil, true, err
-		}
-		return document, true, nil
-	default:
-		return nil, false, nil
-	}
-}
-
 // decodeYAMLFile reads the YAML file at path and decodes its root mapping
 // into out, a pointer to a schema struct. Problems with values come back as
 // *DecodeError values, several joined with errors.Join in file order.
```
