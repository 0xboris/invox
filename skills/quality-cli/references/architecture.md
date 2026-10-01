# Architecture

How gh (`cli/cli`) and docker (`docker/cli`) structure a Go CLI. Paths in
parentheses point at the upstream source for deeper reading.

## Contents
- [Entrypoint](#entrypoint)
- [Errors and exit codes](#errors-and-exit-codes)
- [The Factory (dependency injection)](#the-factory-dependency-injection)
- [Command anatomy](#command-anatomy)
- [Root command setup](#root-command-setup)
- [Command registration and grouping](#command-registration-and-grouping)
- [Signals and cancellation](#signals-and-cancellation)
- [Subprocesses](#subprocesses)
- [Cobra gotchas](#cobra-gotchas)
- [Migrating a hand-rolled CLI](#migrating-a-hand-rolled-cli)

---

## Entrypoint

```go
// cmd/tool/main.go — nothing else lives here
func main() { os.Exit(int(app.Main())) }
```

`app.Main()` (gh: `internal/ghcmd/cmd.go`) does, in order:
1. Build `IOStreams` from the environment and apply env/config overrides (pager, prompt disabled, spinner disabled).
2. Build the Factory (`factory.New(buildVersion)`).
3. Start the async update check (see `build-and-release.md`).
4. Build the root command, set up signal-aware context.
5. `cmd, err := rootCmd.ExecuteContextC(ctx)` → map `err` to an exit code (below).
6. Cancel and drain the update check; print a notice to stderr if any.
7. Return the code. Deferred flushes (telemetry) still run because nothing called `os.Exit`.

Small, important details:
- A failure to load config is a **warning**, not fatal — commands that need config fail lazily with context.
- `cobra.MousetrapHelpText = ""` lets the binary run from Windows Explorer.
- Resolve your own executable path (`os.Executable`, follow Homebrew symlinks) if the tool will re-invoke itself (credential helper, extensions).

## Errors and exit codes

Commands return values; one place decides what to print and which code to return.

```go
// pkg/cmdutil/errors.go
var SilentError = errors.New("SilentError")       // exit 1, already reported
var PendingError = errors.New("PendingError")     // exit 8, nothing failed but not done
type CancelError struct{ error }                   // exit 2
type FlagError struct{ err error }                 // usage error → print usage
func FlagErrorf(format string, a ...any) error { return &FlagError{fmt.Errorf(format, a...)} }
type NoResultsError struct{ message string }       // exit 0, message only on TTY
```

```go
// app.Main — error mapping (gh)
switch {
case err == nil:                                  return exitOK
case errors.Is(err, cmdutil.SilentError):         return exitError
case errors.Is(err, cmdutil.PendingError):        return exitPending
case cmdutil.IsUserCancellation(err):
    if errors.Is(err, terminal.InterruptErr) { fmt.Fprint(stderr, "\n") } // keep shell prompt on its own line
    return exitCancel
case errors.As(err, &authErr):                    return exitAuth
case errors.As(err, &pagerErr):                   return exitOK   // user quit `less`
case errors.As(err, &noResults):
    if io.IsStdoutTTY() { fmt.Fprintln(stderr, noResults.Error()) }
    return exitOK                                                 // empty ≠ failure
case errors.As(err, &extErr):                     return exitCode(extErr.ExitCode()) // pass through
}
printError(stderr, err, cmd, debug)
return exitError
```

`printError`:
- `FlagError` or unknown command → error, blank line, `cmd.UsageString()`.
- DNS/connection errors → friendly "error connecting to HOST / check your internet connection"; raw error only with `TOOL_DEBUG`.
- HTTP 401 → "Try authenticating with: tool auth login".
- If an AI agent is detected (env-based, gh `internal/agents/detect.go`), print the **full help** to stderr on a flag error so the agent can self-correct in one round trip. Keep it on stderr so one failure isn't split across streams.

**Exit codes (gh set — prefer this):** `0` ok · `1` error · `2` cancelled · `4` auth required · `8` pending. Document them in a `help exit-codes` topic. Pass through exit codes of extensions/aliases/subprocesses unchanged.
Docker additionally uses `125` for usage errors, `126/127` for "cannot exec / not found", and `128+signal` on signal termination — adopt `130` for SIGINT if your tool runs long operations; otherwise gh's `2` is fine.

Never panic for user errors. Panics are reserved for programmer errors ("unreachable state"). Never return `0` for an error.

## The Factory (dependency injection)

```go
// pkg/cmdutil/factory.go (gh)
type Factory struct {
    AppVersion     string
    ExecutablePath string
    IOStreams      *iostreams.IOStreams   // eager: cheap and always needed
    Prompter       prompter.Prompter
    Browser        browser.Browser

    // lazy: cost or can fail; only evaluated by commands that need them
    Config     func() (config.Config, error)
    HttpClient func() (*http.Client, error)
    BaseRepo   func() (ghrepo.Interface, error)
    Remotes    func() (context.Remotes, error)
}
```

Why lazy funcs:
1. `tool version` / `--help` must not fail on broken config or offline.
2. Errors surface inside the command that needs the resource, with context.
3. Fields can be **replaced after flag parsing**: gh's `-R/--repo` swaps `f.BaseRepo` in a `PersistentPreRunE` (`pkg/cmdutil/repo_override.go`). Therefore commands copy `opts.BaseRepo = f.BaseRepo` **inside `RunE`**, not in the constructor.
4. Tests replace any field with a closure.

Wiring lives in one file (`pkg/cmd/factory/default.go`) with dependency order documented:
```go
f.IOStreams  = ioStreams(f)            // depends on Config (pager, prompt settings)
f.HttpClient = httpClientFunc(f)       // depends on Config, IOStreams
f.Prompter   = newPrompter(f)          // depends on Config, IOStreams
```

Variations:
- Copy the Factory to give a subtree a different strategy (`repoResolvingFactory := *f; repoResolvingFactory.BaseRepo = smartBaseRepo(f)`).
- Keep separate HTTP clients when credentials must not leak (authenticated / unauthenticated / third-party host).
- Cache lazy values with `sync.Once` when they're used more than once (docker's `DockerCli.initialize()` with a 2s ping timeout).
- Docker's alternative is a `command.Cli` interface (`Out()`, `Err()`, `In()`, `Client()`, `ConfigFile()`) with functional options (`WithOutputStream`, `WithAPIClient`). Fine too — **gh's struct-of-funcs is preferred** because each command copies only what it uses into its Options.
- Helpers should accept the narrowest interface they need (docker passes `config.Provider` or `command.Streams`, not the whole CLI).

## Command anatomy

Every leaf command: **Options struct → constructor → run function**.

```go
type CreateOptions struct {
    // dependencies (copied from Factory)
    IO         *iostreams.IOStreams
    HttpClient func() (*http.Client, error)
    BaseRepo   func() (ghrepo.Interface, error)
    Prompter   prompter.Prompter
    Browser    browser.Browser
    // flags/args
    Title, Body string
    BodyFile    string
    WebMode     bool
    Interactive bool
}

func NewCmdCreate(f *cmdutil.Factory, runF func(*CreateOptions) error) *cobra.Command {
    opts := &CreateOptions{IO: f.IOStreams, HttpClient: f.HttpClient, Prompter: f.Prompter, Browser: f.Browser}
    cmd := &cobra.Command{
        Use:   "create",
        Short: "Create a new issue",
        Long:  heredoc.Doc(`...`),
        Example: heredoc.Doc(`
            $ tool issue create --title "I found a bug" --body "Nothing works"
            $ tool issue create --label "bug,help wanted"
        `),
        Args: cmdutil.NoArgsQuoteReminder,
        RunE: func(cmd *cobra.Command, args []string) error {
            opts.BaseRepo = f.BaseRepo // late-bound for -R override

            titleProvided := cmd.Flags().Changed("title")
            bodyProvided := cmd.Flags().Changed("body") || opts.BodyFile != ""
            if opts.BodyFile != "" {
                b, err := cmdutil.ReadFile(opts.BodyFile, opts.IO.In) // "-" means stdin
                if err != nil { return err }
                opts.Body = string(b)
            }
            opts.Interactive = !(titleProvided && bodyProvided)
            if opts.Interactive && !opts.IO.CanPrompt() {
                return cmdutil.FlagErrorf("must provide `--title` and `--body` when not running interactively")
            }
            if err := cmdutil.MutuallyExclusive("specify only one of `--body` or `--web`",
                bodyProvided, opts.WebMode); err != nil {
                return err
            }
            if runF != nil { return runF(opts) }
            return createRun(opts)
        },
    }
    cmd.Flags().StringVarP(&opts.Title, "title", "t", "", "Supply a title. Will prompt for one otherwise.")
    cmd.Flags().StringVarP(&opts.Body, "body", "b", "", "Supply a body. Will prompt for one otherwise.")
    cmd.Flags().StringVarP(&opts.BodyFile, "body-file", "F", "", "Read body text from `file` (use \"-\" to read from standard input)")
    cmd.Flags().BoolVarP(&opts.WebMode, "web", "w", false, "Open the browser to create an issue")
    return cmd
}

func createRun(opts *CreateOptions) (err error) {
    httpClient, err := opts.HttpClient()
    if err != nil { return err }
    // ... prompts if opts.Interactive, API call wrapped in progress indicator ...
    fmt.Fprintln(opts.IO.Out, created.URL)               // data → stdout
    return nil
}
```

Rules embedded above:
- All validation lives in `RunE` before `runF` so `TestNewCmdCreate` can cover it with zero I/O.
- `cmd.Flags().Changed("x")` distinguishes "explicitly set" from "default".
- `xRun` doesn't import cobra; it can be called from tests, aliases, or other commands.
- Inject time (`Now func() time.Time`) and anything nondeterministic.
- Partial failure: `defer func() { err = errors.Join(cleanupErr, err) }()`.
- Multi-target commands (docker `rm a b c`): process all, print each success, return `errors.Join(errs...)`.
- Preserve typed user input on failure (gh `PreserveInput` → temp file + `--recover` flag).

## Root command setup

```go
func NewCmdRoot(f *cmdutil.Factory, version, date string) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "tool <command> <subcommand> [flags]",
        Short: "Tool CLI",
        Long:  "Work seamlessly with X from the command line.",
        SilenceErrors: true, // Main prints errors
        SilenceUsage:  true, // usage only for FlagError
        Annotations: map[string]string{"versionInfo": versionFormat(version, date)},
    }
    cmd.SetFlagErrorFunc(func(c *cobra.Command, err error) error {
        if err == pflag.ErrHelp { return err }
        return cmdutil.FlagErrorWrap(err) // every pflag parse error shows usage
    })
    cmd.SetHelpFunc(rootHelpFunc)   // custom sections, see ux-and-help.md
    cmd.SetUsageFunc(rootUsageFunc) // terse usage on stderr
    cmd.PersistentFlags().Bool("help", false, "Show help for command")
    cmd.Flags().Bool("version", false, "Show tool version")
    cmd.AddGroup(&cobra.Group{ID: "core", Title: "Core commands"},
                 &cobra.Group{ID: "extension", Title: "Extension commands"})
    // auth gate: PersistentPreRunE checks auth unless cmdutil.DisableAuthCheck(cmd) annotation
    return cmd
}
```

- Auth gate in `PersistentPreRunE`: commands like `auth login`, `version`, `help`, `completion`, `config` opt out via an annotation (`cmdutil.DisableAuthCheck(cmd)`). Exit `4` with a context-aware hint (in CI: "set TOOL_TOKEN"; locally: "run tool auth login").
- Version: `tool --version` and `tool version` print the same string, plus a release URL.
- Global flags are rare. gh has almost none besides `--help`; `-R/--repo` is added per-command via `cmdutil.EnableRepoOverride(cmd, f)`. Docker parses global flags (`--context`, `--host`, `--config`, `--debug`) before the subcommand with `SetInterspersed(false)`. Prefer gh: per-command flags where they apply, env vars for global overrides.

## Command registration and grouping

- Noun command (`pkg/cmd/pr/pr.go`) creates a group command, sets `GroupID: "core"`, and adds verbs, optionally in subgroups (`cmdutil.AddGroup(cmd, "General commands", ...)`, `"Targeted commands"`).
- A noun with no verb shows its help (gh) — docker does the same via `RunE: ShowHelp(stderr)` with `Args: NoArgs`.
- Aliases on verbs: `Aliases: []string{"ls"}` for `list`, `"rm"` for `delete`. Keep them consistent across nouns.
- Hidden commands for help topics, telemetry senders, deprecated forms.
- Docker's flat→noun migration: `docker ps` and `docker container ls` share one constructor (`cmd := *newPsCommand(cli); cmd.Use = "ls"`), with legacy shortcuts hideable via env. Use this if you must rename the command tree without breaking users.

## Signals and cancellation

Use `cmd.Context()` everywhere; derive it from signals:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
cmd, err := rootCmd.ExecuteContextC(ctx)
```

Docker's refinements worth adopting for long-running/streaming tools:
- Keep the signal as the cancellation **cause** (`context.WithCancelCause`) and map it to `128+signum` with no message.
- Force-exit after 3 signals, restoring the terminal (raw mode/echo) first: "got 3 SIGTERM/SIGINTs, forcefully exiting".
- Only the main goroutine calls `os.Exit` (signal goroutine closes a channel).
- Prompts select on `ctx.Done()` so Ctrl-C at a prompt exits cleanly.
- When running a child process in the foreground on a TTY, don't re-send SIGINT — the kernel already delivered it to the process group.

## Subprocesses

- Route all `exec.Cmd` creation through one seam (gh `internal/run`: `var PrepareCmd = func(cmd *exec.Cmd) Runnable`) so tests can stub it.
- Never resolve executables from the current directory (Go ≥1.19 `exec.LookPath` returns `exec.ErrDot` for that; gh additionally uses `github.com/cli/safeexec`). Check for required external tools up front and name how to install them in the error.
- Include the child's stderr in the returned error; log the command line when `TOOL_DEBUG` is set.
- Editor/pager commands are shell-split (`shlex`) so `code --wait` works.

## Cobra gotchas

Bugs that are easy to write and that tests reliably catch:
- **Bare `--json` listing fields**: don't use `NoOptDefVal` on a value flag — then
  `--json id,title` parses `id,title` as a positional argument. Instead give the command
  its own `SetFlagErrorFunc` that turns `flag needs an argument: --json` into the field
  list and delegates everything else to `c.Parent().FlagErrorFunc()` (gh does this).
- **Nested typo suggestions**: cobra only suggests at the root. For `tool item lsit`
  cobra calls the noun's help func, so the help func must detect the unknown verb.
  It receives the **full argv**: skip as many positional args as the command's depth
  from root, ignore flags, and treat the next positional as unknown. Set
  `SuggestionsMinimumDistance = 2` on that command before `SuggestionsFor` (default 0
  disables distance matching).
- **Help funcs can't return errors**: record failure in a package variable checked by
  `Main` (gh `root.HasFailed()`), and **reset it when the root command is built**,
  or tests that build the root repeatedly leak state into each other.
- **Usage output**: replace cobra's default usage template with a terse one on stderr
  (usage line, local flags, "Run 'tool x --help' for more information") — the default
  dumps examples and global flags after every flag error.
- **`SetOut`/`SetErr` on the root** to IOStreams so help, usage and completion go
  through the same streams tests capture.
- **Validation in `PreRunE`** (e.g. `AddJSONFlags`) still runs before `runF`, so flag
  tests cover it; chain to any existing `PreRunE` instead of overwriting it.
- **Package named `delete`** shadows the builtin; fine (gh does it) but don't call the
  builtin inside that package.

## Migrating a hand-rolled CLI

Do it in safe, separately-shippable steps, keeping output byte-identical (golden tests first):
1. **Freeze behavior**: add golden/exact-output tests for each command's stdout, stderr and exit code (both TTY modes if they differ).
2. **IOStreams**: replace `os.Stdout`/`os.Stderr`/`os.Stdin` references with an `*IOStreams` passed in. Tests stop swapping globals.
3. **Typed errors**: commands return `error`; add `FlagError`/`SilentError`/`CancelError` and one mapping in `Main()`.
4. **Factory**: move config/client/subprocess construction into lazy Factory funcs.
5. **Cobra, one noun at a time**: root on cobra with unknown commands delegated to the legacy dispatcher until each noun moves. Gains: nested help, suggestions, completion, generated docs.
6. **Then** add `--json`, non-TTY formats, help topics, generated docs.
