//go:build target

// The target architecture of the clean-architecture experiment, written as
// tests. docs/design/target/README.md is the spec these check. The file is
// frozen: scripts/verify-target.sh fails if it changes after the commit that
// added it. Run with
//
//	go test -tags target -count=1 ./internal/archtest -run TestTarget -v
package archtest

import (
	"fmt"
	"go/importer"
	"go/token"
	"go/types"
	"reflect"
	"slices"
	"strings"
	"testing"
)

var (
	core       = []string{mod + "internal/invoice", mod + "internal/numbering", mod + "internal/epc", mod + "internal/money", mod + "internal/billing/..."}
	driven     = []string{mod + "internal/store/...", mod + "internal/archive", mod + "internal/render/...", mod + "internal/email", mod + "internal/adapters/tectonic", mod + "internal/adapters/applemail"}
	driving    = []string{mod + "internal/cli/...", mod + "internal/cmd/...", mod + "internal/tableprinter", mod + "internal/adapters/editor", mod + "internal/adapters/opener"}
	libraries  = []string{mod + "internal/config", mod + "internal/fsutil", mod + "internal/adapters/run", mod + "internal/iostreams", mod + "internal/env", mod + "internal/build"}
	frameworks = []string{"github.com/spf13/cobra/...", "github.com/spf13/pflag/...", "gopkg.in/yaml.v3"}
	noDisk     = []string{"os", "io/fs", "os/exec", "os/signal", "net/...", "syscall"}
)

func join(lists ...[]string) []string { return slices.Concat(lists...) }

var targetRules = []rule{
	{
		name:        "entities: invoice imports only money and touches no disk",
		pkgs:        []string{mod + "internal/invoice"},
		only:        []string{mod + "internal/money"},
		denyImports: noDisk,
	},
	{
		name:        "entities: numbering imports only the standard library and touches no disk",
		pkgs:        []string{mod + "internal/numbering"},
		only:        []string{},
		denyImports: noDisk,
	},
	{
		name:        "entities: epc and money import only the standard library and touch no disk",
		pkgs:        []string{mod + "internal/epc", mod + "internal/money"},
		only:        []string{},
		denyImports: noDisk,
	},
	{
		name:        "use cases: billing imports only the entities and touches no disk",
		pkgs:        []string{mod + "internal/billing/..."},
		only:        core,
		denyImports: noDisk,
	},
	{
		name: "the core depends on no adapter, library, framework or CLI package, even indirectly",
		pkgs: core,
		deny: join(driven, driving, libraries, frameworks, []string{mod + "internal/factory/...", mod + "cmd/...", mod + "internal/docs/..."}),
	},
	{
		name: "driven: store imports the core, config, fsutil and yaml.v3",
		pkgs: []string{mod + "internal/store/..."},
		only: join(core, []string{mod + "internal/store/...", mod + "internal/config", mod + "internal/fsutil", "gopkg.in/yaml.v3"}),
	},
	{
		name: "driven: archive, render/latex and email import the core and fsutil",
		pkgs: []string{mod + "internal/archive", mod + "internal/render/...", mod + "internal/email"},
		only: join(core, []string{mod + "internal/fsutil"}),
	},
	{
		name: "driven: tectonic and applemail import the core, run and iostreams",
		pkgs: []string{mod + "internal/adapters/tectonic", mod + "internal/adapters/applemail"},
		only: join(core, []string{mod + "internal/adapters/run", mod + "internal/iostreams"}),
	},
	{
		name: "driving adapters import the core, each other, run, iostreams, env, build, cobra and pflag, never a driven adapter, config, fsutil or factory",
		pkgs: driving,
		only: join(core, driving, []string{mod + "internal/adapters/run", mod + "internal/iostreams", mod + "internal/env", mod + "internal/build", "github.com/spf13/cobra/...", "github.com/spf13/pflag/..."}),
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
		pkgs: libraries,
		only: []string{"gopkg.in/yaml.v3"},
	},
}

func TestTargetImports(t *testing.T) {
	graph := loadGraph(t)
	for _, r := range targetRules {
		t.Run(r.name, func(t *testing.T) {
			matched, found := check(r, graph)
			if !matched {
				t.Fatalf("no package matches %v yet", r.pkgs)
			}
			if len(found) > 0 {
				t.Fatalf("%d violations:\n%s", len(found), strings.Join(found, "\n"))
			}
		})
	}
}

