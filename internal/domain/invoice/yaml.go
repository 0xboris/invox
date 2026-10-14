package invoice

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// maxYAMLAliasNodes caps how many nodes a document may reach through
// aliases, counting every node inside each expansion. A real invoice reuses a
// few small anchors and stays in the thousands; a "billion laughs" file of a
// few hundred bytes reaches hundreds of millions.
const maxYAMLAliasNodes = 100_000

// YAMLAliasError reports an alias that the decoder cannot expand: one
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

// checkYAMLMappings rejects what the decoder cannot represent
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

// checkYAMLAliases walks the document the way the decoder expands it,
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

// DecodeError is a value in a YAML file that does not fit the schema: an
// unknown key, a value of the wrong kind, or a malformed number or date.
type DecodeError struct {
	File string
	Line int
	// Path is the field, such as positions[2].unit_price. It is "" when
	// Problem names the field itself.
	Path    string
	Problem string
	// Field is the field that did not decode, which validation then
	// skips. It is "" for an unknown or removed key.
	Field string
	// UnknownKey is set when the problem is a key the schema does not
	// define. Schema then names the kind of file: "customers", "issuer" or
	// "invoice".
	UnknownKey bool
	Schema     string
}

// failedFields returns the Field of every *DecodeError in err.
func failedFields(err error) map[string]bool {
	failed := map[string]bool{}
	var walk func(error)
	walk = func(err error) {
		switch e := err.(type) {
		case *DecodeError:
			if e.Field != "" {
				failed[e.Field] = true
			}
		case interface{ Unwrap() []error }:
			for _, inner := range e.Unwrap() {
				walk(inner)
			}
		}
	}
	walk(err)
	return failed
}

// within reports whether path is one of fields or lies inside one of them.
func within(path string, fields map[string]bool) bool {
	for field := range fields {
		if path == field || strings.HasPrefix(path, field+".") || strings.HasPrefix(path, field+"[") {
			return true
		}
	}
	return false
}

func (e *DecodeError) Error() string {
	if e.Path == "" {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Problem)
	}
	return fmt.Sprintf("%s:%d: %s: %s", e.File, e.Line, e.Path, e.Problem)
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

// decodeYAMLNode decodes n, an entry inside the file label, into out. A
// strict decode rejects keys the schema does not define.
func decodeYAMLNode(n *yaml.Node, label string, out any, strict bool) error {
	d := newYAMLDecoder(label, out, strict)
	d.decode(n, reflect.ValueOf(out).Elem(), d.rootPath())
	return errors.Join(d.errs...)
}

type yamlDecoder struct {
	label  string
	strict bool
	schema string
	// root is the root mapping of the file, where a key whose value
	// defines an anchor is a holder of definitions, not an unknown key.
	root *yaml.Node
	errs []error
}

func newYAMLDecoder(label string, out any, strict bool) *yamlDecoder {
	d := &yamlDecoder{label: label, strict: strict}
	switch out.(type) {
	case *Customer:
		d.schema = "customers"
	case *IssuerFile:
		d.schema = "issuer"
	case *InvoiceFile:
		d.schema = "invoice"
	}
	return d
}

// rootPath is the prefix field paths start with, the same one validation
// messages use: customer.name, issuer.payment.iban, invoice.number.
func (d *yamlDecoder) rootPath() string {
	switch d.schema {
	case "customers":
		return "customer"
	case "issuer":
		return "issuer"
	}
	return ""
}

func (d *yamlDecoder) fail(n *yaml.Node, path, problem string) {
	d.errs = append(d.errs, &DecodeError{File: d.label, Line: n.Line, Path: path, Problem: problem, Field: path})
}

// failShape reports a value of the wrong kind for the struct or list at
// path; problem names the path itself.
func (d *yamlDecoder) failShape(n *yaml.Node, path, problem string) {
	d.errs = append(d.errs, &DecodeError{File: d.label, Line: n.Line, Problem: problem, Field: path})
}

