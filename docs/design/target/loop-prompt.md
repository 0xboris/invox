# Loop prompt

Paste everything below the line into a Claude Code session started on branch
`experiment/clean-architecture` (for example `/loop <prompt>`, or as the first message of a cloud
session).

---

Refactor invox on branch `experiment/clean-architecture` until `scripts/verify-target.sh --target`
exits 0. Work in poteto-mode, Autonomous run playbook. The exit condition is that command; never
relax it.

Read first, in this order:
1. `docs/design/target/README.md`: the spec. Its package table is normative.
2. `docs/design/target-progress.md`: what earlier sessions did. Resume from the first step not
   marked done.
3. `internal/archtest/target_test.go` and `scripts/verify-target.sh`: how done is checked.

Rules for this run (they settle where CLAUDE.md's PR rules don't fit an experiment branch):
- Commit and push only to `experiment/clean-architecture`. Never push elsewhere, open a PR or an
  issue, merge, force-push, or rewrite history.
- One spec step at a time, in the README's order. A step may take several commits; every commit
  must pass `scripts/verify-target.sh` (without `--target`). Push after every commit, because the
  session can be reclaimed at any time.
- After every commit, add a row to `docs/design/target-progress.md`: step, commit, what moved, the
  `--target` count ("N passed, M failed"), and anything learned.
- Never edit `docs/design/target/`, `internal/archtest/target_test.go` or `scripts/verify-target.sh`.
  Never run testscript with `-update`, add `t.Skip`, or delete a test to get green. A test that no
  longer applies gets a `- TestName: reason` line in `docs/design/target-removed-tests.md`.
- Keep the CLI's behavior identical: same stdout, stderr, exit codes, files written, and help text.
- If a spec rule looks wrong or a step is blocked after two different approaches, write it under
  "Blockers" in the progress log with the evidence, and continue with the next piece of work that
  doesn't depend on it.

When `scripts/verify-target.sh --target` exits 0, write a final summary at the top of the progress
log (what changed, what was hard, open blockers) and stop.
