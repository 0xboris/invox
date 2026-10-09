package archtest

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// narrowPorts is the exact method set of each port billing owns, and
// serviceMethods that of *billing.Service. docs/design/ports-v2.md
// explains each method; a new one belongs there first. The compiler is
// render/latex's, not a billing port.
var (
	narrowPorts = map[string][]string{
		"Invoices":  {"Create", "Drafts", "Load", "Update"},
		"Directory": {"Customer", "Customers", "Defaults", "EditablePath", "Init", "Issuer", "Locate", "Paths", "Template", "Templates"},
		"Archive":   {"Add", "Checkout", "Duplicate", "Entries", "Source"},
		"Renderer":  {"Build", "Render", "Write"},
		"Mailer":    {"Draft"},
	}
	serviceMethods = []string{
		"Archive", "Build", "DraftEmail", "EditArchived", "EditablePath", "Increment", "Init",
		"ListArchive", "ListCustomers", "ListTemplates", "New", "Paths", "Render", "Validate",
	}
)

func TestPortMethodSets(t *testing.T) {
	billing, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(mod + "internal/billing")
	if err != nil {
		t.Fatalf("type-check billing: %v", err)
	}
	if billing.Scope().Lookup("Compiler") != nil {
		t.Error("billing.Compiler is declared; the renderer owns its compiler")
	}
	for name, want := range narrowPorts {
		tn, _ := billing.Scope().Lookup(name).(*types.TypeName)
		if tn == nil {
			t.Errorf("billing.%s is not declared", name)
			continue
		}
		iface, ok := tn.Type().Underlying().(*types.Interface)
		if !ok {
			t.Errorf("billing.%s is not an interface", name)
			continue
		}
		var have []string
		for i := range iface.NumMethods() {
			have = append(have, iface.Method(i).Name())
		}
		if !slices.Equal(have, want) {
			t.Errorf("billing.%s methods = %v, want %v", name, have, want)
		}
	}
	tn, _ := billing.Scope().Lookup("Service").(*types.TypeName)
	if tn == nil {
		t.Fatal("billing.Service is not declared")
	}
	ms := types.NewMethodSet(types.NewPointer(tn.Type()))
	var have []string
	for i := range ms.Len() {
		if ms.At(i).Obj().Exported() {
			have = append(have, ms.At(i).Obj().Name())
		}
	}
	slices.Sort(have)
	if !slices.Equal(have, serviceMethods) {
		t.Errorf("*billing.Service methods = %v, want %v", have, serviceMethods)
	}
}

// fileExtension matches a string literal that names a file extension, the
// sign of a use case deriving a path by hand.
var fileExtension = regexp.MustCompile(`\.(?i:ya?ml|pdf|eml|tex|md|markdown)\b`)

func TestBillingHandlesNoPaths(t *testing.T) {
	dir := filepath.Join("..", "billing")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range file.Imports {
			if path, _ := strconv.Unquote(imp.Path.Value); path == "path" || path == "path/filepath" {
				t.Errorf("%s imports %s; a driven adapter builds paths", name, path)
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.STRING && fileExtension.MatchString(lit.Value) {
				t.Errorf("%s: string %s names a file extension; a driven adapter names files", fset.Position(lit.Pos()), lit.Value)
			}
			return true
		})
	}
}

// TestPortsCarryNoAdapterData checks the two ways adapter data has passed
// through billing. A port result that billing hands back to the same port
// (Placement from Archive.Place to Archive.Add) is the adapter's own state
// taking a detour. A func in a port result (Template.FindAsset,
// Draft.Discard) is adapter behavior that billing carries for someone else.
func TestPortsCarryNoAdapterData(t *testing.T) {
	billing, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(mod + "internal/billing")
	if err != nil {
		t.Fatalf("type-check billing: %v", err)
	}
	for name := range narrowPorts {
		iface := billing.Scope().Lookup(name).Type().Underlying().(*types.Interface)
		results, params := map[string]bool{}, map[string]bool{}
		for i := range iface.NumMethods() {
			sig := iface.Method(i).Type().(*types.Signature)
			for j := range sig.Results().Len() {
				collect(sig.Results().At(j).Type(), billing, results)
				if path := funcField(sig.Results().At(j).Type(), map[types.Type]bool{}); path != "" {
					t.Errorf("billing.%s.%s returns %s, a func: adapter behavior passing through billing", name, iface.Method(i).Name(), path)
				}
			}
			for j := range sig.Params().Len() {
				collect(sig.Params().At(j).Type(), billing, params)
			}
		}
		for typ := range results {
			if params[typ] {
				t.Errorf("billing.%s both returns and takes billing.%s: the adapter's own data takes a detour through billing", name, typ)
			}
		}
	}
}

// collect adds the billing struct types t names, through pointers and
// slices, to into.
func collect(t types.Type, billing *types.Package, into map[string]bool) {
	switch t := t.(type) {
	case *types.Pointer:
		collect(t.Elem(), billing, into)
	case *types.Slice:
		collect(t.Elem(), billing, into)
	case *types.Named:
		if _, ok := t.Underlying().(*types.Struct); ok && t.Obj().Pkg() == billing {
			into[t.Obj().Name()] = true
		}
	}
}

// funcField returns the field path of the first func-typed field in t,
// following structs, pointers and slices, or "".
func funcField(t types.Type, seen map[types.Type]bool) string {
	if seen[t] {
		return ""
	}
	seen[t] = true
	switch u := t.(type) {
	case *types.Pointer:
		return funcField(u.Elem(), seen)
	case *types.Slice:
		return funcField(u.Elem(), seen)
	case *types.Named:
		st, ok := u.Underlying().(*types.Struct)
		if !ok {
			return ""
		}
		for i := range st.NumFields() {
			f := st.Field(i)
			if _, isFunc := f.Type().Underlying().(*types.Signature); isFunc {
				return u.Obj().Name() + "." + f.Name()
			}
			if path := funcField(f.Type(), seen); path != "" {
				return path
			}
		}
	}
	return ""
}
