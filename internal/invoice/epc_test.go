package invoice

import (
	"strings"
	"testing"
)

func TestQRCodePayloadTeXSourceUsesQrcodeEscapesForReservedCharacters(t *testing.T) {
	payload := []byte("A B\\C^D~E%F#G&H_I$J{K}L\n")

	got := qrcodePayloadTeXSource(payload)
	want := strings.Join([]string{
		"A",
		`\noexpand\ `,
		"B",
		`\noexpand\\`,
		"C",
		`\noexpand\^`,
		"D",
		`\noexpand\~`,
		"E",
		`\noexpand\%`,
		"F",
		`\noexpand\#`,
		"G",
		`\noexpand\&`,
		"H",
		`\noexpand\_`,
		"I",
		`\noexpand\$`,
		"J",
		`\noexpand\{`,
		"K",
		`\noexpand\}`,
		"L",
		`\noexpand\?`,
	}, "")
	if got != want {
		t.Fatalf("qrcodePayloadTeXSource(%q) = %q, want %q", payload, got, want)
	}
}
