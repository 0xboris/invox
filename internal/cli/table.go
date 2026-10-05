package cli

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/0xboris/invox/internal/iostreams"
)

// column is one table column. maxWidth caps the column on a terminal, in
// cells, and 0 never truncates. Piped output is never truncated.
type column struct {
	header   string
	maxWidth int
}

// table is the output of a list command: one row of fields per record. On a
// terminal it prints a header and aligned columns, and an empty table prints
// emptyHint to stderr. Piped, it prints tab-separated rows with no header,
// and nothing for an empty table.
type table struct {
	columns   []column
	rows      [][]string
	emptyHint string
}

func (t *table) addRow(fields ...string) {
	t.rows = append(t.rows, fields)
}

func (t *table) print(ios *iostreams.IOStreams) {
	if !ios.IsStdoutTTY() {
		for _, row := range t.rows {
			fields := make([]string, len(row))
			for i, field := range row {
				fields[i] = escapeTSVField(field)
			}
			fmt.Fprintln(ios.Out, strings.Join(fields, "\t"))
		}
		return
	}

	if len(t.rows) == 0 {
		fmt.Fprintln(ios.ErrOut, t.emptyHint)
		return
	}

	cells := make([][]string, 0, len(t.rows)+1)
	header := make([]string, len(t.columns))
	for i, col := range t.columns {
		header[i] = col.header
	}
	cells = append(cells, header)
	for _, row := range t.rows {
		fields := make([]string, len(row))
		for i, field := range row {
			fields[i] = truncateCells(terminalField(field), t.columns[i].maxWidth)
		}
		cells = append(cells, fields)
	}

	widths := make([]int, len(t.columns))
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
		fmt.Fprintln(ios.Out, line.String())
	}
}

// escapeTSVField keeps a piped record on one line: backslash, tab, CR and LF
// are written as \\, \t, \r and \n, and terminal escape sequences and other
// control characters are dropped.
func escapeTSVField(field string) string {
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
			if !unicode.IsControl(r) {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// terminalField makes a field safe to print in a terminal column: tabs and
// line breaks become spaces, and terminal escape sequences and other control
// characters are dropped.
func terminalField(field string) string {
	var b strings.Builder
	for _, r := range stripEscapeSequences(field) {
		switch {
		case r == '\t' || r == '\r' || r == '\n':
			b.WriteRune(' ')
		case !unicode.IsControl(r):
			b.WriteRune(r)
		}
	}
	return b.String()
}

// stripEscapeSequences removes ANSI escape sequences (CSI, OSC and other
// ESC-introduced sequences, and their single-byte C1 forms) so text from data
// files cannot move the cursor, recolor or retitle the user's terminal.
// A lone control character left behind is dropped by the caller.
func stripEscapeSequences(s string) string {
	runes := []rune(s)
	var b strings.Builder
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		var introducer rune
		switch {
		case r == 0x1b && i+1 < len(runes):
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
// marks and zero-width format characters, 2 for East Asian wide and
// fullwidth characters and common emoji, 1 otherwise.
func runeCells(r rune) int {
	if unicode.In(r, unicode.Mn, unicode.Me, unicode.Cf) {
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

// wideRanges are the East Asian Wide and Fullwidth blocks (Hangul Jamo, CJK,
// Hiragana, Katakana, Hangul syllables, fullwidth forms) and the main emoji
// blocks, in ascending order.
var wideRanges = [][2]rune{
	{0x1100, 0x115f},
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
	{0x1f300, 0x1f64f},
	{0x1f900, 0x1f9ff},
	{0x20000, 0x2fffd},
	{0x30000, 0x3fffd},
}
