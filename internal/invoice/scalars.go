package invoice

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/money"
)

// Text is a value read as the text it was written as: 01067 stays "01067",
// 12.50 stays "12.50" and 0x10 stays "0x10". A YAML timestamp reads as its
// date, YYYY-MM-DD. A mapping or a list is an error, never its Go string.
type Text string

// Trim returns t without surrounding space, the form fields are compared and
// looked up in.
func (t Text) Trim() string { return strings.TrimSpace(string(t)) }

func (t Text) IsSet() bool { return t.Trim() != "" }

// Decimal is an amount, quantity or rate written as an optional minus sign,
// digits and an optional fraction, such as 12 or 12.50. Leading zeros are
// decimal. An empty value leaves it unset.
type Decimal struct {
	value *big.Rat
	// text is the value as written.
	text string
}

func (d Decimal) IsSet() bool { return d.value != nil }

// Rat returns the value, or nil when it is unset.
func (d Decimal) Rat() *big.Rat { return d.value }

// String returns the decimal as written, or "" when it is unset.
func (d Decimal) String() string { return d.text }

// Rate is a VAT rate in percent, written as a decimal with an optional
// trailing %, such as 20, 7.7 or "19%". An empty value leaves it unset.
type Rate struct {
	value *big.Rat
	// text is the value as written, without surrounding space.
	text string
}

func (r Rate) IsSet() bool { return r.value != nil }

// Percent returns the rate, or nil when it is unset.
func (r Rate) Percent() *big.Rat { return r.value }

// Date is a calendar date written as YYYY-MM-DD, or as a YAML timestamp. An
// empty value leaves it unset.
type Date struct {
	t time.Time
}

func (d Date) IsSet() bool { return !d.t.IsZero() }

// String returns the date as YYYY-MM-DD, or "" when it is unset.
func (d Date) String() string {
	if !d.IsSet() {
		return ""
	}
	return d.t.Format(time.DateOnly)
}

// Display returns the date as DD.MM.YYYY, the form invoices print.
func (d Date) Display() string {
	if !d.IsSet() {
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

func (c Count) IsSet() bool { return c.set }

// Int returns the value, 0 when it is unset.
func (c Count) Int() int64 { return c.value }

// ParseDecimal reads a Decimal from its text. Empty text leaves it unset.
func ParseDecimal(text string) (Decimal, error) {
	if strings.TrimSpace(text) == "" {
		return Decimal{}, nil
	}
	value, ok := money.ParseDecimal(text)
	if !ok {
		return Decimal{}, fmt.Errorf("expected a decimal number such as 12 or 12.50, got `%s`", text)
	}
	return Decimal{value: value, text: text}, nil
}

// ParseRate reads a Rate from its text. Empty text leaves it unset.
func ParseRate(raw string) (Rate, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return Rate{}, nil
	}
	value, ok := money.ParseDecimal(strings.TrimSuffix(text, "%"))
	if !ok {
		return Rate{}, fmt.Errorf("expected a number or percent string, got `%s`", raw)
	}
	return Rate{value: value, text: text}, nil
}

// String returns the rate as written, without surrounding space, or "" when
// it is unset.
func (r Rate) String() string { return r.text }

// ParseDate reads a Date from YYYY-MM-DD. Empty text leaves it unset.
func ParseDate(text string) (Date, error) {
	if strings.TrimSpace(text) == "" {
		return Date{}, nil
	}
	t, err := time.Parse(time.DateOnly, text)
	if err != nil {
		return Date{}, fmt.Errorf("expected YYYY-MM-DD, got `%s`", text)
	}
	return Date{t: t}, nil
}

// ParseCount reads a Count from base-10 digits. Empty text leaves it unset.
func ParseCount(text string) (Count, error) {
	if strings.TrimSpace(text) == "" {
		return Count{}, nil
	}
	value, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return Count{}, fmt.Errorf("expected an integer, got `%s`", text)
	}
	return Count{value: value, set: true}, nil
}
