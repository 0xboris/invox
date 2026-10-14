# Agent instructions

This codebase follows the architecture in `docs/ARCHITECTURE.md`. Read it before you add or move code. `docs/architecture/go.md` shows how each part looks in this language, and `docs/architecture/modules.md` lists every module with its kind.

## Commands

- `./check` verifies everything: the architecture, the build, the types, and the tests. Run it after each step. A task is done only when it passes.
- `python3 tools/new.py slice <Name>` creates a feature slice with its UI module, use case module, tests, and wiring.
- `python3 tools/new.py usecase <Slice> <Name>` adds a use case to a slice.
- `python3 tools/new.py port <Name>` adds a port with its shared fake and contract test.

Use the generators instead of creating modules by hand. They produce the structure that the check expects.

## How to work here

- Copy the structure of the closest existing code. The Notes slice is the reference for every pattern until real slices replace it.
- When the check reports a violation, apply the fix it names. Do not change `architecture.json`, `architecture-baseline.json`, `check`, `tools/`, `.claude/`, or `.github/` to make a check pass. Those are gate files, and changes to them need the owner.
- Use the names in `GLOSSARY.md` for business concepts, and add new concepts to it.
- Every behavior change gets a test that fails without the change. A bug fix starts with a test that reproduces the bug.
- End with a short report: what changed, what you could not verify, and follow-ups you noticed but did not do.
