package archtest

import (
	"go/importer"
	"go/token"
	"go/types"
	"slices"
	"testing"
)

// narrowPorts is the exact method set of each port billing owns, and
// serviceMethods that of *billing.Service. docs/design/ports-narrowing.md
// explains each method; a new one belongs there first.
var (
	narrowPorts = map[string][]string{
		"Invoices":  {"ArchivedHead", "CheckOutput", "Create", "Drafts", "Exists", "Head", "Load", "Stat", "Update"},
		"Directory": {"CopyLegacy", "Customer", "Customers", "Defaults", "EditablePath", "Init", "Issuer", "LegacyFiles", "LegacyFilesUsed", "Locate", "Locations", "Paths", "Template", "Templates"},
		"Archive":   {"Add", "Checkout", "Dir", "Entries", "Existing", "FindFile", "HistoryDir", "Protects", "Resolve"},
		"Renderer":  {"Build", "Render", "Write"},
		"Compiler":  {"Compile"},
		"Mailer":    {"Check", "Draft"},
	}
	serviceMethods = []string{
		"Archive", "Build", "CheckNumberUnique", "CopyLegacyFiles", "DefaultTemplate", "DraftEmail",
		"EditArchived", "EditablePath", "EmailPaths", "Increment", "Init", "LegacyFiles", "LegacyFilesUsed",
		"ListArchive", "ListCustomers", "ListTemplates", "Locations", "New", "NextNumber", "Paths",
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
