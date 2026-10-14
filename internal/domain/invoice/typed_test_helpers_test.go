package invoice

import (
	"testing"

	yaml "gopkg.in/yaml.v3"
)

// decodeForTest decodes source, a YAML document with a mapping at its root,
// into a T the way invox decodes its files, failing the test on any error.
func decodeForTest[T any](t *testing.T, source string) T {
	t.Helper()
	document, err := parseYAMLDocumentSource([]byte(source), "test.yaml")
	if err != nil {
		t.Fatalf("parseYAMLDocumentSource returned error: %v", err)
	}
	var value T
	if err := decodeYAMLDocument(document, "test.yaml", &value, true); err != nil {
		t.Fatalf("decodeYAMLDocument returned error: %v", err)
	}
	return value
}

// mergedText is the text of key in the mapping n, with its merge key
// applied, or "" when the mapping has no such key.
func mergedText(n *yaml.Node, key string) string {
	if n == nil || n.Kind != yaml.MappingNode {
		return ""
	}
	for _, pair := range mappingPairs(n) {
		if pair.key.Value == key {
			return nodeText(pair.value)
		}
	}
	return ""
}
