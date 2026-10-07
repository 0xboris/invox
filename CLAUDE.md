# invox

Go CLI for YAML- and LaTeX-driven invoices: `new`, `validate`, `render`, `build` (via
`tectonic`), `email`, `archive`. One dependency (`gopkg.in/yaml.v3`).

## Commands

```sh
go build ./...                 # or: make build  (-> ./bin/invox)
go test -race ./...            # or: make test
go vet ./...
gofmt -l .                     # must print nothing
go mod tidy -diff              # must print nothing
```

CI (`.github/workflows/ci.yml`) runs the tests on linux/macos/windows with Go 1.24 (the
minimum in `go.mod`) and stable, and gofmt/vet/tidy on Linux. Keep all of it green.

## Layout

- `cmd/invox/main.go`: shim, `os.Exit(cli.Main(os.Args[1:], iostreams.System()))`.
- `internal/iostreams`: stdin, stdout and stderr plus TTY detection. Only `iostreams.System()`
  touches the process streams; everything else writes to the `IOStreams` it is given.
- `internal/cli`: argument parsing (`command_specs.go`, `parsing.go`), commands
  (`commands_*.go`), hand-written help (`help.go`), editor/opener/mail launchers (`editor.go`).
- `internal/invoice`: domain logic (loading and validation, money and VAT, numbering,
  rendering, EPC QR, email drafts, archive). `service.go` is large and being split up.
- `internal/invoice/starter`: files embedded for `invox init`.

## Quality roadmap

Work is tracked in the roadmap tracking issue #9 and its phase epics (#10–#16) and sub-issues.
The standards come from the `quality-cli` skill (`.claude/skills/quality-cli`). The
assessment behind the roadmap is in `docs/quality/ASSESSMENT.md`.

Settled decisions (#9):
- usage errors exit 2
- the CLI moves to cobra one noun at a time
- stdout becomes data-only (status goes to stderr)
- `fonts/` is removed

## Conventions

- One issue per PR. Reference it with `Fixes #N` and keep the diff to what the issue asks.
- Every bug fix comes with a regression test that fails without the fix.
- Treat stdout/stderr content, exit codes, flag names and file formats as a contract. Change
  them only when the issue calls for it, and update the tests that pin them.
- Domain code (`internal/invoice`) must not print, read env vars, call `os.Exit`, or mention
  CLI flags in new code. Return typed errors; the CLI layer words them.
- Match the surrounding style: early returns, `fmt.Errorf("...: %w", err)`, table tests where
  several cases share a shape.

## Testing

- CLI tests: `captureRun(t, args)` returns `(exitCode, stdout, stderr)`; assert all three. It runs
  `Main` with `iostreams.Test()` buffers; `captureRunStreams` takes streams you set up (stdin input,
  TTY flags). Never swap `os.Stdin`, `os.Stdout` or `os.Stderr`.
- Never depend on the developer's real config: point `XDG_CONFIG_HOME` (and for paths derived
  from home, `setPlatformHome(t, dir)` in `internal/invoice`) at `t.TempDir()`.
- `build` tests use `installFakeTectonic(t, fakeTectonicWritePDF|fakeTectonicFail)`, which
  puts the test binary on PATH as `tectonic`. No shell scripts, so tests run on Windows.
- Use `chdirForTest` for working-directory changes. Swapped package-level hooks
  (`openTextFile`, `openDocument`, `currentDate`, ...) must be restored with `t.Cleanup`.
- The e2e suite (`cmd/invox/script_test.go`, scripts in `cmd/invox/testdata/script/*.txtar`)
  pins every command's stdout, stderr and exit code with testscript. Run it with
  `go test ./cmd/invox -run TestScript` (one script: `-run TestScript/archive`). After an
  intentional output change, rewrite the goldens with `go test ./cmd/invox -run TestScript -update`
  and review the diff. Use `exits CODE invox ...` for exit codes and `scrubpaths` before `cmp`
  when output contains sandbox paths.
- Tests must pass on Windows and macOS too. Build paths with `filepath.Join`, and remember that
  `os.UserHomeDir` reads `USERPROFILE` on Windows.

## Agent workflow: poteto-mode (pstack)

