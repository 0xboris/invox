package invoice

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadContextAppliesVATFromMergeKey(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, invoicePath, "customer_id: CUST-001\n", "customer_id: CUST-001\nreduced: &reduced {vat_percent: 10}\n")
	replaceInFixture(t, invoicePath, "  - name: Support\n", "  - <<: *reduced\n    name: Support\n")

	ctx, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err != nil {
		t.Fatalf("LoadContext returned error: %v", err)
	}
	// 200.00 at the invoice's 20% plus 10.00 at the merged 10%.
	if got := ctx.TotalCents; got != 25100 {
		t.Fatalf("total = %d cents, want 25100", got)
	}
}

func TestParseYAMLSourceMergeKeys(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   map[string]any
	}{
		{
			name:   "single alias",
			source: "base: &base {a: 1, b: x}\nitem:\n  <<: *base\n  c: y\n",
			want:   map[string]any{"a": "1", "b": "x", "c": "y"},
		},
		{
			name:   "explicit key wins over merged key after it",
			source: "base: &base {a: 1, b: x}\nitem:\n  b: own\n  <<: *base\n",
			want:   map[string]any{"a": "1", "b": "own"},
		},
		{
			name:   "explicit key wins over merged key before it",
			source: "base: &base {a: 1, b: x}\nitem:\n  <<: *base\n  b: own\n",
			want:   map[string]any{"a": "1", "b": "own"},
		},
		{
			name:   "list of aliases, earlier wins",
			source: "one: &one {a: 1, b: one}\ntwo: &two {b: two, c: 3}\nitem:\n  <<: [*one, *two]\n",
			want:   map[string]any{"a": "1", "b": "one", "c": "3"},
		},
		{
			name:   "inline mapping",
			source: "item:\n  <<: {a: 1}\n  b: 2\n",
			want:   map[string]any{"a": "1", "b": "2"},
		},
		{
			name:   "nested merge",
			source: "base: &base {a: 1}\nmid: &mid\n  <<: *base\n  b: 2\nitem:\n  <<: *mid\n  c: 3\n",
			want:   map[string]any{"a": "1", "b": "2", "c": "3"},
		},
		{
			name:   "quoted << is a plain key",
			source: "item:\n  \"<<\": text\n",
			want:   map[string]any{"<<": "text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := parseYAMLSource([]byte(tt.source), "test.yaml")
			if err != nil {
				t.Fatalf("parseYAMLSource returned error: %v", err)
			}
			got := value.(map[string]any)["item"]
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("item = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseYAMLSourceRejectsInvalidMappings(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		wantErr string
	}{
		{
			name:    "duplicate key",
			source:  "positions:\n  - name: Book\n    unit_price: 1\n    unit_price: 2\n",
			wantErr: `test.yaml:4: duplicate key "unit_price" (first defined on line 3)`,
		},
		{
			name:    "duplicate top-level key",
			source:  "invoice: {}\ncustomer_id: A\ncustomer_id: B\n",
			wantErr: `test.yaml:3: duplicate key "customer_id" (first defined on line 2)`,
		},
		{
			name:    "duplicate key inside an anchor",
			source:  "base: &base\n  a: 1\n  a: 2\nitem: *base\n",
			wantErr: `test.yaml:3: duplicate key "a" (first defined on line 2)`,
		},
		{
			name:    "two merge keys",
			source:  "one: &one {a: 1}\ntwo: &two {b: 2}\nitem:\n  <<: *one\n  <<: *two\n",
			wantErr: `test.yaml:5: duplicate key "<<" (first defined on line 4)`,
		},
		{
			name:    "merge of a scalar",
			source:  "base: &base text\nitem:\n  <<: *base\n",
			wantErr: "test.yaml:3: merge key `<<` must refer to a mapping or a list of mappings",
		},
		{
			name:    "merge of a list with a scalar",
			source:  "base: &base {a: 1}\nitem:\n  <<: [*base, text]\n",
			wantErr: "test.yaml:3: merge key `<<` must refer to a mapping or a list of mappings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseYAMLSource([]byte(tt.source), "test.yaml")
			if err == nil {
				t.Fatalf("parseYAMLSource returned nil error, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseYAMLSourceKeepsAnchorsAndAliases(t *testing.T) {
	source := `
address: &address
  street: Ring 1
  city: Vienna
customers:
  A:
    address: *address
    tags: &tags [x, y]
  B:
    nested:
      address: *address
      tags: *tags
`
	value, err := parseYAMLSource([]byte(source), "test.yaml")
	if err != nil {
		t.Fatalf("parseYAMLSource returned error: %v", err)
	}

	address := map[string]any{"street": "Ring 1", "city": "Vienna"}
	customers := value.(map[string]any)["customers"].(map[string]any)
	if got := getPath(customers["A"].(map[string]any), "address"); !reflect.DeepEqual(got, address) {
		t.Fatalf("A.address = %#v, want %#v", got, address)
	}
	if got := getPath(customers["B"].(map[string]any), "nested.address"); !reflect.DeepEqual(got, address) {
		t.Fatalf("B.nested.address = %#v, want %#v", got, address)
	}
	if got := getPath(customers["B"].(map[string]any), "nested.tags"); !reflect.DeepEqual(got, []any{"x", "y"}) {
		t.Fatalf("B.nested.tags = %#v, want [x y]", got)
	}
}

func TestLoadYAMLDocumentRejectsDuplicateKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invoice_defaults.yaml")
	if err := os.WriteFile(path, []byte("invoice:\n  vat_percent: 20\n  vat_percent: 10\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, err := loadYAMLDocument(path)
	if err == nil {
		t.Fatal("loadYAMLDocument returned nil error, want duplicate key error")
	}
	if want := path + `:3: duplicate key "vat_percent" (first defined on line 2)`; err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestLoadContextRejectsDuplicateKeyInCustomers(t *testing.T) {
	customersPath, issuerPath, invoicePath, _, _, _ := writeContextFixtures(t)
	replaceInFixture(t, customersPath, "  status: active\n", "  status: active\n  status: inactive\n")

	_, err := LoadContext(customersPath, issuerPath, invoicePath)
	if err == nil {
		t.Fatal("LoadContext returned nil error, want duplicate key error")
	}
	if want := customersPath + `:4: duplicate key "status"`; !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err.Error(), want)
	}
}
