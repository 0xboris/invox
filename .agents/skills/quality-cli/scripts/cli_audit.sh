#!/usr/bin/env bash
# cli_audit.sh: heuristic checks of a Go CLI against the quality-cli rules.
#
# Usage: bash cli_audit.sh [repo-root] [--strict]
#   repo-root  defaults to "."
#   --strict   exit 1 if any WARN was reported (for CI)
# Exit codes: 0 ok, 1 warnings with --strict, 2 bad root / no Go sources.
#
# Grep-based. A WARN means "look here", not "this is wrong".
# Composition roots may touch the process directly: files in `package main` and
# ./internal/{app,run,build}/, ./{pkg,internal}/iostreams/, ./{pkg/cmd,internal}/factory/.
# Env lookups are also allowed in ./{internal,pkg}/config/.
# Layering checks treat command, wiring, I/O, presentation, docs, plugin and test-support
# directories (cmd, cli, app, cmdutil, factory, root, iostreams, prompter, tableprinter,
# formatter, ...) as the command side; everything else is "below the commands".
# Skipped: vendor/, testdata/, dot-directories, and generated files ("DO NOT EDIT").
# Portable: bash 3.2+ (macOS), GNU and BSD grep/xargs.

set -uo pipefail
export LC_ALL=C

ROOT="."
STRICT=0
for arg in "$@"; do
  case "$arg" in
    --strict) STRICT=1 ;;
    -h|--help) sed -n '2,15p' "$0"; exit 0 ;;
    *) ROOT="$arg" ;;
  esac
done
cd "$ROOT" 2>/dev/null || { echo "cannot cd to $ROOT" >&2; exit 2; }

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  Y=$'\033[33m'; G=$'\033[32m'; B=$'\033[1m'; M=$'\033[2m'; R=$'\033[0m'
else
  Y=""; G=""; B=""; M=""; R=""
fi

WARNS=0
MAX_HITS=${MAX_HITS:-15}
TMP=$(mktemp -d "${TMPDIR:-/tmp}/cli_audit.XXXXXX") || exit 2
trap 'rm -rf "$TMP"' EXIT

# File lists live in files (one path per line) instead of arrays: works on bash 3.2
# and avoids empty-array errors under `set -u`.
find . -type f -name '*.go' -not -path '*/vendor/*' -not -path '*/testdata/*' \
  -not -path './.*' -not -path '*/.*/*' | sort > "$TMP/all"
# Drop generated files.
tr '\n' '\0' < "$TMP/all" | xargs -0 grep -lE '^// Code generated .* DO NOT EDIT\.$' /dev/null \
  2>/dev/null | sort > "$TMP/generated"
comm -23 "$TMP/all" "$TMP/generated" > "$TMP/go"
grep -v '_test\.go$' "$TMP/go" > "$TMP/src" || true
grep '_test\.go$' "$TMP/go" > "$TMP/tests" || true

n_src=$(wc -l < "$TMP/src" | tr -d ' ')
n_tests=$(wc -l < "$TMP/tests" | tr -d ' ')
if [ "$n_src" -eq 0 ]; then
  echo "no Go sources found under $ROOT" >&2
  exit 2
fi

# grep over a list file; /dev/null keeps grep from ever reading stdin and forces filenames.
lgrep() { # lgrep <list-file> <grep args...>
  local list="$1"; shift
  tr '\n' '\0' < "$list" | xargs -0 grep "$@" /dev/null 2>/dev/null
}

