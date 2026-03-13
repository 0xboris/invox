---
name: latex-expert
description: >
  Expert LaTeX guidance for `.tex`, `.cls`, `.sty`, `.bib`, and LaTeX build
  logs. Use when Codex needs to answer LaTeX syntax questions, fix compilation
  errors, explain commands or environments, adjust document structure or layout,
  work on math/tables/graphics, or author classes and packages with the bundled
  LaTeX2e reference.
---

# LaTeX Expert

Work from the user's actual source and logs first. Use the bundled LaTeX2e reference to confirm syntax and command behavior before editing.

## Do this first

- Read the relevant `.tex`, `.cls`, `.sty`, `.bib`, and `.log` files before proposing changes.
- Identify the request type early: compile error, syntax question, layout issue, math issue, table/float issue, or macro/class/package authoring.
- If the user names a specific command, environment, or topic, search `references/` with `rg -n` before opening large files.
- If the question depends on package-specific behavior outside base LaTeX, say that the bundled reference is core LaTeX2e only and inspect the package docs separately.

## Triage

- Compile error: find the first real error in the log and ignore later cascade errors until that one is resolved.
- Syntax question: answer with a minimal example and name any required package, class, or engine.
- Layout issue: inspect the document class, relevant packages, sectioning, page layout, floats, and spacing commands before changing markup.
- Macro/class/package authoring: use the definitions and document-class references first, and preserve the existing public interface unless the user asks for a redesign.

## Routing map

- Start with `references/_index.md` when the right chapter is not obvious.
- Use `references/02-overview-of-latex.md` for command syntax, environment syntax, special characters, engines, and document start/end structure.
- Use `references/03-document-classes.md` and `references/12-definitions-class-and-package-commands.md` for class/package authoring.
- Use `references/04-fonts.md` and `references/23-special-insertions.md` for fonts, encodings, symbols, accents, and text-level special characters.
- Use `references/05-layout.md`, `references/09-line-breaking.md`, `references/10-page-breaking.md`, `references/14-lengths.md`, `references/15-making-paragraphs.md`, `references/19-spaces.md`, and `references/20-boxes.md` for layout and spacing.
- Use `references/06-sectioning.md`, `references/07-cross-references.md`, and the `references/25-front-back-matter-*.md` files for structure, cross-references, TOC, indexes, and glossaries.
- Use the `references/08-environments-*.md` files for environments, floats, lists, tables, bibliography, verbatim, and theorem-like structures.
- Use the `references/16-math-*.md` files plus `references/17-modes.md` for math syntax, symbols, spacing, and styles.
- Use `references/21-graphics.md` and `references/22-color.md` for figures, graphics inclusion, and color handling.
- Use `references/24-splitting-the-input.md`, `references/27-input-output.md`, and `references/28-command-line-interface.md` for multi-file projects, I/O primitives, and engine CLI behavior.
- Use `references/26-letters.md` and `references/A-document-templates.md` for letters and starter templates.

## Working rules

- Prefer the smallest change that resolves the problem.
- Do not add packages unless they are required for the requested behavior.
- Preserve the existing engine (`pdflatex`, `xelatex`, `lualatex`, etc.) unless the user asks to switch or the current setup is incompatible with the goal.
- Call out multi-pass requirements when relevant: cross-references, TOC, indexes, glossaries, and bibliographies often need repeated runs or external tools.
- For explanations, give a minimal compilable example when that is faster than prose.
- For package-specific advice not covered by the bundled reference, say so explicitly instead of overstating confidence.

## Verification

- If build tools are available, use the existing project build command or the document's current engine to verify edits.
- When reading logs, report the first actionable error with its file and line if available.
- After structural edits, re-check labels, references, TOC/index generation, float placement, and any engine-specific packages involved.
