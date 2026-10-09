package factory_test

import (
	"regexp"
	"slices"
	"testing"

	"github.com/0xboris/invox/internal/cli/helptext"
	"github.com/0xboris/invox/internal/render/latex"
)

// The placeholder table in the help is written by hand; the renderer's
// table decides what a template may use. They must list the same names.
func TestPlaceholderDocsMatch(t *testing.T) {
	var documented []string
	row := regexp.MustCompile(`(?m)^    (@@[A-Z0-9_]+@@) `)
	for _, match := range row.FindAllStringSubmatch(helptext.TemplatePlaceholderReference(), -1) {
		documented = append(documented, match[1])
	}
	known := latex.Placeholders()
	for _, name := range known {
		if !slices.Contains(documented, name) {
			t.Errorf("%s is filled by the renderer but missing from the help's placeholder table", name)
		}
	}
	for _, name := range documented {
		if !slices.Contains(known, name) {
			t.Errorf("%s is in the help's placeholder table but the renderer does not know it", name)
		}
	}
	if len(documented) != len(known) {
		t.Errorf("the help documents %d placeholders, the renderer knows %d", len(documented), len(known))
	}
}
