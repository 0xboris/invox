# Clean-architecture experiment: progress log

Spec: `docs/design/target/README.md`. Done when `scripts/verify-target.sh --target` exits 0.

## Summary

In progress. Steps 1 to 7 of the spec are worked in order; see the log.

## Blockers

None yet.

## Log

| Date | Step | Commit | What moved | Target checks | Notes |
|---|---|---|---|---|---|
| 2026-10-08 | 1 | (next row's commit) | Split `drafts.go` into `new.go`, `increment.go`, `archive_add.go`, `archive_edit.go`, `status.go`, `output.go` and `archive_metadata.go`; merged `archive_replace.go` into them. Moved the YAML node helpers and `loadArchivedInvoiceDocument` into `yamldoc.go`, `removedKey` into `models.go`, the `Source` constants and `configDirName` into `host.go`, the file helpers into `abs_path.go`. | 5 passed, 14 failed | A small AST tool (scratchpad, not committed) finds file-level reference cycles; none remain in `internal/invoice` apart from matches on the method name `Error`. |
