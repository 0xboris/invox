# Narrowing the billing ports

Follow-up unit B of `docs/design/target-progress.md`. Base: `47aef7e` on `experiment/clean-architecture`
(unit A landed). Every `file:line` below refers to that commit.

## Summary

After the refactor `internal/billing` still decides where files go. It joins, cleans and splits
paths with 15 `filepath` calls, makes and removes a scratch directory, asks `Exists` and `Stat`, and calls
39 port methods. This design moves every path, temp-directory, copy and existence decision into
the driven adapters. The six ports shrink from 39 to 36 methods, and the file-system helpers among them
(`Exists`, `Stat`, `CheckOutput`, `Resolve`, `Existing`, `HistoryDir`, `FindFile`, `Scratch`,
`Copy`) give way to six domain-level methods (`Destination`, `Place`, `Duplicate`, `Source`,
`Build`, `CheckAttachment`). `*billing.Service` drops `EmailPaths` and keeps 21 methods.

No user sees a difference. stdout, stderr, exit codes and the files invox writes stay
byte-for-byte the same. A prototype of the whole design at `47aef7e` passes the full test suite with
`-race`, the target test, golangci-lint, the 35 testscript goldens and the docs regeneration. A
differential run of 50 order-sensitive scenarios against the base binary matched in 49. Each
match covers the exit code, stdout, stderr, and every file's path, bytes and mode. The 50th
differs only in the `.eml` timestamp, and two runs of the base binary differ the same way. See [How this design was checked](#how-this-design-was-checked).

The next engineer inherits a `billing` that reads as business rules. Where an invoice goes,
what counts as "the same file", and how a PDF finds its YAML are each answered in the one
adapter that owns that directory.

## The rule

`billing` may receive a path from the CLI or from an adapter, keep it, pass it to another port and
put it into an error message. It treats a path as an opaque reference. It may test a reference for
presence (`req.Output != ""`, `strings.TrimSpace(dir) == ""`).

`billing` may not:

- import `path` or `path/filepath`,
- join, clean, split or compare paths, or derive one path from another (extension, base name,
  directory),
- name a file (`<number>.yaml`, `<name>.tex`),
- create, remove or copy files or directories, temporary ones included,
- ask whether a file exists or can be read.

The CLI keeps what it already does at the boundary: it makes paths absolute, fills in the
default PDF and `.eml` names (`internal/cmd/invoice/email/email.go:147-148`) and checks
extensions (`email.go:117-133`). With this design it also says whether the `email` input is the
invoice or its PDF, a fact it already parsed in `validate`.

## Disposition of every method

"Order" says why the first error a user sees stays the same. The [error-order section](#how-error-order-stays-the-same)
works through each use case.

### Invoices (9 methods, 7 after)

| Method | Decision | Where its logic goes | Order |
|---|---|---|---|
| `Load(path)` | Keep | `store` | Unchanged. |
| `ArchivedHead(path)` | Keep (unit A) | `store` | Unchanged. |
| `Head(path)` | Keep | `store` | Unchanged. |
| `Drafts(dirs []string)` | Change to `Drafts(workDir, output string)` | `store.Drafts` builds `[workDir, filepath.Dir(output)]`, which `billing/new.go:78-81` builds today. | Same directory list, same de-duplication, and the scan never errors. |
| `Create(path, from, inv, opts)` | Keep | `store` | Unchanged. |
| `Update(path, change)` | Keep. Archiving stops calling it. | `store.Update` becomes `Rewrite` plus one write. | Callers `markBuilt` and `Increment` are unchanged. |
| `CheckOutput(path, overwrite)` | Merge into new `Destination(path, workDir, number, overwrite)` | `store.Destination` names the default file (`filepath.Join(workDir, number+".yaml")`, today `new.go:92`) and runs the same two checks. | Called at the same three points (`new.go:56`, `new.go:94`, `archive.go:346`) with the same path. |
| `Exists(path)` | Remove | Archiving: `archive.Place`. Email: `archive.Source`. | Each check runs at the same point in the sequence as the call it replaces. |
| `Stat(path)` | Remove | `Mailer.CheckAttachment` | Same point in `DraftEmail` (`email.go:64`). `billing` keeps the `read %s: %w` wrap. |

### Directory (14 methods, 14 after)

`billing` passes every Directory result through as an opaque reference or a label, so this design
changes none of them. The `Locate` label that `billing` puts into messages (`new.go:69`, `new.go:159`,
`Bundle.IssuerPath` in `validate.go:91`) is a reference from an adapter, which the rule allows.

| Method | Decision | Reason |
|---|---|---|
| `Locate(f)` | Keep | Presence checks in a fixed order (`validate.go:39-49`, `numbering.go:117`) and labels for messages. Folding it into `Customers` or `Issuer` would report a decode error of `customers.yaml` before a missing template. |
| `Customer`, `Customers`, `Issuer`, `Defaults` | Keep | Required by the target test. `Defaults` returns the file `New` starts from, passed through to `Load` and `Create`. |
| `Template`, `Templates` | Keep | Required. `Template.Path` and `FindAsset` are read only by `render/latex`. |
| `Paths(start)` | Keep | `start` is the CLI's working directory, passed through. Changing it to the spec's `Paths()` is optional and outside this unit. |
| `EditablePath(f)`, `Init()` | Keep | Required. Their results go to the CLI. |
| `LegacyFiles`, `CopyLegacy`, `LegacyFilesUsed`, `Locations` | Keep | CLI support queries that `billing` passes through (`directory.go:102-110`, `service.go:39-47`). They handle no path inside `billing`. |

### Archive (9 methods, 8 after)

| Method | Decision | Where its logic goes | Order |
|---|---|---|---|
| `Entries()` | Keep | `archive` | Unchanged. |
| `Add(src, Placement)` | Change to `Add(src, Placement, AddOptions)` | `archive.Add` also finds the replaced files (today `Existing`, `archive.go:114`), returns `*ArchiveReplaceError` unless `Replace`, returns the dry-run result, and writes the archived file once with `opts.Change` applied. | `Existing`, the replace refusal and the dry-run return keep their order and stay after the number check. The single write is covered in [Archive.Add owns the whole move](#archiveadd-owns-the-whole-move). |
| `Checkout(ref, workDir)` | Keep | `archive` | Unchanged. |
| `Dir()` | Keep | `archive` | `archive` and `ListArchive` still need it first (`archive.go:59`, `archive.go:305`). `CheckNumberUnique` stops calling it, because `Duplicate` locates the archive in the same position. |
| `HistoryDir()` | Remove | `Place` puts it into `Placement.HistoryDir`. | Its only error is the one `Locate` returns, and `Dir` already returned that error one line earlier (`archive.go:59-69`). |
| `Resolve(name)` | Remove | `Place` (the link's `archive_path` and `archive_replace_path`) and `Duplicate` (the names to exclude). | Same calls, same order. The `archive_replace_path` resolve moves ahead of the number check. That move is proven in the [Archive walkthrough](#archive-archive-add-build---archive). |
| `Existing(paths...)` | Remove | `Add` | Still after the number check. |
| `Protects(path)` | Keep | `archive` | Unchanged. `billing` asks a yes-or-no question and owns the policy (`ArchivedOutputError`). |
| `FindFile(names...)` | Replace with `Source(pdf)` | `archive.Source` looks for `<pdf base>.yaml` and `.yml` next to the PDF, then searches the archive (today `email.go:156-172`). | Same order: `.yaml` next to it, `.yml` next to it, then the archive walk. |
| *(new)* `Place(src, head)` | Add | Moves the target path (`archive.go:71-72`, `79`, `90-96`, `103-111`) into `archive`. | See the walkthrough. |
| *(new)* `Duplicate(src, head)` | Add | Moves the exclusion set and `filepath.Clean` comparisons (`archive.go:198-234`) into `archive`. | Locate, then the empty-directory check, then the empty-number check, then resolves, the walk and the first match: today's order. |

### Renderer (4 methods, 3 after)

| Method | Decision | Where its logic goes | Order |
|---|---|---|---|
| `Render` | Keep | `render/latex` | Unchanged. |
| `Write` | Keep | `render/latex` | Unchanged. `Render` (the use case) still writes straight to the output. |
| `Scratch()` | Remove | `Renderer.Build` | Same steps in the same order: `os.MkdirTemp`, `Write`, `Compile`, read, `fsutil.WriteFile`. Cleanup is still deferred. |
| `Copy(src, dst)` | Remove | `Renderer.Build` | As above. |
| *(new)* `Build(ctx, c Compiler, t, source, output)` | Add | `render.go:133-149` moves verbatim, including the `.tex` naming. `billing` passes `s.Compiler`, so `Service.Compiler` keeps its job. | As above. |

### Compiler (1 method, 1 after)

| Method | Decision | Reason |
|---|---|---|
| `Compile(ctx, sourcePath)` | Keep | `tectonic` returns the PDF next to the source (`tectonic.go:48-53`). Only `Renderer.Build` calls it now, with a path `Build` made. |

### Mailer (2 methods, 3 after)

| Method | Decision | Reason |
|---|---|---|
| `Draft(ctx, m)` | Keep | Unchanged. The temporary-draft directory was already the adapter's (`email/mailer.go:40-58`). |
| `Check(m)` | Keep | Checks a kept `.eml` output in a dry run. It runs after the subject is built, so it cannot merge with the attachment check. |
| *(new)* `CheckAttachment(path)` | Add | Replaces `Invoices.Stat`. The PDF is the attachment, so the mailer answers whether it can be attached. `email.Mailer` and `applemail.Composer` each implement it as `os.Stat`. |

### Service methods outside the 14 use cases (8)

| Method | Decision | Reason |
|---|---|---|
| `EmailPaths` | Remove | It did three things. It defaulted the PDF and `.eml` names, which the CLI already does (`email.go:147-148`). It classified the input by extension, which moves to the CLI next to its existing check (`email.go:124`). It looked up the YAML of a PDF, which becomes `Archive.Source`. Its own errors, "input path is required" and "input must end with .yaml, .yml, or .pdf", cannot be reached from the CLI. `validate` rejects both cases first, and only the store shim test `TestResolveEmailDraftPaths` pins the second one. |
| `CheckNumberUnique` | Keep | A step of `Validate` that the store tests call directly (`legacy_api_test.go:143-145`). It now calls `Archives.Duplicate`. |
| `NextNumber` | Keep | Shared by `New` and `Increment`. Tests call it. No paths. |
| `DefaultTemplate` | Keep | `template list --json` (`internal/cmd/template/list/list.go:113`). Folding it into `ListTemplates` would make a `Locate` failure fail the plain listing as well. |
| `Locations`, `LegacyFiles`, `CopyLegacyFiles`, `LegacyFilesUsed` | Keep | Pass-throughs that keep "every command calls a `Service` method". No path handling. |

## Archive.Add owns the whole move

Yes, `Archive.Add` should own the whole move, including the status write. Today `archive` copies
the bytes into the archive (`archive/adapter.go:95-128`, which also removes the source) and then
`billing` rewrites the copy with `Invoices.Update` (`billing/archive.go:140-147`). If that
`Update` fails, the archive holds a copy with `status: built` and the working file is gone. The
pre-refactor code wrote the archived file once.

The design writes it once again, without letting `archive` read YAML or own the status rule:

- `billing` passes the rule as data: `AddOptions.Change` sets `status: archived` and clears the
  `_invox` link, the same function it passes to `Update` today.
- `store` gains `Rewrite(path, change) ([]byte, error)`, the body of today's `Update` minus the
  write. `Update` becomes `Rewrite` followed by `fsutil.WriteFile(path, data, fsutil.Public)`.
- `factory` hands `st.Rewrite` to `archive.Archive{Rewrite: ...}`, next to the `Read` function
  it already takes from `store` (`factory/factory.go:54`). The spec sanctions that pattern for
  `Read`.
- `archive.Add` calls `Rewrite(src, opts.Change)` after the dry-run return and before it creates
  directories or backups. It then writes those bytes where it writes the copied bytes today.

The outputs stay identical for these reasons:

- **Bytes.** `Update` today loads the archived copy, whose bytes equal `src`, applies the change
  and encodes it. `Rewrite` applies the same change to the same bytes with the same code. The
  differential run compared the archived file's SHA-256 after `archive add` with comments and an
  empty `_invox: {}` link, after re-archiving with `--yes`, and after `build --archive`. All
  matched.
- **Mode.** A new archived file is still created by `fsutil.WriteNewFile(..., fsutil.Private)`.
  Re-archiving still uses `fsutil.WriteFile`, which keeps the existing file's mode
  (`internal/fsutil/fsutil.go:65-67`). Today's `Update` then rewrote the file with `WriteFile`,
  which also kept the mode. The differential run compared modes. All matched.
- **stdout and stderr.** `ArchiveResult` is built from the same fields.
- **Errors.** `Rewrite` can only fail on the source file. Every check it makes ran earlier on
  the same file. `Head` parses the YAML and rejects a root that is not a mapping ("root value
  must be a mapping"), and `requireHeader` rejects an `invoice` key that is not a mapping or is an
  alias (unit A's `HeaderShape`). Scenarios `archive-list-root`, `archive-scalar-root` and
  `archive-aliased-header` pin those three. The read error is the raw `os.ReadFile` error that `Add` returns
  today (`store/yamldoc.go:17-21`). The one difference is when a file changes between `Head` and
  `Add`. Then the error names the source, not the archived copy, and nothing has been moved yet.
  Today the source is gone by that point.

## How error order stays the same

### Archive (`archive add`, `build --archive`)

Today (`billing/archive.go:50-149`):

1. `Head`, then `requireHeader`.
2. `Dir`: the error, or "archive directory is unavailable".
3. `HistoryDir`.
4. For a working copy: the re-archiving status check, then `Resolve(archive_path)`. For any
   other invoice: "status: missing value", the archiving status check, then `Exists`, which
   reports "… already exists".
5. "… is already in the archive directory".
6. `numberUnique`: return early without a number, `Resolve` both link names, `Entries`, then the
   duplicate error.
7. `Resolve(archive_replace_path)`.
8. `Existing`.
9. `*ArchiveReplaceError`, then the dry-run return.
10. `Add`, then `Update`.

After (prototype):

1. `Head`, then `requireHeader`.
2. `Dir`: the error, or "unavailable".
3. `archivable`: the same status checks in the same order, pulled into a pure function.
4. `Place`: `Resolve(archive_path)` or `Exists`, then the same-file check, then
   `Resolve(archive_replace_path)`.
5. `Duplicate`: locate the archive, return early without a directory or a number, `Resolve` the
   link names, then the walk.
6. `Add`: `Existing`, `*ArchiveReplaceError`, the dry-run return, `Rewrite`, then the write.

Three steps move:

- **`HistoryDir`.** Its only error is the `Locate` error that step 2 already returned.
  `Locate` is the same function, `h.ResolveArchiveDir`, on both calls.
- **`Resolve(archive_replace_path)`.** It moves from step 7 to inside `Place`, just before the
  number check. With a number, today's step 6 resolves `archive_path` and then
  `archive_replace_path` before anything else can fail. The `archive_path` resolve already
  succeeded in step 4, and `Resolve` is a pure function of the archive directory and the name.
  So the first error is the same `archive_replace_path` error either way. Without a number, step 6
  returns nil at once, so nothing can fail between the old position and the new one. The
  `archive_replace_path == archive_path` case resolves to a path that already succeeded. The
  scenarios `replace-resolve-no-number` and `replace-resolve-with-number` check both branches.
- **`Rewrite`.** It replaces both today's `os.ReadFile(src)` inside `Add` and the trailing
  `Update`. See the previous section.

`Existing` stays after the number check. A directory at the target path, together with a
duplicate number, still reports the duplicate (scenario `target-dir-vs-duplicate`).

The `Confirm` loop in `Service.Archive` (`archive.go:37-48`) is unchanged. It still matches
`*ArchiveReplaceError`, which `Add` now returns, and calls `archive` again with `Replace` set.

### Validate (`CheckNumberUnique`)

Today: `Head`, the no-header check, `Dir` (error, then empty means nil), then `numberUnique`, which
returns early without a number before resolving. After: `Head`, the no-header check, then
`Duplicate`, which does locate, the empty-directory check, the empty-number check, resolve and
walk in that order. That is the same sequence. Scenario `validate-bad-link` pins the warning for an
absolute `archive_path`.

### New

`Destination(req.Output, …)` replaces the early `CheckOutput(req.Output)` at the same line.
`Drafts(req.WorkDir, req.Output)` scans the same directories. `Destination(req.Output,
req.WorkDir, number, …)` replaces the default-name join and the second `CheckOutput`, before
`issuerDueDays`, as today. Scenarios `new-default-exists-and-bad-due` and
`new-output-exists-and-bad-due` pin "already exists" ahead of the `due_days` error.

### EditArchived

`Destination(checkout.Path, workDir, "", overwrite)` replaces `CheckOutput(checkout.Path,
overwrite)` at `archive.go:346`. Nothing else changes.

### Build

`Renderer.Build` runs the five steps of `billing.compile` (`render.go:133-149`) in the same
order, with the same `.tex` name and the same deferred `os.RemoveAll`. `TestBuildInvoicePDF*` and
the fake-tectonic CLI tests pin the results.

### DraftEmail

Today: `locateParties`, `EmailPaths` (lookup errors), `loadContext`, status, `Stat`, recipient,
settings, subject, then in a dry run with `-o` the `Mailer.Check`. After: `locateParties`,
`emailInvoice` (`req.Invoice`, else `Archives.Source(req.FromPDF)` and "no matching invoice YAML
found …"), `loadContext`, status, `Mailer.CheckAttachment`, recipient, settings, subject, then
`Mailer.Check`.

The CLI classifies the input with `strings.EqualFold(filepath.Ext(invoicePath), ".pdf")`. Today
`billing` uses `strings.ToLower(filepath.Ext(input))` on the same absolute path. The two agree
for every input that `validate` lets through. `EmailPaths` also called `strings.TrimSpace` on the
input, the PDF and the output. That is a no-op on what the CLI sends. Every path is absolute, so
it starts with a separator or a volume. A trailing space would make the extension `.pdf ` or
`.eml `, which `validate` (`email.go:121-131`) and `shared.RequireExtension` reject first.
Scenarios `email-upper-PDF`, `email-pdf-with-p`, `email-missing-pdf-and-bad-subject` and
`email-dry-run-output-exists-bad-subject` pin the boundaries.

## Final interfaces

Only the types that change are shown. `Directory`, `CustomerTable`, `Template`, `Locations`,
`PathReport`, `InitFile`, `Head`'s fields, `CreateOptions`, `Check`, `ArchiveEntry`, `Backup`,
`ArchiveResult`, `EPC`, `Message` and `Draft` stay as they are at `47aef7e`.

```go
// WorkingCopy reports whether h is a working copy from `archive edit`,
// which re-archiving writes back over the archived files it names.
func (h Head) WorkingCopy() bool { return h.ArchivePath != "" }

// Invoices reads and writes invoice files. Update keeps the file's comments
// and layout; TestInvoiceWritesKeepComments pins that.
type Invoices interface {
	// Load decodes the invoice at path strictly. Values that do not fit
	// come back as joined *DecodeError values, with the rest decoded.
	Load(path string) (invoice.Invoice, error)
	// ArchivedHead is Head for an archived invoice, which may be the front
	// matter of a Markdown file. It decodes the whole invoice strictly, as
	// Load does, and returns those errors.
	ArchivedHead(path string) (Head, error)
	// Head reads what numbering and the archive need of the invoice at
	// path. Values that do not decode are left unset and reported as
	// *DecodeError values.
	Head(path string) (Head, error)
	// Drafts returns the invoices directly in workDir and, when output is
	// set, in the directory output is in, best effort: files that cannot
	// be read are left out.
	Drafts(workDir, output string) []Head
	// Destination returns the file a new invoice is written to: path, or
	// when path is "", <number>.yaml in workDir. It returns an
	// *OutputIsDirError when that is a directory and, unless overwrite is
	// set, an *OutputExistsError when it exists.
	Destination(path, workDir, number string, overwrite bool) (string, error)
	// Create writes a new invoice to path from the document at from, which
	// may be an archived invoice, keeping its comments and keys. Every
	// customer and header field set in inv is written, as text. Positions
	// is added when inv's is not nil and from has none. The `_invox` link
	// is replaced by inv's.
	Create(path, from string, inv invoice.Invoice, opts CreateOptions) error
	// Update rewrites the invoice at path with change applied, writing
	// back only the fields that changed.
	Update(path string, change func(*invoice.Invoice) error) error
}

// Placement is where Archive.Add puts an invoice. Archive.Place makes it.
type Placement struct {
	// Path is the archived file to write.
	Path string
	// Overwrite allows Path to exist, when a working copy is re-archived.
	Overwrite bool
	// Remove is an archived file the invoice supersedes, removed after it
	// is written, or "".
	Remove string
	// HistoryDir is where replaced files' previous versions are kept.
	HistoryDir string
}

// AddOptions control Archive.Add.
type AddOptions struct {
	// Replace allows writing over archived files. Without it, Add returns
	// an *ArchiveReplaceError when it would replace any.
	Replace bool
	// DryRun runs every check and returns the result without writing. The
	// result's backups have no BackupPath.
	DryRun bool
	Now    time.Time
	// Change is applied to the invoice as it is archived, keeping its
	// comments and layout, so the archived file is written once.
	Change func(*invoice.Invoice) error
}

// Archive holds finished invoices.
type Archive interface {
	// Entries reads every archived invoice, in the lexical order of a
	// directory walk.
	Entries() ([]ArchiveEntry, error)
	// Dir returns the archive directory, "" when there is none.
	Dir() (string, error)
	// Place says where archiving the invoice at src, whose head is head,
	// writes it: over the archived file a working copy names, else under
	// src's name in the archive directory, which must not exist yet. It
	// refuses src when it is that file already.
	Place(src string, head Head) (Placement, error)
	// Duplicate returns the archived invoice, in file name order, that has
	// head's number, other than src and, for a working copy, the archived
	// files it replaces. It returns "" when there is none, when head has
	// no number, or when there is no archive directory.
	Duplicate(src string, head Head) (string, error)
	// Add moves the invoice at src into the archive at p with opts.Change
	// applied, in one write, after backing up the archived files it
	// replaces. Without opts.Replace it refuses to replace any.
	Add(src string, p Placement, opts AddOptions) (ArchiveResult, error)
	// Checkout resolves ref, an archived invoice relative to the archive
	// directory, and says where its working copy in workDir goes.
	Checkout(ref, workDir string) (Checkout, error)
	// Protects reports whether path is an existing file inside the archive
	// directory, which nothing but re-archiving overwrites.
	Protects(path string) (bool, error)
	// Source returns the invoice YAML file the PDF at pdf was built from:
	// the one with its name next to it, else the one in the archive. It
	// returns "" when there is none, and an error when the archive has
	// several.
	Source(pdf string) (string, error)
}

// Renderer turns an invoice into the source the Compiler reads.
type Renderer interface {
	// Render checks template t and fills it in. It writes nothing.
	Render(t Template, inv *invoice.Context, epc EPC) (string, error)
	// Write writes source to path and copies t's assets next to it.
	Write(t Template, source, path string) error
	// Build writes source with t's assets to a scratch directory, compiles
	// it there with c, and copies the PDF to output.
	Build(ctx context.Context, c Compiler, t Template, source, output string) error
}

// Compiler turns rendered source into a PDF and returns its path.
type Compiler interface {
	Compile(ctx context.Context, sourcePath string) (string, error)
}

// Mailer drafts an email with the PDF attached.
type Mailer interface {
	Draft(ctx context.Context, m Message) (Draft, error)
	// Check runs the checks Draft runs on m.Output without writing.
	Check(m Message) error
	// CheckAttachment returns why the file at path cannot be attached, or
	// nil.
	CheckAttachment(path string) error
}

// EmailRequest says which invoice DraftEmail drafts an email for.
type EmailRequest struct {
	// Invoice is the invoice file. It is "" when the user named the built
	// PDF instead: FromPDF is then that PDF, and the invoice is the YAML
	// file with its name next to it, else in the archive.
	Invoice string
	FromPDF string
	// PDF is the attachment, Output the .eml draft.
	PDF    string
	Output string
	// Keep, To, Subject, Overwrite and DryRun are unchanged.
	Keep      bool
	To        string
	Subject   string
	Overwrite bool
	DryRun    bool
}
```

The adapter structs gain or change these exported parts:

- `archive.Archive` gains `Rewrite func(path string, change func(*invoice.Invoice) error) ([]byte, error)`.
- `store.Store` gains `Rewrite`. `store.Store.Drafts` and `Destination` change as above. `CheckOutput`, `Exists` and `Stat` go.
- `latex.Renderer` gains `Build`. `Scratch` and `Copy` go.
- `email.Mailer` and `applemail.Composer` gain `CheckAttachment`.

### Method counts

| Port | 47aef7e | After |
|---|---|---|
| `Invoices` | 9 | 7: `ArchivedHead Create Destination Drafts Head Load Update` |
| `Directory` | 14 | 14: unchanged |
| `Archive` | 9 | 8: `Add Checkout Dir Duplicate Entries Place Protects Source` |
| `Renderer` | 4 | 3: `Build Render Write` |
| `Compiler` | 1 | 1: `Compile` |
| `Mailer` | 2 | 3: `Check CheckAttachment Draft` |
| total | 39 | 36 |

`*billing.Service` keeps 21 methods. The 14 use cases are `New Increment Validate Render Build
Archive EditArchived DraftEmail ListCustomers ListArchive Paths Init ListTemplates
EditablePath`. The 7 others are `CheckNumberUnique CopyLegacyFiles DefaultTemplate LegacyFiles
LegacyFilesUsed Locations NextNumber`.

## Acceptance checks

Add `internal/archtest/ports_test.go`, which has no build tag, so `go test ./...` and
`verify-target.sh` run it. `target_test.go` is behind the `target` tag and owns the names
`typecheck` and `methodNames`. This file therefore uses its own names. It holds three checks:

1. **Exact method sets.** Each port's methods, and the exported methods of `*billing.Service`,
   equal the lists above (sorted, compared with `slices.Equal`). A method added to a port fails
   the test until this doc and the list change together.
2. **No path packages.** No non-test file in `internal/billing` imports `path` or
   `path/filepath`. The test parses the files with `go/parser`, so it checks direct imports, as
   depguard does.
3. **No file names.** No string literal in a non-test `internal/billing` file matches
   `\.(?i:ya?ml|pdf|eml|tex|md|markdown)\b`. Without this check, `number + ".yaml"` or
   `strings.TrimSuffix(p, ".pdf")` would pass check 2. At `47aef7e` it flags 10 literals in
   `email.go`, `new.go` and `render.go`. The prototype has none. It does not catch
   `strings.LastIndex(p, ".")`. Review has to catch that.

Also add a depguard rule to `.golangci.yml`, so the linter flags the import before the tests run.
Put it before the driven-adapter rules:

```yaml
        # Use cases hand paths to the adapters; they never build one.
        use-cases-no-paths:
          files:
            - "**/internal/billing/**"
            - "!$test"
          deny:
            - pkg: path$
              desc: a driven adapter builds paths
            - pkg: path/filepath
              desc: a driven adapter builds paths
```

It is a separate rule because the `use-cases` rule is an allow list with `$gostd`. On the
prototype it reports 0 issues. At `47aef7e` it reports the 4 files that import `path/filepath`.

The test as validated on the prototype (it passes there and fails at `47aef7e` with the expected
diffs):

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
// serviceMethods that of *billing.Service. docs/design/ports-narrowing.md
// explains each method; a new one belongs there first.
var (
	narrowPorts = map[string][]string{
		"Invoices":  {"ArchivedHead", "Create", "Destination", "Drafts", "Head", "Load", "Update"},
		"Directory": {"CopyLegacy", "Customer", "Customers", "Defaults", "EditablePath", "Init", "Issuer", "LegacyFiles", "LegacyFilesUsed", "Locate", "Locations", "Paths", "Template", "Templates"},
		"Archive":   {"Add", "Checkout", "Dir", "Duplicate", "Entries", "Place", "Protects", "Source"},
		"Renderer":  {"Build", "Render", "Write"},
		"Compiler":  {"Compile"},
		"Mailer":    {"Check", "CheckAttachment", "Draft"},
	}
	serviceMethods = []string{
		"Archive", "Build", "CheckNumberUnique", "CopyLegacyFiles", "DefaultTemplate", "DraftEmail",
		"EditArchived", "EditablePath", "Increment", "Init", "LegacyFiles", "LegacyFilesUsed",
		"ListArchive", "ListCustomers", "ListTemplates", "Locations", "New", "NextNumber", "Paths",
		"Render", "Validate",
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
```

## Implementation plan

Seven commits. Each one passes `scripts/verify-target.sh --target` on its own and leaves the
goldens untouched. The prototype patch in [Appendix C](#appendix-c-prototype-patch) is the end
state. Each commit below is a slice of it.

1. **Pin the orders that no test pins.** Add `internal/cli/port_order_test.go` with
   `captureRun` tests for the scenarios in [Appendix A](#appendix-a-scenarios) that no test covers
   today. These include `status-before-resolve`, both `replace-resolve-*` cases,
   `target-dir-vs-duplicate`, `exists-vs-duplicate`, `already-in-archive`,
   `new-default-exists-and-bad-due`, `new-output-in-other-dir-drafts`, `email-upper-PDF`,
   `email-pdf-with-p`, `email-missing-pdf-and-bad-subject`,
   `email-dry-run-output-exists-bad-subject`, and a byte-and-mode check of the archived file
   for `rearchive-yes`. Add `TestPortMethodSets` with today's lists (39 and 22) and change the
   lists in each later commit, so every port change shows up in review. A golden file cannot be
   added, because `verify-target.sh` diffs `cmd/invox/testdata` against the base.
2. **`Renderer.Build`.** Move `billing.compile` into `latex.Renderer.Build`, and remove `Scratch`
   and `Copy`. Rewrite the shim's `BuildInvoicePDF` as one `Renderer.Build` call.
3. **`Invoices.Drafts` and `Destination`.** Change `Drafts` to `(workDir, output)` and replace
   `CheckOutput` with `Destination` in `New` and `EditArchived`.
4. **Email lookup.** Replace `EmailRequest.Input` with `Invoice` and `FromPDF` and set them in
   `emailRun`. Add `Archive.Source` (from `FindFile` and the sibling check) and
   `Mailer.CheckAttachment`. Remove `Invoices.Stat`, `Archive.FindFile` and
   `Service.EmailPaths`. The shim's `ResolveEmailDraftPaths` and `PrepareInvoiceEmail` take the
   defaulting the CLI does. `Invoices.Exists` stays until commit 5.
5. **Archive placement (the riskiest step).** Add `Head.WorkingCopy`, `Placement.HistoryDir`,
   `AddOptions`, `Place` and `Duplicate`. Move `Existing`, the replace refusal and the dry-run
   result into `Add`. Remove `HistoryDir`, `Resolve`, `Existing` and `Invoices.Exists`. `billing`
   still calls `Update` after a non-dry-run `Add` in this commit.
6. **Single write.** Add `store.Store.Rewrite`, `archive.Archive.Rewrite` and
   `AddOptions.Change`. Wire `Rewrite` in `factory` and the shim. Drop the `Update` call from
   `billing.archive`.
7. **Lock it in.** Add `TestBillingHandlesNoPaths`, set the final method lists, add the depguard
   deny, and add a unit B row to `docs/design/target-progress.md`.

**Riskiest step: commit 5.** It splits one 100-line sequence of interleaved domain checks and
file checks across four port calls (`Dir`, `Place`, `Duplicate`, `Add`) and the `Confirm` retry.
Most of the error orders it touches have no test today: "already in the archive directory",
"must stay within" for `archive_replace_path`, and a directory at the target path. That is why
commit 1 pins them first.

If commit 5 fights back, keep `Archive.Resolve` and `Archive.Existing` on the port and do only the
moves that need no reordering: `Join` and `Clean` and `Exists` into `Place`, and the exclusion set
into `Duplicate`. That fallback leaves two file-level methods on `Archive` and fails check 1, so
record it under Blockers in the progress log. Commit 6 can be skipped on its own. Without it,
`billing` keeps the `Update` after `Add`, and the gap unit A recorded stays open.

## Size and the split question

Measured with `wc -l` on non-test files, and as non-blank, non-comment lines:

| Package | 47aef7e | Prototype |
|---|---|---|
| `internal/billing` | 2,085 (1,534) | 1,955 (1,396) |
| `internal/store` | 2,480 (2,055) | 2,489 (2,064) |
| `internal/archive` | 516 (403) | 600 (478) |
| `internal/render/latex` | 691 (587) | 698 (594) |

**Don't split `billing`.** It ends at 1,955 lines, over the spec's 1,500, but it has no real
seam. The target test requires all 14 use cases to stay methods on `billing.Service`, so a package
per use case group would be a layer of methods that call through. `ports.go` (359 lines) and
`errors.go` (193 lines) are `billing`'s contract with the adapters and the CLI. They are not a
second concern. The only code that could leave without a pass-through layer is the pure text
helpers, `emailtext.go` and `epc.go` (198 lines together). The spec puts both in `billing` by
name. Moving them would leave 1,757 lines and add an import for no reader gain. Revisit the
question if a use case grows a second rule set of its own, such as numbering patterns per
customer.

## Moving the store tests off the shim

`internal/store` has 42 test files at `47aef7e`. They are 41 test files plus the shim
`legacy_api_test.go`, with 154 `Test` and `Fuzz` functions. The shim keeps the old `Host` API
alive in test code. It needs to go, because it hides which layer each test pins. In the prototype
it already had to copy the CLI's email defaulting to keep `TestResolveEmailDraftPaths*` passing.

Sort the files by what they test:

- **Format tests stay in `store`.** They need no shim: `config_paths`, `yaml_alias`,
  `scalar_decoding`, `templates`, `yaml`, `starter_template`, `legacy_copy_mode_unix`, `host`.
  `yaml_writeback` and `config_sources` call `(&Store{}).Update` or `Host` directly instead of
  the shim. Decoding tests that enter through `LoadContext` stay in `store`: `typed_models`,
  `scalar_parsing`, `yaml_merge`, `fuzz`, `fuzz_findings`. A 15-line helper `validate(t, files)`
  builds `billing.Service` from `Store` and `archive.Archive`, as the shim's `service` does
  today. An internal `store` test cannot import `factory`, because `factory` imports `store`.
- **Use-case tests move to `internal/billing` as `package billing_test`.** They are
  `drafts`, `archive_replace`, `archive_edit`, `archive_unknown_keys`, `number_uniqueness`,
  `numbering_skipped`, `email`, `email_output`, `paid_amount`, `one_run_problems`, `context` and
  `golden_inputs`. Each builds the real `Service` with
  `factorytest.New(t, nil, factorytest.Options{...}).Service(cmdutil.Files{...})` and calls
  `svc.New`, `svc.Archive` and the rest. A probe test in the prototype confirmed that
  `billing_test` can import `factorytest` without a cycle and run `Init` and `New`.
  `factorytest.Options` needs a `Now time.Time` field, because the shim passes a clock per call.
  The XDG and config-dir setup maps onto `Options.Vars` and `Options.ConfigDir`.
- **Adapter behavior moves to its adapter.** `email_archive_walk*` tests `Archive.Source` and
  goes to `internal/archive`. `build_pdf` tests `Renderer.Build` and goes to
  `internal/render/latex`. `render`, `render_epc`, `render_placeholders`, `latex_escape`,
  `unit_price` and `assets` test the template output and go to `internal/render/latex` as
  `package latex_test`. `render_epc_eligibility` tests `billing.EPCFor` and goes to
  `billing_test`. They get their `*invoice.Context` from `factorytest` and `Service.Validate`, or
  through `store` (no cycle: `store` does not import `latex`).

This fits the layering checks. Every depguard rule excludes `$test`. `archtest` lists packages
with `go list -deps` and no `-test`. The target test type-checks non-test sources only.
`ambient_test.go` skips `_test.go` files (`internal/env/ambient_test.go:49`). forbidigo and gosec
exclude `_test.go`.

No test name disappears. `verify-target.sh` greps `^func (Test|Fuzz)` across `cmd` and
`internal` regardless of package, so moving a function keeps it. Keep each name. Avoid clashes
with `billing`'s own `TestEntryNewer` and `FuzzEmail*`. Fixtures under `testdata/golden` may
move, because the fixture check matches by content. Migrate one group per commit, delete the shim
functions each group was the last caller of, and delete `legacy_api_test.go` with the last
group. Unit D owns this work. Doing it after unit B means the moved tests are written once,
against the narrowed ports.

## How this design was checked

- **Prototype.** `git archive 47aef7e` into a scratch directory, with the design applied by
  scripts: 14 files, 324 lines added and 308 removed. The checks that passed:
  `go build`, `go vet`, `gofmt -l`, `go mod tidy -diff`, `go test -race -count=1 ./...`,
  `go test -tags target ./internal/archtest -run TestTarget`, golangci-lint (0 issues), `go run
  ./internal/docs/gen` with `docs/cli` and `share/man` byte-identical, `cmd/invox/testdata`
  untouched, and `TestScript`. The harness refused to run `verify-target.sh` itself outside this
  worktree, because the script calls git. I ran each of its steps by hand instead. Local
  golangci-lint is v2.5.0. CI pins v2.14.0.
- **Differential run.** A script ran 50 scenarios against binaries built from `47aef7e` and from
  the prototype. It compared the exit code, stdout, stderr, and every file's path, mode and
  SHA-256. 49 matched exactly. `email-write` differed in the `.eml` bytes only, and two runs of
  the base binary differ the same way, because of the `Date` header and the MIME boundary. The
  script is [Appendix B](#appendix-b-differential-script). Rerun it after each commit.
- **Acceptance test.** It passes on the prototype and fails at `47aef7e`, listing 4 ports, the
  `Service` set, 2 imports and the extension literals.

## Open questions

1. `billing.archive` keeps `strings.TrimSpace(dir) == ""` to word "archive directory is
   unavailable" before the status checks. The rule above allows it as a presence check. Moving
   it into the adapter would need one more `Archive` method, because the status checks sit
   between `Dir` and `Place`.
2. `Mailer.CheckAttachment` is the same four lines in `email` and `applemail`. They are two
   driven packages that may not import each other. If a third mailer appears, a shared helper in
   `fsutil` would fit.
3. `Directory` keeps 14 methods. `Locate` and `EditablePath` overlap for support files, and the
   legacy trio plus `Locations` are CLI support. None of them handles a path inside `billing`, so
   they are outside this unit. They are a candidate for a later cleanup.
4. Commits 4 to 6 keep the shim passing by moving the CLI's email defaulting into
   `ResolveEmailDraftPaths` in test code. Unit D removes that copy. If unit D slips, the copy
   stays as a known smell.

## Appendix A: scenarios

First stderr line from the `47aef7e` binary. The prototype matched every row's exit code, stdout
and stderr, and every row's files except the `.eml` timestamp in `email-write`. Setup uses the
`archive.txtar` fixtures, with `HOME` and `XDG_CONFIG_HOME` inside the scenario directory.
`$W` is that directory. A row whose message is not the one its name suggests shows that an
earlier check wins, and that order is what it pins.

| Scenario | Command | Exit | First stderr line |
|---|---|---|---|
| `dir-error-before-status` | `invox archive add invoice.yaml` | 1 | error: home/.config/invox/config.yaml: yaml: line 1: did not find expected node content |
| `unavailable-dir-vs-status` | `invox archive add invoice.yaml` | 1 | error: invoice.yaml: invoice.status must be 'built' before archiving, got 'draft' |
| `status-before-resolve` | `invox archive add invoice.yaml` | 1 | error: invoice.yaml: invoice.status must be 'editing' or 'built' before re-archiving, got 'draft' |
| `resolve-archive-path` | `invox archive add invoice.yaml` | 1 | error: ../escape.yaml must stay within home/.config/invox/archive |
| `replace-resolve-no-number` | `invox archive add invoice.yaml` | 1 | error: ../escape.md must stay within home/.config/invox/archive |
| `replace-resolve-with-number` | `invox archive add invoice.yaml` | 1 | error: ../escape.md must stay within home/.config/invox/archive |
| `target-dir-vs-duplicate` | `invox archive add invoice.yaml --yes` | 1 | error: invoice.yaml: invoice number CUST-001-001 is already used by archived invoice home/.config/invox/archive/other.yaml |
| `exists-vs-duplicate` | `invox archive add invoice.yaml` | 1 | error: home/.config/invox/archive/invoice.yaml already exists |
| `duplicate` | `invox archive add invoice.yaml` | 1 | error: invoice.yaml: invoice number CUST-001-001 is already used by archived invoice home/.config/invox/archive/other.yaml |
| `already-in-archive` | `invox archive add invoice.yaml --yes` | 1 | error: invoice.yaml is already in the archive directory |
| `rearchive-needs-yes` | `invox archive add invoice.yaml` | 2 | error: archiving invoice.yaml replaces archived invoice home/.config/invox/archive/invoice.yaml; pass --yes to replace it (stdin is not a terminal) |
| `rearchive-dry-run` | `invox archive add invoice.yaml -n` | 0 | Would replace archived invoice home/.config/invox/archive/invoice.yaml; the previous version would be kept in home/.config/invox/archive/.history |
| `rearchive-yes` | `invox archive add invoice.yaml --yes` | 0 | Replaced archived invoice home/.config/invox/archive/invoice.yaml; previous version kept at home/.config/invox/archive/.history/invoice.20261009T081135Z.yaml |
| `archive-new-ok` | `invox archive add invoice.yaml` | 0 | Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml |
| `build-archive-dry-run` | `invox build invoice.yaml --archive --dry-run` | 0 | Would build invoice.pdf for CUST-001 (CUST-001-001) |
| `build-no-tectonic` | `invox build invoice.yaml` | 1 | error: tectonic not found in PATH |
| `validate-duplicate` | `invox validate invoice.yaml` | 0 | warning: invoice number CUST-001-001 is already used by archived invoice home/.config/invox/archive/other.yaml; run 'invox increment -i invoice.yaml' before archiving |
| `validate-working-copy-own-number` | `invox validate invoice.yaml` | 0 | Validation OK: CUST-001-001 for CUST-001, 1 line item(s), total 120,00 € |
| `validate-bad-link` | `invox validate invoice.yaml` | 0 | warning: could not check the archive for duplicate invoice numbers: archive filename must be relative to archive.dir, got /abs.yaml |
| `edit-force-into-archive` | `invox archive edit invoice.yaml --force` | 1 | error: invoice.yaml is in the archive directory and is never overwritten; archived invoices change only by re-archiving an edited copy |
| `edit-exists` | `invox archive edit invoice.yaml` | 1 | error: invoice.yaml already exists; pass --force to replace it or choose a different working directory |
| `edit-ok-dry-run` | `invox archive edit other.yaml -n` | 0 | Would copy home/.config/invox/archive/other.yaml -> other.yaml |
| `email-missing-pdf-and-recipient` | `invox email invoice.yaml -o d.eml` | 1 | error: customer.email: missing value |
| `email-missing-pdf` | `invox email invoice.yaml -o d.eml` | 1 | error: read invoice.pdf: stat invoice.pdf: no such file or directory |
| `email-orphan-pdf` | `invox email orphan.pdf -o d.eml -n` | 1 | error: orphan.pdf: no matching invoice YAML found next to the PDF or in archive.dir |
| `email-pdf-next-to-draft` | `invox email invoice.pdf -o d.eml -n` | 1 | error: invoice.yaml: invoice.status must be 'built' or 'archived' before creating an email draft, got 'draft' |
| `email-pdf-from-archive` | `invox email out/inv.pdf -o d.eml -n` | 0 | Would open email draft for CUST-001 (CUST-001-001) to office@appsters.example |
| `email-pdf-ambiguous-archive` | `invox email out/inv.pdf -o d.eml -n` | 1 | error: home/.config/invox/archive: multiple archived invoice YAML files match inv.yaml; pass the YAML path explicitly: home/.config/invox/archive/a/inv.yaml, home/.config/invox/archive/b/inv.yaml |
| `email-upper-PDF` | `invox email INV.PDF -o d.eml -n` | 1 | error: read INV.pdf: stat INV.pdf: no such file or directory |
| `email-pdf-with-p` | `invox email invoice.pdf -p other.pdf -o d.eml -n` | 0 | Would open email draft for CUST-001 (CUST-001-001) to office@appsters.example |
| `email-dry-run-output-exists-no-recipient` | `invox email invoice.yaml -o d.eml -n` | 1 | error: customer.email: missing value |
| `email-dry-run-output-exists` | `invox email invoice.yaml -o d.eml -n` | 1 | error: d.eml already exists; pass --force or choose another -o path |
| `email-write` | `invox email invoice.yaml -o d.eml` | 1 | error: created d.eml but failed to open it: exec: "xdg-open": executable file not found in $PATH |
| `new-default-exists-and-bad-due` | `invox new CUST-001` | 1 | error: CUST-001-010.yaml already exists; pass --force to replace it or choose a different -o/--output path |
| `new-output-exists-and-bad-due` | `invox new CUST-001 -o o.yaml` | 1 | error: o.yaml already exists; pass --force to replace it or choose a different -o/--output path |
| `new-output-dir` | `invox new CUST-001 -o o.yaml` | 1 | error: o.yaml is a directory; choose a different -o/--output path |
| `new-default-dir` | `invox new CUST-001` | 1 | error: CUST-001-010.yaml is a directory; choose a different -o/--output path |
| `new-ok` | `invox new CUST-001` | 0 | Created CUST-001-010.yaml for CUST-001 (CUST-001-010) |
| `new-output-in-other-dir-drafts` | `invox new CUST-001 -o sub/n.yaml` | 0 | Created sub/n.yaml for CUST-001 (CUST-001-010) |
| `new-force-into-archive` | `invox new CUST-001 -o home/.config/invox/archive/x.yaml --force` | 1 | error: home/.config/invox/archive/x.yaml is in the archive directory and is never overwritten; archived invoices change only by re-archiving an edited copy |
| `new-from-last` | `invox new CUST-001 --from-last -n` | 0 | Would create CUST-001-010.yaml for CUST-001 (CUST-001-010) |
| `email-missing-pdf-and-bad-subject` | `invox email invoice.yaml -o d.eml --subject {nope}` | 1 | error: read invoice.pdf: stat invoice.pdf: no such file or directory |
| `email-dry-run-output-exists-bad-subject` | `invox email invoice.yaml -o d.eml -n --subject {nope}` | 1 | error: d.eml already exists; pass --force or choose another -o path |
| `email-temp-dry-run` | `invox email invoice.yaml -n` | 0 | Would open email draft for CUST-001 (CUST-001-001) to office@appsters.example |
| `archive-comments-and-empty-link` | `invox archive add invoice.yaml` | 0 | Archived invoice.yaml -> home/.config/invox/archive/invoice.yaml |
| `rearchive-comments-mode` | `invox archive add invoice.yaml --yes` | 0 | Replaced archived invoice home/.config/invox/archive/invoice.yaml; previous version kept at home/.config/invox/archive/.history/invoice.20261009T081135Z.yaml |
| `archive-list-root` | `invox archive add invoice.yaml` | 1 | error: invoice.yaml: root value must be a mapping |
| `archive-scalar-root` | `invox archive add invoice.yaml` | 1 | error: invoice.yaml: root value must be a mapping |
| `archive-aliased-header` | `invox archive add invoice.yaml` | 1 | error: invoice.yaml: 'invoice' must be a mapping |
| `render-ok` | `invox render invoice.yaml` | 0 | Rendered invoice.tex for CUST-001 (CUST-001-001) |

## Appendix B: differential script

`python3 difftest.py cmd/invox/testdata/script/archive.txtar <base-binary> <new-binary>`, with
both binaries built by `go build -o <file> ./cmd/invox`.

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


def run(binary, scenario):
    name, extra, cwd, args = scenario
    work = tempfile.mkdtemp(prefix='invox-diff-')
    try:
        for path, content in list(FIX.items()) + list(extra.items()):
            full = os.path.join(work, path)
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
    a, b = run(base_bin, scenario), run(new_bin, scenario)
    first = a.splitlines()[0] + ' | ' + next((l for l in a.splitlines()[3:] if l and not l.startswith('files')), '')
    if a == b:
        print('same  %-42s %s' % (scenario[0], first[:110]))
    else:
        diffs += 1
        print('DIFF  %s\n--- base\n%s--- new\n%s' % (scenario[0], a, b))
print('%d scenarios, %d differ' % (len(SCENARIOS), diffs))
```

## Appendix C: prototype patch

The end state of commits 2 to 6 at `47aef7e` (`diff -ruN` of `internal` and `cmd`). It omits the
acceptance test, which is shown above, and the commit 1 tests.

<details>
<summary>Patch (14 files)</summary>

```diff
diff -ruN -x ports_test.go base2/internal/adapters/applemail/applemail.go proto2/internal/adapters/applemail/applemail.go
--- base2/internal/adapters/applemail/applemail.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/adapters/applemail/applemail.go	2026-10-09 08:02:13.490201608 +0000
@@ -4,6 +4,7 @@
 import (
 	"context"
 	"fmt"
+	"os"
 
 	"github.com/0xboris/invox/internal/adapters/run"
 	"github.com/0xboris/invox/internal/billing"
@@ -88,5 +89,11 @@
 	return billing.Draft{}, nil
 }
 
+// CheckAttachment returns why the file at path cannot be attached.
+func (c *Composer) CheckAttachment(path string) error {
+	_, err := os.Stat(path)
+	return err
+}
+
 // Check has nothing to check: Apple Mail writes no file.
 func (c *Composer) Check(billing.Message) error { return nil }
diff -ruN -x ports_test.go base2/internal/archive/adapter.go proto2/internal/archive/adapter.go
--- base2/internal/archive/adapter.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/archive/adapter.go	2026-10-09 08:02:29.218195131 +0000
@@ -19,6 +19,9 @@
 	Locate func() (string, error)
 	// Read reads what an archived invoice says about itself.
 	Read Reader
+	// Rewrite returns the invoice at path with change applied, keeping its
+	// comments and layout.
+	Rewrite func(path string, change func(*invoice.Invoice) error) ([]byte, error)
 }
 
 var _ billing.Archive = Archive{}
@@ -45,28 +48,73 @@
 	return a.Locate()
 }
 
-// HistoryDir returns where backups are kept.
-func (a Archive) HistoryDir() (string, error) {
+// Place says where archiving the invoice at src writes it.
+func (a Archive) Place(src string, head billing.Head) (billing.Placement, error) {
 	s, err := a.store()
 	if err != nil {
-		return "", err
+		return billing.Placement{}, err
+	}
+	p := billing.Placement{Path: filepath.Join(s.Dir, filepath.Base(src)), HistoryDir: s.HistoryDir()}
+	if head.WorkingCopy() {
+		target, err := s.Resolve(head.ArchivePath)
+		if err != nil {
+			return billing.Placement{}, err
+		}
+		p.Path, p.Overwrite = target.Path, true
+	} else if isFile(p.Path) {
+		return billing.Placement{}, fmt.Errorf("%s already exists", p.Path)
+	}
+	if filepath.Clean(src) == p.Path {
+		return billing.Placement{}, fmt.Errorf("%s is already in the archive directory", src)
+	}
+	if head.WorkingCopy() && head.ReplacePath != "" && head.ReplacePath != head.ArchivePath {
+		target, err := s.Resolve(head.ReplacePath)
+		if err != nil {
+			return billing.Placement{}, err
+		}
+		if target.Path != p.Path {
+			p.Remove = target.Path
+		}
 	}
-	return s.HistoryDir(), nil
+	return p, nil
 }
 
-// Resolve turns a name relative to the archive directory into a path.
-func (a Archive) Resolve(name string) (string, error) {
+// Duplicate returns the archived invoice, in file name order, that has
+// head's number, other than src and, for a working copy, the archived
+// files it replaces.
+func (a Archive) Duplicate(src string, head billing.Head) (string, error) {
 	s, err := a.store()
 	if err != nil {
 		return "", err
 	}
-	target, err := s.Resolve(name)
-	return target.Path, err
-}
-
-// Existing returns those of paths that exist.
-func (a Archive) Existing(paths ...string) ([]string, error) {
-	return ExistingFiles(paths...)
+	if strings.TrimSpace(s.Dir) == "" || head.Number == "" {
+		return "", nil
+	}
+	excluded := map[string]bool{filepath.Clean(src): true}
+	// Only a working copy from `archive edit` may reuse the number of the
+	// archived file it replaces.
+	if head.WorkingCopy() {
+		for _, name := range []string{head.ArchivePath, head.ReplacePath} {
+			if name == "" {
+				continue
+			}
+			target, err := s.Resolve(name)
+			if err != nil {
+				return "", err
+			}
+			excluded[target.Path] = true
+		}
+	}
+	entries, err := s.List(a.Read)
+	if err != nil {
+		return "", err
+	}
+	for _, entry := range entries {
+		if entry.Number == head.Number && !excluded[filepath.Clean(entry.Path)] {
+			return filepath.Clean(entry.Path), nil
+		}
+	}
+	return "", nil
 }
 
 // Checkout resolves ref and says where its working copy in workDir goes.
@@ -90,21 +138,38 @@
 	}, nil
 }
 
-// Add writes the invoice at src to p.Path, after backing up p.Replaced,
-// then removes p.Remove and src.
-func (a Archive) Add(src string, p billing.Placement) (billing.ArchiveResult, error) {
-	s, err := a.store()
+// Add writes the invoice at src to p.Path, after backing up the archived
+// files it replaces, then removes p.Remove and src.
+func (a Archive) Add(src string, p billing.Placement, opts billing.AddOptions) (billing.ArchiveResult, error) {
+	var replaced []string
+	if p.Overwrite {
+		var err error
+		if replaced, err = ExistingFiles(p.Path, p.Remove); err != nil {
+			return billing.ArchiveResult{}, err
+		}
+	}
+	if len(replaced) > 0 && !opts.Replace {
+		return billing.ArchiveResult{}, &billing.ArchiveReplaceError{InvoicePath: src, Paths: replaced, HistoryDir: p.HistoryDir}
+	}
+	if opts.DryRun {
+		result := billing.ArchiveResult{Path: p.Path, HistoryDir: p.HistoryDir}
+		for _, path := range replaced {
+			result.Replaced = append(result.Replaced, billing.Backup{Path: path})
+		}
+		return result, nil
+	}
+	data, err := a.Rewrite(src, opts.Change)
 	if err != nil {
 		return billing.ArchiveResult{}, err
 	}
-	if err := fsutil.MkdirAll(s.Dir, fsutil.Private); err != nil {
+	s, err := a.store()
+	if err != nil {
 		return billing.ArchiveResult{}, err
 	}
-	backups, err := s.Backup(p.Replaced, p.Now)
-	if err != nil {
+	if err := fsutil.MkdirAll(s.Dir, fsutil.Private); err != nil {
 		return billing.ArchiveResult{}, err
 	}
-	data, err := os.ReadFile(src)
+	backups, err := s.Backup(replaced, opts.Now)
 	if err != nil {
 		return billing.ArchiveResult{}, err
 	}
@@ -124,7 +189,7 @@
 	if err := os.Remove(filepath.Clean(src)); err != nil {
 		return billing.ArchiveResult{}, fmt.Errorf("remove %s: %w", filepath.Clean(src), err)
 	}
-	return billing.ArchiveResult{Path: p.Path, Replaced: backups, HistoryDir: s.HistoryDir()}, nil
+	return billing.ArchiveResult{Path: p.Path, Replaced: backups, HistoryDir: p.HistoryDir}, nil
 }
 
 // Protects reports whether path is an existing file inside the archive
@@ -167,10 +232,29 @@
 	}
 }
 
-// FindFile returns the archived file named one of names: one directly in
+// Source returns the invoice YAML file the PDF at pdf was built from: next
+// to it, else in the archive.
+func (a Archive) Source(pdf string) (string, error) {
+	base := strings.TrimSuffix(pdf, filepath.Ext(pdf))
+	candidates := []string{base + ".yaml", base + ".yml"}
+	for _, candidate := range candidates {
+		if isFile(candidate) {
+			return candidate, nil
+		}
+	}
+	return a.findFile(filepath.Base(candidates[0]), filepath.Base(candidates[1]))
+}
+
+// isFile reports whether path is a file.
+func isFile(path string) bool {
+	info, err := os.Stat(path)
+	return err == nil && !info.IsDir()
+}
+
+// findFile returns the archived file named one of names: one directly in
 // the archive directory, else the only one below it. It returns "" when
 // there is none or no archive directory, and an error when several match.
-func (a Archive) FindFile(names ...string) (string, error) {
+func (a Archive) findFile(names ...string) (string, error) {
 	s, err := a.store()
 	if err != nil {
 		return "", err
diff -ruN -x ports_test.go base2/internal/billing/archive.go proto2/internal/billing/archive.go
--- base2/internal/billing/archive.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/billing/archive.go	2026-10-09 08:02:13.472492709 +0000
@@ -3,7 +3,6 @@
 import (
 	"errors"
 	"fmt"
-	"path/filepath"
 	"sort"
 	"strings"
 	"time"
@@ -63,89 +62,43 @@
 	if strings.TrimSpace(dir) == "" {
 		return ArchiveResult{}, errors.New("archive directory is unavailable")
 	}
-	historyDir, err := s.Archives.HistoryDir()
-	if err != nil {
+	if err := archivable(path, head, status); err != nil {
 		return ArchiveResult{}, err
 	}
-
-	archivePath := filepath.Join(dir, filepath.Base(path))
-	sourcePath := filepath.Clean(path)
-
-	editing := head.ArchivePath != ""
-	if editing {
-		if !status.Allows(invoice.Rearchiving) {
-			return ArchiveResult{}, fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", path, status)
-		}
-		archivePath, err = s.Archives.Resolve(head.ArchivePath)
-		if err != nil {
-			return ArchiveResult{}, err
-		}
-	} else {
-		switch {
-		case status == "":
-			return ArchiveResult{}, fmt.Errorf("%s: invoice.status: missing value", path)
-		case !status.Allows(invoice.Archiving):
-			return ArchiveResult{}, fmt.Errorf("%s: invoice.status must be `built` before archiving, got `%s`", path, status)
-		}
-		if s.Invoices.Exists(archivePath) {
-			return ArchiveResult{}, fmt.Errorf("%s already exists", archivePath)
-		}
-	}
-	if sourcePath == archivePath {
-		return ArchiveResult{}, fmt.Errorf("%s is already in the archive directory", path)
+	place, err := s.Archives.Place(path, head)
+	if err != nil {
+		return ArchiveResult{}, err
 	}
-
 	if err := s.numberUnique(path, head); err != nil {
 		return ArchiveResult{}, err
 	}
-
-	var replacePath string
-	if editing && head.ReplacePath != "" && head.ReplacePath != head.ArchivePath {
-		replacePath, err = s.Archives.Resolve(head.ReplacePath)
-		if err != nil {
-			return ArchiveResult{}, err
-		}
-		if replacePath == archivePath {
-			replacePath = ""
-		}
-	}
-	var replaced []string
-	if editing {
-		replaced, err = s.Archives.Existing(archivePath, replacePath)
-		if err != nil {
-			return ArchiveResult{}, err
-		}
-	}
-	if len(replaced) > 0 && !opts.Replace {
-		return ArchiveResult{}, &ArchiveReplaceError{InvoicePath: path, Paths: replaced, HistoryDir: historyDir}
-	}
-	if opts.DryRun {
-		result := ArchiveResult{Path: archivePath, HistoryDir: historyDir}
-		for _, replacedPath := range replaced {
-			result.Replaced = append(result.Replaced, Backup{Path: replacedPath})
-		}
-		return result, nil
-	}
-
-	result, err := s.Archives.Add(path, Placement{
-		Path:      archivePath,
-		Overwrite: editing,
-		Replaced:  replaced,
-		Remove:    replacePath,
-		Now:       s.Now(),
+	return s.Archives.Add(path, place, AddOptions{
+		Replace: opts.Replace,
+		DryRun:  opts.DryRun,
+		Now:     s.Now(),
+		Change: func(inv *invoice.Invoice) error {
+			inv.Header.Status = invoice.Text(invoice.Archived)
+			inv.Archive = nil
+			return nil
+		},
 	})
-	if err != nil {
-		return ArchiveResult{}, err
-	}
-	err = s.Invoices.Update(archivePath, func(inv *invoice.Invoice) error {
-		inv.Header.Status = invoice.Text(invoice.Archived)
-		inv.Archive = nil
+}
+
+// archivable returns why the invoice at path, whose head is head, cannot be
+// archived with status: a working copy is re-archived when editing or
+// built, any other invoice archived when built.
+func archivable(path string, head Head, status invoice.Status) error {
+	switch {
+	case head.WorkingCopy() && !status.Allows(invoice.Rearchiving):
+		return fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", path, status)
+	case head.WorkingCopy():
 		return nil
-	})
-	if err != nil {
-		return ArchiveResult{}, err
+	case status == "":
+		return fmt.Errorf("%s: invoice.status: missing value", path)
+	case !status.Allows(invoice.Archiving):
+		return fmt.Errorf("%s: invoice.status must be `built` before archiving, got `%s`", path, status)
 	}
-	return result, nil
+	return nil
 }
 
 // headWithInvoice reads the head of the invoice at path, which must have an
@@ -185,52 +138,15 @@
 	if !head.HasHeader {
 		return nil
 	}
-	dir, err := s.Archives.Dir()
-	if err != nil {
-		return err
-	}
-	if strings.TrimSpace(dir) == "" {
-		return nil
-	}
 	return s.numberUnique(path, head)
 }
 
 func (s *Service) numberUnique(path string, head Head) error {
-	if head.Number == "" {
-		return nil
-	}
-
-	excluded := map[string]bool{filepath.Clean(path): true}
-	// Only a working copy from `archive edit` (archive_path set) may reuse
-	// the number of the archived file it replaces.
-	if head.ArchivePath != "" {
-		for _, name := range []string{head.ArchivePath, head.ReplacePath} {
-			if name == "" {
-				continue
-			}
-			resolved, err := s.Archives.Resolve(name)
-			if err != nil {
-				return err
-			}
-			excluded[resolved] = true
-		}
-	}
-
-	entries, err := s.archiveEntries()
-	if err != nil {
+	archived, err := s.Archives.Duplicate(path, head)
+	if err != nil || archived == "" {
 		return err
 	}
-	for _, entry := range entries {
-		if entry.Number != head.Number {
-			continue
-		}
-		entryPath := filepath.Clean(entry.Path)
-		if excluded[entryPath] {
-			continue
-		}
-		return &invoice.DuplicateInvoiceNumberError{InvoicePath: path, InvoiceNumber: head.Number, ArchivedPath: entryPath}
-	}
-	return nil
+	return &invoice.DuplicateInvoiceNumberError{InvoicePath: path, InvoiceNumber: head.Number, ArchivedPath: archived}
 }
 
 // archiveEntries returns the archived invoices sorted by Filename.
@@ -343,7 +259,7 @@
 	if err := requireHeader(checkout.Archived, archived); err != nil {
 		return Edited{}, err
 	}
-	if err := s.Invoices.CheckOutput(checkout.Path, opts.Overwrite); err != nil {
+	if _, err := s.Invoices.Destination(checkout.Path, workDir, "", opts.Overwrite); err != nil {
 		return Edited{}, err
 	}
 	if err := s.refuseArchivedOverwrite(checkout.Path, opts.Overwrite); err != nil {
diff -ruN -x ports_test.go base2/internal/billing/email.go proto2/internal/billing/email.go
--- base2/internal/billing/email.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/billing/email.go	2026-10-09 08:02:13.472720236 +0000
@@ -3,7 +3,6 @@
 import (
 	"context"
 	"fmt"
-	"path/filepath"
 	"strings"
 
 	"github.com/0xboris/invox/internal/invoice"
@@ -12,9 +11,11 @@
 
 // EmailRequest says which invoice DraftEmail drafts an email for.
 type EmailRequest struct {
-	// Input is the invoice, or its built PDF, whose invoice is found next
-	// to it or in the archive.
-	Input string
+	// Invoice is the invoice file. It is "" when the user named the built
+	// PDF instead: FromPDF is then that PDF, and the invoice is the YAML
+	// file with its name next to it, else in the archive.
+	Invoice string
+	FromPDF string
 	// PDF is the attachment, Output the .eml draft.
 	PDF    string
 	Output string
@@ -46,10 +47,11 @@
 	if err != nil {
 		return EmailResult{}, err
 	}
-	invoicePath, pdfPath, outputPath, err := s.EmailPaths(req.Input, req.PDF, req.Output)
+	invoicePath, err := s.emailInvoice(req)
 	if err != nil {
 		return EmailResult{}, err
 	}
+	pdfPath, outputPath := req.PDF, req.Output
 	inv, err := s.loadContext(customersPath, issuerPath, invoicePath)
 	if err != nil {
 		return EmailResult{}, err
@@ -61,7 +63,7 @@
 		}
 		return EmailResult{}, fmt.Errorf("%s: invoice.status must be `built` or `archived` before creating an email draft, got `%s`", invoicePath, status)
 	}
-	if err := s.Invoices.Stat(pdfPath); err != nil {
+	if err := s.Mailer.CheckAttachment(pdfPath); err != nil {
 		return EmailResult{}, fmt.Errorf("read %s: %w", pdfPath, err)
 	}
 
@@ -117,58 +119,20 @@
 	return result, nil
 }
 
-// EmailPaths returns the invoice, PDF and draft of an email for input, an
-// invoice YAML file or its PDF.
-func (s *Service) EmailPaths(input, pdf, output string) (string, string, string, error) {
-	input = strings.TrimSpace(input)
-	if input == "" {
-		return "", "", "", fmt.Errorf("input path is required")
-	}
-	pdf, output = strings.TrimSpace(pdf), strings.TrimSpace(output)
-
-	var invoicePath string
-	switch strings.ToLower(filepath.Ext(input)) {
-	case ".pdf":
-		resolved, err := s.invoiceForPDF(input)
-		if err != nil {
-			return "", "", "", err
-		}
-		invoicePath = resolved
-		if pdf == "" {
-			pdf = input
-		}
-	case ".yaml", ".yml":
-		invoicePath = input
-		if pdf == "" {
-			pdf = replaceExt(input, ".pdf")
-		}
-	default:
-		return "", "", "", fmt.Errorf("%s: input must end with .yaml, .yml, or .pdf", input)
-	}
-	if output == "" {
-		output = replaceExt(input, ".eml")
-	}
-	return invoicePath, pdf, output, nil
-}
-
-// invoiceForPDF finds the invoice of the PDF at pdfPath: next to it, else in
-// the archive.
-func (s *Service) invoiceForPDF(pdfPath string) (string, error) {
-	base := strings.TrimSuffix(pdfPath, filepath.Ext(pdfPath))
-	candidates := []string{base + ".yaml", base + ".yml"}
-	for _, candidate := range candidates {
-		if s.Invoices.Exists(candidate) {
-			return candidate, nil
-		}
+// emailInvoice returns the invoice of req: the one it names, else the one
+// its PDF was built from.
+func (s *Service) emailInvoice(req EmailRequest) (string, error) {
+	if req.Invoice != "" {
+		return req.Invoice, nil
 	}
-	archived, err := s.Archives.FindFile(filepath.Base(candidates[0]), filepath.Base(candidates[1]))
+	invoicePath, err := s.Archives.Source(req.FromPDF)
 	if err != nil {
 		return "", err
 	}
-	if archived != "" {
-		return archived, nil
+	if invoicePath == "" {
+		return "", fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", req.FromPDF)
 	}
-	return "", fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", pdfPath)
+	return invoicePath, nil
 }
 
 // emailFields is what the email placeholders stand for in ctx.
@@ -191,8 +155,3 @@
 func emailMoney(cents int64, currency string) string {
 	return money.FormatCents(cents) + " " + currency
 }
-
-// replaceExt returns path with its extension replaced by ext.
-func replaceExt(path, ext string) string {
-	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
-}
diff -ruN -x ports_test.go base2/internal/billing/new.go proto2/internal/billing/new.go
--- base2/internal/billing/new.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/billing/new.go	2026-10-09 08:02:13.471071817 +0000
@@ -3,7 +3,6 @@
 import (
 	"errors"
 	"fmt"
-	"path/filepath"
 	"time"
 
 	"github.com/0xboris/invox/internal/invoice"
@@ -53,7 +52,7 @@
 	}
 
 	if req.Output != "" {
-		if err := s.Invoices.CheckOutput(req.Output, req.Overwrite); err != nil {
+		if _, err := s.Invoices.Destination(req.Output, req.WorkDir, "", req.Overwrite); err != nil {
 			return NewResult{}, err
 		}
 	}
@@ -75,11 +74,7 @@
 
 	now := s.Now().In(time.Local)
 	issueDate := now.Format("2006-01-02")
-	draftDirs := []string{req.WorkDir}
-	if req.Output != "" {
-		draftDirs = append(draftDirs, filepath.Dir(req.Output))
-	}
-	draftCounter, err := s.highestDraftCounter(draftDirs, req.CustomerID, issueDate, customer)
+	draftCounter, err := s.highestDraftCounter(req.WorkDir, req.Output, req.CustomerID, issueDate, customer)
 	if err != nil {
 		return NewResult{}, err
 	}
@@ -87,11 +82,8 @@
 	if err != nil {
 		return NewResult{}, err
 	}
-	output := req.Output
-	if output == "" {
-		output = filepath.Join(req.WorkDir, number+".yaml")
-	}
-	if err := s.Invoices.CheckOutput(output, req.Overwrite); err != nil {
+	output, err := s.Invoices.Destination(req.Output, req.WorkDir, number, req.Overwrite)
+	if err != nil {
 		return NewResult{}, err
 	}
 	dueDays, err := issuerDueDays(issuerPath, *issuer.Payment)
diff -ruN -x ports_test.go base2/internal/billing/numbering.go proto2/internal/billing/numbering.go
--- base2/internal/billing/numbering.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/billing/numbering.go	2026-10-09 08:02:13.472003763 +0000
@@ -79,16 +79,16 @@
 }
 
 // highestDraftCounter returns the highest counter used by unarchived
-// invoices (status draft or built) directly inside dirs, so that two drafts
+// invoices (status draft or built) where a new invoice goes, so that two drafts
 // created before either is archived do not get the same number. Files that
 // cannot be read or do not match the numbering pattern are ignored.
-func (s *Service) highestDraftCounter(dirs []string, customerID, issueDate string, customer invoice.Customer) (int64, error) {
+func (s *Service) highestDraftCounter(workDir, output, customerID, issueDate string, customer invoice.Customer) (int64, error) {
 	settings, err := s.numberingSettings()
 	if err != nil {
 		return 0, err
 	}
 	var highest int64
-	for _, head := range s.Invoices.Drafts(dirs) {
+	for _, head := range s.Invoices.Drafts(workDir, output) {
 		if !head.Status.Allows(invoice.Numbering) || head.Number == "" {
 			continue
 		}
diff -ruN -x ports_test.go base2/internal/billing/ports.go proto2/internal/billing/ports.go
--- base2/internal/billing/ports.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/billing/ports.go	2026-10-09 08:02:13.431100029 +0000
@@ -71,6 +71,10 @@
 	Check  Check
 }
 
+// WorkingCopy reports whether h is a working copy from `archive edit`,
+// which re-archiving writes back over the archived files it names.
+func (h Head) WorkingCopy() bool { return h.ArchivePath != "" }
+
 // Invoices reads and writes invoice files. Update keeps the file's comments
 // and layout; TestInvoiceWritesKeepComments pins that.
 type Invoices interface {
@@ -85,9 +89,15 @@
 	// path. Values that do not decode are left unset and reported as
 	// *DecodeError values.
 	Head(path string) (Head, error)
-	// Drafts returns the invoices directly in dirs, best effort: files
-	// that cannot be read are left out.
-	Drafts(dirs []string) []Head
+	// Drafts returns the invoices directly in workDir and, when output is
+	// set, in the directory output is in, best effort: files that cannot
+	// be read are left out.
+	Drafts(workDir, output string) []Head
+	// Destination returns the file a new invoice is written to: path, or
+	// when path is "", <number>.yaml in workDir. It returns an
+	// *OutputIsDirError when that is a directory and, unless overwrite is
+	// set, an *OutputExistsError when it exists.
+	Destination(path, workDir, number string, overwrite bool) (string, error)
 	// Create writes a new invoice to path from the document at from, which
 	// may be an archived invoice, keeping its comments and keys. Every
 	// customer and header field set in inv is written, as text. Positions
@@ -97,13 +107,6 @@
 	// Update rewrites the invoice at path with change applied, writing
 	// back only the fields that changed.
 	Update(path string, change func(*invoice.Invoice) error) error
-	// CheckOutput returns an *OutputIsDirError when path is a directory,
-	// and, unless overwrite is set, an *OutputExistsError when it exists.
-	CheckOutput(path string, overwrite bool) error
-	// Exists reports whether path is a file.
-	Exists(path string) bool
-	// Stat returns why path cannot be read, or nil.
-	Stat(path string) error
 }
 
 // CustomerTable is customers.yaml. An entry is decoded only when it is
@@ -239,18 +242,31 @@
 	HistoryDir string
 }
 
-// Placement says where Archive.Add puts an invoice.
+// Placement is where Archive.Add puts an invoice. Archive.Place makes it.
 type Placement struct {
 	// Path is the archived file to write.
 	Path string
 	// Overwrite allows Path to exist, when a working copy is re-archived.
 	Overwrite bool
-	// Replaced are the archived files to back up first.
-	Replaced []string
 	// Remove is an archived file the invoice supersedes, removed after it
 	// is written, or "".
 	Remove string
+	// HistoryDir is where replaced files' previous versions are kept.
+	HistoryDir string
+}
+
+// AddOptions control Archive.Add.
+type AddOptions struct {
+	// Replace allows writing over archived files. Without it, Add returns
+	// an *ArchiveReplaceError when it would replace any.
+	Replace bool
+	// DryRun runs every check and returns the result without writing. The
+	// result's backups have no BackupPath.
+	DryRun bool
 	Now    time.Time
+	// Change is applied to the invoice as it is archived, keeping its
+	// comments and layout, so the archived file is written once.
+	Change func(*invoice.Invoice) error
 }
 
 // Checkout is where the working copy of an archived invoice goes.
@@ -268,25 +284,33 @@
 	// Entries reads every archived invoice, in the lexical order of a
 	// directory walk.
 	Entries() ([]ArchiveEntry, error)
-	// Add moves the invoice at src into the archive, as p says.
-	Add(src string, p Placement) (ArchiveResult, error)
+	// Dir returns the archive directory, "" when there is none.
+	Dir() (string, error)
+	// Place says where archiving the invoice at src, whose head is head,
+	// writes it: over the archived file a working copy names, else under
+	// src's name in the archive directory, which must not exist yet. It
+	// refuses src when it is that file already.
+	Place(src string, head Head) (Placement, error)
+	// Duplicate returns the archived invoice, in file name order, that has
+	// head's number, other than src and, for a working copy, the archived
+	// files it replaces. It returns "" when there is none, when head has
+	// no number, or when there is no archive directory.
+	Duplicate(src string, head Head) (string, error)
+	// Add moves the invoice at src into the archive at p with opts.Change
+	// applied, in one write, after backing up the archived files it
+	// replaces. Without opts.Replace it refuses to replace any.
+	Add(src string, p Placement, opts AddOptions) (ArchiveResult, error)
 	// Checkout resolves ref, an archived invoice relative to the archive
 	// directory, and says where its working copy in workDir goes.
 	Checkout(ref, workDir string) (Checkout, error)
-	// Dir returns the archive directory, "" when there is none.
-	Dir() (string, error)
-	// HistoryDir returns where backups are kept.
-	HistoryDir() (string, error)
-	// Resolve turns a name relative to the archive directory into a path.
-	Resolve(name string) (string, error)
-	// Existing returns those of paths that exist; a directory is an error.
-	Existing(paths ...string) ([]string, error)
 	// Protects reports whether path is an existing file inside the archive
 	// directory, which nothing but re-archiving overwrites.
 	Protects(path string) (bool, error)
-	// FindFile returns the archived file with one of names, "" when there
-	// is none, and an error when there are several.
-	FindFile(names ...string) (string, error)
+	// Source returns the invoice YAML file the PDF at pdf was built from:
+	// the one with its name next to it, else the one in the archive. It
+	// returns "" when there is none, and an error when the archive has
+	// several.
+	Source(pdf string) (string, error)
 }
 
 // EPC is what a template's EPC QR code placeholders need.
@@ -306,11 +330,9 @@
 	Render(t Template, inv *invoice.Context, epc EPC) (string, error)
 	// Write writes source to path and copies t's assets next to it.
 	Write(t Template, source, path string) error
-	// Scratch makes a temporary directory and returns it with the func
-	// that removes it.
-	Scratch() (string, func(), error)
-	// Copy copies the compiled document at src to dst.
-	Copy(src, dst string) error
+	// Build writes source with t's assets to a scratch directory, compiles
+	// it there with c, and copies the PDF to output.
+	Build(ctx context.Context, c Compiler, t Template, source, output string) error
 }
 
 // Compiler turns rendered source into a PDF and returns its path.
@@ -348,4 +370,7 @@
 	Draft(ctx context.Context, m Message) (Draft, error)
 	// Check runs the checks Draft runs on m.Output without writing.
 	Check(m Message) error
+	// CheckAttachment returns why the file at path cannot be attached, or
+	// nil.
+	CheckAttachment(path string) error
 }
diff -ruN -x ports_test.go base2/internal/billing/render.go proto2/internal/billing/render.go
--- base2/internal/billing/render.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/billing/render.go	2026-10-09 08:02:13.472180642 +0000
@@ -2,8 +2,6 @@
 
 import (
 	"context"
-	"path/filepath"
-	"strings"
 
 	"github.com/0xboris/invox/internal/invoice"
 )
@@ -112,7 +110,7 @@
 		return result, nil
 	}
 
-	if err := s.compile(ctx, t, source, req.Output); err != nil {
+	if err := s.Renderer.Build(ctx, s.Compiler, t, source, req.Output); err != nil {
 		return BuildResult{}, err
 	}
 	if err := s.markBuilt(req.Invoice); err != nil {
@@ -128,26 +126,6 @@
 	return result, nil
 }
 
-// compile writes source into a scratch directory, compiles it there, and
-// copies the PDF to output.
-func (s *Service) compile(ctx context.Context, t Template, source, output string) error {
-	dir, remove, err := s.Renderer.Scratch()
-	if err != nil {
-		return err
-	}
-	defer remove()
-
-	sourcePath := filepath.Join(dir, strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))+".tex")
-	if err := s.Renderer.Write(t, source, sourcePath); err != nil {
-		return err
-	}
-	pdf, err := s.Compiler.Compile(ctx, sourcePath)
-	if err != nil {
-		return err
-	}
-	return s.Renderer.Copy(pdf, output)
-}
-
 // markBuilt sets invoice.status to built after a successful PDF build. An
 // archived invoice keeps `archived`: rebuilding its PDF does not take it
 // out of the archive.
diff -ruN -x ports_test.go base2/internal/cmd/invoice/email/email.go proto2/internal/cmd/invoice/email/email.go
--- base2/internal/cmd/invoice/email/email.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/cmd/invoice/email/email.go	2026-10-09 08:02:13.490694381 +0000
@@ -147,8 +147,8 @@
 	pdfPath := cmdutil.AbsPath(baseDir, orDefault(opts.PDFPath, cmdutil.ReplaceExt(opts.InvoicePath, ".pdf")))
 	outputPath := cmdutil.AbsPath(baseDir, orDefault(opts.OutputPath, cmdutil.ReplaceExt(opts.InvoicePath, ".eml")))
 
-	result, err := svc.DraftEmail(ctx, billing.EmailRequest{
-		Input:     invoicePath,
+	request := billing.EmailRequest{
+		Invoice:   invoicePath,
 		PDF:       pdfPath,
 		Output:    outputPath,
 		Keep:      explicitOutput,
@@ -156,7 +156,11 @@
 		Subject:   opts.Subject,
 		Overwrite: opts.Force,
 		DryRun:    opts.DryRun,
-	})
+	}
+	if strings.EqualFold(filepath.Ext(invoicePath), ".pdf") {
+		request.Invoice, request.FromPDF = "", invoicePath
+	}
+	result, err := svc.DraftEmail(ctx, request)
 	if err != nil {
 		return outputExists(cmdutil.UsageError("email", err), baseDir)
 	}
diff -ruN -x ports_test.go base2/internal/email/mailer.go proto2/internal/email/mailer.go
--- base2/internal/email/mailer.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/email/mailer.go	2026-10-09 08:02:13.489775662 +0000
@@ -83,6 +83,12 @@
 	return writeFile(path, message, fsutil.Public)
 }
 
+// CheckAttachment returns why the file at path cannot be attached.
+func (Mailer) CheckAttachment(path string) error {
+	_, err := os.Stat(path)
+	return err
+}
+
 func checkOutput(path string, overwrite bool) error {
 	if info, err := os.Stat(path); err == nil && info.IsDir() {
 		return &billing.OutputIsDirError{Path: path}
diff -ruN -x ports_test.go base2/internal/factory/factory.go proto2/internal/factory/factory.go
--- base2/internal/factory/factory.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/factory/factory.go	2026-10-09 08:02:29.219683077 +0000
@@ -51,7 +51,7 @@
 		return &billing.Service{
 			Invoices:  st,
 			Directory: st,
-			Archives:  archive.Archive{Locate: h.ResolveArchiveDir, Read: store.ReadArchived},
+			Archives:  archive.Archive{Locate: h.ResolveArchiveDir, Read: store.ReadArchived, Rewrite: st.Rewrite},
 			Renderer:  latex.Renderer{},
 			Compiler:  compiler,
 			Mailer:    mailer,
diff -ruN -x ports_test.go base2/internal/render/latex/renderer.go proto2/internal/render/latex/renderer.go
--- base2/internal/render/latex/renderer.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/render/latex/renderer.go	2026-10-09 08:02:13.489663896 +0000
@@ -1,6 +1,7 @@
 package latex
 
 import (
+	"context"
 	"fmt"
 	"os"
 	"path/filepath"
@@ -183,20 +184,26 @@
 	return nil
 }
 
-// Scratch makes a temporary directory for a build.
-func (Renderer) Scratch() (string, func(), error) {
+// Build writes source with t's assets to a scratch directory, named after
+// output, compiles it there with c, and copies the PDF to output.
+func (r Renderer) Build(ctx context.Context, c billing.Compiler, t billing.Template, source, output string) error {
 	dir, err := os.MkdirTemp("", "invox-build-")
 	if err != nil {
-		return "", nil, err
+		return err
 	}
-	return dir, func() { _ = os.RemoveAll(dir) }, nil
-}
+	defer func() { _ = os.RemoveAll(dir) }()
 
-// Copy copies the compiled document at src to dst.
-func (Renderer) Copy(src, dst string) error {
-	data, err := os.ReadFile(src)
+	sourcePath := filepath.Join(dir, strings.TrimSuffix(filepath.Base(output), filepath.Ext(output))+".tex")
+	if err := r.Write(t, source, sourcePath); err != nil {
+		return err
+	}
+	pdf, err := c.Compile(ctx, sourcePath)
+	if err != nil {
+		return err
+	}
+	data, err := os.ReadFile(pdf)
 	if err != nil {
 		return err
 	}
-	return fsutil.WriteFile(dst, data, fsutil.Public)
+	return fsutil.WriteFile(output, data, fsutil.Public)
 }
diff -ruN -x ports_test.go base2/internal/store/invoices.go proto2/internal/store/invoices.go
--- base2/internal/store/invoices.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/store/invoices.go	2026-10-09 08:02:29.219396770 +0000
@@ -90,10 +90,14 @@
 // far smaller.
 const maxDraftScanSize = 1 << 20
 
-// Drafts returns the invoices directly inside dirs. The scan is best
-// effort: the archive check still guarantees unique numbers, so an
-// unreadable directory or file is skipped.
-func (s *Store) Drafts(dirs []string) []billing.Head {
+// Drafts returns the invoices directly inside workDir and the directory of
+// output. The scan is best effort: the archive check still guarantees
+// unique numbers, so an unreadable directory or file is skipped.
+func (s *Store) Drafts(workDir, output string) []billing.Head {
+	dirs := []string{workDir}
+	if output != "" {
+		dirs = append(dirs, filepath.Dir(output))
+	}
 	seen := make(map[string]bool, len(dirs))
 	var heads []billing.Head
 	for _, dir := range dirs {
@@ -129,25 +133,20 @@
 	return heads
 }
 
-// CheckOutput refuses a directory at path, and an existing file unless
-// overwrite is set.
-func (s *Store) CheckOutput(path string, overwrite bool) error {
+// Destination returns path, or <number>.yaml in workDir when path is "",
+// after refusing a directory there, and an existing file unless overwrite
+// is set.
+func (s *Store) Destination(path, workDir, number string, overwrite bool) (string, error) {
+	if path == "" {
+		path = filepath.Join(workDir, number+".yaml")
+	}
 	if err := refuseDirOutput(path); err != nil {
-		return err
+		return "", err
 	}
 	if !overwrite && fileExists(path) {
-		return &billing.OutputExistsError{Path: path}
+		return "", &billing.OutputExistsError{Path: path}
 	}
-	return nil
-}
-
-// Exists reports whether path is a file.
-func (s *Store) Exists(path string) bool { return fileExists(path) }
-
-// Stat returns why path cannot be read, or nil.
-func (s *Store) Stat(path string) error {
-	_, err := os.Stat(path)
-	return err
+	return path, nil
 }
 
 // Create writes a new invoice to path from the document at from.
@@ -247,17 +246,27 @@
 // change set to a different value are written, and an archive link it
 // clears is removed; every other key, comment and anchor stays as it is.
 func (s *Store) Update(path string, change func(*invoice.Invoice) error) error {
-	document, err := loadYAMLDocument(path)
+	data, err := s.Rewrite(path, change)
 	if err != nil {
 		return err
 	}
+	return fsutil.WriteFile(path, data, fsutil.Public)
+}
+
+// Rewrite returns the invoice at path with change applied, as Update
+// writes it.
+func (s *Store) Rewrite(path string, change func(*invoice.Invoice) error) ([]byte, error) {
+	document, err := loadYAMLDocument(path)
+	if err != nil {
+		return nil, err
+	}
 	root, err := documentRootMapping(document, path)
 	if err != nil {
-		return err
+		return nil, err
 	}
 	invoiceNode, err := invoiceMapping(root, path)
 	if err != nil {
-		return err
+		return nil, err
 	}
 	// Values that do not decode read as unset; change leaves them alone.
 	var before invoice.Invoice
@@ -278,7 +287,7 @@
 		after.Archive = &link
 	}
 	if err := change(&after); err != nil {
-		return err
+		return nil, err
 	}
 
 	if after.CustomerID != before.CustomerID {
@@ -293,7 +302,7 @@
 	if !sameLink(after.Archive, before.Archive) {
 		setArchiveLink(root, after.Archive)
 	}
-	return writeYAMLDocument(path, document)
+	return encodeYAMLDocument(document)
 }
 
 func sameLink(a, b *invoice.ArchiveLink) bool {
diff -ruN -x ports_test.go base2/internal/store/legacy_api_test.go proto2/internal/store/legacy_api_test.go
--- base2/internal/store/legacy_api_test.go	2026-10-09 08:00:00.000000000 +0000
+++ proto2/internal/store/legacy_api_test.go	2026-10-09 08:02:29.295201903 +0000
@@ -6,8 +6,10 @@
 
 import (
 	"context"
+	"fmt"
 	"os"
 	"path/filepath"
+	"strings"
 	"time"
 
 	"github.com/0xboris/invox/internal/archive"
@@ -32,7 +34,7 @@
 	return &billing.Service{
 		Invoices:  st,
 		Directory: st,
-		Archives:  archive.Archive{Locate: h.ResolveArchiveDir, Read: ReadArchived},
+		Archives:  archive.Archive{Locate: h.ResolveArchiveDir, Read: ReadArchived, Rewrite: st.Rewrite},
 		Renderer:  latex.Renderer{},
 		Mailer:    email.Mailer{},
 		Settings:  h.Settings,
@@ -208,20 +210,7 @@
 	if err != nil {
 		return err
 	}
-	dir, remove, err := svc.Renderer.Scratch()
-	if err != nil {
-		return err
-	}
-	defer remove()
-	sourcePath := filepath.Join(dir, filepath.Base(outputPath[:len(outputPath)-len(filepath.Ext(outputPath))])+".tex")
-	if err := svc.Renderer.Write(h.templateFor(templatePath), source, sourcePath); err != nil {
-		return err
-	}
-	pdf, err := svc.Compiler.Compile(ctx, sourcePath)
-	if err != nil {
-		return err
-	}
-	return svc.Renderer.Copy(pdf, outputPath)
+	return svc.Renderer.Build(ctx, svc.Compiler, h.templateFor(templatePath), source, outputPath)
 }
 
 type compilerFunc func(ctx context.Context, sourcePath string) (string, error)
@@ -276,9 +265,39 @@
 	OutputPath  string
 }
 
+// ResolveEmailDraftPaths derives the invoice, PDF and draft of an email for
+// input as the email command does: the PDF and draft default to input's
+// name, and the invoice of a PDF is looked up next to it or in the archive.
 func (h Host) ResolveEmailDraftPaths(inputPath, pdfPath, outputPath string) (EmailDraftPaths, error) {
-	invoicePath, pdf, output, err := h.service(Files{}, cwd(), time.Time{}).EmailPaths(inputPath, pdfPath, outputPath)
-	return EmailDraftPaths{InvoicePath: invoicePath, PDFPath: pdf, OutputPath: output}, err
+	paths := EmailDraftPaths{InvoicePath: inputPath, PDFPath: pdfPath, OutputPath: outputPath}
+	if paths.OutputPath == "" {
+		paths.OutputPath = replaceExt(inputPath, ".eml")
+	}
+	switch strings.ToLower(filepath.Ext(inputPath)) {
+	case ".pdf":
+		found, err := h.service(Files{}, cwd(), time.Time{}).Archives.Source(inputPath)
+		if err != nil {
+			return EmailDraftPaths{}, err
+		}
+		if found == "" {
+			return EmailDraftPaths{}, fmt.Errorf("%s: no matching invoice YAML found next to the PDF or in archive.dir", inputPath)
+		}
+		paths.InvoicePath = found
+		if paths.PDFPath == "" {
+			paths.PDFPath = inputPath
+		}
+	case ".yaml", ".yml":
+		if paths.PDFPath == "" {
+			paths.PDFPath = replaceExt(inputPath, ".pdf")
+		}
+	default:
+		return EmailDraftPaths{}, fmt.Errorf("%s: input must end with .yaml, .yml, or .pdf", inputPath)
+	}
+	return paths, nil
+}
+
+func replaceExt(path, ext string) string {
+	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
 }
 
 type EmailParams struct {
@@ -299,9 +318,13 @@
 
 func (h Host) PrepareInvoiceEmail(p EmailParams) (EmailMessage, error) {
 	svc := h.service(Files{Customers: p.CustomersPath, Issuer: p.IssuerPath}, cwd(), time.Time{})
+	pdf := p.PDFPath
+	if pdf == "" {
+		pdf = replaceExt(p.InvoicePath, ".pdf")
+	}
 	result, err := svc.DraftEmail(context.Background(), billing.EmailRequest{
-		Input:   p.InvoicePath,
-		PDF:     p.PDFPath,
+		Invoice: p.InvoicePath,
+		PDF:     pdf,
 		To:      p.Recipient,
 		Subject: p.Subject,
 		DryRun:  true,
```

</details>
