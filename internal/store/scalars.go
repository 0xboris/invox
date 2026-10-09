package store

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/config"
	"github.com/0xboris/invox/internal/invoice"
	yaml "gopkg.in/yaml.v3"
)

// decodeScalar fills out, a pointer to one of the invoice or config value
// types, from n and reports whether out is such a type. It reports a wrong
// kind or a malformed value without the field path, which the decoder adds.
func decodeScalar(out any, n *yaml.Node) (bool, error) {
	var expected string
	var parse func(text string) error
	switch v := out.(type) {
	case *invoice.Text:
		expected = "a string"
		parse = func(text string) error { *v = invoice.Text(text); return nil }
	case *invoice.Status:
		expected = "a string"
		parse = func(text string) error { *v = invoice.ParseStatus(text); return nil }
	case *invoice.Decimal:
		expected = "a decimal number such as 12 or 12.50"
		parse = func(text string) (err error) { *v, err = invoice.ParseDecimal(text); return err }
	case *invoice.Rate:
		expected = "a number or percent string"
		parse = func(text string) (err error) { *v, err = invoice.ParseRate(text); return err }
	case *invoice.Date:
		expected = "YYYY-MM-DD"
		parse = func(text string) (err error) { *v, err = invoice.ParseDate(text); return err }
	case *invoice.Count:
		expected = "an integer"
		parse = func(text string) (err error) { *v, err = invoice.ParseCount(text); return err }
	case *config.Text:
		expected = "a string"
		parse = func(text string) error { *v = config.Text(strings.TrimSpace(text)); return nil }
	case *config.Path:
		expected = "a string"
		parse = func(text string) error { *v = config.NewPath(text); return nil }
	case *config.Int:
		expected = "an integer"
		parse = func(text string) error {
			count, err := invoice.ParseCount(text)
			if err == nil && !count.IsSet() {
				err = fmt.Errorf("expected an integer, got `%s`", text)
			}
			*v = config.Int(count.Int())
			return err
		}
	default:
		return false, nil
	}
	if n.Kind != yaml.ScalarNode {
		return true, fmt.Errorf("expected %s, got %s", expected, describeNode(n))
	}
	return true, parse(scalarText(n))
}

// scalarText is the text of a scalar node: its source text, except that a
// null is "", a boolean is true or false, and a timestamp is its date.
// Numbers keep their source text, because yaml.v3 would read 01067 or 0042
// as octal and 12.50 as 12.5.
func scalarText(n *yaml.Node) string {
	switch n.ShortTag() {
	case "!!null":
		return ""
	case "!!bool":
		var value bool
		if n.Decode(&value) == nil {
			return strconv.FormatBool(value)
		}
	case "!!timestamp":
		var value time.Time
		if n.Decode(&value) == nil {
			return value.Format(time.DateOnly)
		}
	case "!!binary":
		var value string
		if n.Decode(&value) == nil {
			return value
		}
	}
	return n.Value
}

// describeNode names the kind of value n holds, for "expected X, got Y".
func describeNode(n *yaml.Node) string {
	switch n.Kind {
	case yaml.MappingNode:
		return "a mapping"
	case yaml.SequenceNode:
		return "a list"
	}
	switch n.ShortTag() {
	case "!!str":
		return "a string"
	case "!!int":
		return "an integer"
	case "!!float":
		return "a number"
	case "!!bool":
		return "a boolean"
	case "!!null":
		return "null"
	}
	return "a " + strings.TrimPrefix(n.ShortTag(), "!!")
}
