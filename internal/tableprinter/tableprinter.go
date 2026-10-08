// Package tableprinter prints the output of list commands.
package tableprinter

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/0xboris/invox/internal/iostreams"
)

// Column is one table column. MaxWidth caps the column on a terminal, in
// cells, and 0 never truncates. Piped output is never truncated.
type Column struct {
	Header   string
	MaxWidth int
}

// Table is the output of a list command: one row of fields per record. On a
// terminal it prints a header and aligned columns, and an empty table prints
// EmptyHint to stderr. Piped, it prints tab-separated rows with no header,
// and nothing for an empty table.
type Table struct {
	Columns   []Column
	rows      [][]string
	EmptyHint string
}

// AddRow adds one record.
func (t *Table) AddRow(fields ...string) {
	t.rows = append(t.rows, fields)
}

// Print writes the table to ios.Out, or the empty hint to ios.ErrOut.
func (t *Table) Print(ios *iostreams.IOStreams) {
	if !ios.IsStdoutTTY() {
		for _, row := range t.rows {
			fields := make([]string, len(row))
			for i, field := range row {
				fields[i] = EscapeTSVField(field)
			}
			fmt.Fprintln(ios.Out, strings.Join(fields, "\t"))
		}
		return
	}

	if len(t.rows) == 0 {
		fmt.Fprintln(ios.ErrOut, t.EmptyHint)
		return
	}

	cells := make([][]string, 0, len(t.rows)+1)
	header := make([]string, len(t.Columns))
	for i, col := range t.Columns {
		header[i] = col.Header
	}
	cells = append(cells, header)
	for _, row := range t.rows {
		fields := make([]string, len(row))
		for i, field := range row {
			fields[i] = truncateCells(terminalField(field), t.Columns[i].MaxWidth)
		}
		cells = append(cells, fields)
	}

	widths := make([]int, len(t.Columns))
	for _, row := range cells {
		for i, field := range row {
			widths[i] = max(widths[i], cellWidth(field))
		}
	}

	for _, row := range cells {
		var line strings.Builder
		for i, field := range row {
			line.WriteString(field)
			if i < len(row)-1 {
				line.WriteString(strings.Repeat(" ", widths[i]-cellWidth(field)+2))
			}
		}
		fmt.Fprintln(ios.Out, strings.TrimRight(line.String(), " "))
	}
}

