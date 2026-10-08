// Package archtest asserts invox's import layering on the import graph that
// go list reports. depguard in .golangci.yml states the same rules for each
// file's direct imports; these also catch an import that arrives through
// another package.
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

var rules = []rule{
	{
		name: "the release binary does not contain the docs generator",
		pkgs: []string{mod + "cmd/invox"},
		deny: []string{mod + "internal/docs/gen/..."},
	},
	{
		name:   "nothing below the command layer depends on commands, the CLI or flag parsing",
		pkgs:   []string{mod + "..."},
		except: []string{mod + "cmd/...", mod + "internal/cli/...", mod + "internal/cmd/...", mod + "internal/docs/gen/...", mod + "internal/archtest"},
		deny:   []string{mod + "cmd/...", mod + "internal/cli/...", mod + "internal/cmd/...", "github.com/spf13/cobra/...", "github.com/spf13/pflag/...", "flag"},
	},
	{
		name: "the domain neither writes to the terminal nor runs programs",
		pkgs: []string{mod + "internal/archive", mod + "internal/config", mod + "internal/email", mod + "internal/epc", mod + "internal/invoice/...", mod + "internal/money", mod + "internal/numbering", mod + "internal/render/..."},
		deny: []string{mod + "internal/iostreams", mod + "internal/tableprinter", mod + "internal/adapters/..."},
	},
	{
		name: "adapters import only run and iostreams",
		pkgs: []string{mod + "internal/adapters/...", mod + "internal/tableprinter"},
		only: []string{mod + "internal/adapters/run", mod + "internal/iostreams"},
	},
	{
		name:        "only internal/adapters/run starts programs",
		pkgs:        []string{mod + "..."},
		except:      []string{mod + "internal/adapters/run"},
		denyImports: []string{"os/exec"},
	},
	{
		name: "leaf packages import only the standard library",
		pkgs: []string{mod + "internal/build", mod + "internal/email", mod + "internal/env", mod + "internal/epc", mod + "internal/fsutil", mod + "internal/iostreams", mod + "internal/money", mod + "internal/numbering"},
		only: []string{},
	},
	{
		name: "internal/render/latex imports only money",
		pkgs: []string{mod + "internal/render/latex"},
		only: []string{mod + "internal/money"},
	},
	{
		name: "internal/archive imports only fsutil",
		pkgs: []string{mod + "internal/archive"},
		only: []string{mod + "internal/fsutil"},
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
