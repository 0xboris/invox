// Package archtest asserts invox's import layering, the package table of
// docs/design/target/README.md, on the import graph that go list reports.
// depguard in .golangci.yml states the same rules for each file's direct
// imports; these also catch an import that arrives through another package.
package archtest

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const mod = "github.com/0xboris/invox/"

// A pattern is an import path, or a path ending in /... for it and every
// package below it.
type rule struct {
	name   string
	pkgs   []string
	except []string
	// deny applies to every dependency, direct or not.
	deny []string
	// denyImports applies to direct imports only.
	denyImports []string
	// only, when set, lists the only non-standard packages that pkgs may
	// import directly. An empty, non-nil list allows the standard library
	// alone.
	only []string
}

// The rings of docs/design/target/README.md. target_test.go checks the
// target with its own copy of these lists; this file keeps the table once
// the experiment is over.
var (
	ringCore      = []string{mod + "internal/invoice", mod + "internal/numbering", mod + "internal/epc", mod + "internal/money", mod + "internal/billing/..."}
	ringDriven    = []string{mod + "internal/store/...", mod + "internal/archive", mod + "internal/render/...", mod + "internal/email", mod + "internal/adapters/tectonic", mod + "internal/adapters/applemail"}
	ringDriving   = []string{mod + "internal/cli/...", mod + "internal/cmd/...", mod + "internal/tableprinter", mod + "internal/adapters/editor", mod + "internal/adapters/opener"}
	ringLibraries = []string{mod + "internal/config", mod + "internal/fsutil", mod + "internal/adapters/run", mod + "internal/iostreams", mod + "internal/env", mod + "internal/build"}
	ringMain      = []string{mod + "internal/factory/...", mod + "cmd/...", mod + "internal/docs/..."}
	testSupport   = []string{mod + "internal/adapters/run/runtest"}
	frameworkPkgs = []string{"github.com/spf13/cobra/...", "github.com/spf13/pflag/...", "gopkg.in/yaml.v3"}
	diskPkgs      = []string{"os", "io/fs", "os/exec", "os/signal", "net/...", "syscall"}
)

func concat(lists ...[]string) []string { return slices.Concat(lists...) }

