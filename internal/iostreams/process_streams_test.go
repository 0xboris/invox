package iostreams

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

// TestOnlySystemTouchesProcessStreams fails when non-test code outside
// System uses os.Stdin, os.Stdout or os.Stderr, or prints with fmt.Print*,
// log's package-level output functions or the print and println builtins.
// Everything else gets its streams from an IOStreams.
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
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			inThisPackage := filepath.Base(filepath.Dir(path)) == "iostreams"
			violations = append(violations, processStreamUses(fset, file, inThisPackage)...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}

	if len(violations) > 0 {
		t.Fatalf("use the IOStreams passed in instead of the process streams:\n%s", strings.Join(violations, "\n"))
	}
}

func TestProcessStreamUsesFindsEveryForm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "streams and fmt.Print",
			src:  "import (\n\t\"fmt\"\n\t\"os\"\n)\n\nfunc f() { fmt.Println(os.Stdin, os.Stdout, os.Stderr) }\n",
			want: []string{"fmt.Println", "os.Stdin", "os.Stdout", "os.Stderr"},
		},
		{
			name: "aliased import",
			src:  "import stdos \"os\"\n\nvar w = stdos.Stderr\n",
			want: []string{"stdos.Stderr"},
		},
		{
			name: "same package imported twice",
			src:  "import (\n\t\"os\"\n\to \"os\"\n)\n\nvar a, b = os.Args, o.Stdout\n",
			want: []string{"o.Stdout"},
		},
		{
			name: "dot import",
			src:  "import . \"os\"\n\nvar w = Stdout\n",
			want: []string{`import . "os"`},
		},
		{
			name: "log output",
			src:  "import \"log\"\n\nfunc f() { log.Printf(\"x\"); log.Fatal(\"x\") }\n",
			want: []string{"log.Printf", "log.Fatal"},
		},
		{
			name: "print builtins",
			src:  "func f() { print(1); println(2) }\n",
			want: []string{"print", "println"},
		},
		{
			name: "writer-based output",
			src:  "import (\n\t\"fmt\"\n\t\"io\"\n\t\"log\"\n)\n\nfunc f(w io.Writer) { fmt.Fprintln(w); log.New(w, \"\", 0) }\n",
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
			for _, violation := range processStreamUses(fset, file, false) {
				got = append(got, violation[strings.Index(violation, ": ")+2:])
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("violations = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestProcessStreamUsesAllowsOnlySystem(t *testing.T) {
	t.Parallel()

	src := "package iostreams\n\nimport \"os\"\n\nfunc System() any { return os.Stdout }\n\nfunc other() any { return os.Stderr }\n"
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "probe.go", src, 0)
	if err != nil {
		t.Fatalf("parse probe: %v", err)
	}

	got := processStreamUses(fset, file, true)
	if len(got) != 1 || !strings.HasSuffix(got[0], ": os.Stderr") {
		t.Fatalf("violations = %q, want only os.Stderr in other", got)
	}
}

// printingFuncs lists, per import path, the package-level identifiers that
// reach the process streams. A dot import of any of these paths counts too,
// since its uses can't be told apart from local names.
var printingFuncs = map[string][]string{
	"os":  {"Stdin", "Stdout", "Stderr"},
	"fmt": {"Print", "Printf", "Println"},
	"log": {"Print", "Printf", "Println", "Fatal", "Fatalf", "Fatalln", "Panic", "Panicf", "Panicln"},
}

// processStreamUses reports each use of the process streams in file. When
// allowSystem is set, the System function is exempt.
func processStreamUses(fset *token.FileSet, file *ast.File, allowSystem bool) []string {
	var violations []string
	report := func(pos token.Pos, what string) {
		violations = append(violations, fset.Position(pos).String()+": "+what)
	}

	// names maps each local name of a watched import to its import path.
	names := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if _, watched := printingFuncs[path]; err != nil || !watched {
			continue
		}
		name := path
		if spec.Name != nil {
			name = spec.Name.Name
		}
		if name == "." {
			report(spec.Pos(), `import . "`+path+`"`)
			continue
		}
		names[name] = path
	}

	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && allowSystem && fn.Recv == nil && fn.Name.Name == "System" {
			continue
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.SelectorExpr:
				pkg, ok := node.X.(*ast.Ident)
				if !ok {
					return true
				}
				if path, ok := names[pkg.Name]; ok && slices.Contains(printingFuncs[path], node.Sel.Name) {
					report(node.Pos(), pkg.Name+"."+node.Sel.Name)
				}
			case *ast.CallExpr:
				if fun, ok := node.Fun.(*ast.Ident); ok && (fun.Name == "print" || fun.Name == "println") {
					report(fun.Pos(), fun.Name)
				}
			}
			return true
		})
	}
	return violations
}
