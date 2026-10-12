package store

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/testfixture"
)

const (
	aliasLoaderEnv = "INVOX_TEST_YAML_ALIAS_LOADER"
	aliasFileEnv   = "INVOX_TEST_YAML_ALIAS_FILE"
)

// aliasLoaders are the entry points every YAML file goes through: typed
// decoding (invoices, customers, issuer, config), node documents (drafts,
// defaults, numbering), and archived invoices.
var aliasLoaders = map[string]struct {
	load func(path string) error
}{
	"decodeYAMLFile": {load: func(path string) error {
		return decodeYAMLFile(path, &invoiceIdentity{}, false)
	}},
	"loadYAMLDocument": {load: func(path string) error {
		_, err := loadYAMLDocument(path)
		return err
	}},
	"archivedInvoiceIdentity": {load: func(path string) error {
		_, _, err := archivedInvoiceIdentity(path)
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

func TestYAMLLoadersRejectRecursiveAndExplosiveAliases(t *testing.T) {
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
			source:  testfixture.Source("billion-laughs.yaml"),
			line:    7,
			message: "aliases expand to more than 100000 nodes",
		},
	}
	for _, tt := range tests {
		for name := range aliasLoaders {
			t.Run(tt.name+"/"+name, func(t *testing.T) {
				t.Parallel()
				dir := t.TempDir()
				path := filepath.Join(dir, "invoice.yaml")
				if err := os.WriteFile(path, []byte(tt.source), 0o644); err != nil {
					t.Fatalf("WriteFile returned error: %v", err)
				}

				want := fmt.Sprintf("%s:%d: %s", path, tt.line, tt.message)
				if got := runAliasLoader(t, name, path); got != want {
					t.Fatalf("error = %q, want %q", got, want)
				}
			})
		}
	}
}

func TestParseYAMLDocumentSourceAliasErrorIsTyped(t *testing.T) {
	_, err := parseYAMLDocumentSource([]byte("positions: &p [*p]\n"), "invoice.yaml")

	var aliasErr *yamlAliasError
	if !errors.As(err, &aliasErr) {
		t.Fatalf("error = %v, want a *yamlAliasError", err)
	}
	want := yamlAliasError{Label: "invoice.yaml", Line: 1, Alias: "p", Recursive: true}
	if *aliasErr != want {
		t.Fatalf("error = %+v, want %+v", *aliasErr, want)
	}
}

type sharedAliasPosition struct {
	Name       invoice.Text `yaml:"name"`
	Unit       invoice.Text `yaml:"unit"`
	Quantity   invoice.Text `yaml:"quantity"`
	UnitPrice  invoice.Text `yaml:"unit_price"`
	VATPercent invoice.Text `yaml:"vat_percent"`
	A          invoice.Text `yaml:"a"`
	B          invoice.Text `yaml:"b"`
	C          invoice.Text `yaml:"c"`
	D          invoice.Text `yaml:"d"`
	E          invoice.Text `yaml:"e"`
}

func TestDecodeYAMLExpandsSharedAliases(t *testing.T) {
	// One anchor of 10 fields reused by 1,000 positions reaches 21,000 nodes
	// through aliases, more than a real invoice needs.
	var source strings.Builder
	source.WriteString("line: &line {name: Work, unit: h, quantity: 1, unit_price: 100, vat_percent: 20, a: 1, b: 2, c: 3, d: 4, e: 5}\npositions:\n")
	for range 1000 {
		source.WriteString("  - *line\n")
	}

	positions := decodeForTest[struct {
		Positions []sharedAliasPosition `yaml:"positions"`
	}](t, source.String()).Positions
	if len(positions) != 1000 {
		t.Fatalf("len(positions) = %d, want 1000", len(positions))
	}
	want := sharedAliasPosition{Name: "Work", Unit: "h", Quantity: "1", UnitPrice: "100", VATPercent: "20", A: "1", B: "2", C: "3", D: "4", E: "5"}
	if positions[999] != want {
		t.Fatalf("positions[999] = %#v, want %#v", positions[999], want)
	}
}

func TestDecodeYAMLAcceptsLargeDocumentWithoutAliases(t *testing.T) {
	source := "items: [" + strings.Repeat("x, ", 200_000) + "x]\n"

	items := decodeForTest[struct {
		Items []invoice.Text `yaml:"items"`
	}](t, source).Items
	if got := len(items); got != 200_001 {
		t.Fatalf("len(items) = %d, want 200001", got)
	}
}