// decode fills out from n. Schema structs map YAML keys to fields with yaml
// tags, a pointer field stays nil when its key is missing or null, and a
// scalarField decodes its own node.
func (d *yamlDecoder) decode(n *yaml.Node, out reflect.Value, path string) {
	n = resolveYAMLAlias(n)
	if field, ok := out.Addr().Interface().(scalarField); ok {
		if err := field.decodeScalar(n); err != nil {
			d.fail(n, path, err.Error())
		}
		return
	}
	if n.Kind == yaml.ScalarNode && n.ShortTag() == "!!null" {
		return
	}
	switch out.Kind() {
	case reflect.Pointer:
		value := reflect.New(out.Type().Elem())
		d.decode(n, value.Elem(), path)
		out.Set(value)
	case reflect.Struct:
		if n.Kind != yaml.MappingNode {
			d.failShape(n, path, fmt.Sprintf("%s must be a mapping, got %s", path, describeNode(n)))
			return
		}
		d.decodeStruct(n, out, path)
	case reflect.Slice:
		if n.Kind != yaml.SequenceNode {
			d.failShape(n, path, fmt.Sprintf("%s must be a list, got %s", path, describeNode(n)))
			return
		}
		items := reflect.MakeSlice(out.Type(), len(n.Content), len(n.Content))
		for index, child := range n.Content {
			d.decode(child, items.Index(index), fmt.Sprintf("%s[%d]", path, index+1))
		}
		out.Set(items)
	default:
		panic(fmt.Sprintf("invoice: no YAML decoding for %s", out.Type()))
	}
}

// removedKey marks a key invox no longer reads. Its replacement tag names
// the key that took its place.
type removedKey struct{}

var removedKeyType = reflect.TypeOf(removedKey{})

func (d *yamlDecoder) decodeStruct(n *yaml.Node, out reflect.Value, path string) {
	fields := yamlFields(out.Type())
	for _, pair := range mappingPairs(n) {
		key := pair.key.Value
		fieldPath := key
		if path != "" {
			fieldPath = path + "." + key
		}
		index, ok := fields[key]
		switch {
		case !ok:
			// A top-level key whose value defines an anchor holds a
			// definition for aliases elsewhere, as in
			// `reduced: &reduced {vat_percent: 10}`.
			if d.strict && !(n == d.root && pair.value.Anchor != "") {
				problem := fmt.Sprintf("unknown key %q", key)
				if path != "" {
					problem += " in " + path
				}
				d.errs = append(d.errs, &DecodeError{File: d.label, Line: pair.key.Line, Problem: problem, UnknownKey: true, Schema: d.schema})
			}
		case out.Type().Field(index).Type == removedKeyType:
			d.errs = append(d.errs, &DecodeError{File: d.label, Line: pair.key.Line, Path: fieldPath, Problem: "unsupported key; use " + out.Type().Field(index).Tag.Get("replacement")})
		default:
			d.decode(pair.value, out.Field(index), fieldPath)
		}
	}
}

// yamlFields maps the yaml tag of each field of a schema struct to the
// field's index.
func yamlFields(t reflect.Type) map[string]int {
	fields := make(map[string]int, t.NumField())
	for index := range t.NumField() {
		if name := t.Field(index).Tag.Get("yaml"); name != "" {
			fields[name] = index
		}
	}
	return fields
}

type yamlPair struct {
	key, value *yaml.Node
}

// mappingPairs returns the entries of a mapping with its merge key applied
// as YAML 1.1 defines it: the mapping's own keys win over merged ones, and
// in `<<: [*a, *b]` a key from *a wins over the same key from *b.
// checkYAMLMappings has already rejected merge values that are not
// mappings.
func mappingPairs(n *yaml.Node) []yamlPair {
	var pairs []yamlPair
	var merges []*yaml.Node
	seen := make(map[string]bool, len(n.Content)/2)
	for index := 0; index+1 < len(n.Content); index += 2 {
		key, value := n.Content[index], n.Content[index+1]
		if isYAMLMergeKey(key) {
			merges = append(merges, value)
			continue
		}
		seen[key.Value] = true
		pairs = append(pairs, yamlPair{key: key, value: value})
	}
	for _, merge := range merges {
		sources := []*yaml.Node{merge}
		if merge.Kind == yaml.SequenceNode {
			sources = merge.Content
		}
		for _, source := range sources {
			for _, pair := range mappingPairs(resolveYAMLAlias(source)) {
				if !seen[pair.key.Value] {
					seen[pair.key.Value] = true
					pairs = append(pairs, pair)
				}
			}
		}
	}
	return pairs
}
