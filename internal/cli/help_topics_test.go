package cli

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

func TestHelpTopicsWorkWithAndWithoutConfig(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStdout []string
	}{
		{
			name: "environment",
			args: []string{"help", "environment"},
			wantStdout: []string{
				"Environment variables and default directories.\n",
				"  XDG_CONFIG_HOME\n",
				"  1. explicit flag (-c, -u, --defaults, -t)\n",
			},
		},
		{
			name: "exit-codes",
			args: []string{"help", "exit-codes"},
			wantStdout: []string{
				"  0    Success.\n",
				"  1    The command failed",
				"  2    Usage error",
				"  130  Interrupted by Ctrl-C (SIGINT).",
				"  143  Stopped by SIGTERM",
			},
		},
	}
	for _, tt := range tests {
		for _, broken := range []bool{false, true} {
			name := tt.name
			if broken {
				name += " with broken config"
			}
			t.Run(name, func(t *testing.T) {
				if broken {
					setupBrokenConfig(t)
				} else {
					chdirForTest(t, t.TempDir())
				}

				exitCode, stdout, stderr := captureRun(t, tt.args)
				if exitCode != 0 {
					t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
				}
				for _, want := range tt.wantStdout {
					if !strings.Contains(stdout, want) {
						t.Fatalf("stdout = %q, want it to contain %q", stdout, want)
					}
				}
				if stderr != "" {
					t.Fatalf("stderr = %q, want empty", stderr)
				}
			})
		}
	}
}

