package invoice

import (
	"context"
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
var aliasLoaders = map[string]func(path string) error{
	"loadYAML": func(path string) error {
		_, err := loadYAML(path)
		return err
	},
	"loadYAMLDocument": func(path string) error {
		_, err := loadYAMLDocument(path)
		return err
	},
	"archivedInvoiceValue": func(path string) error {
		_, _, err := archivedInvoiceValue(path)
		return err
	},
	"loadArchivedInvoiceDocument": func(path string) error {
		_, _, err := loadArchivedInvoiceDocument(path)
		return err
	},
}

// TestYAMLAliasLoaderProcess is the child side of runAliasLoader. A loader
// that overflows the stack kills the process, so it runs in its own.
func TestYAMLAliasLoaderProcess(t *testing.T) {
	name := os.Getenv(aliasLoaderEnv)
	if name == "" {
		t.Skip("runs only as a child of runAliasLoader")
	}
	err := aliasLoaders[name](os.Getenv(aliasFileEnv))
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
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "alias to its enclosing sequence",
			source: "customer_id: CUST-001\npositions: &p [*p]\n",
			want:   "%s:2: alias *p refers to a node that contains it",
		},
		{
			name:   "alias to an enclosing mapping",
			source: "customer_id: CUST-001\ninvoice: &inv\n  number: X\n  copy:\n    nested: *inv\n",
			want:   "%s:5: alias *inv refers to a node that contains it",
		},
		{
			name:   "merge key to its own mapping",
			source: "customer_id: CUST-001\nbase: &base\n  a: 1\n  <<: *base\n",
			want:   "%s:4: alias *base refers to a node that contains it",
		},
		{
			name:   "billion laughs",
			source: billionLaughs(),
			want:   "%s:7: aliases expand to more than 100000 nodes",
		},
	}
	for _, tt := range tests {
		for loader := range aliasLoaders {
			t.Run(tt.name+"/"+loader, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				path := filepath.Join(dir, "invoice.yaml")
				source := tt.source
				label := path
				if strings.HasPrefix(loader, "archived") || strings.HasPrefix(loader, "loadArchived") {
					path = filepath.Join(dir, "invoice.md")
					source = "---\n" + source + "---\n"
					label = "front matter in " + path
				}
				if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
					t.Fatalf("WriteFile returned error: %v", err)
				}

				if got, want := runAliasLoader(t, loader, path), fmt.Sprintf(tt.want, label); got != want {
					t.Fatalf("error = %q, want %q", got, want)
				}
			})
		}
	}
}

func TestParseYAMLSourceExpandsSharedAliases(t *testing.T) {
	// One anchor used by 1,000 positions of 10 fields each reaches 11,000
	// nodes through aliases, which a real invoice could plausibly do.
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