// ports lists, for each interface billing owns, the methods it must have at
// least. Signatures are left to the implementation; names are fixed.
var ports = map[string][]string{
	"Invoices":  {"Load", "Create", "Update"},
	"Directory": {"Customer", "Customers", "Issuer", "Defaults", "Template", "Templates", "Paths", "EditablePath", "Init"},
	"Archive":   {"Entries", "Add", "Checkout"},
	"Renderer":  {"Render"},
	"Mailer":    {"Draft"},
}

var useCases = []string{"New", "Increment", "Validate", "Render", "Build", "Archive", "EditArchived", "DraftEmail", "ListCustomers", "ListArchive", "Paths", "Init", "ListTemplates", "EditablePath"}

// implementations lists which package implements which interface, and the
// package that owns the interface. Compiler belongs to render/latex, the only
// package that calls it; billing never sees a compiler.
var implementations = []struct{ pkg, owner, port string }{
	{"internal/store", "internal/billing", "Invoices"},
	{"internal/store", "internal/billing", "Directory"},
	{"internal/archive", "internal/billing", "Archive"},
	{"internal/render/latex", "internal/billing", "Renderer"},
	{"internal/adapters/tectonic", "internal/render/latex", "Compiler"},
	{"internal/email", "internal/billing", "Mailer"},
	{"internal/adapters/applemail", "internal/billing", "Mailer"},
}

// sourceImporter is shared so that every check sees one copy of each package;
// types from two importers never compare identical.
var sourceImporter = importer.ForCompiler(token.NewFileSet(), "source", nil)

func typecheck(t *testing.T, path string) *types.Package {
	t.Helper()
	p, err := sourceImporter.Import(mod + path)
	if err != nil {
		t.Fatalf("type-check %s: %v", path, err)
	}
	return p
}

func lookupType(p *types.Package, name string) *types.TypeName {
	tn, _ := p.Scope().Lookup(name).(*types.TypeName)
	return tn
}

func methodNames(typ types.Type) []string {
	var names []string
	ms := types.NewMethodSet(typ)
	for i := range ms.Len() {
		names = append(names, ms.At(i).Obj().Name())
	}
	return names
}

func TestTargetPorts(t *testing.T) {
	billing := typecheck(t, "internal/billing")
	if billing.Scope().Lookup("Compiler") != nil {
		t.Error("billing.Compiler exists; the compiler interface belongs to render/latex")
	}
	for name, want := range ports {
		t.Run(name, func(t *testing.T) {
			tn := lookupType(billing, name)
			if tn == nil {
				t.Fatalf("billing.%s is not declared", name)
			}
			iface, ok := tn.Type().Underlying().(*types.Interface)
			if !ok || tn.IsAlias() {
				t.Fatalf("billing.%s must be an interface declared in billing, got %s", name, tn.Type())
			}
			var have []string
			for i := range iface.NumMethods() {
				have = append(have, iface.Method(i).Name())
			}
			for _, m := range want {
				if !slices.Contains(have, m) {
					t.Errorf("billing.%s has no method %s (has %v)", name, m, have)
				}
			}
		})
	}
}

func TestTargetService(t *testing.T) {
	billing := typecheck(t, "internal/billing")
	tn := lookupType(billing, "Service")
	if tn == nil {
		t.Fatal("billing.Service is not declared")
	}
	st, ok := tn.Type().Underlying().(*types.Struct)
	if !ok {
		t.Fatalf("billing.Service must be a struct, got %s", tn.Type().Underlying())
	}
	for name := range ports {
		port := lookupType(billing, name)
		found := false
		for i := range st.NumFields() {
			if port != nil && types.Identical(st.Field(i).Type(), port.Type()) {
				found = true
			}
		}
		if !found {
			t.Errorf("billing.Service has no field of type billing.%s", name)
		}
	}
	have := methodNames(types.NewPointer(tn.Type()))
	for _, m := range useCases {
		if !slices.Contains(have, m) {
			t.Errorf("billing.Service has no use case method %s", m)
		}
	}
}

