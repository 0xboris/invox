# Layers and package design

`architecture.md` covers the top of the stack: entrypoint, Factory, command anatomy.
This file covers what sits **under and beside the commands**. That is where most of a
CLI's code lives, and it decides whether the tool stays maintainable.

Sources: cli/cli @ fc4b137 and docker/cli @ 48988f5; every path below was checked
against them. Neither repo enforces its layering with a linter (no depguard rules for
import direction), and both leak in places. The leaks are listed so you don't copy
them. For a new tool, enforce the rules instead (§Enforcing the layering).

## Contents
- [The layer stack](#the-layer-stack)
- [Domain / client layer](#domain--client-layer)
- [Typed models, value types and output shapes](#typed-models-value-types-and-output-shapes)
- [Adapters for external programs](#adapters-for-external-programs)
- [Noun-level shared packages](#noun-level-shared-packages)
- [Presentation beside the domain](#presentation-beside-the-domain)
- [Typed flag values](#typed-flag-values)
- [Interfaces: where and how big](#interfaces-where-and-how-big)
- [Config as a dependency](#config-as-a-dependency)
- [Errors across layers](#errors-across-layers)
- [Package and file size](#package-and-file-size)
- [Tools without a remote API](#tools-without-a-remote-api)
- [Anti-patterns](#anti-patterns)
- [Enforcing the layering](#enforcing-the-layering)

---

## The layer stack

```
cmd/tool/main.go               shim: os.Exit(int(app.Main()))
internal/app                   composition root: config load, IOStreams, Factory wiring,
                               signals, error → exit code                  (gh internal/ghcmd)
pkg/cmd/root, pkg/cmd/<noun>   command tree and registration
pkg/cmd/<noun>/<verb>          COMMAND: flags → Options → run fn → presentation
pkg/cmd/<noun>/shared          logic several verbs share: finders, listers, display
                               helpers, flags→request mappers
──────── below this line: no cobra, no IOStreams/ColorScheme, no os.Std*, no os.Exit ────────
domain / client                typed models, operations, field lists, typed errors
                               (gh api/; docker: the moby client SDK)
adapters                       external programs and OS services behind a small type
                               (gh git/, internal/browser, internal/keyring;
                                docker cli/config/credentials with helper binaries
                                behind an injected ProgramFunc)
value types                    identities parsed once, compared safely (gh internal/ghrepo)
config contract                interface in a leaf package (gh internal/gh), impl elsewhere

beside, used by the command layer only:
pkg/iostreams                  streams, TTY, color, pager, spinner    (docker cli/streams)
presentation                   table printer, text/time helpers, JSON exporter
                               (gh internal/tableprinter, internal/text, cmdutil/json_flags.go;
                                docker cli/command/formatter, templates/)
flag value types               pflag.Value implementations with validation (docker opts/)
```

**Imports point down only.** Verified in gh:
- `api` imports only `internal/gh`, `internal/ghrepo`, `internal/safeurl`, `pkg/set` and `utils`.
- `git` imports only `internal/ghinstance` and `internal/run`.
- Nothing in `api`, `git`, `internal/config` or `internal/ghrepo` imports `pkg/cmd`, `iostreams`, `cobra` or `cmdutil`.

Verified in docker:
- `cli/config` doesn't import `cli/command`.
- `cli/command/formatter` and `opts` import no cobra or pflag.
- `cli-plugins/manager` takes only `config.Provider`, never `command.Cli`.

| Layer | Knows about | Never |
|---|---|---|
| Command | cobra, Options, Factory, IOStreams, domain, presentation | business rules beyond orchestration |
| Shared (per noun) | domain, Factory, narrow interfaces | other leaf command packages |
| Domain / client | its models, transport, value types, ctx | cobra, IOStreams, color, `os.Std*`, `os.Exit`, env, global config, `time.Now` |
| Adapters | `exec`, OS APIs, injected writers | prompts, color, exit codes |
| Presentation | IOStreams/writer, ColorScheme, `now`, models | fetching data, mutating state |

The payoff: the domain layer can be reused (by other commands, aliases, an MCP
server, a TUI) and tested without a CLI. Commands become short and uniform:
1. resolve dependencies;
2. call one or two domain operations;
3. render or export.

## Domain / client layer

### gh: `api/`
- **A thin transport client.** `type Client struct{ http *http.Client }` built with
  `NewClientFromHTTP(httpClient)` (`api/client.go:30-41`). Auth, caching and header
  extraction are `http.RoundTripper` decorators in `api/http_client.go`
  (`AddAuthTokenHeader`, `NewCachedHTTPClient`), so the client knows nothing about tokens.
- **Operations are free functions,** grouped one file per domain: `queries_issue.go`,
  `queries_pr.go`, `queries_repo.go`. For example, `func FetchRepository(client *Client,
  repo ghrepo.Interface, fields []string) (*Repository, error)` (`queries_repo.go:286`).
- **Typed models.** `api.Issue`, `api.PullRequest` and `api.Repository` are plain
  structs whose fields mirror the wire names.
- **Field lists are data.** `api.IssueFields` (`query_builder.go:351`) is both the
  `--json` whitelist (`cmdutil.AddJSONFlags(cmd, &opts.Exporter, api.IssueFields)`)
  and the input to `IssueGraphQL(fields)`, which builds the query selection. Asking
  for fewer fields fetches less.
- **Typed errors with remediation data.** `api.HTTPError{*ghAPI.HTTPError;
  scopesSuggestion}` and `api.GraphQLError` with `Match(type, path)`. Callers use
  `errors.As` (`pkg/cmd/cache/delete/delete.go:176`), and the top level turns a 401
  into "Try authenticating with: gh auth login" (`internal/ghcmd/cmd.go:222-233`).
- **A consumer-side interface instead of a config import:**
  ```go
  // api/http_client.go:17
  type config interface {
      ActiveToken(string) (string, string)
      HostForAPIHost(string) (string, bool)
  }
  ```

### docker: the moby client SDK
The domain client is a separate module. Every call has the shape
`Method(ctx, …, XOptions) (XResult, error)`, for example
`ContainerList(ctx, client.ContainerListOptions) (client.ContainerListResult, error)`.
`APIClient` is composed from per-domain sub-interfaces (`ContainerAPIClient`,
`ImageAPIClient`, …). The command layer never builds an HTTP request to the daemon.

### Rules (both repos' good parts, minus their leaks)
1. **`ctx context.Context` first on every operation** that does I/O. The moby SDK
   does this on every call (docker's own `http.Get` for remote build contexts doesn't,
   `image/build/context.go:237`). gh's `api` mostly doesn't (only `QueryWithContext`/`RequestWithContext`
   do), so Ctrl-C can't cancel most gh API calls. Don't copy that.
2. **One params struct per operation** once it takes more than two or three inputs
   (docker `XOptions`). Never use a run of same-typed positional strings
   (`Create(src, out, customers, issuer, id string, …)`): a swapped argument compiles.
3. **Identity values are value types** (`ghrepo.Interface`), not bare strings.
4. **Return typed models and results; never print.** For progress, accept a two-method
   interface, as gh's finder does without depending on IOStreams:
   ```go
   // pkg/cmd/pr/shared/finder.go:33
   type progressIndicator interface {
       StartProgressIndicator()
       StopProgressIndicator()
   }
   ```
5. **Typed errors that wrap the cause** (`Unwrap`), sentinels for states
   (`git.ErrNotOnAnyBranch`), or marker interfaces (docker classifies with
   `errdefs.IsNotFound`). Domain messages state facts. The command layer adds what
   the user should do.
6. **No ambient state.** Env vars, config files, the clock and the working directory
   come in through constructor fields or parameters. The leaks to avoid: gh's `api`
   reads `utils.IsDebugEnabled()`, and docker's `configfile` prints to `os.Stderr`
   (`file.go:334`).
7. **Command-specific queries live next to the command,** not in the global client.
   gh has 28 `pkg/cmd/*/*/http.go` files; `issue/list/http.go` builds its query with
   `api.IssueGraphQL(filters.Fields)`. gh's docs: "Any logic specific to this command
   should be kept within the command's package and not added to any 'global'
   packages like `api` or `utils`" (`docs/project-layout.md:68`).

## Typed models, value types and output shapes

- **Parse at the boundary, then work on types.** Decode a file, response or flag into
  a struct once, validate it, and pass the struct on. Domain data held as
  `map[string]any` hides the schema. It also spreads validation across every reader
  and makes `--json` field lists impossible to declare.
- **User-edited files:** decode strictly so typos fail loudly (yaml.v3
  `Decoder.KnownFields(true)`, `json.Decoder.DisallowUnknownFields()`). Accept
  deliberate aliases explicitly, not by probing maps. (This is this skill's
  recommendation, not a gh/docker pattern: their configs are mostly tool-written.)
- **Value types for identities.** gh's `internal/ghrepo` (121 lines, one internal import):
  ```go
  type Interface interface { RepoName() string; RepoOwner() string; RepoHost() string }
  func New(owner, repo string) Interface
  func FromFullName(nwo string) (Interface, error)   // the only way in from user input
  func FromURL(u *url.URL) (Interface, error)
  func IsSame(a, b Interface) bool                  // compare with this, never ==
  ```
  The value is immutable, normalized at construction (`normalizeHostname`), and
  satisfied by `context.Remote` and `api.Repository` too. 180 non-test files take it.
- **Three shapes, kept apart:** the domain model, the wire or file format, and the
  output (`--json`) shape.
  - gh uses one struct for model and wire, but the model decides the export shape
    via `ExportData(fields []string) map[string]any` (`api/export_pr.go:8`). It
    flattens connection wrappers explicitly and falls back to reflection.
  - Docker derives JSON from the formatter's accessor methods
    (`formatter/reflect.go:16`), so `--format json` emits what the table shows, not
    the raw API struct.
  - Either way, `--json` field names are public API (`io-and-output.md`).

## Adapters for external programs

gh's `git.Client` is the template for wrapping any external program: a compiler, a
renderer, a mail client, `ssh` or `kubectl`.

```go
// git/client.go:53
type Client struct {
    GhPath, RepoDir, GitPath string     // GitPath resolved lazily (under mu) via safe LookPath
    Stderr io.Writer                    // the factory sets these to IOStreams writers
    Stdin  io.Reader
    Stdout io.Writer

    commandContext commandCtx           // unexported seam: func(ctx, name, args...) *exec.Cmd
    mu             sync.Mutex
}
func (c *Client) Command(ctx context.Context, args ...string) (*Command, error)
func (c *Client) Fetch(ctx context.Context, remote, refspec string, mods ...CommandModifier) error

type CommandModifier func(*Command)                       // WithStdout, WithStderr, WithRepoDir
type GitError struct { ExitCode int; Stderr string; err error } // Error() includes Stderr; Unwrap
type NotInstalled struct { message string; err error }
```

Rules taken from it:
- **One type per external program, with a method per operation.** No `exec.Command`
  outside adapters.
- **Every method takes `ctx` first** (all `*git.Client` methods except `Copy`). Use
  `exec.CommandContext` so Ctrl-C reaches the child.
- **Resolve the binary lazily.** A missing tool returns a typed `NotInstalled`-style
  error, and the command turns it into install instructions for the current OS.
- **Streams are injected, never `os.Std*`.** Wire the child's stdout to `ErrOut`
  unless the child's output *is* the data the user asked for. Compiler chatter on
  stdout breaks `tool build x | …`.
- **Failures carry exit code and stderr** (`GitError`).
- **Opaque types for dangerous parameters.** `git.CredentialPattern` is a struct, so a
  bare string can't be passed by accident.
- **The test seam is an injected field.** gh's other seam, the package-global
  `run.PrepareCmd`, is described in gh's own comment as "a hack in order to not break
  the hundreds of existing tests". Don't copy it.

Other side effects follow the same idea as small interfaces:
- `browser.Browser{ Browse(string) error }` (`internal/browser/browser.go`)
- `prompter.Prompter` (see `ux-and-help.md`)
- keyring calls with timeouts (`config-and-auth.md`)
- docker's `credentials.Store{Erase, Get, GetAll, Store}`, with file and native
  (helper-binary) implementations chosen in one place
  (`configfile/file.go:320 GetCredentialsStore`)

## Noun-level shared packages

When several verbs of a noun resolve, list or display the same thing, put it in
`pkg/cmd/<noun>/shared`. gh has 18 of these. The main example is gh's PR finder,
used by 14 `pr` verbs:

```go
// pkg/cmd/pr/shared/finder.go
type PRFinder interface {
    Find(opts FindOptions) (*api.PullRequest, ghrepo.Interface, error)
}
type FindOptions struct {
    Selector string   // number (#123), branch ([owner:]branch) or PR URL
    Fields   []string // GraphQL fields; callers pass opts.Exporter.Fields() for --json
    // BaseBranch, States, DisableProgress, Detector …
}
type GitConfigClient interface { // narrow, consumer-side view of *git.Client
    ReadBranchConfig(ctx context.Context, branch string) (git.BranchConfig, error)
    PushDefault(ctx context.Context) (git.PushDefault, error)
    // …
}
func NewFinder(f *cmdutil.Factory) PRFinder
func NewMockFinder(selector string, pr *api.PullRequest, repo ghrepo.Interface) *mockFinder
```

A command holds `Finder shared.PRFinder` in its Options and sets it **inside `RunE`**
(`opts.Finder = shared.NewFinder(f)`, `pr/view/view.go:65`). Tests inject
`NewMockFinder(...)`.

Docker's equivalents:
- **A pure flags → request mapper:** `buildContainerListOptions(*psOptions)
  (client.ContainerListOptions, error)` (`container/list.go:71`), unit-tested on its
  own.
- **A shared flag set:** `addFlags(*pflag.FlagSet) *containerOptions` and
  `parse(...) (*containerConfig, error)`, used by both `run` and `create`
  (`container/opts.go`).

Avoid gh's own leaks:
- **A package-global override** of the finder for "run-command-style" tests
  (`finder.go:80`: "This is a bad pattern").
- **Production files importing `testing`.** Mocks belong in `_test.go` or a separate
  `…test` package.
- **Packages importing another command's package.** Examples:
  `extension/browse` → `repo/view`, `auth/shared` → `ssh-key/add`,
  `repo/autolink/delete` → `repo/autolink/view`, and `pr` mounting `issue/lock`. Move
  such code to `shared` or the domain. (`scripts/cli_audit.sh` reports exactly these four.)
- **Cross-noun reuse of a shared package.** `issue/*` imports `pr/shared`. Once two
  nouns need the same code, it belongs in the domain layer.

## Presentation beside the domain

Rendering is a separate concern with its own small helpers. Domain code never
formats for humans, beyond `String()` on identities.

**gh:**
- `internal/tableprinter` holds the TTY decisions, e.g. `AddTimeField(now, t, color)`:
  fuzzy time on a TTY, RFC3339 when piped.
- `pkg/cmd/<noun>/shared/display.go` helpers take what they need as parameters
  (`StateTitleWithColor(cs *iostreams.ColorScheme, pr api.PullRequest)`).
- `internal/text` holds pure string and time helpers.
- The `cmdutil.Exporter` interface (`Fields()`, `Write(io, data)`) writes JSON, and
  the model's `ExportData` decides the shape. The model owns the shape; the exporter
  owns the format.

**Docker:** `cli/command/formatter` is a cobra-free package. Adding a resource takes
four pieces (smallest complete example: `formatter/context.go`):
1. constants for the default table and quiet formats, plus header strings;
2. `NewXFormat(source string, quiet bool) Format`;
3. `XWrite(ctx formatter.Context, items []T) error`, which calls `ctx.Write(newXContext(), render)`;
4. `type xContext struct{ HeaderContext; v T }`, with one exported method per
   template field and `MarshalJSON() { return MarshalJSON(c) }`.

Headers come from running the same template against a header map, so custom
`--format "table {{.ID}}"` gets correct headers for free.

**Rule:** presentation functions take a writer (or IOStreams), the TTY state or
ColorScheme, `now`, and the typed data. They are pure and table-testable
(`formatter/volume_test.go`). Keep the output contract from `io-and-output.md`
(gh's TSV and `--json` over docker's templates).

## Typed flag values

Validate structured flag input **at parse time**, inside a `pflag.Value`. Docker's
`opts/` package does this without importing cobra or pflag (it satisfies the interface
structurally):

```go
// opts/opts.go:28 (abridged)
type ValidatorFctType func(val string) (string, error) // validates and normalizes
type ListOpts struct { values *[]string; validator ValidatorFctType }
func (o *ListOpts) Set(v string) error {
    if o.validator != nil {
        var err error
        if v, err = o.validator(v); err != nil { return err }
    }
    *o.values = append(*o.values, v)
    return nil
}
func (o *ListOpts) Type() string { return "list" }

// binding (container/opts.go): flags.VarP(&copts.env, "env", "e", "Set environment variables")
```

Other examples:
- `MapOpts` for `key=value`.
- `FilterOpt` for `name=value` filters that return typed `client.Filters`.
- `MemBytes` for sizes.
- `DurationOpt`, which is pointer-backed so "unset" differs from `0`.
- `PositiveDurationOpt`, which embeds `DurationOpt` and adds a check in `Set`.

Use gh's `StringEnumFlag` helpers for enums (`ux-and-help.md` §Flags) and `opts`-style
types for anything with structure.

## Interfaces: where and how big

- **Define interfaces at the consumer, small.** Examples:
  - gh's `GitConfigClient` (4 methods over `*git.Client`), `api`'s private 2-method
    `config`, and `progressIndicator`;
  - docker's `resizeClient interface{ client.ExecAPIClient; client.ContainerAPIClient }`
    (`container/tty.go:21`);
  - docker's `APIClientProvider interface{ Client() client.APIClient }`
    (`completion/functions.go:19`), whose comment explains it lets completion
    "postpone initializing the client until it's used".
- **Helpers take the narrowest dependency.** Docker passes `command.Streams` or
  `config.Provider` instead of the whole `command.Cli`. Examples:
  `PromptUserForCredentials(ctx, cli Streams, …)` and `PruneFilters(dockerCLI
  config.Provider, …)`.
- **Small interfaces break import cycles.** Docker's `credentials` package defines its
  own `store` interface instead of importing `configfile`.
- **Producer-side interfaces only for a whole contract that many consumers mock.**
  Examples: gh's `gh.Config` and `prompter.Prompter` (moq-generated via `go:generate`),
  and docker's `APIClient`.
- **Concrete types on the Factory are fine** when the type has its own injected seam
  (gh: `GitClient *git.Client`).

Pick the test seam deliberately:

| Dependency | Seam | Example |
|---|---|---|
| HTTP API you own the client for | stub the transport, keep the real client | gh `httpmock.Registry` (unused stub fails) |
| SDK or large client interface | fake implementing the interface with func fields | docker `fakeClient` |
| External program | injected command factory or stub runner | gh `git.Client.commandContext`, `run.Stub` |
| Files | real files under `t.TempDir()` + golden output; in-memory objects where the format isn't under test | docker `configfile.New("")` ("so that tests don't create configfiles") |
| Prompts, browser, clock | interface or func field | `PrompterMock`, `browser.Stub`, `Now func() time.Time` |

Docker's fakes embed the concrete `client.Client` struct, so a forgotten stub silently
calls a zero-value client. Embed the **interface** instead (then a missing stub
panics with a nil method), or generate a mock that fails on unexpected calls.

## Config as a dependency

- **Interface in a leaf package, implementation elsewhere.** gh's `internal/gh/gh.go`
  says so in its package doc. `gh.Config` returns
  `ConfigEntry{Value, Source}` (default vs user) and is mocked with moq.
  `internal/config` implements it.
- **Loaded once in `Main`.** A failure is a warning:
  `cfg, cfgErr := config.NewConfig(); cfgFunc := func() (gh.Config, error) { return
  cfg, cfgErr }` (`internal/ghcmd/cmd.go:57-60`). Commands get it through
  `opts.Config()`.
  - gh's root command setup still calls `f.Config()` and fails
    (`pkg/cmd/root/root.go:66`, and the comment in `pkg/cmdutil/factory.go` admits
    it). Do better: nothing at root-build time may require config.
- **The domain receives values, not the config object.** That means resolved paths,
  hostnames and settings, or at most a consumer-side two-method interface (gh `api`'s
  `config`). A domain function that calls `os.Getenv` or re-reads the config file
  breaks precedence (flag > env > file) and tests.
- **Versioned migrations.** `Migration{PreVersion(); PostVersion(); Do(cfg) error}`,
  applied by `cfg.Migrate` (`internal/config/config.go:190`). See
  `config-and-auth.md` §Schema evolution.

## Errors across layers

| Layer | Does |
|---|---|
| Adapter / domain | returns typed errors that wrap the cause (`GitError`, `HTTPError`, `NotFoundError{ID}`); no remediation text, no printing |
| Shared / command | translates: user-input problems → `FlagError`; known domain errors → actionable messages ("run `tool auth login`"); declined prompt → `CancelError`; empty result → `NoResultsError` |
| `Main` | maps to exit codes and prints once (`architecture.md` §Errors) |

Docker's `Client()` calls `os.Exit(1)` when initialization fails
(`cli/command/cli.go:98`). That skips deferred cleanup and the exit-code mapping. Lazy
getters return `(T, error)`, as gh's Factory does.

**Multi-target commands** (docker `rm a b c`, `container/rm.go:72-99`):
- `parallelOperation` runs up to 50 at once but returns results **in input order**.
- Each success prints its name on stdout.
- Failures are collected and returned as `errors.Join(errs...)`.
- With `--force`, not-found is skipped.

Copy this; docker itself is inconsistent here (`start.go` prints each error and
returns a summary string instead).

## Package and file size

- **Measured in gh:** `pkg/cmd` packages have a median of 173 non-test lines (p90 659)
  in about 1.6 files, with about 1.65 lines of test per line of code. A leaf is
  `<verb>.go` + `<verb>_test.go`, plus `http.go` when it has its own queries
  (`docs/command-development.md`).
- **Large files exist** (`api/queries_repo.go` 1765, `pr/create/create.go` 1408). Size
  alone isn't the problem. `queries_repo.go`, which mixes repos, labels, milestones and
  projects, is the least cohesive file in `api`.
- **Split by concept, not by line count.** One file per domain concept
  (`queries_issue.go`, `queries_pr.go`, `export_pr.go`, `http_client.go`). A
  `service.go`, `utils.go`, `helpers.go` or `common.go` that keeps growing is the
  smell. gh's own `utils` package is down to one function.

## Tools without a remote API

Most of the above was built around an HTTP API, but the shapes carry over to tools
that work on local files and run local programs:

| gh / docker | Local-files tool |
|---|---|
| `api.Client` over `*http.Client` | `Store` rooted at a directory (or `fs.FS` for reads) |
| `httpmock.Registry` | fixture trees under `t.TempDir()`, golden files |
| `ghrepo.Interface` | identity types (`DocumentID`, archive-relative path) parsed once |
| `api.IssueFields` + `ExportData` | the same, for `--json` |
| `git.Client` | adapter per external program (renderer, compiler, mail client, opener) |
| `gh.Config` interface + loaded-once config | the same; domain gets resolved paths, not the config |
| `queries_*.go` per domain | one file per concept: `store.go`, `numbering.go`, `render.go` |

```go
// internal/doc/doc.go — domain: no cobra, no iostreams, no os.Getenv, no time.Now
type Document struct {
    ID       DocumentID
    Customer CustomerID
    Issued   time.Time
    Lines    []Line
    Status   Status // typed enum with String() and ParseStatus
}
var Fields = []string{"id", "customer", "issued", "status", "total"}
func (d Document) ExportData(fields []string) map[string]any

type Store struct{ Root string } // the composition root resolves Root; the domain never looks it up
func (s Store) Load(ctx context.Context, id DocumentID) (*Document, error)       // NotFoundError{ID}
func (s Store) Save(ctx context.Context, d *Document) error                      // atomic write
func (s Store) List(ctx context.Context, f ListFilter) ([]Document, error)

// internal/doc/render.go — pure
func Render(w io.Writer, tmpl *Template, d *Document, now time.Time) error

// internal/compiler/compiler.go — adapter, git.Client-shaped
type Compiler struct { Path string; Stdout, Stderr io.Writer; commandContext commandCtx }
func (c *Compiler) Build(ctx context.Context, dir, file string) error // NotInstalledError, *ExitError{Code, Stderr}
```

The command's run function then looks the same as gh's `issueList`: load → operate →
`Exporter.Write` or render.

## Anti-patterns

Found in hand-rolled CLIs; each one has a gh/docker replacement:

| Anti-pattern | Why it hurts | Replace with |
|---|---|---|
| A command-spec table of booleans (`NeedsX`, `SupportsY`) driving one generic parser | Every flag is restated in the parser, help, root help, completion and arg reordering, and they drift | Options + `NewCmdX` per command; help and completion generated from flag definitions |
| One Options type shared by all commands, defined in the domain package | CLI concerns (`--edit`, `--archive`) leak into the domain; every command sees every field | Per-command Options in the command package; domain params structs |
| Parse helpers that print and return exit codes | Usage printed for runtime errors; can't test parsing alone | Return `FlagError`/typed errors; one mapping in `Main` |
| Package-level `var doX = defaultDoX` swapped by tests | Hidden global state, tests can't run in parallel (gh calls its own a "hack") | Injected Options/Factory fields |
| Domain code wiring a child process to `os.Stdout` | Diagnostics land in the data stream | Adapter with injected writers, stdout → `ErrOut` |
| Domain functions re-reading env or config | Precedence breaks, tests need env juggling | Load once; pass resolved values |
| `map[string]any` domain models | Schema hidden; validation scattered; no `--json` field list | Typed structs, strict decoding |
| Runs of same-typed positional params | Swapped arguments compile | Params struct, value types |
| One `service.go` holding unrelated concepts | Unreviewable diffs, merge conflicts, hidden coupling | One file or package per concept |
| A leaf command importing another leaf | Cycles and accidental API between commands | `shared/` or the domain |

## Enforcing the layering

gh and docker rely on convention, and both leak (gh `context` → `iostreams`; docker
`configfile` → `os.Stderr`). A new tool should enforce the rules mechanically from day
one. golangci-lint v2 (adjust module paths and globs; `$test` excludes test files from depguard):

```yaml
# .golangci.yml
version: "2"
linters:
  enable: [depguard, forbidigo]
  settings:
    depguard:
      rules:
        below-commands:
          files: ["**/internal/doc/**", "**/internal/compiler/**", "!$test"]
          deny:
            - pkg: github.com/spf13/cobra
              desc: domain and adapters must not know about commands
            - pkg: github.com/spf13/pflag
              desc: domain and adapters must not know about flags
            - pkg: example.com/tool/pkg/iostreams
              desc: return data; the command renders it
            - pkg: example.com/tool/pkg/cmd
              desc: imports point down only
    forbidigo:
      forbid:
        - pattern: ^(fmt\.Print(f|ln)?|print|println)$
          msg: write to an injected io.Writer (IOStreams)
        - pattern: ^os\.(Stdout|Stderr|Stdin|Exit)$
          msg: use IOStreams; return errors to Main
  exclusions:
    rules:
      # Anchor the paths: an unanchored "cmd/" also matches pkg/cmd/ and silently
      # exempts every command (go-cli-starter v0.2.0 shipped exactly that bug).
      - path: ^(cmd/|internal/app/|pkg/iostreams/)|_test\.go$
        linters: [forbidigo]
```
Checked with golangci-lint 2.14 on go-cli-starter v0.3.0 (globs pointed at its
`internal/api` and `internal/browser`, only these two linters on): no findings as shipped. Planted
violations (`fmt.Println`, `os.Stdout`, `os.Exit` and a cobra import in
`internal/api`; `fmt.Println` in `pkg/cmd/item/list`) were all reported, and the same
calls in `internal/app` and `cmd/` were not. When you test a rule, plant the call
in a function something uses, on its own line. golangci-lint drops issues in unused
code and reports one issue per line.

Also fail `go test` (or CI) if a domain package depends on the command layer, even
transitively. go-cli-starter's `internal/archtest` runs `go list -deps` per rule; the
one-line version is `! go list -deps ./internal/doc/... | grep -E 'spf13/cobra|/pkg/cmd'`.
Run `bash scripts/cli_audit.sh` for the remaining heuristics.
