# invox: build, release, lint, CI and docs audit

## (a) Verdict
invox compiles cleanly for linux/darwin/windows on amd64 and arm64, builds byte-for-byte reproducibly with `-trimpath`, really does build with Go 1.22, passes `go vet`/`gofmt`, and has a single small dependency. It is not shippable yet. There is no version command. The module path `invox` makes `go install …@latest` fail. There is no CI (so a red `go test ./...` on Linux got merged), no linter config, no LICENSE (also for the redistributed Ubuntu fonts), and no release pipeline. The README is mostly accurate but carries a stale "Last reviewed" date and leaves out install-by-path, exit codes, env vars and licensing. The Makefile can silently run a stale binary.

## (b) Findings

**1. [blocker] No version reporting. VERIFIED**
Rule: "`--help`, `version` and completion work offline"; "`tool --version` / `tool version`" with ldflags plus a `debug.ReadBuildInfo` fallback.
Evidence: `invox --version`, `invox version` and `invox -v` each print `error: unknown subcommand "…"` followed by 80 lines of root help on stderr, then exit 2. No `Version` variable exists anywhere in the code. The binary already carries `mod invox v0.0.0-20261002115421-c345065516b5` and `vcs.revision` (from `go version -m`), so a ReadBuildInfo fallback would work right away.
Fix: snippet C4.

**2. [blocker] Module path is not go-installable. VERIFIED**
Evidence: `go install github.com/0xboris/invox/cmd/invox@latest` fails with `module declares its path as: invox but was required as: github.com/0xboris/invox`. Today the only install route is a local checkout. It also blocks pkg.go.dev, the `go install` route for GoReleaser/Homebrew, and govulncheck by module.
Fix: set `module github.com/0xboris/invox` and rewrite 10 imports (`invox/internal/…`) in 10 files with `gofmt -r` or sed. Then tag `v0.1.0`.

**3. [blocker] No CI; tests are red on main. VERIFIED**
Rule: "`go test -race ./...` + `go build` on ubuntu/macos/windows; lint; tidy; govulncheck; docs drift."
Evidence: there is no `.github/`. `go test ./...` fails with `TestEditableConfigPathCreatesCommentedTemplate` (it expects `~/Library/Application Support/invox/invoices` on Linux). A Linux CI job would have blocked the merge.
Fix: snippet C2.

**4. [major] No LICENSE, and the Ubuntu fonts ship without their licence. VERIFIED**
Evidence: `ls LICENSE*` finds nothing. `fonts/Ubuntu-*.ttf` (1.4 MB, added in the initial commit) has no UFL-1.0 text, which the Ubuntu Font Licence requires with every copy. The fonts are **not** embedded (`go:embed` lists only `starter/*.yaml` and `template.tex`). The starter template does not use them either (no `fontspec`/`\setmainfont`), and `invox init` does not install them. At runtime, `copyTemplateAssets` copies `Path=fonts/` relative to the *user's template dir*, so this repo's `fonts/` is unused.
Fix: add LICENSE (MIT or Apache-2.0), plus `fonts/UFL.txt`. Or delete `fonts/` and document how to get them. If fonts should ship, embed them and have `init` write them out.

**5. [major] No release pipeline, tags or changelog. VERIFIED**
Evidence: `git tag` is empty. There is no `.goreleaser.yaml` and no CHANGELOG. The Makefile builds without `-trimpath`, ldflags or `CGO_ENABLED=0`; the default build records `CGO_ENABLED=1` and 22 embedded `/home/user/invox` paths. With `-trimpath CGO_ENABLED=0` the path count is 0 and two builds hash identically (`45857bc1…`). All 6 GOOS/GOARCH targets build, and `GOOS=windows|darwin go vet ./...` (tests included) passes.
Fix: snippet C3. Release from `v*` tags via CI.

**6. [major] No lint config; dead code and a layering leak. VERIFIED**
Rule: "gh's set + staticcheck + errcheck + gosec; forbidigo for `os.Exit`, `fmt.Print*`, `os.Stdout` outside main/iostreams"; "domain packages never… print, read env".
Evidence: staticcheck and golangci-lint v2.5.0 were both run with config C1 on a copy (33 issues):
- 6 unused funcs: `renameMappingKey`, `nodeIsEmpty`, `invoiceEmailBody`, `firstPresentPath`, `firstNonEmptyPath`, `prependPath`.
- An ineffectual `archivePath` assignment (drafts.go:292).
- `nilerr` at numbering.go:204, which swallows the parse error; it is probably meant to skip non-matching files, so it needs a comment.
- ST1005: a multi-line error string that hard-codes `brew install tectonic` (service.go:936), which is wrong advice on Linux and Windows.
- 23 forbidigo hits: 17 `fmt.Printf/Println` in `internal/cli/commands_*.go`, and in the **domain** package `internal/invoice/service.go:941-943`, which wires tectonic to `os.Stdout`/`os.Stdin` (compiler chatter lands on the data stream) and reads `XDG_CONFIG_HOME`, `XDG_DATA_HOME` and `APPDATA` itself.
- errcheck, gosec (with the listed excludes) and depguard report nothing: `internal/invoice` does not import `internal/cli`.
Fix: commit C1, delete the dead code, inject streams and env.