// EscapeTSVField keeps a piped record on one line: backslash, tab, CR and LF
// are written as \\, \t, \r and \n, and terminal escape sequences and other
// control and format characters are dropped.
func EscapeTSVField(field string) string {
	var b strings.Builder
	for _, r := range stripEscapeSequences(field) {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\t':
			b.WriteString(`\t`)
		case '\r':
			b.WriteString(`\r`)
		case '\n':
			b.WriteString(`\n`)
		default:
			if printable(r) {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// terminalField makes a field safe to print in a terminal column: tabs and
// line breaks become spaces, and terminal escape sequences and other control
// and format characters are dropped.
func terminalField(field string) string {
	var b strings.Builder
	for _, r := range stripEscapeSequences(field) {
		switch {
		case r == '\t' || r == '\r' || r == '\n':
			b.WriteRune(' ')
		case printable(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// printable reports whether r may reach the terminal. Control characters
// and format characters (bidi overrides, zero-width spaces, soft hyphens) are
// dropped so data cannot reorder or hide text.
func printable(r rune) bool {
	return !unicode.IsControl(r) && !unicode.Is(unicode.Cf, r)
}

// stripEscapeSequences removes ANSI escape sequences (CSI, OSC and other
// ESC-introduced sequences, and their single-byte C1 forms) so text from data
// files cannot move the cursor, recolor or retitle the user's terminal.
// A lone ESC left behind is dropped by the caller.
func stripEscapeSequences(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		var introducer rune
		switch {
		case r == 0x1b && i+1 < len(runes) && runes[i+1] >= 0x20 && runes[i+1] <= 0x7e:
			i++
			introducer = runes[i]
		case r == 0x9b:
			introducer = '['
		case r == 0x9d:
			introducer = ']'
		case r == 0x90 || r == 0x98 || r == 0x9e || r == 0x9f:
			introducer = 'P'
		default:
			b.WriteRune(r)
			continue
		}

		switch introducer {
		case '[':
			// CSI: parameter and intermediate bytes, then one final byte.
			for i+1 < len(runes) {
				i++
				if runes[i] >= 0x40 && runes[i] <= 0x7e {
					break
				}
			}
		case ']', 'P', 'X', '^', '_':
			// OSC, DCS, SOS, PM and APC run to BEL or ST (ESC \ or 0x9c).
			for i+1 < len(runes) {
				i++
				if runes[i] == 0x07 || runes[i] == 0x9c {
					break
				}
				if runes[i] == 0x1b && i+1 < len(runes) && runes[i+1] == '\\' {
					i++
					break
				}
			}
		default:
			// nF sequences such as ESC ( B: intermediate bytes, then one
			// final byte.
			for introducer >= 0x20 && introducer <= 0x2f && i+1 < len(runes) {
				i++
				introducer = runes[i]
			}
		}
	}
	return b.String()
}

// truncateCells shortens s to at most maxWidth cells, ending in an ellipsis
// when it cuts. A maxWidth of 0 leaves s alone.
func truncateCells(s string, maxWidth int) string {
	if maxWidth <= 0 || cellWidth(s) <= maxWidth {
		return s
	}
	var b strings.Builder
	width := 0
	for _, r := range s {
		w := runeCells(r)
		if width+w > maxWidth-1 {
			break
		}
		b.WriteRune(r)
		width += w
	}
	b.WriteRune('…')
	return b.String()
}

func cellWidth(s string) int {
	width := 0
	for _, r := range s {
		width += runeCells(r)
	}
	return width
}

// runeCells approximates how many terminal cells r takes: 0 for combining
// marks and emoji skin-tone modifiers, 2 for East Asian Wide and Fullwidth
// characters (which include emoji with emoji presentation), 1 otherwise.
func runeCells(r rune) int {
	if unicode.In(r, unicode.Mn, unicode.Me) || (r >= 0x1f3fb && r <= 0x1f3ff) {
		return 0
	}
	for _, wide := range wideRanges {
		if r < wide[0] {
			return 1
		}
		if r <= wide[1] {
			return 2
		}
	}
	return 1
}

// wideRanges are the East Asian Wide and Fullwidth ranges of Unicode's
// EastAsianWidth.txt, slightly coarsened in the emoji blocks, in ascending
// order.
var wideRanges = [][2]rune{
	{0x1100, 0x115f},
	{0x231a, 0x231b},
	{0x2329, 0x232a},
	{0x23e9, 0x23ec},
	{0x23f0, 0x23f0},
	{0x23f3, 0x23f3},
	{0x25fd, 0x25fe},
	{0x2614, 0x2615},
	{0x2648, 0x2653},
	{0x267f, 0x267f},
	{0x2693, 0x2693},
	{0x26a1, 0x26a1},
	{0x26aa, 0x26ab},
	{0x26bd, 0x26be},
	{0x26c4, 0x26c5},
	{0x26ce, 0x26ce},
	{0x26d4, 0x26d4},
	{0x26ea, 0x26ea},
	{0x26f2, 0x26f3},
	{0x26f5, 0x26f5},
	{0x26fa, 0x26fa},
	{0x26fd, 0x26fd},
	{0x2705, 0x2705},
	{0x270a, 0x270b},
	{0x2728, 0x2728},
	{0x274c, 0x274c},
	{0x274e, 0x274e},
	{0x2753, 0x2755},
	{0x2757, 0x2757},
	{0x2795, 0x2797},
	{0x27b0, 0x27b0},
	{0x27bf, 0x27bf},
	{0x2b1b, 0x2b1c},
	{0x2b50, 0x2b50},
	{0x2b55, 0x2b55},
	{0x2e80, 0x303e},
	{0x3041, 0x33ff},
	{0x3400, 0x4dbf},
	{0x4e00, 0x9fff},
	{0xa000, 0xa4cf},
	{0xac00, 0xd7a3},
	{0xf900, 0xfaff},
	{0xfe30, 0xfe4f},
	{0xff00, 0xff60},
	{0xffe0, 0xffe6},
	{0x1f004, 0x1f004},
	{0x1f0cf, 0x1f0cf},
	{0x1f18e, 0x1f18e},
	{0x1f191, 0x1f19a},
	{0x1f200, 0x1f202},
	{0x1f210, 0x1f23b},
	{0x1f240, 0x1f248},
	{0x1f250, 0x1f251},
	{0x1f260, 0x1f265},
	{0x1f300, 0x1f64f},
	{0x1f680, 0x1f6ff},
	{0x1f7e0, 0x1f7eb},
	{0x1f90c, 0x1f9ff},
	{0x1fa70, 0x1faff},
	{0x20000, 0x2fffd},
	{0x30000, 0x3fffd},
}
