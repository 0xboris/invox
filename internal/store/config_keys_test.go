package store

import (
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/config"
)

// TestConfigKeysDocumented checks that the settings `invox help config`
// and the starter config.yaml list are exactly the keys config.yaml
// decodes. The help is read from its generated page, which CI keeps equal
// to the command's help.
func TestConfigKeysDocumented(t *testing.T) {
	want := configKeys(reflect.TypeFor[config.Config](), "")
	page, err := os.ReadFile(filepath.Join("..", "..", "docs", "cli", "invox_config.md"))
	if err != nil {
		t.Fatal(err)
	}
	help := settingsList(t, string(page), "Supported settings:", regexp.MustCompile(`^  (\S+)  `))
	if !slices.Equal(help, want) {
		t.Errorf("config help lists %v, want the config.yaml keys %v", help, want)
	}

	starter := settingsList(t, NewHost(HostInputs{GOOS: "linux"}).ConfigTemplate(), "# Supported settings:", regexp.MustCompile(`^#   (\S+)$`))
	if !slices.Equal(starter, want) {
		t.Errorf("starter config.yaml lists %v, want the config.yaml keys %v", starter, want)
	}
}

// configKeys returns the dotted keys of the settings in the schema struct
// t, sorted.
func configKeys(t reflect.Type, prefix string) []string {
	var keys []string
	for key, name := range schemaKeys[t] {
		field, _ := t.FieldByName(name)
		ft := field.Type
		if ft.Kind() == reflect.Pointer {
			ft = ft.Elem()
		}
		if _, nested := schemaKeys[ft]; nested {
			keys = append(keys, configKeys(ft, prefix+key+".")...)
		} else {
			keys = append(keys, prefix+key)
		}
	}
	slices.Sort(keys)
	return keys
}

// settingsList returns the keys that item matches in the lines after
// heading, up to the first line it does not match and that is not
// indented further, sorted.
func settingsList(t *testing.T, text, heading string, item *regexp.Regexp) []string {
	t.Helper()
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	start := slices.Index(lines, heading)
	if start < 0 {
		t.Fatalf("no %q line in:\n%s", heading, text)
	}
	var keys []string
	for _, line := range lines[start+1:] {
		if m := item.FindStringSubmatch(line); m != nil {
			keys = append(keys, m[1])
			continue
		}
		if indent := strings.TrimPrefix(line, "#"); !strings.HasPrefix(indent, "    ") {
			break
		}
	}
	slices.Sort(keys)
	return keys
}
