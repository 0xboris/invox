# Clean-architecture experiment: progress log

Spec: `docs/design/target/README.md`. Done when `scripts/verify-target.sh --target` exits 0.

## Summary

Done: `scripts/verify-target.sh --target` passes with all 32 target checks, from the commit
that closes step 7 on. Seven steps, one commit each, in the README's order.

What changed. `internal/invoice` holds only entities: the schema types without YAML tags,
value types with `Parse*` constructors, `Status` with a transition table, `Validate` returning
`[]Problem`, and the VAT totals. `internal/numbering` formats and parses invoice numbers.
`internal/billing` holds the 14 use cases on `Service`, the six ports and the error types the
CLI matches; it imports only the entities. `internal/store` decodes and writes the YAML files
and resolves paths; `archive`, `render/latex`, `email`, `adapters/tectonic` and
`adapters/applemail` implement the other ports. `internal/factory` wires them, and the
commands call `Factory.Service(...)` and print. `.golangci.yml` and `archtest_test.go` state
the package table, one depguard rule per row.

Behavior. The 35 testscript goldens, `docs/cli` and `share/man` are byte-for-byte unchanged,
no test was removed or skipped, and the CLI tests passed unchanged once the code compiled.
Two untested edge cases changed; follow-up unit A (70c583d) restored them, with tests.

What was hard. Comment-keeping writes: `new` has always rewritten fields such as
`paid_amount: 0` as `"0"` even when the value is the same, so a diff of the entity cannot
express it. `Invoices.Create` therefore writes every header field it is given, and
`Invoices.Update` writes only the fields that changed. Keeping every error message and its
order meant porting each use case step by step, which made the ports wider than the README's
draft (`Head`, `Drafts`, `CheckOutput`, `Locate`, `Checkout`, `Protects`, `FindFile` and
others). The archive's walk order differs from its sorted list order, and numbering depends
on walk order for its warning, so `Archive.Entries` keeps walk order and billing sorts where
the old code listed.

How. Two scratchpad tools did the moves (one moves declarations between files, one qualifies
identifiers left unresolved after a package split), gopls did the renames, and the 45 store
test files reach the new code through a test-only shim of the old `Host` API instead of being
rewritten call by call.

Open blockers: none. Follow-ups after the run (see the Follow-up units section): narrowing the
ports, sizing `billing`, and moving the store tests off the shim onto `billing.Service` directly.

## Blockers

None yet.

## Known differences

Edge cases where the refactor changes output and no test pins it. Each is a
candidate for a follow-up fix.

None.

## Log