**7. [major] Makefile `$(BINARY_PATH)` rule goes stale. VERIFIED**
Evidence: in a copy I ran `make build`, edited `internal/cli/help.go`, and ran `make build` again. It printed "Nothing to be done for 'build'", and `./bin/invox help` still showed the old text, while a fresh `go build` showed the edit. Every workflow target (`validate`, `render`, `pdf`, `archive`, …) depends on this file target, so it runs the stale binary.
Further issues (READ):
- 13 variables mirror CLI flags, with no tests.
- `make email` always passes `-o $(EMAIL_OUTPUT)`, which per `invox email --help` turns off the macOS Apple Mail compose path. The Makefile behaves differently from the CLI it wraps.
- `make pdf CLI=invox` still builds `bin/`.
- There are no `lint`/`fmt`/`tidy` targets.
Fix: snippet C5. Keep only dev targets; drop the workflow wrappers.

**8. [minor] `go.sum` not tidy. VERIFIED**
Evidence: `go mod tidy -diff` exits 1 and adds the `gopkg.in/check.v1 … h1:` line. `go mod verify` reports "all modules verified". govulncheck is installed, but the proxy blocks `vuln.go.dev` (403), so it should run in CI.

**9. [minor] go.mod says `go 1.22`, toolchain is go1.24.7. VERIFIED**
`GOTOOLCHAIN=go1.22.12 go build ./... && go vet ./...` passes, and no 1.23+ APIs were found by grep, so the README's "Go 1.22+" is honest. 1.22 is past end of support (only 1.24 and 1.25 are supported). Bump to `go 1.24` when convenient, and keep a matrix job on the minimum version as long as you claim it.

**10. [major] README gaps. VERIFIED against `--help`**
Every command and flag in the README tables exists and matches its help: init, config, customer list/config, template list, completion zsh, new, increment, validate, render, build, email, archive, archive list/edit, and help config/customers/issuer/defaults/template. Gaps:
- "Last reviewed: 2026-03-30" predates #5, #6 (line-item blocks, not in the README) and #8.
- Install covers only `go install ./cmd/invox`; there is no remote install.
- Tectonic install is given only for macOS.
- No exit codes: usage errors return 2 and runtime errors 1, and `help exit-codes` does not exist.
- No env vars: `XDG_CONFIG_HOME`, `XDG_DATA_HOME`, `APPDATA`, `VISUAL`/`EDITOR` (`help environment` does not exist).
- The hidden `send` alias is undocumented.
- Missing sections: invoice YAML reference, troubleshooting, License, Contributing.
- `completion` supports zsh only.
- Docs are not generated: help is about 850 lines of hand-written `Fprintf` in `help.go`, alongside a `commandSpec` table.
Fix: add those sections; generate `docs/invox.md` and a man page from `commandSpec`/help output (`go run ./internal/cli/gendocs`) and check for drift in CI.

**11. [minor] Repo hygiene. VERIFIED**
- `.gitignore` is just `bin/`.
- 84 of 126 tracked files are agent skills under `.agents/`:
  - `latex-expert`: 47 files, 756 KB, relevant to template work.
  - `coding-standards`: TypeScript/React, 32 hits for typescript|react, not relevant.
  - `cli-guidline` (sic) and `golang-patterns` overlap `quality-cli`.
- `skills-lock.json` pins only 2 of the 7 skills.
- `.claude/skills/quality-cli` is a **symlink** to `.agents/skills/quality-cli` (`diff -r` shows them identical), so there is no duplication. Symlinks break on Windows checkouts without `core.symlinks`.
- `features/multi_vat/prd.md` (376 lines) describes a feature already shipped in #1.

Fix:
- Drop `coding-standards`, merge `cli-guidline` into `quality-cli`, and lock or vendor the rest consistently.
- Move the PRD to `docs/design/`.
- Ignore: `/invox`, `/dist/`, `*.test`, `coverage.out`, `/invoice.tex`, `/*.pdf`, `*.eml`, `*.aux`, `*.log`, `*.xdv`, `.DS_Store`. Keep `internal/invoice/starter/*.tex` tracked.

## (c) Snippets

