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

// maxYAMLAliasNodes caps how many nodes a document may reach through
// aliases, counting every node inside each expansion. A real invoice reuses a
// few small anchors and stays in the thousands; a "billion laughs" file of a
// few hundred bytes reaches hundreds of millions.
const maxYAMLAliasNodes = 100_000

// YAMLAliasError reports an alias that normalizeYAMLNode cannot expand: one
// that refers to a node containing it, or one that takes the document past
// maxYAMLAliasNodes nodes reached through aliases.
type YAMLAliasError struct {
	Label     string
	Line      int
	Alias     string
	Recursive bool
}

func (e *YAMLAliasError) Error() string {
	if e.Recursive {
		return fmt.Sprintf("%s:%d: alias *%s refers to a node that contains it", e.Label, e.Line, e.Alias)
	}
	return fmt.Sprintf("%s:%d: aliases expand to more than %d nodes", e.Label, e.Line, maxYAMLAliasNodes)
}

// checkYAMLMappings rejects what normalizeYAMLNode cannot represent
// faithfully: a key defined twice in one mapping, a merge key whose value is
// not a mapping or a list of mappings, and aliases that recurse or expand
// past maxYAMLAliasNodes.
func checkYAMLMappings(node *yaml.Node, label string) error {
	if err := checkYAMLMappingKeys(node, label); err != nil {
		return err
	}
	return checkYAMLAliases(node, label)
}

// checkYAMLMappingKeys does not follow aliases, because the anchored node is
// checked where it is defined.
func checkYAMLMappingKeys(node *yaml.Node, label string) error {
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
		if err := checkYAMLMappingKeys(child, label); err != nil {
			return err
		}
	}
	return nil
}

// checkYAMLAliases walks the document the way normalizeYAMLNode expands it,
// keeping the nodes on the current path to catch an alias to one of them, and
// stops once aliases have reached maxYAMLAliasNodes nodes. An error names the
// outermost alias being expanded.
func checkYAMLAliases(document *yaml.Node, label string) error {
	reached := 0
	enclosing := map[*yaml.Node]bool{}
	var walk func(node, outerAlias *yaml.Node) error
	walk = func(node, outerAlias *yaml.Node) error {
		if node == nil {
			return nil
		}
		if node.Kind == yaml.AliasNode {
			if enclosing[node.Alias] {
				return &YAMLAliasError{Label: label, Line: node.Line, Alias: node.Value, Recursive: true}
			}
			if outerAlias == nil {
				outerAlias = node
			}
			return walk(node.Alias, outerAlias)
		}
		if outerAlias != nil {
			reached++
			if reached > maxYAMLAliasNodes {
				return &YAMLAliasError{Label: label, Line: outerAlias.Line, Alias: outerAlias.Value}
			}
		}
		enclosing[node] = true
		defer delete(enclosing, node)
		for _, child := range node.Content {
			if err := walk(child, outerAlias); err != nil {
				return err
			}
		}
		return nil
	}
	return walk(document, nil)
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
