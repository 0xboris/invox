package iostreams

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestOnlySystemTouchesProcessStreams fails when non-test code outside
// System uses os.Stdin, os.Stdout, os.Stderr or fmt.Print*. Everything else
// gets its streams from an IOStreams.
func TestOnlySystemTouchesProcessStreams(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	var violations []string
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			found, err := processStreamUses(path)
			violations = append(violations, found...)
			return err
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("use the IOStreams passed in instead of the process streams:\n%s", strings.Join(violations, "\n"))
	}
}

func processStreamUses(path string) ([]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}
	osName, fmtName := importName(file, "os"), importName(file, "fmt")
	inThisPackage := filepath.Base(filepath.Dir(path)) == "iostreams"

	var violations []string
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && inThisPackage && fn.Recv == nil && fn.Name.Name == "System" {
			continue
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			sel, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok {
				return true
			}
			name := sel.Sel.Name
			isStream := pkg.Name == osName && (name == "Stdin" || name == "Stdout" || name == "Stderr")
			isPrint := pkg.Name == fmtName && (name == "Print" || name == "Printf" || name == "Println")
			if isStream || isPrint {
				violations = append(violations, fset.Position(sel.Pos()).String()+": "+pkg.Name+"."+name)
			}
			return true
		})
	}
	return violations, nil
}

// importName returns the name file refers to the imported package path by,
// or "" when file does not import it.
func importName(file *ast.File, path string) string {
	for _, spec := range file.Imports {
		importPath, err := strconv.Unquote(spec.Path.Value)
		if err != nil || importPath != path {
			continue
		}
		if spec.Name != nil {
			return spec.Name.Name
		}
		return path
	}
	return ""
}
