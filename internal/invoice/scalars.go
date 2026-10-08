package invoice

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/money"
	yaml "gopkg.in/yaml.v3"
)

// A scalarField decodes itself from one YAML node. decodeScalar reports a
// wrong kind or a malformed value without the field path, which the decoder
// adds.
type scalarField interface {
	decodeScalar(n *yaml.Node) error
}

// Text is a value read as the text it was written as: 01067 stays "01067",
// 12.50 stays "12.50" and 0x10 stays "0x10". A YAML timestamp reads as its
// date, YYYY-MM-DD. A mapping or a list is an error, never its Go string.
type Text string

func (t *Text) decodeScalar(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected a string, got %s", describeNode(n))
	}
	*t = Text(scalarText(n))
	return nil
}

// Trim returns t without surrounding space, the form fields are compared and
// looked up in.
func (t Text) Trim() string { return strings.TrimSpace(string(t)) }

func (t Text) isSet() bool { return t.Trim() != "" }

// Decimal is an amount, quantity or rate written as an optional minus sign,
// digits and an optional fraction, such as 12 or 12.50. Leading zeros are
// decimal. An empty value leaves it unset.
type Decimal struct {
	value *big.Rat
}

func (d *Decimal) decodeScalar(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected a decimal number such as 12 or 12.50, got %s", describeNode(n))
	}
	text := scalarText(n)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	value, ok := money.ParseDecimal(text)
	if !ok {
		return fmt.Errorf("expected a decimal number such as 12 or 12.50, got `%s`", text)
	}
	d.value = value
	return nil
}

func (d Decimal) isSet() bool { return d.value != nil }

// Rat returns the value, or nil when it is unset.
func (d Decimal) Rat() *big.Rat { return d.value }

// Rate is a VAT rate in percent, written as a decimal with an optional
// trailing %, such as 20, 7.7 or "19%". An empty value leaves it unset.
type Rate struct {
	value *big.Rat
	// text is the value as written, without surrounding space.
	text string
}

func (r *Rate) decodeScalar(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected a number or percent string, got %s", describeNode(n))
	}
	raw := scalarText(n)
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil
	}
	value, ok := money.ParseDecimal(strings.TrimSuffix(text, "%"))
	if !ok {
		return fmt.Errorf("expected a number or percent string, got `%s`", raw)
	}
	r.value, r.text = value, text
	return nil
}

func (r Rate) isSet() bool { return r.value != nil }

// Percent returns the rate, or nil when it is unset.
func (r Rate) Percent() *big.Rat { return r.value }

// Date is a calendar date written as YYYY-MM-DD, or as a YAML timestamp. An
// empty value leaves it unset.
type Date struct {
	t time.Time
}

func (d *Date) decodeScalar(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected YYYY-MM-DD, got %s", describeNode(n))
	}
	text := scalarText(n)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", text)
	if err != nil {
		return fmt.Errorf("expected YYYY-MM-DD, got `%s`", text)
	}
	d.t = t
	return nil
}

func (d Date) isSet() bool { return !d.t.IsZero() }

// String returns the date as YYYY-MM-DD, or "" when it is unset.
func (d Date) String() string {
	if !d.isSet() {
		return ""
	}
	return d.t.Format("2006-01-02")
}

// Display returns the date as DD.MM.YYYY, the form invoices print.
func (d Date) Display() string {
	if !d.isSet() {
		return ""
	}
	return d.t.Format("02.01.2006")
}

// Count is a whole number written in base 10, such as 30. An empty value
// leaves it unset.
type Count struct {
	value int64
	set   bool
}

func (c *Count) decodeScalar(n *yaml.Node) error {
	if n.Kind != yaml.ScalarNode {
		return fmt.Errorf("expected an integer, got %s", describeNode(n))
	}
	text := scalarText(n)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return fmt.Errorf("expected an integer, got `%s`", text)
	}
	c.value, c.set = value, true
	return nil
}

func (c Count) isSet() bool { return c.set }

// Int returns the value, 0 when it is unset.
func (c Count) Int() int64 { return c.value }

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
			return value.Format("2006-01-02")
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
