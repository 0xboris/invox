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
	var document yaml.Node
	if err := yaml.Unmarshal(source, &document); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	return normalizeYAMLNode(&document), nil
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
		for index := 0; index+1 < len(node.Content); index += 2 {
			keyNode := node.Content[index]
			valueNode := node.Content[index+1]
			normalized[keyNode.Value] = normalizeYAMLNode(valueNode)
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

func normalizeYAMLScalar(node *yaml.Node) any {
	var value any
	if err := node.Decode(&value); err != nil {
		return node.Value
	}
	if typed, ok := value.(time.Time); ok {
		return typed.Format("2006-01-02")
	}
	return value
}
