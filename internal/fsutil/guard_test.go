package fsutil

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// fileWrites lists, per import path, the package-level identifiers that
// create, replace or rename files. A dot import of any of these paths counts
// too, since its uses can't be told apart from local names.
var fileWrites = map[string][]string{
	"os":        {"WriteFile", "Create", "CreateTemp", "OpenFile", "Rename", "Link"},
	"io/ioutil": {"WriteFile", "TempFile"},
}

// TestOnlyFsutilWritesFiles fails when non-test code under cmd/ or internal/
// outside internal/fsutil uses one of fileWrites. Everything else writes
// through fsutil, so writes stay atomic and keep modes and symlinks.
func TestOnlyFsutilWritesFiles(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	var violations []string
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if filepath.ToSlash(rel) == "internal/fsutil" {
					return filepath.SkipDir
				}
				return nil
			}
			if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			violations = append(violations, fileWriteUses(fset, file)...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("write files through internal/fsutil instead:\n%s", strings.Join(violations, "\n"))
	}
}

func TestFileWriteUsesFindsEveryForm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "call",
			src:  "import \"os\"\n\nfunc f() error { return os.WriteFile(\"a\", nil, 0o644) }\n",
			want: []string{"os.WriteFile"},
		},
		{
			name: "aliased import and function value",
			src:  "import stdos \"os\"\n\nvar rename = stdos.Rename\n",
			want: []string{"stdos.Rename"},
		},
		{
			name: "dot import",
			src:  "import . \"os\"\n\nfunc f() { _, _ = Create(\"a\") }\n",
			want: []string{`import . "os"`},
		},
		{
			name: "ioutil",
			src:  "import \"io/ioutil\"\n\nfunc f() error { return ioutil.WriteFile(\"a\", nil, 0o644) }\n",
			want: []string{"ioutil.WriteFile"},
		},
		{
			name: "every os write",
			src:  "import \"os\"\n\nvar _ = []any{os.Create, os.CreateTemp, os.OpenFile, os.Link}\n",
			want: []string{"os.Create", "os.CreateTemp", "os.OpenFile", "os.Link"},
		},
		{
			name: "reads, removes and directories",
			src:  "import \"os\"\n\nfunc f() { _, _ = os.ReadFile(\"a\"); _ = os.Remove(\"a\"); _ = os.MkdirAll(\"d\", 0o755); _, _ = os.Open(\"a\") }\n",
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "probe.go", "package probe\n\n"+tc.src, 0)
			if err != nil {
				t.Fatalf("parse probe: %v", err)
			}
			var got []string
			for _, violation := range fileWriteUses(fset, file) {
				got = append(got, violation[strings.Index(violation, ": ")+2:])
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("violations = %q, want %q", got, tc.want)
			}
		})
	}
}

// fileWriteUses reports each use of fileWrites in file, calls and function
// values alike.
func fileWriteUses(fset *token.FileSet, file *ast.File) []string {
	var violations []string
	report := func(pos token.Pos, what string) {
		violations = append(violations, fset.Position(pos).String()+": "+what)
	}

	names := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if _, watched := fileWrites[path]; err != nil || !watched {
			continue
		}
		name := path[strings.LastIndex(path, "/")+1:]
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." {
			report(spec.Pos(), `import . "`+path+`"`)
			continue
		}
		names[name] = path
	}

	ast.Inspect(file, func(node ast.Node) bool {
		selector, ok := node.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		pkg, ok := selector.X.(*ast.Ident)
		if !ok {
			return true
		}
		if path, ok := names[pkg.Name]; ok && slices.Contains(fileWrites[path], selector.Sel.Name) {
			report(selector.Pos(), pkg.Name+"."+selector.Sel.Name)
		}
		return true
	})
	return violations
}
