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

- `cmd/invox/main.go`: shim, `os.Exit(cli.Run(os.Args[1:]))`.
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

- CLI tests: `captureRun(t, args)` returns `(exitCode, stdout, stderr)`; assert all three.
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

`.claude/skills/` vendors [pstack](https://github.com/backnotprop/pstack) at `124f622`
(MIT, `.claude/skills/PSTACK-LICENSE`), unedited. Left out: `make-bot-ui`,
`typescript-best-practices`, `setup-pstack`, `poteto-mode/scripts/`, and the `shipping`,
`autopilot-full` and `autopilot-stack` playbooks. To update, re-copy from a newer pstack
commit with the same exclusions and bump the commit here.

Every task in this repo runs in poteto-mode. Before any work, read
`.claude/skills/poteto-mode/SKILL.md` in full (it can't be invoked as a skill by the model),
then follow it: match a playbook, and read each `principle-*` leaf you apply. Subagents use
`subagent_type: poteto-agent`. Models per role come from `.claude/pstack-models.md`, which the
session-start hook installs as `~/.agents/pstack-models.md`.

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
- Use the GitHub MCP tools, not `gh` or `gt`. No `bun`, `npx`, `curl | sh` or Cursor cloud
  agents; use local subagents.
- Only Claude models exist here. Read any grok, gpt or `claude-*-max` name in a skill as the
  matching role in `.claude/pstack-models.md`. "A different model family" means a different
  Claude model.
- `~/` and `/tmp` don't persist across cloud sessions. Keep notes, plans and decision logs in
  the PR or the repo.
- Don't post to chat, tickets or other external services unless asked. poteto-mode's "just do
  it" covers local, reversible work only.
- References to the left-out pieces are expected. For landing a PR, the coordinator merges
  after the maintainer's go-ahead (instead of `shipping`). Do the steps that name
  `poteto-mode/scripts/` (`watch-pr`, `check-plan.mjs`, `orch`) by hand.
