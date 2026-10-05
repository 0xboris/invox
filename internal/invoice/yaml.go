package invoice

import (
	"fmt"
	"os"
	"time"

	yaml "gopkg.in/yaml.v3"
)

func loadYAML(path string) (any, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parseYAMLSource(source, path)
}

func parseYAMLSource(source []byte, label string) (any, error) {
	document, err := parseYAMLDocumentSource(source, label)
	if err != nil {
		return nil, err
	}
	return normalizeYAMLNode(document), nil
}

// checkYAMLMappings rejects what normalizeYAMLNode cannot represent
// faithfully: a key defined twice in one mapping, and a merge key whose value
// is not a mapping or a list of mappings. Aliases are not followed, because
// the anchored node is checked where it is defined.
func checkYAMLMappings(node *yaml.Node, label string) error {
	if node == nil {
		return nil
	}

	if node.Kind == yaml.MappingNode {
		firstLines := make(map[string]int, len(node.Content)/2)
		for index := 0; index+1 < len(node.Content); index += 2 {
			keyNode := node.Content[index]
			if keyNode.Kind != yaml.ScalarNode {
				continue
			}
			if firstLine, exists := firstLines[keyNode.Value]; exists {
				return fmt.Errorf("%s:%d: duplicate key %q (first defined on line %d)", label, keyNode.Line, keyNode.Value, firstLine)
			}
			firstLines[keyNode.Value] = keyNode.Line

			if isYAMLMergeKey(keyNode) && !isYAMLMergeValue(node.Content[index+1]) {
				return fmt.Errorf("%s:%d: merge key `<<` must refer to a mapping or a list of mappings", label, keyNode.Line)
			}
		}
	}

	if node.Kind == yaml.AliasNode {
		return nil
	}
	for _, child := range node.Content {
		if err := checkYAMLMappings(child, label); err != nil {
			return err
		}
	}
	return nil
}

func isYAMLMergeKey(node *yaml.Node) bool {
	return node.Kind == yaml.ScalarNode && node.Value == "<<" && node.ShortTag() == "!!merge"
}

// isYAMLMergeValue accepts what yaml.v3's typed decoding accepts: a mapping
// or an alias to one, or a literal sequence of those. An alias to a sequence
// is not a merge value.
func isYAMLMergeValue(node *yaml.Node) bool {
	if node.Kind != yaml.SequenceNode {
		return resolveYAMLAlias(node).Kind == yaml.MappingNode
	}
	for _, child := range node.Content {
		if resolveYAMLAlias(child).Kind != yaml.MappingNode {
			return false
		}
	}
	return true
}

func resolveYAMLAlias(node *yaml.Node) *yaml.Node {
	for node.Kind == yaml.AliasNode && node.Alias != nil {
		node = node.Alias
	}
	return node
}

// clearYAMLMergeTags drops the resolved `!!merge` tag from merge keys, which
// yaml.v3's encoder would otherwise write out as `!!merge <<: *anchor`.
func clearYAMLMergeTags(node *yaml.Node) {
	if node == nil || node.Kind == yaml.AliasNode {
		return
	}
	if node.Kind == yaml.MappingNode {
		for index := 0; index+1 < len(node.Content); index += 2 {
			if isYAMLMergeKey(node.Content[index]) {
				node.Content[index].Tag = ""
			}
		}
	}
	for _, child := range node.Content {
		clearYAMLMergeTags(child)
	}
}

func normalizeYAMLNode(node *yaml.Node) any {
	if node == nil {
		return nil
	}

	switch node.Kind {
	case yaml.DocumentNode:
		if len(node.Content) == 0 {
			return nil
		}
		return normalizeYAMLNode(node.Content[0])
	case yaml.MappingNode:
		normalized := make(map[string]any, len(node.Content)/2)
		// Merge keys follow YAML 1.1: the mapping's own keys win over merged
		// ones, and in `<<: [*a, *b]` a key from *a wins over the same key
		// from *b.
		for index := 0; index+1 < len(node.Content); index += 2 {
			if isYAMLMergeKey(node.Content[index]) {
				mergeYAMLValue(normalized, node.Content[index+1])
			}
		}
		for index := 0; index+1 < len(node.Content); index += 2 {
			keyNode := node.Content[index]
			if isYAMLMergeKey(keyNode) {
				continue
			}
			normalized[keyNode.Value] = normalizeYAMLNode(node.Content[index+1])
		}
		return normalized
	case yaml.SequenceNode:
		normalized := make([]any, len(node.Content))
		for index, child := range node.Content {
			normalized[index] = normalizeYAMLNode(child)
		}
		return normalized
	case yaml.AliasNode:
		return normalizeYAMLNode(node.Alias)
	case yaml.ScalarNode:
		return normalizeYAMLScalar(node)
	default:
		return nil
	}
}

// mergeYAMLValue copies the keys of a merge key's value into target, keeping
// keys target already has.
func mergeYAMLValue(target map[string]any, value *yaml.Node) {
	sources := []*yaml.Node{value}
	if value.Kind == yaml.SequenceNode {
		sources = value.Content
	}
	for _, source := range sources {
		merged, ok := normalizeYAMLNode(source).(map[string]any)
		if !ok {
			continue
		}
		for key, item := range merged {
			if _, exists := target[key]; !exists {
				target[key] = item
			}
		}
	}
}

func normalizeYAMLScalar(node *yaml.Node) any {
	// Numbers keep their source text: yaml.v3 would read `01067` or `0042` as
	// octal and `12.50` as 12.5. Fields that need a number parse the text with
	// a strict decimal grammar instead.
	switch node.ShortTag() {
	case "!!int", "!!float":
		return node.Value
	}

	var value any
	if err := node.Decode(&value); err != nil {
		return node.Value
	}
	if typed, ok := value.(time.Time); ok {
		return typed.Format("2006-01-02")
	}
	return value
}
