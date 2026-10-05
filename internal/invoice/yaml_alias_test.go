package invoice

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	aliasLoaderEnv = "INVOX_TEST_YAML_ALIAS_LOADER"
	aliasFileEnv   = "INVOX_TEST_YAML_ALIAS_FILE"
)

// aliasLoaders are the entry points every YAML file goes through: plain
// values (invoices, customers, issuer, config), node documents (drafts,
// defaults, numbering), and front matter of archived Markdown invoices.
var aliasLoaders = map[string]struct {
	markdown bool
	load     func(path string) error
}{
	"loadYAML": {load: func(path string) error {
		_, err := loadYAML(path)
		return err
	}},
	"loadYAMLDocument": {load: func(path string) error {
		_, err := loadYAMLDocument(path)
		return err
	}},
	"archivedInvoiceValue": {markdown: true, load: func(path string) error {
		_, _, err := archivedInvoiceValue(path)
		return err
	}},
	"loadArchivedInvoiceDocument": {markdown: true, load: func(path string) error {
		_, _, err := loadArchivedInvoiceDocument(path)
		return err
	}},
}

// TestYAMLAliasLoaderProcess is the child side of runAliasLoader. A loader
// that overflows the stack kills the process, so it runs in its own.
func TestYAMLAliasLoaderProcess(t *testing.T) {
	name := os.Getenv(aliasLoaderEnv)
	if name == "" {
		t.Skip("runs only as a child of runAliasLoader")
	}
	err := aliasLoaders[name].load(os.Getenv(aliasFileEnv))
	fmt.Fprint(os.Stdout, err)
	os.Exit(0)
}

// runAliasLoader loads path with the named loader in a child process and
// returns the error it printed, or fails the test if the child crashes or
// does not finish in time.
func runAliasLoader(t *testing.T, loader, path string) string {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestYAMLAliasLoaderProcess$")
	cmd.Env = append(os.Environ(), aliasLoaderEnv+"="+loader, aliasFileEnv+"="+path)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()
	if ctx.Err() != nil {
		t.Fatalf("%s(%s) did not finish within 20s", loader, filepath.Base(path))
	}
	if err != nil {
		message := stderr.String()
		if len(message) > 300 {
			message = message[:300]
		}
		t.Fatalf("%s(%s) crashed: %v\n%s", loader, filepath.Base(path), err, message)
	}
	return string(stdout)
}

// billionLaughs nests nine levels of nine aliases each: 364 bytes that
// expand to about 430 million nodes.
func billionLaughs() string {
	lines := []string{"customer_id: CUST-001", `a: &a ["lol","lol","lol","lol","lol","lol","lol","lol","lol"]`}
	previous := "a"
	for _, name := range strings.Split("bcdefghi", "") {
		lines = append(lines, fmt.Sprintf("%s: &%s [%s]", name, name, strings.TrimSuffix(strings.Repeat("*"+previous+",", 9), ",")))
		previous = name
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestYAMLLoadersRejectRecursiveAndExplosiveAliases(t *testing.T) {
	// Front matter errors count lines from the top of the Markdown file.
	tests := []struct {
		name    string
		source  string
		line    int
		message string
	}{
		{
			name:    "alias to its enclosing sequence",
			source:  "customer_id: CUST-001\npositions: &p [*p]\n",
			line:    2,
			message: "alias *p refers to a node that contains it",
		},
		{
			name:    "alias to an enclosing mapping",
			source:  "customer_id: CUST-001\ninvoice: &inv\n  number: X\n  copy:\n    nested: *inv\n",
			line:    5,
			message: "alias *inv refers to a node that contains it",
		},
		{
			name:    "merge key to its own mapping",
			source:  "customer_id: CUST-001\nbase: &base\n  a: 1\n  <<: *base\n",
			line:    4,
			message: "alias *base refers to a node that contains it",
		},
		{
			name:    "billion laughs",
			source:  billionLaughs(),
			line:    7,
			message: "aliases expand to more than 100000 nodes",
		},
	}
	for _, tt := range tests {
		for name, loader := range aliasLoaders {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				path := filepath.Join(dir, "invoice.yaml")
				source := tt.source
				label, line := path, tt.line
				if loader.markdown {
					path = filepath.Join(dir, "invoice.md")
					source = "---\n" + source + "---\n"
					label = "front matter in " + path
					line++
				}
				if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
					t.Fatalf("WriteFile returned error: %v", err)
				}

				want := fmt.Sprintf("%s:%d: %s", label, line, tt.message)
				if got := runAliasLoader(t, name, path); got != want {
					t.Fatalf("error = %q, want %q", got, want)
				}
			})
		}
	}
}

func TestParseYAMLSourceAliasErrorIsTyped(t *testing.T) {
	_, err := parseYAMLSource([]byte("positions: &p [*p]\n"), "invoice.yaml")

	var aliasErr *YAMLAliasError
	if !errors.As(err, &aliasErr) {
		t.Fatalf("error = %v, want a *YAMLAliasError", err)
	}
	want := YAMLAliasError{Label: "invoice.yaml", Line: 1, Alias: "p", Recursive: true}
	if *aliasErr != want {
		t.Fatalf("error = %+v, want %+v", *aliasErr, want)
	}
}

func TestParseYAMLSourceExpandsSharedAliases(t *testing.T) {
	// One anchor of 10 fields reused by 1,000 positions reaches 21,000 nodes
	// through aliases, more than a real invoice needs.
	var source strings.Builder
	source.WriteString("line: &line {name: Work, unit: h, quantity: 1, unit_price: 100, vat_percent: 20, a: 1, b: 2, c: 3, d: 4, e: 5}\npositions:\n")
	for range 1000 {
		source.WriteString("  - *line\n")
	}

	value, err := parseYAMLSource([]byte(source.String()), "invoice.yaml")
	if err != nil {
		t.Fatalf("parseYAMLSource returned error: %v", err)
	}
	positions := value.(map[string]any)["positions"].([]any)
	if len(positions) != 1000 {
		t.Fatalf("len(positions) = %d, want 1000", len(positions))
	}
	want := map[string]any{"name": "Work", "unit": "h", "quantity": "1", "unit_price": "100", "vat_percent": "20", "a": "1", "b": "2", "c": "3", "d": "4", "e": "5"}
	if !reflect.DeepEqual(positions[999], want) {
		t.Fatalf("positions[999] = %#v, want %#v", positions[999], want)
	}
}

func TestParseYAMLSourceAcceptsLargeDocumentWithoutAliases(t *testing.T) {
	source := "items: [" + strings.Repeat("x, ", 200_000) + "x]\n"

	value, err := parseYAMLSource([]byte(source), "invoice.yaml")
	if err != nil {
		t.Fatalf("parseYAMLSource returned error: %v", err)
	}
	if got := len(value.(map[string]any)["items"].([]any)); got != 200_001 {
		t.Fatalf("len(items) = %d, want 200001", got)
	}
}
