package store

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/invoice"
)

// mergeItem is the schema of `item` in the merge key cases. The `<<` field
// reads a quoted "<<" key, which is a plain key and not a merge.
type mergeItem struct {
	A      invoice.Text `yaml:"a"`
	B      invoice.Text `yaml:"b"`
	C      invoice.Text `yaml:"c"`
	List   []mergeItem  `yaml:"list"`
	Quoted invoice.Text `yaml:"<<"`
}

func TestDecodeYAMLMergeKeys(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   mergeItem
	}{
		{
			name:   "single alias",
			source: "base: &base {a: 1, b: x}\nitem:\n  <<: *base\n  c: y\n",
			want:   mergeItem{A: "1", B: "x", C: "y"},
		},
		{
			name:   "explicit key wins over merged key after it",
			source: "base: &base {a: 1, b: x}\nitem:\n  b: own\n  <<: *base\n",
			want:   mergeItem{A: "1", B: "own"},
		},
		{
			name:   "explicit key wins over merged key before it",
			source: "base: &base {a: 1, b: x}\nitem:\n  <<: *base\n  b: own\n",
			want:   mergeItem{A: "1", B: "own"},
		},
		{
			name:   "list of aliases, earlier wins",
			source: "one: &one {a: 1, b: one}\ntwo: &two {b: two, c: 3}\nitem:\n  <<: [*one, *two]\n",
			want:   mergeItem{A: "1", B: "one", C: "3"},
		},
		{
			name:   "inline mapping",
			source: "item:\n  <<: {a: 1}\n  b: 2\n",
			want:   mergeItem{A: "1", B: "2"},
		},
		{
			name:   "nested merge",
			source: "base: &base {a: 1}\nmid: &mid\n  <<: *base\n  b: 2\nitem:\n  <<: *mid\n  c: 3\n",
			want:   mergeItem{A: "1", B: "2", C: "3"},
		},
		{
			name:   "merge inside a sequence item",
			source: "base: &base {a: 1}\nitem:\n  <<: *base\n  list:\n    - <<: *base\n      b: 2\n",
			want:   mergeItem{A: "1", List: []mergeItem{{A: "1", B: "2"}}},
		},
		{
			name:   "quoted << is a plain key",
			source: "item:\n  \"<<\": text\n",
			want:   mergeItem{Quoted: "text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeForTest[struct {
				Item mergeItem `yaml:"item"`
			}](t, tt.source).Item
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("item = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseYAMLDocumentSourceRejectsInvalidMappings(t *testing.T) {
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
			name:    "quoted and unquoted duplicate",
			source:  "a: 1\n\"a\": 2\n",
			wantErr: `test.yaml:2: duplicate key "a" (first defined on line 1)`,
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
		{
			name:    "merge of an alias to a list",
			source:  "list: &list [{a: 1}]\nitem:\n  <<: *list\n",
			wantErr: "test.yaml:3: merge key `<<` must refer to a mapping or a list of mappings",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseYAMLDocumentSource([]byte(tt.source), "test.yaml")
			if err == nil {
				t.Fatalf("parseYAMLDocumentSource returned nil error, want %q", tt.wantErr)
			}
			if err.Error() != tt.wantErr {
				t.Fatalf("error = %q, want %q", err.Error(), tt.wantErr)
			}
		})
	}
}

type anchorAddress struct {
	Street invoice.Text `yaml:"street"`
	City   invoice.Text `yaml:"city"`
}

func TestDecodeYAMLKeepsAnchorsAndAliases(t *testing.T) {
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
	customers := decodeForTest[struct {
		Address   anchorAddress `yaml:"address"`
		Customers struct {
			A struct {
				Address anchorAddress  `yaml:"address"`
				Tags    []invoice.Text `yaml:"tags"`
			} `yaml:"A"`
			B struct {
				Nested struct {
					Address anchorAddress  `yaml:"address"`
					Tags    []invoice.Text `yaml:"tags"`
				} `yaml:"nested"`
			} `yaml:"B"`
		} `yaml:"customers"`
	}](t, source).Customers

	address := anchorAddress{Street: "Ring 1", City: "Vienna"}
	if got := customers.A.Address; got != address {
		t.Fatalf("A.address = %#v, want %#v", got, address)
	}
	if got := customers.B.Nested.Address; got != address {
		t.Fatalf("B.nested.address = %#v, want %#v", got, address)
	}
	if got := customers.B.Nested.Tags; !reflect.DeepEqual(got, []invoice.Text{"x", "y"}) {
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

func TestArchivedInvoiceIdentityReportsMarkdownFileLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invoice.md")
	source := "---\ninvoice:\n  number: A-1\n  number: A-2\n---\n# Invoice\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	_, _, err := archivedInvoiceIdentity(path)
	if err == nil {
		t.Fatal("archivedInvoiceIdentity returned nil error, want duplicate key error")
	}
	if want := "front matter in " + path + `:4: duplicate key "number" (first defined on line 3)`; err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestWriteInvoiceFieldsKeepMergeKeySyntax(t *testing.T) {
	source := "defaults: &d\n  currency: EUR\ninvoice:\n  <<: *d\n  number: A-1\n"
	tests := []struct {
		name  string
		write func(path string) error
	}{
		{name: "writeInvoiceNumber", write: func(path string) error { return writeInvoiceNumber(path, "A-2") }},
		{name: "writeInvoiceStringField", write: func(path string) error { return setInvoiceStatus(path, "sent") }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "invoice.yaml")
			if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			if err := tt.write(path); err != nil {
				t.Fatalf("write returned error: %v", err)
			}

			written, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile returned error: %v", err)
			}
			if strings.Contains(string(written), "!!merge") || !strings.Contains(string(written), "  <<: *d\n") {
				t.Fatalf("written file does not keep `<<: *d`:\n%s", written)
			}
			document, err := loadYAMLDocument(path)
			if err != nil {
				t.Fatalf("loadYAMLDocument returned error: %v", err)
			}
			root, err := documentRootMapping(document, path)
			if err != nil {
				t.Fatalf("documentRootMapping returned error: %v", err)
			}
			if got := mergedText(findMappingValue(root, "invoice"), "currency"); got != "EUR" {
				t.Fatalf("invoice.currency = %q, want %q", got, "EUR")
			}
		})
	}
}

func TestWriteInvoiceNumberRejectsDuplicateKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "invoice.yaml")
	if err := os.WriteFile(path, []byte("invoice:\n  number: A-1\n  number: A-2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	err := writeInvoiceNumber(path, "A-3")
	if want := path + `:3: duplicate key "number" (first defined on line 2)`; err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}
