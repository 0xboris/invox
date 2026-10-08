# Clean-architecture experiment: progress log

Spec: `docs/design/target/README.md`. Done when `scripts/verify-target.sh --target` exits 0.

## Summary

In progress. Steps 1 to 7 of the spec are worked in order; see the log.

## Blockers

None yet.

## Known differences

Edge cases where the refactor changes output and no test pins it. Each is a
candidate for a follow-up fix.

- `invoice:` with no value (YAML null): `archive add` and `archive edit`
  said "`invoice` must be a mapping" and now say "missing `invoice`
  mapping". The decoder reads null as an absent mapping, and billing only
  sees the decoded invoice.
- `_invox: {}` (an empty link) on an invoice that is archived: archiving
  removed the key and now keeps it, because `Invoices.Update` writes back
  only fields that changed and an empty link reads the same as none.

## Log

| Date | Step | Commit | What moved | Target checks | Notes |
|---|---|---|---|---|---|
| 2026-10-08 | 1 | f7fb7bf | Split `drafts.go` into `new.go`, `increment.go`, `archive_add.go`, `archive_edit.go`, `status.go`, `output.go` and `archive_metadata.go`; merged `archive_replace.go` into them. Moved the YAML node helpers and `loadArchivedInvoiceDocument` into `yamldoc.go`, `removedKey` into `models.go`, the `Source` constants and `configDirName` into `host.go`, the file helpers into `abs_path.go`. | 5 passed, 14 failed | A small AST tool (scratchpad, not committed) finds file-level reference cycles; none remain in `internal/invoice` apart from matches on the method name `Error`. |
| 2026-10-08 | 2 | 2910c5a | New `internal/numbering`: `Settings.Validate`, `Format`, `Parse`, `InPeriod`, `Next`. `invoice/numbering.go` keeps the archive and draft scans and calls it. Moved `FuzzInvoiceNumberRoundTrip` and three tests. Added numbering to the leaf and domain rules in `.golangci.yml` and `archtest_test.go`. | 6 passed, 13 failed | Callers pass the customer code trimmed, so the fuzz target trims it too. |
| 2026-10-08 | 3 | 41e6590 | `invoice/lifecycle.go`: `Status` (draft, built, editing, archived) and a transitions table over the actions building, archiving, re-archiving, emailing and numbering. `invoice/validate.go`: `Validate(Bundle) []Problem` with the checks from `LoadContext`. New `TestStatusTransitions`. | 6 passed, 13 failed | `TestTargetEntities` no longer reports Status or Problem; it waits on step 4 (names, tags, Host). |
| 2026-10-08 | 4 | 113b62a | Moved 21 files, `starter/`, `testdata/` and 45 test files from `internal/invoice` to `internal/store`. `invoice` keeps `models.go` (no tags), `scalars.go` (`ParseDecimal`, `ParseRate`, `ParseDate`, `ParseCount`), `validate.go`, `lifecycle.go`, `context.go` (`NewContext`) and the business errors; it imports only `money`. `store/schema.go` maps YAML keys and removed keys to fields; `store/scalars.go` decodes the value types with one type switch. Renamed `InvoiceFile`, `IssuerFile`, `InvoiceHeader` to `Invoice`, `Issuer`, `Header` with gopls. | 9 passed, 10 failed | Two scratchpad tools did the move: one moves declarations between files, one qualifies identifiers left unresolved after a package split. `store` still imports `archive`, `email`, `render/latex` and `epc`; that goes away as the orchestration moves to `billing` in step 5. |
| 2026-10-08 | 5 | 08d1941 | New `internal/billing` (ports, `Service`, use cases, errors, EPC payload, email templates). `store.Store` implements `Directory` and `Invoices`; `archive.Archive`, `latex.Renderer`, `email.Mailer`, `tectonic.Compiler.Compile` and `applemail.Composer.Draft` implement the other ports. Deleted the orchestration from `store` (new, increment, archive add and edit, status, email, render, numbering scans, LoadContext). Commands call `f.Service(cmdutil.Files{...})`; path display helpers moved to `cmdutil`. Updated `archtest` and depguard rules toward the table. | 29 passed, 3 failed | The testscript goldens and CLI tests passed unchanged on the first run. 160 store test call sites go through a test-only shim of the old API rather than being rewritten. `Invoices.Create` writes the header fields it is given even when unchanged, because `new` has always rewritten `paid_amount: 0` as `"0"`; `Update` writes only changed fields. Two untested edge cases changed; see Known differences. |
| 2026-10-08 | 6 | pending | Moved `cmdutil.NewFactory` and the user-directory resolution to `factory.New`; `cmd/invox`, `docs/gen`, `factorytest` and the tests call it. `cmdutil` keeps `Factory`, `Files` and the editor and opener. | 32 passed, 0 failed | Every target check passes from here; step 7 makes `.golangci.yml` and `archtest_test.go` state the full table. |
