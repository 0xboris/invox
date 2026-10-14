---
name: new-slice
description: Adds a command or a new command noun to invox in the shape of the existing commands. Use when a task adds a new capability, such as a new `invox <noun>` group or a new verb under an existing noun.
---

1. Pick the noun from `GLOSSARY.md`. Add the concept to the glossary if it is new. A new noun is a new slice; a new verb joins its noun's slice.
2. Create `internal/cmd/<noun>/<verb>` by hand. Copy the structure of `internal/cmd/invoice/validate`: an Options struct, `NewCmdX(f, runF)`, a run function, and tests that pass a `runF` for parsing. Do not run `tools/new.py`, which targets a layout invox does not have.
3. Register the command in its noun's parent command. A new noun is registered in `internal/cli/root.go`.
4. Put every business decision in `internal/invoice` or the domain module that owns it, and call it from the run function. Commands only parse input, call the domain, and write output to the `IOStreams` they are given.
5. When the command needs a program from outside the process, follow the new-port skill.
6. When another slice needs the same rule or type, move it into the domain. Slices never import each other.
7. Pin the command's stdout, stderr and exit code in a testscript under `cmd/invox/testdata/script`, and regenerate the docs with `make docs`.
8. Run `python3 tools/archcheck.py --module-map docs/architecture/modules.md` for a new noun, then `./check` after each step, and finish when it passes.
