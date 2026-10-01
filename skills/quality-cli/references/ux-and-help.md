# UX, help, errors and interactivity

Command language, flags, help text, error messages, prompts and accessibility.
Mostly from gh's design guide (`docs/primer/`, `docs/command-line-syntax.md`) and code.

## Contents
- [Command language](#command-language)
- [Flags](#flags)
- [Arguments](#arguments)
- [Help](#help)
- [Errors](#errors)
- [Interactivity](#interactivity)
- [Destructive operations](#destructive-operations)
- [Editor integration](#editor-integration)
- [Browser integration](#browser-integration)
- [Accessibility and AI agents](#accessibility-and-ai-agents)

---

## Command language

- **`tool <noun> <verb> [args] [flags]`** — `gh pr create`, `gh repo view`. Nouns group, verbs act.
- **Modifiers are flags, not commands**: `gh pr review --approve`, not `gh pr approve`.
- **Unambiguous verbs**, consistent across nouns: `list`, `view`, `create`, `edit`, `delete`, `close`/`reopen`, `status`. Pick `create` over `new`/`open`; use the product's own vocabulary (`close`, not `delete`, if that's what the UI says).
- **Understood short forms**: `repo`, not `repository`; aliases `ls` for `list`, `rm` for `delete`.
- **Parallel siblings**: if `issue list` has `--state`, `--label`, `--limit`, `--json`, `--web`, so does `pr list`, with the same short letters.
- **Sentence case** in descriptions, no trailing period in `Short`.
- **Anticipate next steps**: after `create`, print the URL; after a state change, hint at the follow-up command (TTY only). Docker generalizes this into plugin "What's next" hooks.
- **Bias to terminal, easy escape to browser**: `--web` on create/view/list.
- **Escape hatch**: a raw `tool api <endpoint>` command (see `extensibility.md`) so users are never blocked on missing commands.

## Flags

| Want | Use |
|---|---|
| Enum value | `cmdutil.StringEnumFlag(cmd, &opts.State, "state", "s", "open", []string{"open","closed","all"}, "Filter by state")` → validates, appends `{open\|closed\|all}` to usage, registers completion |
| Enum list | `StringSliceEnumFlag` |
| Tri-state bool (unset/true/false) | `cmdutil.NilBoolFlag(cmd, &opts.Draft, "draft", "d", "Filter by draft state")` → `*bool` |
| "Was it set?" | `cmd.Flags().Changed("author")` |
| Mutually exclusive | `cmdutil.MutuallyExclusive("specify only one of `--a` or `--b`", aSet, bSet)` |
| Read from file or stdin | `--body-file file` where `-` = stdin (`cmdutil.ReadFile(path, opts.IO.In)`) |
| Limit | `-L, --limit int` with validation `< 1 → FlagErrorf("invalid value for --limit: %v")` |
| Repeatable values | `StringSliceVarP` (comma-separated **and** repeatable) |
| Value placeholder in help | backticks in usage: ``"Read body text from `file`"`` → `--body-file file` |
| Rename | keep old name: `MarkDeprecated("confirm", "use `--yes` instead")` or `MarkHidden` |

Conventions:
- Every flag has a long form; short forms only for frequent flags; don't reuse a letter for different meanings across commands.
- Common names: `-R/--repo`, `-L/--limit`, `-w/--web`, `-y/--yes`, `-q/--jq`, `-t/--template`, `--json`, `-e/--editor`, `-F/--body-file`, `-s/--state`.
- Flag help: sentence case, describe the effect, mention the default only when not obvious.
- Completion for flag values: `cmd.RegisterFlagCompletionFunc("label", ...)` (dynamic from API/git), `cobra.NoFileCompletions` where files make no sense.

## Arguments

- Validators with good messages (cobra `NoArgs`, `ExactArgs(n)`, `MaximumNArgs(n)`, or custom).
- gh's `NoArgsQuoteReminder`: on unexpected positional args when a value flag was used, append "please quote all values that have spaces" — catches `--title my title`.
- Docker's validator message shape (good model for custom validators):
  ```
  tool: 'tool item rm' requires at least 1 argument

  Usage:  tool item rm [OPTIONS] ITEM [ITEM...]

  See 'tool item rm --help' for more information
  ```
- Accept every reasonable identifier form for a target and document them in a `help:arguments` annotation: number, URL, name (`gh pr view 123 | https://.../pull/123 | branch-name`).
- Optional positional target defaults to context (current repo/dir) — but see destructive ops.
- `Use` syntax (`docs/command-line-syntax.md`): `<required>`, `[optional]`, `{a | b}` required choice, `...` repeatable, dash-case names: `"view [<number> | <url> | <branch>]"`.

## Help

### Per-command text
```go
cmd := &cobra.Command{
    Use:   "list",
    Short: "List items in a project",               // one line, no period
    Long: heredoc.Docf(`
        List items in a project.

        The search query syntax is documented here:
        <https://example.com/docs/search>

        Use %[1]s--json%[1]s for machine-readable output.
    `, "`"),
    Example: heredoc.Doc(`
        # List open items assigned to you
        $ tool item list --assignee @me

        # Output JSON for scripting
        $ tool item list --json id,title --jq '.[].title'
    `),
    Annotations: map[string]string{
        "help:arguments":   "An item can be supplied as a number or URL.",
        "help:environment": "TOOL_TOKEN: authentication token",
    },
}
```
- `heredoc.Doc` keeps indentation sane; `%[1]s` injects backticks inside raw strings.
- Examples: `# comment` line then `$ command`. Show the common path first, then scripting.

### Rendered help layout (gh `pkg/cmd/root/help.go`)
Bold section titles, two-space indented bodies:
```
<Long>

USAGE
  tool item list [flags]

ALIASES
  tool item ls

FLAGS
  -L, --limit int   Maximum number of items to fetch (default 30)
  ...

INHERITED FLAGS
  --help   Show help for command

JSON FIELDS
  id, title, state, url

ARGUMENTS / ENVIRONMENT VARIABLES   (from annotations)

EXAMPLES
  $ tool item list

LEARN MORE
  Use `tool <command> <subcommand> --help` for more information about a command.
  Read the manual at https://example.com/manual
  Learn about exit codes using `tool help exit-codes`
```
- Root help lists commands **by group** (`Core commands`, `Additional commands`, `Extension commands`, `Alias commands`) then `HELP TOPICS`.
- Commands with `--jq` automatically get "see `tool help formatting`".
- Wrap flag usages to terminal width (docker `FlagUsagesWrapped(width-1)`).
- Typo suggestions must work for **nested** commands too (cobra only does root): gh sets `SuggestionsMinimumDistance = 2` and runs suggestions on unknown subcommands, then prints `unknown command "x" for "tool item"\n\nDid you mean this?\n\tlist`.
- `tool <noun>` with no verb prints the noun's help.
- `-h` and `--help` work everywhere; `tool help <cmd>` works too, and for extensions maps to `tool <ext> --help`.

### Help topics
Hidden commands rendered as help pages: `tool help environment`, `formatting`,
`exit-codes`, `reference` (full command reference generated from the tree, paged).
`environment` lists **every** env var with precedence, e.g. "`TOOL_TOKEN`, `TOOL_API_TOKEN` (in order of precedence)".

### Help from data
Where a table already exists, generate help from it so it can't drift: gh builds
`gh config --help` from the `config.Options` table (key, description, default, allowed values).

## Errors

Shape: **what happened → why (if known) → what to do**.
- `must provide `--title` and `--body` when not running interactively`
- `invalid value for --limit: 0`
- `Unknown JSON field: "nme"\nAvailable fields:\n  name\n  ...`
- `HTTP 401: Bad credentials\nTry authenticating with:  tool auth login`
- `error connecting to api.example.com\ncheck your internet connection or https://status.example.com`
- `X Could not find extension 'foo' for this platform\n  To request support, run: tool issue create -R owner/foo --title "Add support for linux-arm64"`
- Requirement mismatch (docker): `"--platform" requires API version 1.50, but the server API version is 1.47`.

Rules:
- Lowercase start, no trailing period, backticks around flags/commands.
- Usage block only for usage errors.
- Errors to stderr, once. Wrap with context (`fmt.Errorf("failed to read config: %w", err)`); don't stack five prefixes.
- Context-aware remediation: in CI, suggest env vars; on a TTY, suggest interactive commands.
- Partial failures: report each, return non-zero at the end.

## Interactivity

**Prompter interface** (gh `internal/prompter`): one interface, swappable implementations, generated mock.
```go
//go:generate moq -rm -out prompter_mock.go . Prompter
type Prompter interface {
    Select(prompt, defaultValue string, options []string) (int, error)
    MultiSelect(prompt string, defaults, options []string) ([]int, error)
    Input(prompt, defaultValue string) (string, error)
    Password(prompt string) (string, error)
    Confirm(prompt string, defaultValue bool) (bool, error)
    ConfirmDeletion(requiredValue string) error
    MarkdownEditor(prompt, defaultValue string, blankAllowed bool) (string, error)
}
```
Implementations: default (survey/huh), **accessible** (line-based, no redraws, defaults spelled out "(default: X)", echo off for secrets).

Rules:
- Gate with `opts.IO.CanPrompt()`; compute `opts.Interactive` in `RunE`; without TTY return a `FlagError` naming the flags.
- **Every prompt has a flag.** Interactive mode only fills what flags didn't supply.
- Yes/no: default in caps `(Y/n)`. Text: show default. Long text: offer "press e to open editor".
- Prompt cancellation (Ctrl-C) → `CancelError` → exit 2, newline printed so the shell prompt isn't glued.
- `TOOL_PROMPT_DISABLED` / config `prompt: disabled` → never prompt even on a TTY.
- Docker's cancellable confirm: read stdin in a goroutine, `select` on `ctx.Done()`; default **No** for risky prompts.

## Destructive operations

gh `repo delete` is the reference implementation:
```go
// Ignore --yes when no argument provided to prevent accidental deletion
if len(args) == 0 && opts.Confirmed {
    if !opts.IO.CanPrompt() {
        return cmdutil.FlagErrorf("cannot non-interactively delete current repository. Please specify a repository or run interactively")
    }
    fmt.Fprintln(opts.IO.ErrOut, "Warning: `--yes` is ignored since no repository was specified")
    opts.Confirmed = false
}
if !opts.IO.CanPrompt() && !opts.Confirmed {
    return cmdutil.FlagErrorf("--yes required when not running interactively")
}
// later, in deleteRun:
if !opts.Confirmed {
    if err := opts.Prompter.ConfirmDeletion(fullName); err != nil { return err } // user types the name
}
```
- `-y/--yes` skips confirmation (gh). Docker uses `-f/--force`; prefer `--yes` for "don't ask", reserve `--force` for "override a safety check".
- Old `--confirm` kept as deprecated alias.
- Declining is a quiet cancel, not an error message.
- Batch destructive ops (docker `system prune`) collect a dry-run summary and confirm once.

## Editor integration

- Resolution: `TOOL_EDITOR` > config `editor` > `GIT_EDITOR` > `VISUAL` > `EDITOR` > `nano` (`notepad` on Windows).
- Shell-split the command (`code --wait`, `subl -w`).
- Temp file with a meaningful extension (`*.md`), UTF-8 BOM on Windows for Notepad, strip on read.
- `-e/--editor` flag and config `prefer_editor_prompt` for editor-first flows; error without a TTY.
- On failure after editing, preserve the text (temp file + `--recover`).

## Browser integration

- Injected `Browser` interface (`Browse(url) error`) — tests use a stub that records the URL.
- `TOOL_BROWSER` > config `browser` > `BROWSER` > OS default.
- Print `Opening <url> in your browser.` to stderr on TTY.

## Accessibility and AI agents

- Accessible prompter, spinner-off textual progress, accessible colors — each toggleable by config **and** env; gh documents them under a hidden `accessibility` command.
- Punctuation and icons readable by screen readers; never color-only meaning.
- Detect AI agents from env markers (gh `internal/agents/detect.go` checks e.g. `AI_AGENT`, `CLAUDECODE`, `CODEX_SANDBOX`, `CURSOR_AGENT`, `GEMINI_CLI`, `COPILOT_CLI`) and: disable spinners, print **full help** on flag errors (to stderr), keep output deterministic. Add the agent to the User-Agent/telemetry dimensions if you collect them.
- Make every interactive flow fully drivable by flags — that's what agents and scripts use.
