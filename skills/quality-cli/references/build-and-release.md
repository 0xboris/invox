# Build, release, docs generation, quality gates, background work

## Contents
- [Versioning](#versioning)
- [Build tooling](#build-tooling)
- [Release](#release)
- [Docs generation](#docs-generation)
- [Shell completion](#shell-completion)
- [Lint](#lint)
- [CI](#ci)
- [Deprecation policy](#deprecation-policy)
- [Update notifier](#update-notifier)
- [Telemetry](#telemetry)
- [Repo hygiene for contributors and agents](#repo-hygiene-for-contributors-and-agents)

---

## Versioning

```go
// internal/build/build.go
var Version = "DEV"
var Date = "" // YYYY-MM-DD

func init() {
    if Version == "DEV" {
        if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
            Version = info.Main.Version // `go install module@vX.Y.Z` builds
        }
    }
}
```
```make
VERSION ?= $(shell git describe --tags 2>/dev/null || echo DEV)
DATE    ?= $(shell date -u -d "@$${SOURCE_DATE_EPOCH:-$$(date +%s)}" +%Y-%m-%d 2>/dev/null || date -u +%Y-%m-%d)
LDFLAGS := -X example.com/tool/internal/build.Version=$(VERSION) -X example.com/tool/internal/build.Date=$(DATE)
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/tool ./cmd/tool
```
- `tool --version` / `tool version`: `tool version 2.40.0 (2024-01-15)\nhttps://github.com/org/tool/releases/tag/v2.40.0`.
- SemVer for the CLI surface: minor for features, major for breaking changes to the script-facing contract.
- Reproducible: `-trimpath`, `SOURCE_DATE_EPOCH`.
- Inject build-time constants (OAuth client IDs) the same way, never commit them in code paths that run in forks.

## Build tooling

- gh: `Makefile` delegates to `script/build.go` (a Go task runner) so Windows works without make; skips rebuild when no source is newer than the binary.
- Docker: containerized builds (`docker buildx bake`) with pinned tool versions in small Dockerfiles. Overkill for most tools; pin tool versions anyway (golangci-lint version file).
- Self-documenting Makefile (`make help` greps `## ` comments).
- Targets: `build`, `test`, `lint`, `manpages`, `completions`, `docs`, `acceptance`, `install`.

## Release

GoReleaser (gh `.goreleaser.yml`):
- Builds: linux/darwin/windows × amd64/arm64 (+386/arm if users need it), `CGO_ENABLED=0` unless required.
- `before.hooks`: generate man pages and completions so they ship in archives/packages.
- Archives include `LICENSE`, `share/man/man1/*`, completions.
- `nfpms` for deb/rpm (declare runtime deps, e.g. `git`); sign packages.
- Homebrew tap/formula (gh relies on homebrew-core autobump), Scoop/WinGet for Windows, MSI if enterprise.
- macOS signing + notarization; Windows signing; build provenance (`actions/attest-build-provenance`), checksums.
- Draft release until all artifacts are uploaded; then publish and update docs site.
- Release from tags via CI with an approval gate (environment protection).

## Docs generation

One source of truth: the cobra tree.
- gh `cmd/gen-docs`: builds the real root with stub factory (mock config, no-op browser) and emits `--website` markdown (Jekyll front matter) and `--man-page` (go-md2man, custom sections). Hidden help topics opt in via annotations (`markdown:generate`).
- Docker `docs/generate` + `cli-docs-tool`: per-command `.md` files with **generated blocks between markers** and hand-written examples outside; options table anchors link to example sections. CI regenerates and fails on `git diff` (docs can't drift from flags).
- Recommendation: generate man + markdown (gh), add a CI drift check (docker).
- Also generate: `tool help reference` (full tree in one page) at runtime.

## Shell completion

- `tool completion -s bash|zsh|fish|powershell` (cobra generators; bash V2 with descriptions). Help text explains install per shell. Hide the default cobra `completion` command if you provide your own.
- Without a TTY default to a shell (gh: bash) so `eval "$(tool completion)"` works.
- Dynamic completion mostly through **flag** completion: enum flags (auto), `--json` fields (comma-aware), `-R` from git remotes, branch names, labels/reviewers from API. Use `ValidArgsFunction` for positional targets when cheap (docker `completion.ContainerNames`).
- Default `ShellCompDirectiveNoFileComp` unless the arg is a path.
- Completion must not prompt, print to stdout, or require auth to fail gracefully.
- Ship completions in packages (Debian: install zsh to both `site-functions` and `vendor-completions`).

## Lint

gh `.golangci.yml` (`default: none`, then opt-in): `asasalint`, `asciicheck`, `bidichk`, `bodyclose`, `copyloopvar`, `durationcheck`, `exptostd`, `fatcontext`, `gocheckcompilerdirectives`, `gochecksumtype`, `gocritic`, `gomoddirectives`, `goprintffuncname`, `govet`, `ineffassign`, `nilerr`, `nolintlint`, `nosprintfhostport`, `reassign`, `unused`; formatter `gofmt`.
Docker adds `depguard` (ban packages with named replacements), `forbidigo` (e.g. ban `regexp.MustCompile` at init for startup time), `gosec`, `staticcheck`, `revive`, `errcheck`, `gocyclo`, `importas`, `thelper`, `usetesting`.
Recommended start: gh's set + `staticcheck` + `errcheck` + `gosec`; add `forbidigo` rules for `os.Exit`, `fmt.Print*` and `os.Stdout` outside `main`/`iostreams` to enforce the I/O contract mechanically.

## CI

- `go test -race ./...` + `go build` on ubuntu/macos/windows.
- Lint: `go mod tidy -diff`, `go vet`, golangci-lint (pinned version), `govulncheck` (also scheduled daily).
- Docs/completions drift check.
- CodeQL / security scanning on schedule.
- Pin all GitHub Actions to commit SHAs; `persist-credentials: false` on checkout.
- Acceptance tests on demand or nightly if they hit real services.

## Deprecation policy

- A deprecated flag/command/env var keeps working for **at least one release** (docker `docs/deprecated.md` keeps a table: feature, deprecated-in, removed-in).
- Flags: `MarkDeprecated(name, "use --new instead")` (prints a warning on use, hides from help); compatibility spellings via `MarkHidden`.
- Commands: hidden alias pointing at the new path; warn on stderr.
- Env vars: keep reading; document "(deprecated)" in `help environment`.
- Never change non-TTY output format or JSON field names without a major version.

## Update notifier

gh `internal/update` + `internal/ghcmd`:
```go
updateCtx, updateCancel := context.WithCancel(ctx)
defer updateCancel()
updateMessageChan := make(chan *update.ReleaseInfo)
go func() {
    rel, err := checkForUpdate(updateCtx, f, buildVersion)
    if err != nil && hasDebug { fmt.Fprintf(stderr, "warning: checking for update failed: %v", err) }
    updateMessageChan <- rel
}()
// ... run command ...
updateCancel()                      // abort if not done
newRelease := <-updateMessageChan
if newRelease != nil { /* print to stderr: current → latest, upgrade command, release URL */ }
```
Conditions: `TOOL_NO_UPDATE_NOTIFIER` unset, not CI, not Codespaces-like envs, stdout **and** stderr are TTYs, last check > 24h (timestamp in state dir), build tag enabled for official builds only (`update_enabled.go`/`update_disabled.go`). Tailor the upgrade command to the install method (`brew upgrade tool`); delay Homebrew notices 24h until the formula catches up.

## Telemetry

Only if you need it; be explicit and conservative:
- Controls: `TOOL_TELEMETRY` (`log` = print payload to stderr; falsy = off) > `DO_NOT_TRACK` > config `telemetry`. Auto-off for self-hosted/enterprise and when config fails to load.
- Record command path and **flag names only**, never values, args, alias or extension names. Dimensions: version, OS, is_tty, CI, agent.
- Never block: gh spawns a detached hidden `tool send-telemetry` child reading the payload from stdin; docker flushes OTEL with a 50ms timeout on a fresh context and only reports telemetry errors in debug mode.
- Document in `tool help telemetry`.

## Repo hygiene for contributors and agents

- `docs/command-development.md`-style guide (the Options/Factory/runF pattern), `docs/testing.md`, a design/UX primer.
- `AGENTS.md`/`CLAUDE.md` stating invariants, e.g. gh: "Preserve script-facing contracts: flags, arguments, defaults, exit behavior, error messages, JSON fields, non-TTY output, and stdout/stderr routing" and "bind `opts.BaseRepo = f.BaseRepo` in `RunE`".
- PR template asking for observed behavior (real terminal output for TTY and piped), not just "tests pass".
- CODEOWNERS for sensitive areas (auth, release).