func TestHelpTopicsRejectExtraArguments(t *testing.T) {
	for _, topic := range []string{"environment", "exit-codes"} {
		t.Run(topic, func(t *testing.T) {
			exitCode, stdout, stderr := captureRun(t, []string{"help", topic, "extra"})
			if exitCode != 2 {
				t.Fatalf("exitCode = %d, want 2", exitCode)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if want := `error: unknown help topic "` + topic + ` extra"`; !strings.HasPrefix(stderr, want) {
				t.Fatalf("stderr = %q, want prefix %q", stderr, want)
			}
		})
	}
}

func TestHelpConfigNamesLegacyPathUnderConfigHome(t *testing.T) {
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)

	exitCode, stdout, stderr := captureRun(t, []string{"help", "config"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	want := "  legacy fallback: " + filepath.Join(configHome, "invoice-tool", "config.yaml") + "\n"
	if !strings.Contains(stdout, want) {
		t.Fatalf("stdout = %q, want it to contain %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

// TestHelpEnvironmentDocumentsEveryVariableRead fails when non-test code
// under cmd/ or internal/ reads an environment variable that
// `invox help environment` does not name.
func TestHelpEnvironmentDocumentsEveryVariableRead(t *testing.T) {
	root := filepath.Join("..", "..")
	keys := map[string][]string{}
	for _, dir := range []string{"cmd", "internal"} {
		packages := map[string][]*ast.File{}
		fset := token.NewFileSet()
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			packages[filepath.Dir(path)] = append(packages[filepath.Dir(path)], file)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", dir, err)
		}
		for _, files := range packages {
			for key, positions := range envKeysRead(fset, files) {
				keys[key] = append(keys[key], positions...)
			}
		}
	}
	if len(keys) == 0 {
		t.Fatal("found no environment reads; the scan is broken")
	}

	exitCode, stdout, stderr := captureRun(t, []string{"help", "environment"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	var missing []string
	for key, positions := range keys {
		if !strings.Contains(stdout, "  "+key+"\n") {
			missing = append(missing, key+" (read at "+strings.Join(positions, ", ")+")")
		}
	}
	slices.Sort(missing)
	if len(missing) > 0 {
		t.Fatalf("add these to environmentVariables in internal/cli/helptext/topics.go:\n%s", strings.Join(missing, "\n"))
	}
}

func TestEnvKeysReadFindsEveryForm(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want []string
	}{
		{
			name: "Getenv and LookupEnv literals",
			src:  "import \"os\"\n\nvar a = os.Getenv(\"A\")\nvar _, b = os.LookupEnv(`B`)\n",
			want: []string{"A", "B"},
		},
		{
			name: "aliased import",
			src:  "import stdos \"os\"\n\nvar a = stdos.Getenv(\"A\")\n",
			want: []string{"A"},
		},
		{
			name: "constant key",
			src:  "import \"os\"\n\nconst key = \"K\"\n\nvar a = os.Getenv(key)\n",
			want: []string{"K"},
		},
		{
			name: "computed key",
			src:  "import \"os\"\n\nfunc f(name string) string { return os.Getenv(name) }\n",
			want: []string{"<non-constant key name>"},
		},
		{
			name: "Env value and field",
			src:  "type Env struct{ Getenv func(string) string }\n\nfunc f(e Env, g struct{ Env Env }) { e.Getenv(\"K\"); g.Env.Getenv(\"K2\") }\n",
			want: []string{"K", "K2"},
		},
		{
			name: "unexported getenv field, as in the editor adapter",
			src:  "type Editor struct{ getenv func(string) string }\n\nfunc (e *Editor) f() string { return e.getenv(\"VISUAL\") }\n",
			want: []string{"VISUAL"},
		},
		{
			name: "Env value with computed key",
			src:  "type Env struct{ Getenv func(string) string }\n\nfunc f(e Env, name string) string { return e.Getenv(name) }\n",
			want: []string{"<non-constant key name>"},
		},
		{
			name: "other packages and setters",
			src:  "import (\n\t\"os\"\n\t\"syscall\"\n)\n\nfunc f() { os.Setenv(\"S\", \"1\"); syscall.Getenv(\"Y\"); _ = os.Environ() }\n",
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
			for key := range envKeysRead(fset, []*ast.File{file}) {
				got = append(got, key)
			}
			slices.Sort(got)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("keys = %q, want %q", got, tc.want)
			}
		})
	}
}

// envKeysRead maps each environment variable read in one package's files to
// the positions that read it. A read is a call X.Getenv(key) or
// X.LookupEnv(key) where X is os or a value such as an env.Env
// (e.Getenv, f.Env.Getenv, or an adapter's getenv field), but not another
// imported package such as syscall.
// A key that is neither a string literal nor a package-level string constant
// is reported as "<non-constant key EXPR>", so it can't slip past the check.
func envKeysRead(fset *token.FileSet, files []*ast.File) map[string][]string {
	constants := map[string]string{}
	for _, file := range files {
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				for i, name := range valueSpec.Names {
					if i >= len(valueSpec.Values) {
						continue
					}
					if value, ok := stringLiteral(valueSpec.Values[i]); ok {
						constants[name.Name] = value
					}
				}
			}
		}
	}

	keys := map[string][]string{}
	for _, file := range files {
		otherPackages := map[string]bool{}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil || path == "os" {
				continue
			}
			name := importName(path)
			if spec.Name != nil {
				name = spec.Name.Name
			}
			otherPackages[name] = true
		}

		ast.Inspect(file, func(node ast.Node) bool {
			call, ok := node.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || (selector.Sel.Name != "Getenv" && selector.Sel.Name != "getenv" && selector.Sel.Name != "LookupEnv") {
				return true
			}
			switch x := selector.X.(type) {
			case *ast.Ident:
				if otherPackages[x.Name] {
					return true
				}
			case *ast.SelectorExpr:
			default:
				return true
			}

			key, ok := stringLiteral(call.Args[0])
			if !ok {
				if ident, isIdent := call.Args[0].(*ast.Ident); isIdent {
					key, ok = constants[ident.Name]
				}
			}
			if !ok {
				key = "<non-constant key " + exprString(call.Args[0]) + ">"
			}
			keys[key] = append(keys[key], fset.Position(call.Pos()).String())
			return true
		})
	}
	return keys
}

// importName is the name a package is referred to by when its import has no
// alias: the last path element, without a gopkg.in style ".vN" suffix.
func importName(path string) string {
	name := path[strings.LastIndex(path, "/")+1:]
	if dot := strings.Index(name, ".v"); dot > 0 {
		name = name[:dot]
	}
	return name
}

func stringLiteral(expr ast.Expr) (string, bool) {
	literal, ok := expr.(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return "", false
	}
	value, err := strconv.Unquote(literal.Value)
	return value, err == nil
}

func exprString(expr ast.Expr) string {
	if ident, ok := expr.(*ast.Ident); ok {
		return ident.Name
	}
	return "expression"
}
