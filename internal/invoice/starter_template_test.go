package invoice

import (
	"strings"
	"testing"
)

// The starter is built by tectonic (XeTeX). inputenc/fontenc drop or mangle
// non-ASCII text there, and without \tracinglostchars=3 a missing glyph is
// silently dropped while the build still succeeds (#22).
func TestStarterTemplateUsesUnicodeFontSetup(t *testing.T) {
	source, err := starterFiles.ReadFile("starter/template.tex")
	if err != nil {
		t.Fatalf("ReadFile(starter/template.tex) returned error: %v", err)
	}
	text := string(source)

	for _, want := range []string{
		`\usepackage{fontspec}`,
		`\tracinglostchars=3`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("starter template does not contain %q", want)
		}
	}
	for _, unwanted := range []string{
		`{inputenc}`,
		`{fontenc}`,
	} {
		if strings.Contains(text, unwanted) {
			t.Errorf("starter template still loads %q", unwanted)
		}
	}
}