func TestTargetImplementations(t *testing.T) {
	for _, impl := range implementations {
		t.Run(impl.pkg+" implements "+impl.port, func(t *testing.T) {
			owner := typecheck(t, impl.owner)
			port := lookupType(owner, impl.port)
			if port == nil {
				t.Fatalf("%s.%s is not declared", owner.Name(), impl.port)
			}
			iface, ok := port.Type().Underlying().(*types.Interface)
			if !ok {
				t.Fatalf("%s.%s is not an interface", owner.Name(), impl.port)
			}
			p := typecheck(t, impl.pkg)
			for _, name := range p.Scope().Names() {
				tn, ok := p.Scope().Lookup(name).(*types.TypeName)
				if !ok || !tn.Exported() || tn.IsAlias() || types.IsInterface(tn.Type()) {
					continue
				}
				if types.Implements(tn.Type(), iface) || types.Implements(types.NewPointer(tn.Type()), iface) {
					return
				}
			}
			t.Fatalf("no exported concrete type in %s implements %s.%s", impl.pkg, owner.Name(), impl.port)
		})
	}
}

// TestTargetEntities checks that invoice holds plain domain types: the named
// entities exist, nothing carries a YAML tag, the old Host facade is gone, and
// validation returns []Problem.
func TestTargetEntities(t *testing.T) {
	inv := typecheck(t, "internal/invoice")
	for _, name := range []string{"Invoice", "Position", "Customer", "Issuer", "Payment", "Status", "Problem"} {
		if lookupType(inv, name) == nil {
			t.Errorf("invoice.%s is not declared", name)
		}
	}
	if inv.Scope().Lookup("Host") != nil {
		t.Error("invoice.Host still exists; path resolution belongs to store")
	}
	var tagged []string
	for _, name := range inv.Scope().Names() {
		tn, ok := inv.Scope().Lookup(name).(*types.TypeName)
		if !ok {
			continue
		}
		st, ok := tn.Type().Underlying().(*types.Struct)
		if !ok {
			continue
		}
		n := 0
		for i := range st.NumFields() {
			if _, has := reflect.StructTag(st.Tag(i)).Lookup("yaml"); has {
				n++
			}
		}
		if n > 0 {
			tagged = append(tagged, fmt.Sprintf("%s (%d)", name, n))
		}
	}
	if len(tagged) > 0 {
		t.Errorf("yaml tags remain on invoice types; decoding belongs to store: %s", strings.Join(tagged, ", "))
	}
	if problem := lookupType(inv, "Problem"); problem != nil && !returnsProblems(inv, problem) {
		t.Error("nothing in invoice named Validate returns []Problem")
	}
}

func returnsProblems(p *types.Package, problem *types.TypeName) bool {
	want := types.NewSlice(problem.Type())
	ok := func(sig *types.Signature) bool {
		r := sig.Results()
		return r.Len() > 0 && types.Identical(r.At(0).Type(), want)
	}
	if f, isFunc := p.Scope().Lookup("Validate").(*types.Func); isFunc && ok(f.Type().(*types.Signature)) {
		return true
	}
	for _, name := range p.Scope().Names() {
		tn, isType := p.Scope().Lookup(name).(*types.TypeName)
		if !isType {
			continue
		}
		ms := types.NewMethodSet(types.NewPointer(tn.Type()))
		if sel := ms.Lookup(p, "Validate"); sel != nil && ok(sel.Obj().Type().(*types.Signature)) {
			return true
		}
	}
	return false
}

// TestTargetNoBorrowedTypes stops the core from re-exporting adapter or
// library types through aliases, which would satisfy the import rules on paper
// only.
func TestTargetNoBorrowedTypes(t *testing.T) {
	for _, path := range []string{"internal/invoice", "internal/billing"} {
		p := typecheck(t, path)
		for _, name := range p.Scope().Names() {
			tn, ok := p.Scope().Lookup(name).(*types.TypeName)
			if !ok || !tn.IsAlias() {
				continue
			}
			named, ok := types.Unalias(tn.Type()).(*types.Named)
			if !ok || named.Obj().Pkg() == nil {
				continue
			}
			from := named.Obj().Pkg().Path()
			if !matchesAny(core, from) && strings.Contains(from, ".") {
				t.Errorf("%s.%s aliases %s from outside the core", path, name, fmt.Sprintf("%s.%s", from, named.Obj().Name()))
			}
		}
	}
}
