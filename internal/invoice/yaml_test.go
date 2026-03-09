package invoice

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadYAMLPreservesNumericLookingMappingKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "customers.yaml")
	source := "0021:\n  name: Appsters GmbH\n  nested:\n    0007: yes\nissued_on: 2026-03-06\n"
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	value, err := loadYAML(path)
	if err != nil {
		t.Fatalf("loadYAML returned error: %v", err)
	}

	root, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("root type = %T, want map[string]any", value)
	}
	if _, exists := root["17"]; exists {
		t.Fatalf("root unexpectedly contains coerced key %q", "17")
	}

	customer, ok := root["0021"].(map[string]any)
	if !ok {
		t.Fatalf("root[0021] type = %T, want map[string]any", root["0021"])
	}
	if customer["name"] != "Appsters GmbH" {
		t.Fatalf("customer name = %#v, want %q", customer["name"], "Appsters GmbH")
	}

	nested, ok := customer["nested"].(map[string]any)
	if !ok {
		t.Fatalf("customer[nested] type = %T, want map[string]any", customer["nested"])
	}
	if _, exists := nested["7"]; exists {
		t.Fatalf("nested unexpectedly contains coerced key %q", "7")
	}
	if nested["0007"] != "yes" {
		t.Fatalf("nested[0007] = %#v, want %q", nested["0007"], "yes")
	}
	if root["issued_on"] != "2026-03-06" {
		t.Fatalf("issued_on = %#v, want %q", root["issued_on"], "2026-03-06")
	}
}
