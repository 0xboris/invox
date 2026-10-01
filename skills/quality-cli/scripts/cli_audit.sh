#!/usr/bin/env bash
# cli_audit.sh — mechanical checks of a Go CLI against the quality-cli contract.
#
# Usage: cli_audit.sh [repo-root] [--strict]
#   repo-root  defaults to "."
#   --strict   exit 1 if any WARN was reported (for CI)
#
# Heuristic, grep-based: every finding is a prompt to look, not a verdict.
# Composition roots may touch the process directly: package main (cmd/) and
# packages whose path contains "iostreams", "/app/", "/run/", "/build/" or
# "/factory/". Env lookups are also allowed in "/config/".

set -uo pipefail

ROOT="."
STRICT=0
for arg in "$@"; do
  case "$arg" in
    --strict) STRICT=1 ;;
    -h|--help) sed -n '2,12p' "$0"; exit 0 ;;
    *) ROOT="$arg" ;;
  esac
done
cd "$ROOT" || { echo "cannot cd to $ROOT" >&2; exit 2; }

if [ -t 1 ] && [ -z "${NO_COLOR:-}" ]; then
  Y=$'\033[33m'; G=$'\033[32m'; B=$'\033[1m'; M=$'\033[2m'; R=$'\033[0m'
else
  Y=""; G=""; B=""; M=""; R=""
fi

WARNS=0
MAX_HITS=${MAX_HITS:-15}

# Non-test, non-vendor Go sources.
mapfile -t SRC < <(find . -type f -name '*.go' \
  -not -name '*_test.go' -not -path '*/vendor/*' -not -path '*/testdata/*' \
  -not -path '*/.git/*' | sort)
mapfile -t TESTS < <(find . -type f -name '*_test.go' -not -path '*/vendor/*' -not -path '*/.git/*')

if [ "${#SRC[@]}" -eq 0 ]; then
  echo "no Go sources found under $ROOT" >&2
  exit 2
fi

# Files outside the allowed raw-I/O locations.
is_allowed() {
  case "$1" in
    ./cmd/*|*/main.go|*iostreams*|*/app/*|*/run/*|*/build/*|*/factory/*) return 0 ;;
  esac
  return 1
}
mapfile -t APP_SRC < <(for f in "${SRC[@]}"; do is_allowed "$f" || echo "$f"; done)

section() { printf '\n%s== %s ==%s\n' "$B" "$1" "$R"; }
ok()      { printf '  %sok%s   %s\n' "$G" "$R" "$1"; }
info()    { printf '  %sinfo%s %s\n' "$M" "$R" "$1"; }
warn()    { WARNS=$((WARNS+1)); printf '  %sWARN%s %s\n' "$Y" "$R" "$1"; }

# grep_check <description> <regex> <files...>
grep_check() {
  local desc="$1" re="$2"; shift 2
  [ "$#" -eq 0 ] && { ok "$desc (no files)"; return; }
  local hits
  hits=$(grep -nE "$re" "$@" 2>/dev/null | grep -vE '^\S+:[0-9]+:\s*//' || true)
  if [ -n "$hits" ]; then
    local n; n=$(printf '%s\n' "$hits" | wc -l | tr -d ' ')
    warn "$desc — $n hit(s)"
    printf '%s\n' "$hits" | head -n "$MAX_HITS" | sed 's/^/         /'
    [ "$n" -gt "$MAX_HITS" ] && printf '         … %d more\n' $((n-MAX_HITS))
  else
    ok "$desc"
  fi
}

has() { grep -qE "$1" "${SRC[@]}" 2>/dev/null; }

printf '%squality-cli audit%s: %s (%d source files, %d test files)\n' \
  "$B" "$R" "$(pwd)" "${#SRC[@]}" "${#TESTS[@]}"

section "Process & errors"
grep_check "os.Exit outside package main" '\bos\.Exit\(' "${APP_SRC[@]}"
grep_check "log.Fatal in app code (return an error instead)" '\blog\.Fatal' "${APP_SRC[@]}"
n_panic=$(grep -hE '\bpanic\(' "${APP_SRC[@]}" 2>/dev/null | wc -l | tr -d ' ')
[ "$n_panic" -gt 0 ] && info "$n_panic panic() call(s) — fine for programmer errors, never for user input"
if has 'github.com/spf13/cobra'; then
  has 'SilenceErrors:\s*true|SilenceErrors\s*=\s*true' && ok "cobra SilenceErrors set" \
    || warn "cobra SilenceErrors not set — cobra will print errors itself (double printing)"
  has 'SilenceUsage:\s*true|SilenceUsage\s*=\s*true' && ok "cobra SilenceUsage set" \
    || warn "cobra SilenceUsage not set — usage is dumped on runtime errors"
  has 'SetFlagErrorFunc' && ok "custom flag error func" \
    || info "no SetFlagErrorFunc — consider wrapping flag errors as FlagError"
else
  info "cobra not used — hand-rolled parsing; see references/architecture.md §Migrating"
fi
has 'FlagError' && ok "FlagError-style usage errors present" \
  || warn "no FlagError type — usage vs runtime errors are probably not distinguished"
