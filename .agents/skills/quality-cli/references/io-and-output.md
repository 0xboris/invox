# I/O and output

Terminal handling, color, pagers, progress, tables and machine-readable output.

## Contents
- [IOStreams](#iostreams)
- [stdout vs stderr](#stdout-vs-stderr)
- [Color](#color)
- [Pager](#pager)
- [Progress indicators](#progress-indicators)
- [Tables: TTY vs pipe](#tables-tty-vs-pipe)
- [Structured output: --json / --jq / --template](#structured-output---json----jq----template)
- [View commands and markdown](#view-commands-and-markdown)
- [Broken pipes and write errors](#broken-pipes-and-write-errors)
- [Untrusted content](#untrusted-content)
- [Environment variables for I/O](#environment-variables-for-io)

---

## IOStreams

One object owns stdin/stdout/stderr and answers every terminal question. Commands
never touch `os.Std*` (gh `pkg/iostreams/iostreams.go`).

```go
type IOStreams struct {
    In     io.ReadCloser
    Out    io.Writer
    ErrOut io.Writer

    term term                      // interface over the real terminal (size, color support, theme)
    stdinTTYOverride, stdinIsTTY   bool
    stdoutTTYOverride, stdoutIsTTY bool
    stderrTTYOverride, stderrIsTTY bool
    colorEnabled                   bool
    neverPrompt                    bool
    spinnerDisabled                bool
    pagerCommand                   string
    pagerProcess                   *os.Process
}

func (s *IOStreams) IsStdoutTTY() bool {
    if s.stdoutTTYOverride { return s.stdoutIsTTY }
    return s.term.IsTerminalOutput() // honors TOOL_FORCE_TTY; also detect Cygwin/MinTTY
}
func (s *IOStreams) SetStdoutTTY(v bool) { s.stdoutTTYOverride = true; s.stdoutIsTTY = v } // tests

func (s *IOStreams) CanPrompt() bool {
    if s.neverPrompt { return false }          // --no-input, TOOL_PROMPT_DISABLED, config
    return s.IsStdinTTY() && s.IsStderrTTY()   // prompts render on stderr
}
```
gh checks stdin && stdout and draws prompts on stdout. This skill deliberately checks
**stderr** and draws prompts there, so `tool x > out.txt` can still prompt while the
redirected file only gets data. A CI job or agent may allocate a PTY, so always offer
`--no-input` as a deterministic off switch instead of treating a TTY as proof of a human.

API surface worth having: `IsStdinTTY/IsStdoutTTY/IsStderrTTY`, `SetXTTY` overrides,
`ColorEnabled()`, `ColorScheme()`, `CanPrompt()`, `SetNeverPrompt()`,
`TerminalWidth()` (fallback 80), `TerminalTheme()` (light/dark/none),
`StartPager()/StopPager()`, `StartProgressIndicatorWithLabel()/StopProgressIndicator()`,
`RunWithProgress(label, fn)`, `ReadUserFile(path)` ("-" = stdin).

`System()` builds the real streams; `Test()` returns buffer-backed streams, all non-TTY:
```go
ios, stdin, stdout, stderr := iostreams.Test()
```

Use `github.com/cli/go-gh/v2/pkg/term` (`term.FromEnv()`) for detection — it implements
`NO_COLOR`, `CLICOLOR`, `CLICOLOR_FORCE` and Windows virtual-terminal enablement. Its
force-TTY variable is hard-coded as `GH_FORCE_TTY`, so a tool that wants
`TOOL_FORCE_TTY` must apply that override itself (as the starter's `iostreams.System()` does). On Windows wrap output with `go-colorable`.

Docker's `cli/streams` adds terminal **state**: `SetRawTerminal()` (no-op on non-TTY,
saves state) and `RestoreTerminal()` (always safe). Use that if you ever enter raw mode,
and restore on every exit path including forced exit.

## stdout vs stderr

| stdout | stderr |
|---|---|
| Rows/tables the user asked for | Progress, spinners |
| JSON / template output | "Creating issue in owner/repo" |
| URL or path of what was created | "Opening example.com/... in your browser." |
| Raw file content (`view`) | Warnings, deprecation notices |
| | Update notice, auth hints, debug logs |
| | Prompts (see `CanPrompt`) |

Human chatter is additionally **TTY-gated**: e.g. `pr list` prints "Showing 3 of 3 open
pull requests in owner/repo" only when stdout is a TTY. Success lines use the success icon:
```go
if opts.IO.IsStdoutTTY() {
    fmt.Fprintf(opts.IO.Out, "%s Deleted repository %s\n", cs.SuccessIcon(), name)
}
```
(gh writes these TTY-only confirmations to `Out`; because they are TTY-gated they never
reach a pipe. Writing them to `ErrOut` is equally fine. Pick one and be consistent.)
Prompts also go to stderr and only run when `CanPrompt()` holds (see above).

A command that creates something prints its URL/path as the **last** line of stdout so
`$(tool x create ...)` captures it.

## Color

`ColorScheme` (gh `pkg/iostreams/color.go`) is a value snapshot of capabilities. Every
method is a no-op when color is disabled, so call sites never branch.

```go
cs := opts.IO.ColorScheme()
cs.Bold(s); cs.Muted(s); cs.Red(s); cs.Green(s); cs.Yellow(s); cs.Cyan(s)
cs.SuccessIcon()  // green ✓
cs.WarningIcon()  // yellow !
cs.FailureIcon()  // red X
cs.TableHeader(s) // theme-aware header style
cs.ColorFromString("green")
```

Rules (gh primer):
- Color is off when `NO_COLOR` is non-empty, `TERM=dumb`, `CLICOLOR=0`, `--no-color`
  is passed, or the stream being written isn't a TTY; `CLICOLOR_FORCE` turns it on.
  Decide per stream: stderr can be colored while stdout is piped. Explicit machine
  formats (`--json`) are never colored on a pipe.
- Only the 8 basic ANSI colors (respect user terminal themes). 256/truecolor only for user-supplied colors (labels) and only if supported.
- Color **enhances** meaning; it never carries it alone — pair with icons, words, or a column when piped.
- Semantic usage: green = success/open, red = failure/closed, yellow = warning/pending, magenta = merged/special, muted gray = secondary info, bold = emphasis/titles.
- Icon set: `✓` success, `X`/`✗` failure, `!` alert, `-` neutral, `+` changes.
- Accessible mode (`accessible_colors` config / env): avoid colors that collide with themes.

## Pager

```go
if err := opts.IO.StartPager(); err != nil {
    fmt.Fprintf(opts.IO.ErrOut, "error starting pager: %v\n", err) // warning, not fatal
}
defer opts.IO.StopPager()
```
- Source: `TOOL_PAGER` > config `pager` > `PAGER`. No-op when empty, `cat`, or stdout isn't a TTY.
- Set `LESS=FRX` (quit if one screen, raw colors, no init) and `LV=-c` if unset; strip `PAGER=` from the child env.
- Wrap the pager's stdin; translate `EPIPE`/`io.ErrClosedPipe` into `ErrClosedPagerPipe` → exit 0.
- Detect terminal theme **before** starting the pager (the pager owns the TTY afterwards).

## Progress indicators

- Spinner writes to **stderr**, only when stdout **and** stderr are TTYs.
- Prefer `opts.IO.RunWithProgress("Fetching items", func() error { ... })` — starts, stops (deferred), returns fn's error.
- Reusing an active indicator just updates the label; it's mutex-protected.
- Disabled spinner (`TOOL_SPINNER_DISABLED`, accessibility, AI agent detected) → print one plain line (`Working...`) instead; screen readers and logs can't handle redraws.
- Always stop the indicator before printing anything else or prompting.
- Docker streams progress bars that redraw in place on TTY and degrade to one line per update otherwise (`jsonmessage.DisplayJSONMessagesStream(r, out, fd, isTerminal, ...)`).

## Tables: TTY vs pipe

Use go-gh's table printer (`github.com/cli/go-gh/v2/pkg/tableprinter`:
`tableprinter.New(w, isTTY, maxWidth)`, `AddHeader`, `AddField(s, WithColor/WithTruncate)`,
`EndRow`, `Render`) behind a thin wrapper like gh's `internal/tableprinter`, which adds
header styling and `AddTimeField`. The snippet below uses that wrapper's API:

```go
isTTY := opts.IO.IsStdoutTTY()
headers := []string{"ID", "TITLE", "UPDATED"}
if !isTTY {
    headers = []string{"ID", "TITLE", "STATE", "UPDATED"} // header is dropped when piped, but keep columns aligned
}
tp := tableprinter.New(opts.IO, tableprinter.WithHeader(headers...))
for _, it := range items {
    id := strconv.Itoa(it.Number)
    if isTTY {
        id = "#" + id
    }
    tp.AddField(id, tableprinter.WithColor(stateColor(cs, it.State)))
    tp.AddField(text.RemoveExcessiveWhitespace(it.Title))
    if !isTTY {
        tp.AddField(it.State) // color carried state on TTY; spell it out when piped
    }
    tp.AddTimeField(opts.Now(), it.UpdatedAt, cs.Muted)
    tp.EndRow()
}
return tp.Render()
```
gh builds its headers and `#` prefix conditionally the same way (`pkg/cmd/issue/shared/display.go`).

| | TTY | Piped |
|---|---|---|
| Header | yes, uppercase, styled | **no** |
| Separator | aligned columns | **tab** (`cut -f2` works) |
| Truncation | to terminal width | **none** |
| Color | yes | **none** |
| Time | `about 3 hours ago` (`text.FuzzyAgo`) | **RFC3339** |
| IDs | `#32` | `32` |
| State | via color | explicit column |

Test both modes against the same fixture.

**Escaping in non-TTY output.** Raw TSV can't hold a tab or newline inside a field.
Escape `\` as `\\`, tab as `\t`, CR as `\r` and LF as `\n` so one record is always one
line, and document it. (`--json` is the lossless format.)

**Display width.** Pad and truncate by terminal cells, not bytes or runes: CJK and
emoji take two cells, combining marks zero. Use `github.com/mattn/go-runewidth` or
`github.com/rivo/uniseg`, never `len(s)`/`utf8.RuneCountInString` for alignment, and
never cut a string in the middle of a UTF-8 sequence.

Docker alternative: `--format "table {{.ID}}\t{{.Name}}"` Go templates with a `table`
prefix that adds a header, plus per-user default formats in config (`psFormat`). Prefer gh's
fixed TTY/TSV contract + `--json/--template`; consider docker's config-level default format
only for heavily-customized list commands.

## Structured output: --json / --jq / --template

gh `pkg/cmdutil/json_flags.go`:

```go
var itemFields = []string{"id", "title", "state", "author", "createdAt", "url"}
cmdutil.AddJSONFlags(cmd, &opts.Exporter, itemFields)

// in xRun:
if opts.Exporter != nil {
    return opts.Exporter.Write(opts.IO, items) // honors --jq / --template
}
```

Behavior to replicate:
- `--json` takes a comma-separated field list; **bare `--json` lists available fields** instead of erroring opaquely.
- Unknown field → `Unknown JSON field: "foo"\nAvailable fields:\n  id\n  title ...`.
- `--jq <expr>` and `-t/--template <tmpl>` require `--json`; `--web` and `--json` are mutually exclusive.
- Fields are stored in a `help:json-fields` annotation → shown in help under JSON FIELDS and completed by the shell (comma-aware).
- **Fields shape the query**: `opts.Exporter.Fields()` is passed to the API/query builder so you fetch only what's requested (cheaper, faster).
- Serialization: models implement `ExportData(fields []string) map[string]any`; fallback `cmdutil.StructExportData` by reflection. Flatten wrapper types (GraphQL `nodes`).
- Output: `--jq` via `go-gh/pkg/jq` (no external jq needed; pretty only on TTY); `--template` via `go-gh/pkg/template` (helpers `tablerow`, `tablerender`, `timeago`, `timefmt`, `truncate`, `hyperlink`, `color`, `autocolor`, `pluck`, `join`); else colored pretty JSON on TTY (`jsoncolor`), compact JSON when piped. `SetEscapeHTML(false)` so URLs stay readable.
- Empty result with `--json` prints `[]` (exit 0), not the "no results" message.
- **The JSON contract:**
  - One JSON value plus a newline: an array for lists, an object for a single item.
    A missing requested item is an error, not `{}`.
  - Encode the whole document into a buffer before writing, so a fetch failure
    leaves stdout empty. Initialize slices (`items := []Item{}`) so empty is `[]`,
    not `null`.
  - Use dedicated output types with explicit tags; never serialize internal
    clients, config or secrets. No `omitempty` on fields where `false`/`0` mean
    something. Use `json.Number` or strings for IDs that can exceed 2^53.
  - Check the encoder's and writer's errors. A requested result that couldn't be
    written is a failure.
  - Field names, types, units and empty representations are public API; only
    additive changes are compatible.
- Document it all in a `help formatting` topic with examples.

## View commands and markdown

- TTY: render markdown (glamour) using terminal width and theme (`GLAMOUR_STYLE` override), through the pager.
- Piped: emit stable `key:\tvalue` lines, then `--`, then the raw body (gh `repo view`, `issue view`).
- `--web`/`-w` opens the browser instead; print `Opening URL in your browser.` to stderr (TTY only).

## Broken pipes and write errors

- `tool list | head -1` closes the pipe early. For a Unix data producer, Go's
  default SIGPIPE behavior (the process exits quietly) is a fine contract. If you
  handle `EPIPE` yourself, stop producing and exit without a scary message, as gh
  does for the pager (`ErrClosedPagerPipe` → exit 0).
- Never ignore write errors globally. Check the error from the final flush/encode
  of requested output.
- If a mutation succeeded but its result couldn't be written, say so on stderr if
  you can. Don't let a retry repeat the mutation.

## Untrusted content

Remote text (titles, bodies, API errors) may contain ANSI/OSC escape sequences that can
spoof output or manipulate the terminal. gh wraps such strings (`iostreams.Untrusted`) and
writes bodies through a sanitizing writer (`ContentOut`, go-gh `asciisanitizer`). Offer an
explicit opt-out flag only for raw API access (`gh api --allow-escape-sequences`).

## Environment variables for I/O

| Variable | Effect |
|---|---|
| `NO_COLOR` | disable color |
| `TERM=dumb` | disable color and cursor movement (spinners) |
| `CLICOLOR=0` | disable color |
| `CLICOLOR_FORCE=1` | force color even when piped |
| `TOOL_FORCE_TTY` | treat stdout as TTY; value = width in columns or `%` |
| `TOOL_PAGER`, `PAGER` | pager command (`cat` or empty disables) |
| `TOOL_PROMPT_DISABLED` | never prompt (same as `--no-input`; also config `prompt: disabled`) |
| `TOOL_SPINNER_DISABLED` | textual progress instead of spinner |
| `TOOL_ACCESSIBLE_PROMPTER` | line-based prompts for screen readers |
| `TOOL_DEBUG` | verbose logs to stderr (`api` value = HTTP tracing) |

Apply them once in `Main()` with explicit precedence (env > config > default), and
document every one in `tool help environment`.
