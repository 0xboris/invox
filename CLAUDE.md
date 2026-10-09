# invox

Go CLI for YAML- and LaTeX-driven invoices: `new`, `validate`, `render`, `build` (via
`tectonic`), `email`, `archive`. Dependencies: `gopkg.in/yaml.v3`, and `spf13/cobra` with its `pflag`.

## Commands

```sh
go build ./...                 # or: make build  (-> ./bin/invox)
go test -race ./...            # or: make test
go vet ./...
gofmt -l .                     # must print nothing
go mod tidy -diff              # must print nothing
make lint                      # golangci-lint at CI's version, rules in .golangci.yml
make vulncheck                 # govulncheck ./... (needs vuln.go.dev)
```

CI (`.github/workflows/ci.yml`) runs the tests on linux/macos/windows with Go 1.24 (the
minimum in `go.mod`) and stable, and gofmt/vet/tidy, golangci-lint and govulncheck on Linux.
Keep all of it green.

## Layout

The code follows the clean architecture of `docs/design/target/README.md`: source
dependencies point inward, from main to the driving and driven adapters to the use cases
(`billing`) to the entities. Its package table is the rule set.

- `cmd/invox/main.go`: shim. It builds the `cmdutil.Factory` with `factory.New` from
  `iostreams.System()`, `run.Exec{}` and `env.System()`, then exits with
  `cli.Main(os.Args[1:], f)`.
