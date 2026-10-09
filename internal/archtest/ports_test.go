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
// serviceMethods that of *billing.Service. docs/design/ports-narrowing.md
// explains each method; a new one belongs there first.
var (
	narrowPorts = map[string][]string{
		"Invoices":  {"Create", "Drafts", "Load", "Update"},
		"Directory": {"Customer", "Customers", "Defaults", "EditablePath", "Init", "Issuer", "Locate", "Paths", "Template", "Templates"},
		"Archive":   {"Add", "Checkout", "Dir", "Duplicate", "Entries", "Place", "Source"},
		"Renderer":  {"Build", "Render", "Write"},
		"Compiler":  {"Compile"},
		"Mailer":    {"Check", "CheckAttachment", "Draft"},
	}
	serviceMethods = []string{
		"Archive", "Build", "CheckNumberUnique", "DefaultTemplate", "DraftEmail",
		"EditArchived", "EditablePath", "Increment", "Init",
		"ListArchive", "ListCustomers", "ListTemplates", "New", "NextNumber", "Paths",
		"Render", "Validate",
	}
)

func TestPortMethodSets(t *testing.T) {
	billing, err := importer.ForCompiler(token.NewFileSet(), "source", nil).Import(mod + "internal/billing")
	if err != nil {
		t.Fatalf("type-check billing: %v", err)
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