var rules = []rule{
	{
		name:        "entities: invoice imports only money and touches no disk",
		pkgs:        []string{mod + "internal/invoice"},
		only:        []string{mod + "internal/money"},
		denyImports: diskPkgs,
	},
	{
		name:        "entities: numbering, epc and money import only the standard library and touch no disk",
		pkgs:        []string{mod + "internal/numbering", mod + "internal/epc", mod + "internal/money"},
		only:        []string{},
		denyImports: diskPkgs,
	},
	{
		name:        "use cases: billing imports only the entities and touches no disk",
		pkgs:        []string{mod + "internal/billing/..."},
		only:        ringCore,
		denyImports: diskPkgs,
	},
	{
		name: "the core depends on no adapter, library, framework, CLI or main package, even indirectly",
		pkgs: ringCore,
		deny: concat(ringDriven, ringDriving, ringLibraries, ringMain, testSupport, frameworkPkgs),
	},
	{
		name: "driven: store imports the core, config, fsutil and yaml.v3",
		pkgs: []string{mod + "internal/store/..."},
		only: concat(ringCore, []string{mod + "internal/store/...", mod + "internal/config", mod + "internal/fsutil", "gopkg.in/yaml.v3"}),
	},
	{
		name: "driven: archive, render/latex and email import the core and fsutil",
		pkgs: []string{mod + "internal/archive", mod + "internal/render/...", mod + "internal/email"},
		only: concat(ringCore, []string{mod + "internal/fsutil"}),
	},
	{
		name: "driven: tectonic and applemail import the core, run and iostreams",
		pkgs: []string{mod + "internal/adapters/tectonic", mod + "internal/adapters/applemail"},
		only: concat(ringCore, []string{mod + "internal/adapters/run", mod + "internal/iostreams"}),
	},
	{
		name: "driving adapters import the core, each other, run, iostreams, env, build, cobra and pflag, never a driven adapter, config, fsutil or factory",
		pkgs: ringDriving,
		only: concat(ringCore, ringDriving, []string{mod + "internal/adapters/run", mod + "internal/iostreams", mod + "internal/env", mod + "internal/build", "github.com/spf13/cobra/...", "github.com/spf13/pflag/..."}),
	},
	{
		name:        "main: factory builds everything but never imports the CLI root or a command",
		pkgs:        []string{mod + "internal/factory/..."},
		denyImports: []string{mod + "internal/cli", mod + "internal/cmd/..."},
	},
	{
		name: "main: cmd/invox imports factory, cli and the process libraries",
		pkgs: []string{mod + "cmd/invox"},
		only: []string{mod + "internal/factory", mod + "internal/cli", mod + "internal/iostreams", mod + "internal/env", mod + "internal/adapters/run"},
	},
	{
		name: "main: docs/gen imports factory, the CLI, cobra and the process libraries",
		pkgs: []string{mod + "internal/docs/gen"},
		only: []string{mod + "internal/factory", mod + "internal/cli", mod + "internal/cli/cmdutil", mod + "internal/cli/helptext", mod + "internal/fsutil", mod + "internal/iostreams", mod + "internal/env", mod + "internal/adapters/run", "github.com/spf13/cobra"},
	},
	{
		name: "libraries import no application package",
		pkgs: ringLibraries,
		only: []string{"gopkg.in/yaml.v3"},
	},
	{
		name: "test support: runtest imports only run",
		pkgs: testSupport,
		only: []string{mod + "internal/adapters/run"},
	},
	{
		name: "the release binary contains neither the docs generator nor runtest",
		pkgs: []string{mod + "cmd/invox"},
		deny: concat([]string{mod + "internal/docs/gen/..."}, testSupport),
	},
	{
		name:   "only the CLI and main know about commands and flag parsing",
		pkgs:   []string{mod + "..."},
		except: concat(ringDriving, ringMain, []string{mod + "internal/archtest"}),
		deny:   []string{mod + "cmd/...", mod + "internal/cli/...", mod + "internal/cmd/...", "github.com/spf13/cobra/...", "github.com/spf13/pflag/...", "flag"},
	},
	{
		name:        "only internal/adapters/run starts programs",
		pkgs:        []string{mod + "..."},
		except:      []string{mod + "internal/adapters/run"},
		denyImports: []string{"os/exec"},
	},
}

// pkg is the part of go list's output the rules read.
type pkg struct {
	ImportPath string
	Standard   bool
	Imports    []string
	Deps       []string
}

func TestImportRules(t *testing.T) {
	t.Parallel()

	graph := loadGraph(t)
	var violations []string
	for _, r := range rules {
		matched, found := check(r, graph)
		if !matched {
			violations = append(violations, r.name+": matches no package; fix or drop the rule")
		}
		violations = append(violations, found...)
	}
	if len(violations) > 0 {
		t.Fatalf("import rules broken:\n%s", strings.Join(violations, "\n"))
	}
}

// TestCommandsImportOnlyTheirOwnSubcommands keeps commands independent: a
// command may import the packages below it and a shared package beside or
// above it, never another command.
func TestCommandsImportOnlyTheirOwnSubcommands(t *testing.T) {
	t.Parallel()

	var violations []string
	for _, p := range loadGraph(t) {
		if !matches(mod+"internal/cmd/...", p.ImportPath) {
			continue
		}
		for _, imp := range p.Imports {
			if !matches(mod+"internal/cmd/...", imp) || matches(p.ImportPath+"/...", imp) {
				continue
			}
			if path.Base(imp) == "shared" && strings.HasPrefix(p.ImportPath, path.Dir(imp)+"/") {
				continue
			}
			violations = append(violations, p.ImportPath+" imports "+imp)
		}
	}
	if len(violations) > 0 {
		t.Fatalf("commands import other commands:\n%s", strings.Join(violations, "\n"))
	}
}