### C1 `.golangci.yml` (golangci-lint v2, tested)
```yaml
version: "2"
run: { go: "1.22" }
linters:
  default: none
  enable: [asasalint, asciicheck, bidichk, bodyclose, copyloopvar, durationcheck,
    fatcontext, gocheckcompilerdirectives, gocritic, gomoddirectives, goprintffuncname,
    govet, ineffassign, nilerr, nolintlint, reassign, unused,
    staticcheck, errcheck, gosec, depguard, forbidigo]
  settings:
    depguard:
      rules:
        domain-is-a-leaf:
          files: ["**/internal/invoice/**"]
          deny:
            - { pkg: "github.com/0xboris/invox/internal/cli", desc: "domain must not import the CLI layer" }
            - { pkg: "invox/internal/cli", desc: "domain must not import the CLI layer" }
            - { pkg: "flag", desc: "flag parsing belongs in internal/cli" }
    forbidigo:
      analyze-types: true
      forbid:
        - { pattern: '^fmt\.Print(f|ln)?$', msg: "write to an injected io.Writer" }
        - { pattern: '^os\.Exit$', msg: "return an exit code; only cmd/invox/main.go exits" }
        - { pattern: '^os\.(Stdout|Stderr|Stdin)$', msg: "use injected streams" }
        - { pattern: '^os\.(Getenv|LookupEnv)$', msg: "read env in one place and pass it down" }
    gosec:
      excludes: [G204, G304, G306, G301]   # exec of tectonic/editor, user paths, 0644 docs
  exclusions:
    presets: [comments, std-error-handling]
    rules:
      - { path: _test\.go, linters: [forbidigo, gosec, errcheck] }
      - { path: ^cmd/invox/main\.go$, linters: [forbidigo] }
      - { path: ^internal/cli/streams\.go$, linters: [forbidigo] }  # IOStreams constructor
formatters:
  enable: [gofmt, goimports]
```

### C2 `.github/workflows/ci.yml`
```yaml
name: ci
on: { push: { branches: [main] }, pull_request: {}, schedule: [{ cron: "17 5 * * *" }] }
permissions: { contents: read }
jobs:
  test:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
        go: ["1.22", stable]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4          # pin to commit SHA
        with: { persist-credentials: false }
      - uses: actions/setup-go@v5          # pin to commit SHA
        with: { go-version: "${{ matrix.go }}" }
      - run: go build ./...
      - run: go test -race ./...
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with: { persist-credentials: false }
      - uses: actions/setup-go@v5
        with: { go-version: stable }
      - run: go mod tidy -diff
      - run: go vet ./...
      - uses: golangci/golangci-lint-action@v8
        with: { version: v2.5.0 }
      - run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
      - name: docs drift
        run: go run ./internal/cli/gendocs -out docs && git diff --exit-code docs
```

### C3 `.goreleaser.yaml`
```yaml
version: 2
before:
  hooks: [go mod tidy, go run ./internal/cli/gendocs -out docs -man share/man/man1]
builds:
  - main: ./cmd/invox
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    flags: [-trimpath]
    mod_timestamp: "{{ .CommitTimestamp }}"
    ldflags:
      - -s -w -X github.com/0xboris/invox/internal/build.Version={{.Version}}
      - -X github.com/0xboris/invox/internal/build.Date={{.CommitDate}}
archives:
  - formats: [tar.gz]
    format_overrides: [{ goos: windows, formats: [zip] }]
    files: [LICENSE, README.md, share/man/man1/*]
checksum: { name_template: checksums.txt }
changelog: { use: github }
release: { draft: true }
brews:
  - repository: { owner: 0xboris, name: homebrew-tap }
    dependencies: [{ name: tectonic }]
```

### C4 Version wiring
```go
// internal/build/build.go
package build

import "runtime/debug"

var (
	Version = "DEV"
	Date    = "" // YYYY-MM-DD
)

func init() {
	if Version != "DEV" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
}
```
```go
// internal/cli/cli.go, at the top of Run, before help handling
if args[0] == "version" || args[0] == "--version" {
	fmt.Fprintf(stdout, "invox version %s", strings.TrimPrefix(build.Version, "v"))
	if build.Date != "" {
		fmt.Fprintf(stdout, " (%s)", build.Date)
	}
	fmt.Fprintf(stdout, "\nhttps://github.com/0xboris/invox/releases/tag/%s\n", build.Version)
	return 0
}
```
`cmd/invox/main.go` stays `os.Exit(cli.Run(os.Args[1:]))`.

### C5 Makefile (dev targets only)
```make
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo DEV)
DATE    ?= $(shell date -u -d "@$${SOURCE_DATE_EPOCH:-$$(date +%s)}" +%Y-%m-%d 2>/dev/null || date -u +%Y-%m-%d)
PKG     := github.com/0xboris/invox
LDFLAGS := -X $(PKG)/internal/build.Version=$(VERSION) -X $(PKG)/internal/build.Date=$(DATE)
.PHONY: help build test vet lint fmt tidy install clean
build: ## Build ./bin/invox (always; Go's cache makes it cheap; never stale)
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/invox ./cmd/invox
test:    ## Run tests with the race detector
	go test -race ./...
vet:     ; go vet ./...
lint:    ; golangci-lint run
fmt:     ; gofmt -w . && go mod tidy
tidy:    ; go mod tidy -diff
install: ; go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/invox
clean:   ; rm -rf bin dist
```
Drop `validate`, `render`, `email`, `send`, `pdf`, `archive` and `init`: they duplicate the CLI's flags, they diverge (`email -o`), and the README already says to prefer `invox …`.
