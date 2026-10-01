# CLI review checklist

Use for reviewing a diff, auditing an existing CLI, or self-checking before finishing.
Run `bash <skill-dir>/scripts/cli_audit.sh <repo-root>` first for the mechanical checks (Go).
Mark each item ✅ / ❌ / n/a. Severity hints: **[break]** = breaks scripts or users,
**[bug]** = wrong behavior, **[ux]** = quality issue.

## Contract & compatibility
- [ ] **[break]** No removed/renamed flags, args, commands, env vars, JSON fields without deprecation.
- [ ] **[break]** Non-TTY output format unchanged (columns, tabs, no header, timestamps), or a major version.
- [ ] **[break]** Exit codes unchanged for existing scenarios; new codes documented.
- [ ] **[break]** Data still on stdout, diagnostics still on stderr.
- [ ] **[ux]** Deprecated items warn and point to the replacement.

## Structure
- [ ] **[bug]** Command returns errors; no `os.Exit`, `log.Fatal`, `panic` for user errors.
- [ ] **[bug]** No direct `os.Stdout/Stderr/Stdin`, `os.Getenv`, `time.Now`, `exec.Command` in command code — goes through IOStreams/Factory/Options/run seam.
- [ ] **[ux]** Options + `NewCmdX(f, runF)` + `xRun(opts)`; validation in `RunE` before `runF`.
- [ ] **[bug]** Late-bound factory fields (repo/context overrides) read inside `RunE`.
- [ ] **[bug]** Expensive deps lazy; `--help` works offline with broken config.
- [ ] **[bug]** `cmd.Context()` passed to network/subprocess calls (Ctrl-C works).

## Flags & args
- [ ] **[ux]** Long name for every flag; short only for common ones, consistent with siblings.
- [ ] **[bug]** Enums validated (and completed); `--limit < 1` rejected; mutually exclusive flags enforced.
- [ ] **[bug]** Tri-state flags use `*bool`/`Changed()` where "unset" ≠ false.
- [ ] **[ux]** `-` accepted for stdin on file flags where it makes sense.
- [ ] **[ux]** Arg validator present with a clear message; `Use` line follows `<required> [optional]` syntax.

## Output
- [ ] **[bug]** Checked with stdout piped: no color, no header, tab-separated, no truncation, absolute times, explicit state words.
- [ ] **[ux]** Checked on TTY: header, alignment, color via ColorScheme, relative times, success icon lines.
- [ ] **[ux]** `--json` (fields validated, bare `--json` lists fields) + `--jq` + `--template` on list/view commands; JSON field list in help annotation.
- [ ] **[bug]** Empty result: TTY message on stderr, exit 0; `[]` with `--json`.
- [ ] **[bug]** Spinner on stderr, stopped before any other output or prompt; disabled when not TTY.
- [ ] **[bug]** Untrusted/remote text sanitized of escape sequences.
- [ ] **[ux]** Created resource URL/path printed on stdout.

## Interactivity
- [ ] **[bug]** Prompts only when `CanPrompt()`; non-TTY path errors with the flags to use (no hang waiting on stdin).
- [ ] **[bug]** Each prompt has a flag equivalent.
- [ ] **[bug]** Destructive ops: TTY confirmation, `--yes` required otherwise, implicit targets not deletable with `--yes` alone.
- [ ] **[ux]** Cancellation exits 2 quietly with a newline.

## Errors & help
- [ ] **[ux]** Errors: lowercase, specific, say how to fix; usage only for flag errors.
- [ ] **[ux]** `Short` (no period), `Long`, `Example` with `$ ` lines; annotations for arguments/env/JSON fields.
- [ ] **[ux]** New env vars/config keys documented in `help environment` / config options table.
- [ ] **[ux]** Generated docs/completions regenerated.

## Config & secrets
- [ ] **[bug]** Writes atomic, permissions preserved (0600 for secrets); no writes in read-only commands.
- [ ] **[bug]** Secrets never logged or printed (except explicit `auth token`); keyring calls have timeouts.
- [ ] **[ux]** Precedence flag > env > config > default holds.

## Tests
- [ ] **[bug]** `TestNewCmdX` (parsing/validation via `runF`) and `TestXRun` (TTY + non-TTY exact output).
- [ ] **[bug]** HTTP/exec/prompt stubs self-verify (unused stub fails); no real network/home/keyring.
- [ ] **[ux]** Error paths and `--json` covered; acceptance script added for new user-facing flows.

## Background behavior
- [ ] **[bug]** Update checks/telemetry async, cancellable, off in CI/non-TTY, never delay exit.

---

## Report format (for reviews)

```
## CLI review: <scope>
**Breaking**: <list or "none">
**Bugs**: <file:line — issue — fix>
**UX**: <file:line — issue — fix>
**Tests missing**: <list>
**Verdict**: ship / ship after fixes / needs redesign
```