- `internal/factory`: the composition root. `factory.New` resolves the user directories from
  the `env.Env`, builds `store`, `archive` (with the clock that stamps backups),
  `render/latex` (with the tectonic compiler and `store.Host.FindAsset`) and the mailer (Apple
  Mail on macOS without `-o`, otherwise an .eml file it opens with the Factory's opener), and
  wires them into `billing.Service` behind `Factory.Service(cmdutil.Files)`. It hands
  `archive.Archive.Protects` to `store.Store.Protected` and `store.Rewrite` to the archive,
  and fills `Factory.Locations`, the default locations help texts show.
  `factory/factorytest` builds the same Factory on temporary directories for command tests.
- Entities (standard library and `money` only, no disk): `internal/invoice` (the schema types
  without YAML tags, the value types with `Parse*` constructors, `Status` and its transition
  table, `Validate` returning `[]Problem`, VAT totals in `NewContext`), `internal/numbering`
  (number patterns: `Format`, `Parse`, `Next`), `internal/epc` (EPC QR rules and payload) and
  `internal/money` (decimals, cents, 1.234,56 formatting).
- `internal/billing`: the use cases, one method each on `Service` (`New`, `Increment`,
  `Validate`, `Render`, `Build`, `Archive`, `EditArchived`, `DraftEmail`, `ListCustomers`,
  `ListArchive`, `Paths`, `Init`, `ListTemplates`, `EditablePath`) and no other exported
  method, the five ports in `ports.go` (`Invoices`, `Directory`, `Archive`, `Renderer`,
  `Mailer`; `docs/design/ports-v2.md` explains each method), the email subject and body
  templates, and every error type the CLI words (`ConfigError`, `ToolMissingError`,
  `FileNotFoundError`, `DecodeError`, `OutputExistsError` and so on). Ports carry domain data
  only: no func fields in their results, and no result handed back to the same port. Results
  of use cases that walk the archive carry `Unread`, the Markdown invoices it no longer reads.
  It imports only the entities.
- Driven adapters implement the ports. `internal/store` (`store.Store`: `Directory` and
  `Invoices`) owns YAML decoding of the invoice files and `config.yaml` (strict decoder, alias
  limits, duplicate keys, the key table in `schema.go`), comment-keeping writes (`Create`, which names `<number>.yaml` and refuses
  to overwrite an archived file, and `Update`), config and support-file lookup in the invox
  config directory only, and the `init` starter files (`starter/`). `internal/archive`
  (`archive.Archive`) owns the archive directory: walk, list, name resolution, where `Add`
  places an invoice, `.history` backups; it reads invoices through `store.ReadArchived` and
  reports `.md`/`.markdown` files that open with front matter as `billing.Unread` instead of
  reading them. `internal/render/latex` (`latex.Renderer`) owns placeholders, escaping, line
  item blocks, template checks and asset copying, and declares the `Compiler` it builds with.
  `internal/email` (`email.Mailer`) owns the .eml MIME layout, the temporary-draft policy
  and opening the draft. `internal/adapters/tectonic` implements `latex.Compiler` and
  `internal/adapters/applemail` `billing.Mailer`, both on a `run.Runner`.
- Driving adapters: `internal/cli`, `internal/cmd/...`, `internal/tableprinter`,
  `internal/adapters/editor` and `internal/adapters/opener`. They never import a driven
  adapter, `config`, `fsutil` or `factory`.
  - `internal/cli`: `Main`, the cobra root (`root.go`: global flags, the check that
    rejects single-dash long flags, help routing, help groups), exit codes (`exit.go`), signals and the help page
    renderer (`usage.go`); the `completion` command is in `internal/cmd/completion`. Help is generated from each command's `Short`,
    `Long` and `Example`; a `Long` is a text/template that `helptext.Render` fills in from
    the fields of `helptext.Locations` (`{{.Customers}}`), which `cmdutil.Factory.Locations`
    returns. `helptext` holds the shared help content: the topic pages
    (`topics/<name>.tmpl`, each registered by one line in `Topics`), the sections in
    `reference.tmpl` that a `Long` or page includes with `{{template "customer-fields"}}`,
    the reference tables they range over (`reference.go`) and the lookup lines. `cmdutil` holds the `Factory`, `Files`, the error types, `UsageError` (which words
    a missing support file or template as a usage error), `FlagErrorFunc`, the `Args`
    checks (`NoArgs`, `ExactArgs`, `MaximumArgs`), `AddSupportFlags` (the `-c`, `-u`,
    `--defaults` and `-t` flags from one table, into `SupportPaths`), the path display
    helpers and what else commands share (editor, prompt, completion funcs). A usage error
    carries no command name: `Main` names the help of the command that ran, and
    `FlagError.Root` points to the root help for a global flag or a help topic.
  - `internal/cmd/<noun>/<verb>`: one cobra command per package, each with an Options struct,
    `NewCmdX(f, runF)`, where `runF func(context.Context, *XOptions) error` defaults to the
    run function that calls one `Service` method and prints its result. Parse-only tests pass
    a `runF`. The invoice verbs live under
    `internal/cmd/invoice/`, with what they share in `internal/cmd/invoice/shared`, such as
    `WarnUnread`, the one stderr warning that names the Markdown invoices a command skipped.
- `internal/docs/gen`: `go run ./internal/docs/gen` (or `make docs`) rewrites `docs/cli/*.md`
  and `share/man/man1/*.1` from the command tree. CI fails when they are stale, so regenerate
  them with any help change. The release binary doesn't import it.
- Libraries, which import only the standard library: `internal/config` (the typed settings
  of `config.yaml`, which `store` decodes), `internal/fsutil`,
  `internal/adapters/run` (the only package that calls `os/exec`), `internal/iostreams`
  (stdin, stdout and stderr plus TTY detection; only `iostreams.System()` touches the process
  streams), `internal/env` and `internal/build`. `env.Env` holds `GOOS`, `Getenv`, `HomeDir`,
  `Getwd` and `Now`; `env.System()` reads them from the process and everything else uses the
  `Env` it is given. `ambient_test.go` fails on any other read, apart from a short allowlist of
  functions (`iostreams.newSystem` and `run.Exec.Run`, whose child processes inherit the
  environment). `Factory.Env` carries the `Env`.
- The layering is enforced. depguard and forbidigo in `.golangci.yml` check each file, one
  depguard rule per row of the package table; `internal/archtest` checks the transitive import
  graph with `go list`. A new package or import that crosses a ring fails both; update the
  rules in the same PR when the layering itself changes.

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
- The core (`internal/invoice`, `internal/billing` and the other entities) must not print,
  read env vars, call `os.Exit`, touch the disk, or mention CLI flags. Return typed errors; the
  CLI layer words them.
- Match the surrounding style: early returns, `fmt.Errorf("...: %w", err)`, table tests where
  several cases share a shape.

## Testing

- CLI tests: `captureRun(t, args)` returns `(exitCode, stdout, stderr)`; assert all three. It runs
  `Main` with `iostreams.Test()` buffers; `captureRunStreams` takes streams you set up (stdin input,
  TTY flags). Never swap `os.Stdin`, `os.Stdout` or `os.Stderr`.
- Never depend on the developer's real config: in CLI tests point `XDG_CONFIG_HOME` at
  `t.TempDir()`; in command tests build the Factory with `factorytest.New`; in `internal/store`
  build a `store.Host` with `store.NewHost(store.HostInputs{...})` whose directories are under
  `t.TempDir()`. Use-case tests live in `internal/billing` as `package billing_test` and build
  the `Service` with `factorytest.New`; the `internal/store` tests cover decoding, writing and
  path resolution only.
- `build` tests use `installFakeTectonic(t, fakeTectonicWritePDF|fakeTectonicFail)`, which
  puts the test binary on PATH as `tectonic`. No shell scripts, so tests run on Windows.
- Use `chdirForTest` for working-directory changes. Swapped package-level hooks
  must be restored with `t.Cleanup`.
- Tests that reach the editor, the opener or Apple Mail use `testFactory(t)` and
  `captureRunFactory`. Its `runtest.Stub` panics on any program the test did not register with
  `expectEditor`, `expectOpener` or `stub.Register`, and fails the test if one never runs. The
  mailer opens drafts itself, so a use-case test that drafts an email passes a stub runner
  through `factorytest.Options.Runner` (`mailService` in `internal/billing`).
- `internal/archtest/ports_test.go` pins the exact method set of each port and of
  `*billing.Service`, and `TestPortsCarryNoAdapterData` fails when a port result carries a func
  or a port takes back a struct it returned. A new port method updates the lists there.
- invox reads neither the old `invoice-tool` config directory nor Markdown archived invoices;
  write archive fixtures as `.yaml`. A `.md` file in a test archive exercises the warning.
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