| Date | Step | Commit | What moved | Target checks | Notes |
|---|---|---|---|---|---|
| 2026-10-08 | 1 | f7fb7bf | Split `drafts.go` into `new.go`, `increment.go`, `archive_add.go`, `archive_edit.go`, `status.go`, `output.go` and `archive_metadata.go`; merged `archive_replace.go` into them. Moved the YAML node helpers and `loadArchivedInvoiceDocument` into `yamldoc.go`, `removedKey` into `models.go`, the `Source` constants and `configDirName` into `host.go`, the file helpers into `abs_path.go`. | 5 passed, 14 failed | A small AST tool (scratchpad, not committed) finds file-level reference cycles; none remain in `internal/invoice` apart from matches on the method name `Error`. |
| 2026-10-08 | 2 | 2910c5a | New `internal/numbering`: `Settings.Validate`, `Format`, `Parse`, `InPeriod`, `Next`. `invoice/numbering.go` keeps the archive and draft scans and calls it. Moved `FuzzInvoiceNumberRoundTrip` and three tests. Added numbering to the leaf and domain rules in `.golangci.yml` and `archtest_test.go`. | 6 passed, 13 failed | Callers pass the customer code trimmed, so the fuzz target trims it too. |
| 2026-10-08 | 3 | 41e6590 | `invoice/lifecycle.go`: `Status` (draft, built, editing, archived) and a transitions table over the actions building, archiving, re-archiving, emailing and numbering. `invoice/validate.go`: `Validate(Bundle) []Problem` with the checks from `LoadContext`. New `TestStatusTransitions`. | 6 passed, 13 failed | `TestTargetEntities` no longer reports Status or Problem; it waits on step 4 (names, tags, Host). |
| 2026-10-08 | 4 | 113b62a | Moved 21 files, `starter/`, `testdata/` and 45 test files from `internal/invoice` to `internal/store`. `invoice` keeps `models.go` (no tags), `scalars.go` (`ParseDecimal`, `ParseRate`, `ParseDate`, `ParseCount`), `validate.go`, `lifecycle.go`, `context.go` (`NewContext`) and the business errors; it imports only `money`. `store/schema.go` maps YAML keys and removed keys to fields; `store/scalars.go` decodes the value types with one type switch. Renamed `InvoiceFile`, `IssuerFile`, `InvoiceHeader` to `Invoice`, `Issuer`, `Header` with gopls. | 9 passed, 10 failed | Two scratchpad tools did the move: one moves declarations between files, one qualifies identifiers left unresolved after a package split. `store` still imports `archive`, `email`, `render/latex` and `epc`; that goes away as the orchestration moves to `billing` in step 5. |
| 2026-10-08 | 5 | 08d1941 | New `internal/billing` (ports, `Service`, use cases, errors, EPC payload, email templates). `store.Store` implements `Directory` and `Invoices`; `archive.Archive`, `latex.Renderer`, `email.Mailer`, `tectonic.Compiler.Compile` and `applemail.Composer.Draft` implement the other ports. Deleted the orchestration from `store` (new, increment, archive add and edit, status, email, render, numbering scans, LoadContext). Commands call `f.Service(cmdutil.Files{...})`; path display helpers moved to `cmdutil`. Updated `archtest` and depguard rules toward the table. | 29 passed, 3 failed | The testscript goldens and CLI tests passed unchanged on the first run. 160 store test call sites go through a test-only shim of the old API rather than being rewritten. `Invoices.Create` writes the header fields it is given even when unchanged, because `new` has always rewritten `paid_amount: 0` as `"0"`; `Update` writes only changed fields. Two untested edge cases changed; see Known differences. |
| 2026-10-08 | 6 | 298e799 | Moved `cmdutil.NewFactory` and the user-directory resolution to `factory.New`; `cmd/invox`, `docs/gen`, `factorytest` and the tests call it. `cmdutil` keeps `Factory`, `Files` and the editor and opener. | 32 passed, 0 failed | Every target check passes from here; step 7 makes `.golangci.yml` and `archtest_test.go` state the full table. |
| 2026-10-08 | 7 | 0d2ae79 | Rewrote the depguard rules and `archtest_test.go` rules as the package table (entities, use cases, driven, driving, main, libraries), checked that depguard rejects a driven import from a command and a `helptext` import from `cmd/invox`. Updated the Layout, Conventions and Testing parts of `CLAUDE.md`. Wrote the summary. | 32 passed, 0 failed | depguard matches by prefix, so the CLI root and `factory` need a trailing `$` for an exact match. |
| 2026-10-08 | log | this commit | Filled in the step 7 commit. | 32 passed, 0 failed | |

## Follow-up units

All four units landed on 2026-10-09 (A, B, D) or were decided (C). Open: `Directory` keeps 14 methods with some overlap (`Locate` and `EditablePath`, the legacy helpers and `Locations`); narrowing the ports further would let error messages change order, which is a product decision; the context fixtures are duplicated between the billing and latex tests.

Coordinated from the review in the target-architecture comparison: restore the changed edge cases
(A), narrow the ports so path decisions live in the adapters (B), decide on splitting `billing`
(C), and move the `store` tests off the `Host` shim (D). Each lands only after
`scripts/verify-target.sh --target` passes on its commit in a clean worktree.

| Date | Unit | Commit | What moved | Target checks | Notes |
|---|---|---|---|---|---|
| 2026-10-09 | A | 70c583d | Restored "`invoice` must be a mapping" for a null, scalar or aliased `invoice:` on `archive add` and `archive edit`, and the removal of an empty or null `_invox` link on archive. `store.Head` reports the `invoice` key's shape as written (`billing.HeaderShape`); `invoice.Invoice.Archive` is a pointer; `Invoices.LoadArchived` became `ArchivedHead`. New `internal/cli/archive_shape_test.go`: 9 subtests fail on d13b178, all pass here. | 32 passed, 0 failed | Same root cause also fixed an aliased `invoice:` that moved the file into the archive before failing. Open: archiving copies the file before `Invoices.Update` sets its status, so a later failure can leave an archived copy with `status: built`. |
| 2026-10-09 | B design | b45b567 | `docs/design/ports-narrowing.md`: disposition of all 39 port methods and the 8 extra `Service` methods, final interfaces, acceptance checks, a 7-commit plan, and a 50-scenario differential against 47aef7e. Prototyped in full before landing. | 32 passed, 0 failed | Narrows what billing does with paths, not how many methods the ports have: 39 to 36. Going further would let error messages change order. |
| 2026-10-09 | B | dca15ec..995d58a | Seven commits. `Renderer.Build` owns the scratch directory, compile and copy; `Invoices.Destination` and `Drafts(workDir, output)` name files; `Archive.Place`, `Duplicate` and `Source` do placement, the duplicate scan and the PDF-to-YAML lookup; `Archive.Add` owns the whole move and writes the archived invoice once (`store.Rewrite`); `Mailer.CheckAttachment` replaces `Invoices.Stat`. `internal/billing` imports neither `path` nor `path/filepath`. New `internal/archtest/ports_test.go` pins the exact method sets (36 port methods; 21 on `Service`) and bans path imports and file-extension literals in billing; new depguard rule `use-cases-no-paths`. New `internal/cli/port_order_test.go` pins 14 error orders. | 32 passed, 0 failed | Coordinator checks: verify-target OK on 995d58a in a clean worktree; full tests pass on 62c079b (the riskiest step); the differential rebuilt from 47aef7e gives 50 scenarios, 1 differing (`email-write`, the .eml timestamp), the same as base against itself. |
| 2026-10-09 | C | (decision) | Don't split `billing`. It is 1,955 lines after B, but the 14 use cases stay methods on `Service` and the only code that could leave cleanly (`emailtext.go`, `epc.go`, about 200 lines) belongs to billing by the spec. A split would add pass-through layers. | | Revisit if billing grows a second, separable concern. |
| 2026-10-09 | D | 475d398..7937e18 | Deleted the test-only `Host` shim (`store/legacy_api_test.go`). Use-case tests moved to `internal/billing` as `package billing_test` on `factorytest`; render, asset and PDF-build tests to `internal/render/latex`; PDF-to-invoice lookup tests to `internal/archive`; email path defaulting tests to `internal/cmd/invoice/email`. `internal/store` keeps 49 test functions (was 154) on decoding, writing and paths. `factorytest.Options` gained `Now`; the unused `store.writeYAMLDocument` is gone. | 32 passed, 0 failed | Coordinator checks: verify-target OK on 7937e18 in a clean worktree; test and fuzz names identical before and after (491); assertion calls 2,065 before, 2,078 after; production diff limited to factorytest and the deleted helper. |

