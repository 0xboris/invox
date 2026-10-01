# Eval results

## Iteration 1 (2026-10-01)

Skill at invox `7e91ad8`, before the clig.dev / go-cli-development additions.
Model claude-opus-5-5, one run per configuration, isolated workspaces.
Evals 1 and 2 were graded by scripts that run the built binaries; eval 3 by grader agents.

| Eval | With skill | Without | Time with / without | Tokens with / without |
|---|---|---|---|---|
| 1. `invox customer list --json` | 8/8 | 8/8 | 260s / 131s | 126k / 105k |
| 2. Scaffold `notes` CLI | 8/8 | 7/8 | 826s / 281s | 214k / 82k |
| 3. Review `export.go` | 9/9 | 9/9 | 228s / 139s | 109k / 62k |

**Discriminating.** Only eval 2. Without the skill, piped `notes list` kept a header
row and aligned columns.

**Saturated.** Evals 1 and 3 pass both ways. Two harder assertions were then added to
eval 3; they have not been run yet:
- data loss when `--force` deletes after a failed export write;
- missing auth header / unchecked HTTP status.

**Not captured by assertions.** The with-skill `notes` CLI had 12 test files vs 2,
separate TTY and piped output, help topics and `--jq`/`--template`. The with-skill
invox change added zsh completion and exact-output tests.

**Cost.** About 2.4× the time and 1.8× the tokens. Eval 2 dominates because the run
cloned and adapted the full starter template.

**Triggering.**
- skill-creator's `run_loop`: about 0% recall, 100% precision. Its harness counts a
  trigger only if the skill is the session's *first* tool call, and coding agents
  usually look at files first.
- Real sessions with the skill installed in a project: used on 3/3 CLI tasks and
  not used on an unrelated bash one-liner. The description was kept.

**Next iteration.** Re-run with the harder eval 3 assertions, add a `--no-input` /
prompt-on-stderr check to eval 2, and use 3 runs per configuration.