# Composition roots: package main + allowlisted paths.
lgrep "$TMP/src" -lE '^package main([[:space:]]|$)' | sort > "$TMP/main"
while IFS= read -r f; do
  case "$f" in
    ./internal/app/*|./internal/run/*|./internal/build/*|\
    ./pkg/iostreams/*|./internal/iostreams/*|./pkg/cmd/factory/*|./internal/factory/*) ;;
    *) grep -qxF "$f" "$TMP/main" || echo "$f" ;;
  esac
done < "$TMP/src" > "$TMP/app"
while IFS= read -r f; do
  case "$f" in ./internal/config/*|./pkg/config/*) ;; *) echo "$f" ;; esac
done < "$TMP/app" > "$TMP/app_env"

section() { printf '\n%s== %s ==%s\n' "$B" "$1" "$R"; }
ok()      { printf '  %sok%s   %s\n' "$G" "$R" "$1"; }
info()    { printf '  %sinfo%s %s\n' "$M" "$R" "$1"; }
warn()    { WARNS=$((WARNS+1)); printf '  %sWARN%s %s\n' "$Y" "$R" "$1"; }

# W = "not preceded by an identifier character or a dot" (portable \b substitute).
W='(^|[^[:alnum:]_.])'

# grep_check <description> <regex> <list-file>
grep_check() {
  local desc="$1" re="$2" list="$3" hits n
  hits=$(lgrep "$list" -nE "$re" | grep -vE '^[^:]+:[0-9]+:[[:space:]]*//' || true)
  if [ -n "$hits" ]; then
    n=$(printf '%s\n' "$hits" | wc -l | tr -d ' ')
    warn "$desc: $n hit(s)"
    printf '%s\n' "$hits" | head -n "$MAX_HITS" | sed 's/^/         /'
    if [ "$n" -gt "$MAX_HITS" ]; then printf '         … %d more\n' $((n-MAX_HITS)); fi
  else
    ok "$desc"
  fi
}

has()      { lgrep "$TMP/src" -qE "$1"; }
has_test() { lgrep "$TMP/tests" -qE "$1"; }

printf '%squality-cli audit%s: %s (%d source files, %d test files)\n' \
  "$B" "$R" "$(pwd)" "$n_src" "$n_tests"

section "Process & errors"
grep_check "os.Exit outside package main / app" "${W}os\.Exit\(" "$TMP/app"
grep_check "log.Fatal in app code (return an error instead)" "${W}log\.Fatal" "$TMP/app"
n_panic=$(lgrep "$TMP/app" -hE "${W}panic\(" | wc -l | tr -d ' ')
if [ "$n_panic" -gt 0 ]; then info "$n_panic panic() call(s): fine for programmer errors, never for user input"; fi
if has 'github\.com/spf13/cobra'; then
  if has 'SilenceErrors([[:space:]]*:|[[:space:]]*=)[[:space:]]*true'; then ok "cobra SilenceErrors set"
  else warn "cobra SilenceErrors not set: cobra prints errors itself (double printing)"; fi
  if has 'SilenceUsage([[:space:]]*:|[[:space:]]*=)[[:space:]]*true'; then ok "cobra SilenceUsage set"
  else warn "cobra SilenceUsage not set: usage is dumped on runtime errors"; fi
  if has 'SetFlagErrorFunc'; then ok "custom flag error func"
  else info "no SetFlagErrorFunc: consider wrapping flag errors as FlagError"; fi
else
  info "cobra not used: hand-rolled parsing (see references/architecture.md §Migrating)"
fi
if has 'FlagError'; then ok "FlagError-style usage errors present"
else warn "no FlagError type: usage vs runtime errors are probably not distinguished"; fi
if has 'Cancel(led)?Error|IsUserCancellation'; then ok "cancellation error present"
else info "no cancellation error: Ctrl-C/declined prompts may report as failures"; fi

section "I/O discipline"
grep_check "direct os.Stdout/os.Stderr/os.Stdin outside composition roots" \
  "${W}os\.(Stdout|Stderr|Stdin)([^[:alnum:]_]|$)" "$TMP/app"
grep_check "implicit stdout prints (fmt.Print*/println)" \
  "${W}fmt\.Print(f|ln)?\(|^[[:space:]]*println\(" "$TMP/app"
if has 'IsTerminal|isatty|go-gh/v2/pkg/term|IsStdoutTTY'; then ok "TTY detection present"
else warn "no TTY detection: piped output is probably identical to terminal output"; fi
if has 'NO_COLOR|go-gh/v2/pkg/term|fatih/color|termenv|lipgloss'; then ok "NO_COLOR-aware color handling (direct or via library)"
elif has '\\033\[|\\x1b\['; then warn "raw ANSI escapes without NO_COLOR handling"
else info "no color usage detected"; fi
if has 'CanPrompt|neverPrompt|PROMPT_DISABLED'; then ok "prompt gating present"
else info "no CanPrompt-style gate (fine if the tool never prompts)"; fi

