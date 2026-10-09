package env

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

// allowedFuncs lists, per package directory, the functions that may read the
// process environment. A method is named Type.Method.
var allowedFuncs = map[string][]string{
	"internal/adapters/run": {"Exec.Run"}, // child processes inherit os.Environ
	"internal/env":          {"System"},
	"internal/iostreams":    {"newSystem"}, // reads INVOX_FORCE_TTY
}

// ambientReads lists, per import path, the package-level identifiers that
// read the process environment, the working directory or the clock. A dot
// import of any of these paths counts too, since its uses can't be told apart
// from local names.
var ambientReads = map[string][]string{
	"os":      {"Getenv", "LookupEnv", "Environ", "ExpandEnv", "UserHomeDir", "UserConfigDir", "UserCacheDir", "Getwd"},
	"time":    {"Now", "Since", "Until"},
	"syscall": {"Getenv", "LookupEnv", "Environ"},
	"runtime": {"GOOS"},
}

// TestOnlyEnvSystemReadsTheProcessEnvironment fails when non-test code under
// cmd/ or internal/ uses one of ambientReads outside the allowlist.
// Everything else reads these values from the env.Env it is given.
func TestOnlyEnvSystemReadsTheProcessEnvironment(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	var violations []string
	declared := map[string]bool{}
	for _, dir := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			pkgDir := filepath.ToSlash(filepath.Dir(rel))
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			for _, decl := range file.Decls {
				if fn, ok := decl.(*ast.FuncDecl); ok {
					declared[pkgDir+"."+funcName(fn)] = true
				}
			}
			violations = append(violations, ambientUses(fset, file, allowedFuncs[pkgDir])...)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
	}
	for dir, funcs := range allowedFuncs {
		for _, name := range funcs {
			if !declared[dir+"."+name] {
				violations = append(violations, dir+": "+name+" is allowlisted but missing; drop it from allowedFuncs")
			}
		}
	}

	if len(violations) > 0 {
		t.Fatalf("read these from the env.Env passed in instead:\n%s", strings.Join(violations, "\n"))
	}
}

func TestAmbientUsesFindsEveryForm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		src     string
		allowed []string
		want    []string
	}{
		{
			name: "aliased import",
			src:  "import stdos \"os\"\n\nvar home, _ = stdos.UserHomeDir()\n",
			want: []string{"stdos.UserHomeDir"},
		},
		{
			name: "dot import",
			src:  "import . \"time\"\n\nvar now = Now()\n",
			want: []string{`import . "time"`},
		},
		{
			name: "function value",
			src:  "import \"time\"\n\nvar clock = struct{ Now func() time.Time }{Now: time.Now}\n",
			want: []string{"time.Now"},
		},
		{
			name: "clock reads not spelled Now",
			src:  "import \"time\"\n\nvar start time.Time\n\nfunc f() (time.Duration, time.Duration) { return time.Since(start), time.Until(start) }\n",
			want: []string{"time.Since", "time.Until"},
		},
		{
			name: "syscall environment",
			src:  "import \"syscall\"\n\nfunc f() []string { v, _ := syscall.Getenv(\"A\"); _ = v; return syscall.Environ() }\n",
			want: []string{"syscall.Getenv", "syscall.Environ"},
		},
		{
			name: "runtime.GOOS",
			src:  "import \"runtime\"\n\nvar windows = runtime.GOOS == \"windows\"\n",
			want: []string{"runtime.GOOS"},
		},
		{
			name:    "System allowed, other function not",
			src:     "import \"os\"\n\nfunc System() any { return os.Getenv }\n\nfunc other() (string, error) { return os.Getwd() }\n",
			allowed: []string{"System"},
			want:    []string{"os.Getwd"},
		},
		{
			name:    "method allowed by type and name",
			src:     "import \"os\"\n\ntype Exec struct{}\n\nfunc (Exec) Run() []string { return os.Environ() }\n\nfunc (*Exec) Other() []string { return os.Environ() }\n\nfunc Run() []string { return os.Environ() }\n",
			allowed: []string{"Exec.Run"},
			want:    []string{"os.Environ", "os.Environ"},
		},
		{
			name: "System outside an allowed package",
			src:  "import \"os\"\n\nfunc System() string { return os.Getenv(\"A\") }\n",
			want: []string{"os.Getenv"},
		},
		{
			name: "writes and other identifiers",
			src:  "import (\n\t\"os\"\n\t\"runtime\"\n\t\"time\"\n)\n\nfunc f() { os.Setenv(\"S\", \"1\"); _ = os.Args; _ = runtime.GOARCH; _ = time.Duration(0) }\n",
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
			for _, violation := range ambientUses(fset, file, tc.allowed) {
				got = append(got, violation[strings.Index(violation, ": ")+2:])
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("violations = %q, want %q", got, tc.want)
			}
		})
	}
}

// ambientUses reports each use of ambientReads in file, calls and function
// values alike, outside the functions and methods named in allowed.
func ambientUses(fset *token.FileSet, file *ast.File, allowed []string) []string {
	var violations []string
	report := func(pos token.Pos, what string) {
		violations = append(violations, fset.Position(pos).String()+": "+what)
	}

	names := map[string]string{}
	for _, spec := range file.Imports {
		path, err := strconv.Unquote(spec.Path.Value)
		if _, watched := ambientReads[path]; err != nil || !watched {
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
		if fn, ok := decl.(*ast.FuncDecl); ok && slices.Contains(allowed, funcName(fn)) {
			continue
		}
		ast.Inspect(decl, func(node ast.Node) bool {
			selector, ok := node.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			pkg, ok := selector.X.(*ast.Ident)
			if !ok {
				return true
			}
			if path, ok := names[pkg.Name]; ok && slices.Contains(ambientReads[path], selector.Sel.Name) {
				report(selector.Pos(), pkg.Name+"."+selector.Sel.Name)
			}
			return true
		})
	}
	return violations
}

// funcName returns a function's name, or Type.Method for a method.
func funcName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	recv := fn.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	if ident, ok := recv.(*ast.Ident); ok {
		return ident.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}