func TestCheckReportsEachKindOfViolation(t *testing.T) {
	t.Parallel()

	graph := []pkg{
		{ImportPath: "fmt", Standard: true},
		{ImportPath: "os/exec", Standard: true},
		{ImportPath: "m/top", Imports: []string{"m/mid"}, Deps: []string{"fmt", "m/leaf", "m/mid"}},
		{ImportPath: "m/mid", Imports: []string{"fmt", "m/leaf", "os/exec"}, Deps: []string{"fmt", "m/leaf", "os/exec"}},
		{ImportPath: "m/leaf", Imports: []string{"fmt"}, Deps: []string{"fmt"}},
	}
	tests := []struct {
		name string
		rule rule
		want []string
	}{
		{
			name: "deny follows the graph",
			rule: rule{pkgs: []string{"m/top"}, deny: []string{"m/leaf"}},
			want: []string{"m/top depends on m/leaf (via m/mid)"},
		},
		{
			name: "denyImports looks at direct imports only",
			rule: rule{pkgs: []string{"m/..."}, denyImports: []string{"os/exec"}},
			want: []string{"m/mid imports os/exec"},
		},
		{
			name: "except skips a package",
			rule: rule{pkgs: []string{"m/..."}, except: []string{"m/mid"}, denyImports: []string{"os/exec"}},
		},
		{
			name: "only allows the standard library and the listed packages",
			rule: rule{pkgs: []string{"m/mid", "m/top"}, only: []string{}},
			want: []string{"m/top imports m/mid; allowed: the standard library", "m/mid imports m/leaf; allowed: the standard library"},
		},
		{
			name: "only with a listed package",
			rule: rule{pkgs: []string{"m/mid"}, only: []string{"m/leaf"}},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			matched, got := check(tc.rule, graph)
			if !matched {
				t.Fatal("rule matched no package")
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("violations = %q, want %q", got, tc.want)
			}
		})
	}
}

// check reports whether r covers any package in graph, and each package that
// breaks it.
func check(r rule, graph []pkg) (bool, []string) {
	byPath := map[string]pkg{}
	for _, p := range graph {
		byPath[p.ImportPath] = p
	}
	matched := false
	var violations []string
	for _, p := range graph {
		if p.Standard || !matchesAny(r.pkgs, p.ImportPath) || matchesAny(r.except, p.ImportPath) {
			continue
		}
		matched = true
		for _, dep := range p.Deps {
			if matchesAny(r.deny, dep) {
				violations = append(violations, p.ImportPath+" depends on "+dep+via(p, dep, byPath))
			}
		}
		for _, imp := range p.Imports {
			if matchesAny(r.denyImports, imp) {
				violations = append(violations, p.ImportPath+" imports "+imp)
			}
			if r.only != nil && !byPath[imp].Standard && !matchesAny(r.only, imp) {
				allowed := strings.Join(append([]string{"the standard library"}, r.only...), ", ")
				violations = append(violations, p.ImportPath+" imports "+imp+"; allowed: "+allowed)
			}
		}
	}
	return matched, violations
}

// via names the direct import of p that pulls in dep, or "" when p imports
// dep itself.
func via(p pkg, dep string, byPath map[string]pkg) string {
	if slices.Contains(p.Imports, dep) {
		return ""
	}
	for _, imp := range p.Imports {
		if slices.Contains(byPath[imp].Deps, dep) {
			return " (via " + imp + ")"
		}
	}
	return ""
}

func matchesAny(patterns []string, importPath string) bool {
	return slices.ContainsFunc(patterns, func(pattern string) bool { return matches(pattern, importPath) })
}

func matches(pattern, importPath string) bool {
	if base, ok := strings.CutSuffix(pattern, "/..."); ok {
		return importPath == base || strings.HasPrefix(importPath, base+"/")
	}
	return importPath == pattern
}

// loadGraph lists the module's packages and everything they import, without
// tests.
func loadGraph(t *testing.T) []pkg {
	t.Helper()

	cmd := exec.Command("go", "list", "-deps", "-json=ImportPath,Standard,Imports,Deps", "./...")
	cmd.Dir = filepath.Join("..", "..")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go list: %v\n%s", err, stderr.String())
	}
	var graph []pkg
	decoder := json.NewDecoder(bytes.NewReader(out))
	for {
		var p pkg
		err := decoder.Decode(&p)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatalf("decode go list output: %v", err)
		}
		graph = append(graph, p)
	}
	return graph
}
