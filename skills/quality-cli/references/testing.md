# Testing a CLI

Snippets show gh's own test helpers (`iostreams.Test`, `httpmock`, `run.Stub`,
`prompter.NewMockPrompter`). They are not importable; copy the pattern or use the
go-cli-starter equivalents.

Goal: every command is testable without a real terminal, network, keyring, home dir,
subprocess or clock — and tests fail loudly when a stub is wrong *or unused*.

## Contents
- [Test layers](#test-layers)
- [Layer 1: flag parsing via runF](#layer-1-flag-parsing-via-runf)
- [Layer 2: run function with stubs](#layer-2-run-function-with-stubs)
- [HTTP stubs](#http-stubs)
- [Subprocess stubs](#subprocess-stubs)
- [Prompter mocks](#prompter-mocks)
- [Golden files](#golden-files)
- [Acceptance tests with testscript](#acceptance-tests-with-testscript)
- [Hygiene](#hygiene)

---

## Test layers

| Layer | What | Tooling |
|---|---|---|
| 1. `TestNewCmdX` | flag/arg parsing + validation → Options | inject `runF` that captures opts |
| 2. `TestXRun` | behavior + exact stdout/stderr, TTY and non-TTY | `iostreams.Test()`, httpmock, `run.Stub`, prompter mock |
| 3. Formatters/helpers | pure functions, table/JSON rendering | table-driven, golden files |
| 4. Acceptance/e2e | real binary or in-process `Main()` against a sandbox | testscript (`.txtar`) |

Every command gets layers 1 and 2. Docker policy worth copying: each command has at
least one e2e success test that exercises most flags; new flags are added to it.

## Layer 1: flag parsing via runF

```go
func TestNewCmdCreate(t *testing.T) {
    tests := []struct {
        name     string
        tty      bool
        stdin    string
        cli      string
        wantErr  string
        wantOpts CreateOptions
    }{
        {name: "non-tty without title", tty: false, cli: "", wantErr: "must provide `--title` and `--body` when not running interactively"},
        {name: "body from stdin", tty: false, stdin: "from stdin", cli: "-t hello -F -",
            wantOpts: CreateOptions{Title: "hello", Body: "from stdin"}},
        {name: "web and body conflict", tty: true, cli: "--web --body x", wantErr: "specify only one of `--body` or `--web`"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ios, stdin, _, _ := iostreams.Test()
            ios.SetStdinTTY(tt.tty)
            ios.SetStdoutTTY(tt.tty)
            stdin.WriteString(tt.stdin)
            f := &cmdutil.Factory{IOStreams: ios}

            var gotOpts *CreateOptions
            cmd := NewCmdCreate(f, func(o *CreateOptions) error { gotOpts = o; return nil })
            args, _ := shlex.Split(tt.cli)
            cmd.SetArgs(args)
            cmd.SetIn(&bytes.Buffer{}); cmd.SetOut(io.Discard); cmd.SetErr(io.Discard)

            _, err := cmd.ExecuteC()
            if tt.wantErr != "" {
                require.EqualError(t, err, tt.wantErr)
                return
            }
            require.NoError(t, err)
            assert.Equal(t, tt.wantOpts.Title, gotOpts.Title)
            assert.Equal(t, tt.wantOpts.Body, gotOpts.Body)
        })
    }
}
```

## Layer 2: run function with stubs

Assert the **same fixture** in both TTY modes — this is where most CLI regressions hide.

```go
func TestListRun(t *testing.T) {
    tests := []struct {
        name       string
        tty        bool
        wantStdout string
        wantStderr string
    }{
        {name: "tty", tty: true,
            wantStdout: "\nShowing 1 of 1 items\n\nID  TITLE  UPDATED\n#1  First  about 1 hour ago\n"},
        {name: "piped", tty: false,
            wantStdout: "1\tFirst\tOPEN\t2024-01-01T10:00:00Z\n"},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            reg := &httpmock.Registry{}
            defer reg.Verify(t) // fails if a stub was never used
            reg.Register(httpmock.REST("GET", "items"), httpmock.FileResponse("./fixtures/items.json"))

            ios, _, stdout, stderr := iostreams.Test()
            ios.SetStdoutTTY(tt.tty)
            err := listRun(&ListOptions{
                IO:         ios,
                HttpClient: func() (*http.Client, error) { return &http.Client{Transport: reg}, nil },
                Limit:      30,
                Now:        func() time.Time { return fixedNow },
            })
            require.NoError(t, err)
            assert.Equal(t, tt.wantStdout, stdout.String())
            assert.Equal(t, tt.wantStderr, stderr.String())
        })
    }
}
```

Also test: `--json` output (field selection, `--jq`), empty results (`NoResultsError` on TTY,
`[]` with `--json`), error paths (HTTP 404/401 → message + correct error type), `--web`
(`browser.Stub` records `BrowsedURL()`).

## HTTP stubs

gh `pkg/httpmock`: a `Registry` implementing `http.RoundTripper`.
- Matchers: `REST(method, path)`, `GraphQL(regex)`, `QueryMatcher(method, path, url.Values)`, `WithHost`.
- Responders: `StringResponse`, `JSONResponse(v)`, `FileResponse(path)`, `StatusStringResponse(code, body)`, `JSONErrorResponse`, `RESTPayload(code, body, func(payload map[string]any){ assert... })`, `GraphQLQuery(body, func(query string, vars map[string]any){ assert... })`.
- Each stub matches once; unmatched request → error "no registered HTTP stubs matched"; `defer reg.Verify(t)` → fails on unused stubs; `reg.Exclude(t, matcher)` asserts a call must **not** happen.
- Assert request bodies/variables in the responder callback, not just responses.

If you don't vendor gh's package, a 100-line equivalent is worth writing; the self-verification is the point.

## Subprocess stubs

gh `internal/run`: all `exec.Cmd` go through `run.PrepareCmd`. In tests:
```go
cs, teardown := run.Stub()
defer teardown(t) // fails for stubs that never ran
cs.Register(`git rev-parse --verify refs/heads/feature`, 1, "")
cs.Register(`git checkout -b feature --track origin/feature`, 0, "")
```
Unstubbed commands **panic** with the command line — no accidental real `git push` from a test.
Docker-style alternative: inject an interface (`Runner`) with function fields.

## Prompter mocks

- Generated: `//go:generate moq -rm -out prompter_mock.go . Prompter` → `&prompter.PrompterMock{ConfirmFunc: func(p string, d bool) (bool, error) { ... }}`.
- Ordered, self-verifying: `pm := prompter.NewMockPrompter(t)` (registers `t.Cleanup(pm.Verify)`), then
  `pm.RegisterSelect("Select a repo", []string{"a","b"}, func(_, _ string, opts []string) (int, error) { return prompter.IndexFor(opts, "b") })`.
- Assert the prompt text — it's user-facing UX.

## Golden files

Docker (`gotest.tools/v3/golden`): for wide or multi-line output (tables, help, formatted views).
```go
golden.Assert(t, cli.OutBuffer().String(), "container-list-with-format.golden")
// regenerate: go test ./... -update
```
Review golden diffs like code. Prefer inline expected strings for short output (gh style),
golden files for long output (help text, large tables).

Docker's fake API client pattern (when your API client is an interface):
```go
type fakeClient struct {
    client.Client                       // embed to satisfy the interface
    listFunc func(ctx context.Context, opts ListOptions) ([]Item, error)
}
func (f *fakeClient) List(ctx context.Context, o ListOptions) ([]Item, error) {
    if f.listFunc != nil { return f.listFunc(ctx, o) }
    return nil, nil
}
```

## Acceptance tests with testscript

gh `acceptance/` uses `github.com/rogpeppe/go-internal/testscript` (gh uses a fork):
```go
//go:build acceptance

package acceptance

func TestMain(m *testing.M) {
    os.Exit(testscript.RunMain(m, map[string]func() int{
        "tool": func() int { return int(app.Main()) }, // in-process binary
    }))
}
func TestItems(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/items",
        Setup: func(env *testscript.Env) error {
            env.Setenv("TOOL_CONFIG_DIR", env.WorkDir+"/config") // sandbox
            env.Setenv("HOME", env.WorkDir)
            return nil
        },
        RequireExplicitExec: true,
    })
}
```
```
# testdata/items/create-and-list.txtar
exec tool item create --title 'hello'
stdout '^https://example.com/items/\d+$'
exec tool item list --json title --jq '.[].title'
stdout '^hello$'
! exec tool item create
stderr 'must provide `--title`'
```
Custom commands make scripts readable (`defer` cleanup, `stdout2env VAR`, `replace`).
Run separately (`make acceptance`, build tag) since they may hit real services.
For tools with no remote service, run these in normal CI — they're fast.

Docker e2e: real binary via `gotest.tools/v3/icmd` with a temp `--config` dir per test:
`icmd.RunCommand("docker", "--config", dir, "foo").Assert(t, icmd.Expected{ExitCode: 1, Err: "..."})`.

## Hygiene

- Never touch the real home dir, keyring, network, or user config. Point config dir env to `t.TempDir()`; use `config.NewMockConfig()`-style in-memory configs (docker `configfile.New("")`).
- Inject time; use fixed `now` in fixtures.
- `t.Setenv` for env-dependent behavior (and not `t.Parallel()` in those tests).
- `go test -race ./...` in CI, on Linux, macOS **and** Windows (paths, line endings, terminals differ).
- Test the error **message** for user-facing errors; scripts and users rely on them.
- Exit-code mapping and `printError` get their own table tests (gh `internal/ghcmd/cmd_test.go`).
- Signal handling: docker sends a real `syscall.Kill(os.Getpid(), syscall.SIGINT)` in prompt tests to verify cancellation.
