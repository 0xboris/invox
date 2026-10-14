package tableprinter

import "testing"

func TestStripEscapeSequences(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "csi color", in: "a\x1b[1;31mb\x1b[0m", want: "ab"},
		{name: "osc title with bel", in: "a\x1b]0;pwned\x07b", want: "ab"},
		{name: "osc hyperlink with st", in: "\x1b]8;;http://x\x1b\\link\x1b]8;;\x1b\\", want: "link"},
		{name: "two-byte escape", in: "a\x1bcb", want: "ab"},
		{name: "c1 csi", in: "a\u009b2Jb", want: "ab"},
		{name: "unterminated csi", in: "a\x1b[31", want: "a"},
		{name: "charset designation", in: "a\x1b(Bb", want: "ab"},
		{name: "esc before non-ascii", in: "a\x1bäb", want: "a\x1bäb"},
		{name: "plain", in: "Müller & Söhne", want: "Müller & Söhne"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := stripEscapeSequences(tc.in); got != tc.want {
				t.Fatalf("stripEscapeSequences(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