## Round 2

The maintainer lifted byte-identical output ("we can break things if need be") and asked to drop
legacy support: the invoice-tool directory, the deprecated command forms, and Markdown archived
invoices (with a warning naming skipped files). Plans: `docs/design/cleanup-plan.md` (12 units)
and `docs/design/ports-v2.md` (ports 36 to 24 methods, Service to the 14 use cases). Guardrails:
`scripts/verify-target.sh` takes its baseline from `docs/design/target/BASELINE`; output may
change only when each changed file is named in `docs/design/output-changes/`.

| Date | Unit | Commits | What moved | Target checks | Notes |
|---|---|---|---|---|---|
| 2026-10-09 | guardrails | 74d3563 | Movable baseline, output-change and removed-test ledgers per unit, regenerated baselines. | 32 passed, 0 failed | Control: an unlisted golden change fails, a listed one passes. |
| 2026-10-09 | U1 | 52b9678 | Dropped `archive FILE`, `customer config`, `send`, `new -s/--source`, `cmdutil/deprecated.go`; the single-dash normaliser became a check that rejects `-names` with exit 2. 11 goldens changed, `deprecated.txtar` deleted; see `output-changes/U1.md`. | 32 passed, 0 failed | Coordinator ran the binary: removed forms exit 2 with the planned messages; `-output=x.yaml` writes nothing; `-ofile.yaml` still works. |
| 2026-10-09 | U7 | d85a2bd..7d9e2b8 | `time.DateOnly`; one numbering token table and `Parse` without a per-call regexp; `latex.Item`/`VATRow` deleted; `run.Stub` moved to `run/runtest` (not in the release binary); one `fsutil.Abs` for store and factory. | 32 passed, 0 failed | Fixed a Windows path bug, reproduced under wine; see `output-changes/U7.md`. Verified on the combined tip with U1. |
| 2026-10-09 | baseline | 2a68009 | Target test: `Compiler` belongs to `render/latex`, not billing; `ports-v2.md` is normative for the ports. | (fails on the two Compiler checks by design) | |
| 2026-10-09 | verifier | 3ac8d0c | Fixed a SIGPIPE race in the ledger lookups (`cat \| grep -q` under pipefail gave random false failures; found by the ports v2 worker). | | Reproduced 300/300 with a large input, 0/300 after. |
| 2026-10-09 | ports v2 + U2 + U3 | 9d2fc02..0770402 (as cherry-picked) | Legacy invoice-tool directory and `init --force` gone; Markdown archives no longer read, with a warning naming each skipped file; ports at 24 methods (Invoices 4, Directory 10, Archive 5, Renderer 3, Mailer 1, plus `latex.Compiler`); `Service` is the 14 use cases; `TestPortsCarryNoAdapterData`. 5 goldens and 8 generated pages changed; 19 tests removed with reasons. | 31 passed, 0 failed | Coordinator checks: verify OK on 0770402 in a clean worktree; `go doc` method counts; real binary shows the Markdown warning (README.md ignored, `--json` stdout clean); 85-scenario differential rebuilt from 2a68009: the same 32 scenarios differ as the worker reported, each listed in `output-changes/{ports-v2,U2,U3}.md`. |

