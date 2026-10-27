package editor

import (
	"slices"
	"testing"
)

func TestSplitWords(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		in    string
		posix bool
		want  []string
	}{
		{name: "one word", in: "vim", posix: true, want: []string{"vim"}},
		{name: "runs of whitespace", in: "code \t -w", posix: true, want: []string{"code", "-w"}},
		{name: "single quotes are literal", in: `ed '$HOME \x "q"'`, posix: true, want: []string{"ed", `$HOME \x "q"`}},
		{name: "double quotes group", in: `vim -c "set ft=yaml"`, posix: true, want: []string{"vim", "-c", "set ft=yaml"}},
		{name: "backslash in double quotes escapes only some characters", in: `ed "a\"b\\c\d\$"`, posix: true, want: []string{"ed", `a"b\c\d$`}},
		{name: "backslash outside quotes escapes the next character", in: `my\ editor \'x`, posix: true, want: []string{"my editor", "'x"}},
		{name: "quotes join with the word around them", in: `a"b c"'d e'f`, posix: true, want: []string{"ab cd ef"}},
		{name: "empty quotes are a word", in: `ed ''`, posix: true, want: []string{"ed", ""}},
		{name: "trailing backslash is kept", in: `ed x\`, posix: true, want: []string{"ed", `x\`}},
		{name: "windows path in double quotes", in: `"C:\Program Files\Ed\ed.exe" -w`, posix: false, want: []string{`C:\Program Files\Ed\ed.exe`, "-w"}},
		{name: "windows backslash is literal", in: `C:\tools\ed.exe`, posix: false, want: []string{`C:\tools\ed.exe`}},
		{name: "windows single quote is literal", in: `ed 'a b'`, posix: false, want: []string{"ed", "'a", "b'"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := splitWords(tc.in, tc.posix)
			if err != nil {
				t.Fatalf("splitWords(%q) returned error: %v", tc.in, err)
			}
			if !slices.Equal(got, tc.want) {
				t.Fatalf("splitWords(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestSplitWordsUnterminatedQuote(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		in    string
		posix bool
	}{
		{in: `vim 'x`, posix: true},
		{in: `vim "x`, posix: true},
		{in: `vim "x\"`, posix: true},
		{in: `"C:\ed.exe`, posix: false},
	} {
		words, err := splitWords(tc.in, tc.posix)
		if err == nil || err.Error() != "unterminated quote" {
			t.Fatalf("splitWords(%q) = %q, %v; want error %q", tc.in, words, err, "unterminated quote")
		}
	}
}

func TestNeedsShell(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want bool
	}{
		{in: "vim", want: false},
		{in: "code -w", want: false},
		{in: `vim -c "set ft=yaml"`, want: false},
		{in: `"/opt/My Editor/ed" --wait`, want: false},
		{in: "$HOME/bin/ed", want: true},
		{in: "ed `which x`", want: true},
		{in: "ed | tee log", want: true},
		{in: "ed && true", want: true},
		{in: "ed; true", want: true},
		{in: "ed < /dev/tty", want: true},
		{in: "ed > log", want: true},
		{in: "(ed)", want: true},
		{in: "ed *.yaml", want: true},
		{in: "ed ?", want: true},
		{in: "ed [ab]", want: true},
		{in: "ed # note", want: true},
		{in: "~/bin/ed", want: true},
		{in: "ed\nmore", want: true},
		{in: "FOO=1 vim", want: true},
	}
	for _, tc := range tests {
		if got := needsShell(tc.in); got != tc.want {
			t.Errorf("needsShell(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}