has 'Cancel(led)?Error|IsUserCancellation' && ok "cancellation error type present" \
  || info "no cancellation error type — Ctrl-C/declined prompts may report as failures"

section "I/O discipline"
grep_check "direct os.Stdout/os.Stderr/os.Stdin outside iostreams/main" \
  '\bos\.(Stdout|Stderr|Stdin)\b' "${APP_SRC[@]}"
grep_check "implicit stdout prints (fmt.Print*/println)" \
  '\bfmt\.Print(f|ln)?\(|^\s*println\(' "${APP_SRC[@]}"
if has 'IsTerminal|isatty|go-gh/v2/pkg/term|IsStdoutTTY'; then ok "TTY detection present"
else warn "no TTY detection — piped output probably identical to terminal output"; fi
if has 'NO_COLOR|go-gh/v2/pkg/term|fatih/color|termenv|lipgloss'; then ok "NO_COLOR-aware color handling (direct or via library)"
elif has '\\033\[|\\x1b\['; then warn "raw ANSI escapes without NO_COLOR handling"
else info "no color usage detected"; fi
has 'CanPrompt|neverPrompt|PROMPT_DISABLED' && ok "prompt gating present" \
  || info "no CanPrompt-style gate found (fine if the tool never prompts)"

section "Dependencies & determinism"
mapfile -t ENV_SRC < <(for f in "${APP_SRC[@]}"; do case "$f" in */config/*) ;; *) echo "$f" ;; esac; done)
grep_check "os.Getenv/LookupEnv scattered in app code (centralize + document precedence)" \
  '\bos\.(Getenv|LookupEnv)\(' "${ENV_SRC[@]}"
grep_check "exec.Command outside a run/exec seam" '\bexec\.Command(Context)?\(' "${APP_SRC[@]}"
grep_check "time.Now() in app code (inject a clock for testable output)" '\btime\.Now\(\)' "${APP_SRC[@]}"
grep_check "http.DefaultClient / http.Get (inject an http.Client)" \
  '\bhttp\.(DefaultClient|Get|Post)\b' "${APP_SRC[@]}"

section "Output contracts"
if has '"json"|AddJSONFlags|Exporter'; then ok "--json / exporter support found"
else warn "no --json support found — list/view commands should offer machine output"; fi
has 'RFC3339' && ok "RFC3339 timestamps used somewhere (piped output)" \
  || info "no RFC3339 formatting — check piped timestamps are absolute"

section "Help & docs"
if has 'github.com/spf13/cobra'; then
  n_cmds=$(grep -hoE '&cobra\.Command\{' "${SRC[@]}" | wc -l | tr -d ' ')
  n_ex=$(grep -hoE '\bExample:' "${SRC[@]}" | wc -l | tr -d ' ')
  # Group/noun commands and help topics don't need examples; leaf commands do.
  if [ "$n_ex" -eq 0 ]; then warn "no cobra command has an Example"
  else info "$n_ex Example fields for $n_cmds cobra commands — check every leaf command has one"; fi
  has 'GenManTree|GenMarkdownTree|gen-docs|cli-docs-tool' && ok "docs generation present" \
    || info "no man/markdown generation from the command tree"
fi
has 'completion' && ok "shell completion referenced" || warn "no shell completion"

section "Config writes"
if has 'os\.WriteFile|ioutil\.WriteFile'; then
  if has 'os\.Rename|CreateTemp|renameio|atomicwriter'; then ok "atomic write pattern present (verify it is used for config/state)"
  else warn "os.WriteFile without temp+rename — config/state writes may corrupt on crash"; fi
elif has 'os\.Rename|CreateTemp'; then ok "writes use temp file + rename"
else info "no file writes detected"; fi

section "Tests"
mapfile -t PKGS < <(for f in "${APP_SRC[@]}"; do dirname "$f"; done | sort -u)
missing=()
for d in "${PKGS[@]}"; do
  compgen -G "$d/*_test.go" >/dev/null || missing+=("$d")
done
if [ "${#missing[@]}" -gt 0 ]; then
  warn "${#missing[@]} package(s) without tests:"
  printf '         %s\n' "${missing[@]}" | head -n "$MAX_HITS"
else ok "every app package has tests"; fi
if [ "${#TESTS[@]}" -gt 0 ]; then
  grep -qE 'SetStdoutTTY|IsTerminal|isTTY|tty' "${TESTS[@]}" 2>/dev/null \
    && ok "tests reference TTY modes" || warn "tests never toggle TTY — piped vs terminal output untested"
  grep -qE 'os\.Stdout\s*=' "${TESTS[@]}" 2>/dev/null \
    && warn "tests swap global os.Stdout — inject IOStreams instead" || ok "tests don't swap os.Stdout"
  grep -qE 'testscript|icmd\.' "${TESTS[@]}" 2>/dev/null \
    && ok "script/e2e tests present" || info "no testscript/e2e tests"
fi

printf '\n%s%d warning(s)%s. Walk references/review-checklist.md for the judgment calls.\n' "$B" "$WARNS" "$R"
[ "$STRICT" -eq 1 ] && [ "$WARNS" -gt 0 ] && exit 1
exit 0
