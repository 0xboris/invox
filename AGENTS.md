# Agent instructions

`README.md` describes what the project does. This codebase follows the architecture in `docs/ARCHITECTURE.md`. Read it before you add or move code. `docs/architecture/go.md` shows how each part looks in Go, `docs/architecture/stack.md` names the libraries invox uses and where each part of invox sits, `docs/architecture/modules.md` lists every module with its kind, and `docs/architecture/where-code-goes.md` says which module new code belongs in.

## Reference slice

The reference slice is **customer**: `internal/features/customer/ui`, `internal/features/customer/usecases`, and `internal/app/wire_customer.go`. Copy its structure when you add a slice, a command, a use case, a port, an adapter, or their tests. Each verb is a package under `ui/` with an Options struct, `NewCmdX(f, runF)` and a run function, and reaches the ports only through the slice's use cases.

## Commands

- `./check` runs the architecture check, the module map, gofmt, `go mod tidy -diff`, vet, the build, the race tests, and the project checks in `checks/`. Run it after each step. A change is done when the check passes and you have shown the change working in the running app.
- The `verify-invox` skill in `.claude/skills/` runs the built `invox` against a temporary config directory and keeps each command, its output and its exit code as evidence. Its feature map lists what invox does.
- The `new-slice` and `new-port` skills list the files that a new slice or port needs.
- The `deslop` skill removes AI slop from the branch's diff. pstack runs it before each commit.

## How to work here

- Copy the structure of the reference slice, or of the closest existing code that follows it.
- When the check reports a violation, apply the fix it names. A failing `./check` is a finding in the code, never a flaky tool.
- Use the names in `GLOSSARY.md` for business concepts, and add new concepts to it.
- A bug fix starts with a failing reproduction, committed before the fix. Use a test when a cheap one exists, and otherwise a script or a recorded run against the running app.
- Change an API inside the repository in one step. Move every caller and delete the old form in the same change. Stored formats and wire contracts change by expanding first, as `docs/ARCHITECTURE.md` describes.
- Every review of a change here also answers the questions in `docs/architecture/review.md`. When you run `/pstack:interrogate`, pass that file to its reviewers as a context file.

## Gates

`.github/CODEOWNERS` lists the gate files: `architecture.json`, `check`, `tools/`, `.claude/`, `.github/`, the instruction files, the architecture docs, the compiler, linter, and test runner settings, and the composition root. A change to them can loosen a check, so the owner reviews it. The composition root is on the list because it decides which middleware each slice receives.

- Never change a gate file to make a check pass, and never disable or soften a check to get a pull request green.
- A gate change, such as a new module kind, a fix to a check you think is wrong, a lint setting, a row in the rule table, or a skill, goes into its own pull request against main. Link it in your report and keep working on what doesn't depend on it. A slice's wiring and its entries in the verification skill's feature map are the exception. They stay in the slice's pull request, and the owner reviews that pull request.
- To tighten a rule, add an executable to `checks/` whose message names the fix, or lower the baseline with `python3 tools/archcheck.py --write-baseline` after you fix a violation. Neither needs the owner.
- Loosening a check is the owner's call. That covers raising the baseline, changing or removing an existing check, removing a glossary term, and adding a suppression comment such as `oxlint-disable` or `//nolint` to production code. Add a row to the exceptions table below in the pull request that loosens the check: what it allows, the reason, and an expiry date written as YYYY-MM-DD. The gate guard fails a loosening without a new row, and a new row that lacks any of the three. The owner reviews the row, because `AGENTS.md` is a gate file.
- A suppression comment also names its reason on its line, as `/pstack:correct` asks of every exception. The row holds the expiry date and the owner's approval, because the gate guard can enforce them there. A raised baseline or a changed check has no line, so its row holds the reason too.

## Rules and what enforces them

When agents keep making the same mistake, fix it at the highest level that works: the architecture, then a type, then a check whose message names the fix, then a test, and a row here last. A rule whose only enforcement is review is a candidate for a check.

| Rule | Enforced by |
| --- | --- |
| Modules import only the kinds that `architecture.json` allows, and slices never import each other. | `tools/archcheck.py`, depguard in `.golangci.yml`, and `internal/archtest` |
| Every source file belongs to a module, a test, or a script. | `tools/archcheck.py` |
| The domain, use cases, and UI perform no I/O. | `tools/archcheck.py`, `forbiddenCalls`, and forbidigo in `.golangci.yml` |
| UI reaches ports only through its slice's use cases. | `tools/archcheck.py`, `forbidPorts` |
| Utilities and shared UI use no business terms. | `tools/archcheck.py`, `bannedTerms` from `GLOSSARY.md` |
| The composition root holds only wiring. | `tools/archcheck.py`, `composition` |
| Only `internal/adapters/platform` reads the process environment and streams, and only `programs/run` starts programs. | `internal/adapters/platform/ambient_test.go`, `process_streams_test.go`, forbidigo, and `internal/archtest` |
| Every command keeps its stdout, stderr and exit code. | The testscript goldens in `cmd/invox/testdata/script` |
| The help pages in `docs/cli` and `share/man` match the command tree. | CI, which fails when `go run ./internal/docs/gen` changes them |
| Violations in the baseline only go down, except through a row in the exceptions table. | `architecture-baseline.json`, `tools/gateguard.py`, and `tools/archcheck.py` fails while the baseline accepts more than exist |
| Checks in `checks/` and glossary terms are only added, except through a row in the exceptions table. | `tools/gateguard.py`, and CI runs the existing checks as the base branch has them |
| Production code adds no lint or type-check suppression, except through a row in the exceptions table. | `tools/gateguard.py` |
| An exception ends on its expiry date. | `tools/gateguard.py` fails every pull request while an expired row remains |
| A slice that needs a middleware keeps it. | `.github/CODEOWNERS` on the composition root, and `internal/app/dependencies_test.go` |
| A stored format still loads data that an earlier version wrote. | The committed samples in `internal/domain/invoice/testdata/golden` and the testscript fixtures |
| A use case makes no business decision, and each rule exists once. | Review, `docs/architecture/review.md` |

## Exceptions

Each row is a loosening that the owner approved, with the reason and the date it expires. Before a row expires, fix what it allowed and remove the row. Remove it sooner when its mistake can no longer happen.

| Exception | Reason | Expires |
| --- | --- | --- |