section "Dependencies & determinism"
grep_check "os.Getenv/LookupEnv scattered in app code (centralize + document precedence)" \
  "${W}os\.(Getenv|LookupEnv)\(" "$TMP/app_env"
grep_check "exec.Command outside a run/exec seam" "${W}exec\.Command(Context)?\(" "$TMP/app"
grep_check "time.Now() in app code (inject a clock for testable output)" "${W}time\.Now\(\)" "$TMP/app"
grep_check "http.DefaultClient / http.Get / http.Post (inject an http.Client)" \
  "${W}http\.(DefaultClient|Get\(|Post\()" "$TMP/app"

section "Output contracts"
if has 'AddJSONFlags|Flags\(\)\.[A-Za-z]+\(.*"json"|"--json"|\.(Bool|String|StringSlice)(Var)?P?\([^)]*"json"'; then ok "--json support found"
else warn "no --json flag found: list/view commands should offer machine output"; fi
if has 'RFC3339'; then ok "RFC3339 timestamps used somewhere (non-TTY output)"
else info "no RFC3339 formatting: check non-TTY timestamps are absolute"; fi

section "Help & docs"
if has 'github\.com/spf13/cobra'; then
  n_cmds=$(lgrep "$TMP/src" -hoE '&cobra\.Command\{' | wc -l | tr -d ' ')
  n_ex=$(lgrep "$TMP/src" -hoE "${W}Example:" | wc -l | tr -d ' ')
  # Group/noun commands and help topics don't need examples; leaf commands do.
  if [ "$n_ex" -eq 0 ]; then warn "no cobra command has an Example"
  else info "$n_ex Example fields for $n_cmds cobra commands: check every leaf command has one"; fi
  if has 'GenManTree|GenMarkdownTree|gen-docs|cli-docs-tool'; then ok "docs generation present"
  else info "no man/markdown generation from the command tree"; fi
  if has 'DisableDefaultCmd:[[:space:]]*true'; then info "cobra's default completion command is disabled: check another one exists"
  else ok "shell completion (cobra default completion command)"; fi
elif has 'compdef|complete -[CFW]|_arguments|[Cc]ompletion'; then ok "shell completion referenced"
else warn "no shell completion found"; fi

section "Config writes"
if has 'os\.WriteFile|ioutil\.WriteFile'; then
  if has 'os\.Rename|CreateTemp|renameio|atomicwriter'; then ok "atomic write pattern present (verify it is used for config/state)"
  else warn "os.WriteFile without temp+rename: config/state writes may corrupt on crash"; fi
elif has 'os\.Rename|CreateTemp'; then ok "writes use temp file + rename"
else info "no file writes detected"; fi

section "Layering (references/layers.md)"
# "Below the commands" = source files outside command, wiring, I/O and presentation
# packages (and outside package main). Those should not know about cobra or streams.
CMD_LAYER_RE='(^|/)([a-z]*cmd|cli|app|commands?|cmdutil|factory|root|iostreams|streams|prompter|tableprinter|formatter|docs|cli-plugins|plugins?|[^/]*tests?|testutils?|fixtures|mocks?)(/|$)'
while IFS= read -r f; do
  d=$(dirname "$f")
  if ! printf '%s\n' "$d" | grep -qE "$CMD_LAYER_RE"; then echo "$f"; fi
done < "$TMP/app" > "$TMP/below"
n_below=$(wc -l < "$TMP/below" | tr -d ' ')
if [ "$n_below" -eq 0 ]; then
  info "no packages below the command layer: business logic probably lives in command files"
else
  IMPORT_PFX='^[[:space:]]*(import[[:space:]]+)?([A-Za-z_][A-Za-z0-9_]*[[:space:]]+|\.[[:space:]]+|_[[:space:]]+)?'
  grep_check "domain/adapter packages importing cobra/pflag or I/O streams" \
    "${IMPORT_PFX}\"([^\"]*/spf13/(cobra|pflag)|[^\"]*/(iostreams|streams))\"" "$TMP/below"
  n_map=$(lgrep "$TMP/below" -hE '^[[:space:]]+[A-Z][A-Za-z0-9_]*[[:space:]]+(\[\])?map\[string\](any|interface\{\})' | wc -l | tr -d ' ')
  if [ "$n_map" -gt 0 ]; then info "$n_map exported struct field(s) typed map[string]any below the commands: prefer typed models"; fi
