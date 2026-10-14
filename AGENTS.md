# Agent instructions

This codebase follows the architecture in `docs/ARCHITECTURE.md`. Read it before you add or move code. `docs/architecture/go.md` shows where each part sits in invox today, and `docs/architecture/modules.md` lists every module with its kind.

## Commands

- `./check` verifies the architecture, formatting, `go.mod` tidiness, vet, the build, and the tests. Run it after each step. A task is done only when it passes.
- `python3 tools/archcheck.py` runs the architecture check alone.
- `python3 tools/archcheck.py --module-map docs/architecture/modules.md` regenerates the module map after you add or move a package.

`tools/new.py` generates use cases and HTTP handlers, which invox does not use. Do not run it here. `docs/architecture/go.md` says which existing code to copy instead.

## How to work here

- Copy the structure of the closest existing code, as `docs/architecture/go.md` describes.
- When the check reports a violation, apply the fix it names. Do not change `architecture.json`, `check`, `tools/`, `.claude/`, or `.github/` to make a check pass. Those are gate files, and changes to them need the owner. The code has no accepted violations, so never create `architecture-baseline.json` with `--write-baseline`.
- Use the names in `GLOSSARY.md` for business concepts, and add new concepts to it.
- Every behavior change gets a test that fails without the change. A bug fix starts with a test that reproduces the bug.
- End with a short report: what changed, what you could not verify, and follow-ups you noticed but did not do.
