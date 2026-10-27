---
name: new-slice
description: Adds a command or a new command noun to invox in the shape of the reference slice that AGENTS.md names. Use when a task adds a new capability, such as a new `invox <noun>` group or a new verb under an existing noun.
---

1. Pick the noun from `GLOSSARY.md`. Add the concept to the glossary if it is new. A new noun is a new slice; a new verb joins its noun's slice.
2. Read the reference slice that `AGENTS.md` names, and the layout in `docs/architecture/stack.md`.
3. Create the files by copying the reference slice's structure:
   - a use case in `internal/features/<noun>/usecases`, with its test, for each step that reaches a port
   - the command in `internal/features/<noun>/ui/<verb>`: an Options struct, `NewCmdX(f, runF)`, a run function, and tests that pass a `runF` for parsing
   - for a new noun, `internal/features/<noun>/ui/shared/factory.go` with only the fields its commands use, and `internal/app/wire_<noun>.go`, which builds the use cases from `Dependencies`, puts them into the Factory, and returns the noun's commands. Add the wire call to `commands` in `internal/app/app.go` and the noun to `commandGroups` in `internal/features/cli/ui/root.go`.
4. Put every business decision in the domain module that owns it, and call it from the use case. Use cases only orchestrate. Commands only parse input, call use cases and domain functions, and write output to the `IOStreams` they are given. A command never names a port.
5. When the use case needs a program or anything else from outside the process, follow the new-port skill.
6. When another slice needs the same rule or type, move it into the domain. Slices never import each other or share a use case.
7. Test the use case against the real file system in `t.TempDir()`, as the reference slice does, and assert what you read back. Use `run.Stub` for programs, and add a failing port implementation to `internal/testsupport` only for error paths.
8. Pin the command's stdout, stderr and exit code in a testscript under `cmd/invox/testdata/script`, and regenerate the docs with `make docs`.
9. Add the command's entries to the feature map of `.claude/skills/verify-invox/`, in the shape of its feature files. Keep them in the slice's pull request.
10. Run `./check` after each step. Finish when it passes and the verification skill shows each new command working, with the evidence named in your report.