fi
# Dedicated test-support packages (internal/test, testutils, *test) may import testing.
grep -vE '/([^/]*tests?|testutils?|testenv|fixtures)/[^/]*$' "$TMP/src" > "$TMP/prod" || true
grep_check "production files importing \"testing\" (keep mocks in _test.go or a test package)" \
  '^[[:space:]]*(import[[:space:]]+)?"testing"' "$TMP/prod"
MODULE=$(sed -n 's/^module[[:space:]]*//p' go.mod 2>/dev/null | head -n 1)
if [ -n "$MODULE" ] && has 'github\.com/spf13/cobra'; then
  # Command packages = directories with a cobra.Command literal. Flag any package that
  # imports a command package other than its own subcommands (root wiring excepted).
  grep -vE '(^|/)(test|mock[^/]*)\.go$' "$TMP/prod" > "$TMP/prodcmd" || true
  lgrep "$TMP/prodcmd" -lE '&cobra\.Command\{' | while IFS= read -r f; do dirname "$f"; done \
    | sed 's#^\./##' | sort -u > "$TMP/cmdpkgs"
  : > "$TMP/leafhits"
  while IFS= read -r f; do
    own=$(dirname "$f" | sed 's#^\./##')
    case "$own" in */root|root|cmd/*) continue ;; esac
    grep -nE "\"$MODULE/[^\"]+\"" "$f" /dev/null 2>/dev/null | while IFS= read -r line; do
      dep=$(printf '%s\n' "$line" | sed -E "s#.*\"$MODULE/([^\"]+)\".*#\1#")
      case "$dep" in "$own"/*|*/root|root) continue ;; esac
      if grep -qxF "$dep" "$TMP/cmdpkgs"; then echo "$line"; fi
    done >> "$TMP/leafhits"
  done < "$TMP/app"
  n_leaf=$(wc -l < "$TMP/leafhits" | tr -d ' ')
  if [ "$n_leaf" -gt 0 ]; then
    warn "packages importing a command package (not their own subcommand): $n_leaf hit(s) (move shared code to <noun>/shared or the domain)"
    head -n "$MAX_HITS" "$TMP/leafhits" | sed 's/^/         /'
  else ok "no package imports another command package"; fi
fi
: > "$TMP/big"
while IFS= read -r f; do
  n=$(wc -l < "$f" | tr -d ' ')
  if [ "$n" -gt 1000 ]; then echo "$n $f" >> "$TMP/big"; fi
done < "$TMP/src"
if [ -s "$TMP/big" ]; then
  info "files over 1000 lines (split by concept if they mix several):"
  sort -rn "$TMP/big" | head -n "$MAX_HITS" | sed 's/^/         /'
fi

section "Tests"
while IFS= read -r f; do dirname "$f"; done < "$TMP/app" | sort -u > "$TMP/pkgs"
: > "$TMP/missing"
while IFS= read -r d; do
  if ! ls "$d"/*_test.go >/dev/null 2>&1; then echo "$d" >> "$TMP/missing"; fi
done < "$TMP/pkgs"
n_missing=$(wc -l < "$TMP/missing" | tr -d ' ')
if [ "$n_missing" -gt 0 ]; then
  warn "$n_missing package(s) without tests:"
  head -n "$MAX_HITS" "$TMP/missing" | sed 's/^/         /'
else ok "every app package has tests"; fi
if [ "$n_tests" -gt 0 ]; then
  if has_test 'Set(Stdout|Stdin|Stderr)TTY|FORCE_TTY|[Ii]sTTY|tty:'; then ok "tests reference TTY modes"
  else warn "tests never toggle TTY: terminal vs non-TTY output untested"; fi
  if has_test 'os\.Stdout[[:space:]]*='; then warn "tests swap global os.Stdout: inject IOStreams instead"
  else ok "tests don't swap os.Stdout"; fi
  if has_test 'testscript|icmd\.'; then ok "script/e2e tests present"
  else info "no testscript/e2e tests"; fi
fi

printf '\n%s%d warning(s)%s. Walk references/review-checklist.md for the judgment calls.\n' "$B" "$WARNS" "$R"
if [ "$STRICT" -eq 1 ] && [ "$WARNS" -gt 0 ]; then exit 1; fi
exit 0
