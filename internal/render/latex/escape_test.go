package latex

import (
	"testing"
)

func TestLatexEscapeShieldsLeadingBracketAndStar(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "[Q1] 2026", want: "{}[Q1] 2026"},
		{input: "*Hauptstraße 1", want: "{}*Hauptstraße 1"},
		{input: "  [x]", want: "{}  [x]"},
		{input: "Q1 [2026]", want: "Q1 [2026]"},
		{input: "Main St. 1*", want: "Main St. 1*"},
		{input: "A & B_1 50%", want: `A \& B\_1 50\%`},
		{input: "", want: ""},
	}
	for _, tt := range tests {
		if got := Escape(tt.input); got != tt.want {
			t.Errorf("Escape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
