package invoice

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v3"

	"github.com/0xboris/invox/internal/fsutil"
)

func loadYAMLDocument(path string) (*yaml.Node, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return parseYAMLDocumentSource(source, path)
}

func parseYAMLDocumentSource(source []byte, label string) (*yaml.Node, error) {
	var document yaml.Node
	if err := yaml.Unmarshal(source, &document); err != nil {
		return nil, fmt.Errorf("%s: %w", label, err)
	}
	if err := checkYAMLMappings(&document, label); err != nil {
		return nil, err
	}
	return &document, nil
}

// writeYAMLDocument replaces path with document, keeping the file's mode.
func writeYAMLDocument(path string, document *yaml.Node) error {
	data, err := encodeYAMLDocument(document)
	if err != nil {
		return err
	}
	return fsutil.WriteFile(path, data, fsutil.Public)
}

func encodeYAMLDocument(document *yaml.Node) ([]byte, error) {
	clearYAMLMergeTags(document)
	var buffer bytes.Buffer
	encoder := yaml.NewEncoder(&buffer)
	encoder.SetIndent(2)
	if err := encoder.Encode(document); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// nodeText is the text of a scalar node, "" for a missing node, a mapping
// or a list.
func nodeText(node *yaml.Node) string {
	if node == nil {
		return ""
	}
	node = resolveYAMLAlias(node)
	if node.Kind != yaml.ScalarNode {
		return ""
	}
	return scalarText(node)
}

func documentRootMapping(document *yaml.Node, label string) (*yaml.Node, error) {
	if document == nil || len(document.Content) == 0 {
		return nil, fmt.Errorf("%s: root value must be a mapping", label)
	}
	root := document.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: root value must be a mapping", label)
	}
	return root, nil
}

// invoiceMapping returns the `invoice` mapping of root. label starts the
// error, which says whether the key is missing or holds something else.
func invoiceMapping(root *yaml.Node, label string) (*yaml.Node, error) {
	invoiceNode := findMappingValue(root, "invoice")
	if invoiceNode == nil {
		return nil, fmt.Errorf("%s: missing `invoice` mapping", label)
	}
	if invoiceNode.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: `invoice` must be a mapping", label)
	}
	return invoiceNode, nil
}

func getOrCreateMappingNode(parent *yaml.Node, key string) *yaml.Node {
	if existing := findMappingValue(parent, key); existing != nil {
		if existing.Kind == yaml.MappingNode {
			return existing
		}
		existing.Kind = yaml.MappingNode
		existing.Tag = "!!map"
		existing.Value = ""
		existing.Content = nil
		return existing
	}

	child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	appendMappingNode(parent, key, child)
	return child
}

func setMappingString(parent *yaml.Node, key, value string) {
	if existing := findMappingValue(parent, key); existing != nil {
		existing.Kind = yaml.ScalarNode
		existing.Tag = "!!str"
		existing.Value = value
		existing.Content = nil
		return
	}
	appendMappingNode(parent, key, scalarNode(value))
}

func setMappingSequence(parent *yaml.Node, key string, content []*yaml.Node) {
	if existing := findMappingValue(parent, key); existing != nil {
		existing.Kind = yaml.SequenceNode
		existing.Tag = "!!seq"
		existing.Value = ""
		existing.Content = content
		return
	}
	appendMappingNode(parent, key, &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     "!!seq",
		Content: content,
	})
}

func deleteMappingKey(parent *yaml.Node, key string) {
	if parent == nil || parent.Kind != yaml.MappingNode {
		return
	}
	for index := 0; index+1 < len(parent.Content); index += 2 {
		if parent.Content[index].Value == key {
			parent.Content = append(parent.Content[:index], parent.Content[index+2:]...)
			return
		}
	}
}

func markdownFrontMatter(source []byte) ([]byte, bool) {
	text := strings.ReplaceAll(string(source), "\r\n", "\n")
	if !strings.HasPrefix(text, "---\n") {
		return nil, false
	}
	remainder := text[len("---\n"):]
	end := strings.Index(remainder, "\n---\n")
	if end < 0 {
		return nil, false
	}
	// The leading newline stands in for the opening `---`, so YAML line
	// numbers in errors match the lines of the Markdown file.
	return []byte("\n" + remainder[:end]), true
}

func findMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func appendMappingNode(node *yaml.Node, key string, value *yaml.Node) {
	node.Content = append(node.Content, scalarNode(key), value)
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: value,
	}
}

func loadArchivedInvoiceDocument(path string) (*yaml.Node, bool, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		document, err := loadYAMLDocument(path)
		if err != nil {
			return nil, true, err
		}
		return document, true, nil
	case ".md", ".markdown":
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, false, err
		}
		frontMatter, ok := markdownFrontMatter(source)
		if !ok {
			return nil, false, nil
		}
		document, err := parseYAMLDocumentSource(frontMatter, "front matter in "+path)
		if err != nil {
			return nil, true, err
		}
		return document, true, nil
	default:
		return nil, false, nil
	}
}

// decodeYAMLFile reads the YAML file at path and decodes its root mapping
// into out, a pointer to a schema struct. Problems with values come back as
// *DecodeError values, several joined with errors.Join in file order.
func decodeYAMLFile(path string, out any, strict bool) error {
	document, err := loadYAMLDocument(path)
	if err != nil {
		return err
	}
	return decodeYAMLDocument(document, path, out, strict)
}

func decodeYAMLDocument(document *yaml.Node, label string, out any, strict bool) error {
	root, err := documentRootMapping(document, label)
	if err != nil {
		return err
	}
	d := newYAMLDecoder(label, out, strict)
	d.root = root
	d.decode(root, reflect.ValueOf(out).Elem(), d.rootPath())
	return errors.Join(d.errs...)
}