`.claude/skills/` vendors all of [pstack](https://github.com/backnotprop/pstack)'s `skills/`
at `124f622` (MIT, `.claude/skills/PSTACK-LICENSE`). `quality-cli` is this repo's own skill.
`agents/comment-sicko.md` is left out of `.claude/agents/`: "Comment Sicko" isn't a valid
Claude Code agent name, and `no-comments` falls back to a general subagent with its bundled
prompt.

Local edits, all so pstack's GitHub calls work in Claude Code cloud sessions (REST only):

- `poteto-mode/scripts/watch-pr/github.ts` reads PRs, checks, reviews and threads through
  `gh api` REST and the `/ccr/review_threads` route; `github.test.ts` covers it. The check
  source labels `gh-pr-checks` and `graphql-rollup` in `types.ts` are kept unchanged on purpose:
  they are part of the verdict JSON, and renaming them would edit more upstream files.
- `poteto-mode/scripts/orch/`: `frontier set --source rest` reads the open-PR list through
  `gh api`, with tests in `orch.test.ts`.
- `poteto-mode/scripts/worktree-audit.sh` lists PRs through `gh api`, paged by hand.
- The playbooks `shipping`, `babysit`, `opening-a-pr`, `multi-phase-plan`, `autopilot-full`,
  `autopilot-stack` and `orchestrate`, plus `why/SKILL.md` and
  `why/references/sources/code-archaeology.md`, use `gh api` REST commands.
- New: `poteto-mode/references/github-rest.md`, the REST and `/ccr/` command sheet.

To update to a newer pstack commit:

1. Clone pstack at the old and the new commit. `diff -ruN -x quality-cli -x PSTACK-LICENSE
   -x node_modules <old>/skills .claude/skills` must show only the local edits above.
2. Copy `<new>/skills/` over `.claude/skills/`, keeping `quality-cli` and `PSTACK-LICENSE`.
3. Re-apply each local edit. `git diff HEAD -- <path>` shows what the copy reverted.
4. Check: the diff against the new commit shows only the list above;
   `grep -rnE 'gh (pr|issue) |graphql' .claude/` finds only notes that explain the ban; and
   `bun install --frozen-lockfile && bun test orch watch-pr && bun run typecheck` pass in
   `.claude/skills/poteto-mode/scripts`. Bump the commit here.

Every task in this repo runs in poteto-mode. Before any work, read
`.claude/skills/poteto-mode/SKILL.md` in full (it can't be invoked as a skill by the model),
then follow it: match a playbook, and read each `principle-*` leaf you apply. Subagents use
`subagent_type: poteto-agent`. Models per role come from `.claude/pstack-models.md`, which is
the source of truth. The session-start hook copies it over `~/.agents/pstack-models.md` at
every cloud session start, so anything `/setup-pstack` writes there is lost in the next session.
To change models, run `/setup-pstack`, then copy `~/.agents/pstack-models.md` back into
`.claude/pstack-models.md` and commit it in its own PR. Or edit the repo file directly. Locally,
copy it to `~/.agents/` once, as the hook does in the cloud.

This file and the harness instructions take precedence over every vendored skill, playbook
and agent. When they conflict, follow this file and say which rule you overrode:

- Never merge, auto-merge or arm merge-when-ready without the maintainer's explicit go-ahead
  for that specific PR. "Full autonomy", "land", "ship" and swarm verdicts are not that
  go-ahead. Merges are squash merges.
- One issue per PR, `Fixes #N` in the body, diff limited to the issue. No stacked PRs, no
  drive-by PRs for broken skills or nearby bugs; report them as follow-ups instead.
- Never force-push, rebase or `git reset --hard` a branch you didn't create.
- Commit and PR attribution comes from the harness. Playbook title and body formats (for
  example Conventional Commits) don't replace it or `Fixes #N`.
- For GitHub, use `gh api` REST calls (`.claude/skills/poteto-mode/references/github-rest.md`)
  or the GitHub MCP tools. Cloud sessions refuse GraphQL, so no `gh pr`, `gh issue` or
  `gh api graphql`. Page lists by hand (`per_page=100&page=N`); `gh api --paginate` fails
  there past the first page. The `/ccr/` routes exist only in cloud sessions.
- `bun` is allowed for the pstack scripts. No `gt`, `npx`, `curl | sh` or Cursor cloud
  agents; use local subagents.
- Only Claude models exist here. Read any grok, gpt or `claude-*-max` name in a skill as the
  matching role in `.claude/pstack-models.md`. "A different model family" means a different
  Claude model.
- `~/` and `/tmp` don't persist across cloud sessions. Keep notes, plans and decision logs in
  the PR description or the repo.
- Agents may post review findings and verification verdicts on PRs in this repo. This covers
  PR review comments, a review summary, and the Shipping playbook's per-PR PASS / PASS+NOTES /
  FAIL verdict. They end with the harness's Claude Code footer.
- Everything else external still needs the maintainer to ask: other comments or issues, chat,
  and any other service. poteto-mode's "just do it" covers local, reversible work only.
- The Shipping and Autopilot playbooks run under the rules above. Their overrides:
  - Merge and auto-merge wait for the maintainer's go-ahead for that specific PR, and merges
    are squash merges with `sha=<verified head>`.
  - Local subagents replace their Cursor cloud agents.
  - Autopilot-stack's stacked delivery isn't used here, because PRs aren't stacked.
- Run the `poteto-mode/scripts/` tools (`watch-pr`, `check-plan.mjs`, `orch`) with `bun` from
  that directory. This repo doesn't use Graphite, so `orch frontier set` takes `--source rest`.
