package store

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/config"
)

func writeConfig(t *testing.T, source string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	return path
}

func TestLoadReportsProblemsWithFileAndLine(t *testing.T) {
	tests := []struct {
		name   string
		source string
		want   string
	}{
		{
			name:   "unknown top-level key",
			source: "paths:\n  customers: c.yaml\nfoo: 1\n",
			want:   `F:3: unknown key "foo"`,
		},
		{
			name:   "unknown nested key",
			source: "numbering:\n  patern: x\n",
			want:   `F:2: unknown key "patern" in numbering`,
		},
		{
			name:   "list into a string",
			source: "email:\n  subject: [5]\n",
			want:   "F:2: email.subject: expected a string, got a list",
		},
		{
			name:   "mapping into a path",
			source: "archive:\n  dir: {a: 1}\n",
			want:   "F:2: archive.dir: expected a string, got a mapping",
		},
		{
			name:   "text start",
			source: "numbering:\n  start: x\n",
			want:   "F:2: numbering.start: expected an integer, got `x`",
		},
		{
			name:   "empty start",
			source: "numbering:\n  start: ''\n",
			want:   "F:2: numbering.start: expected an integer, got ``",
		},
		{
			name:   "fractional start",
			source: "numbering:\n  start: 1.9\n",
			want:   "F:2: numbering.start: expected an integer, got `1.9`",
		},
		{
			name:   "section not a mapping",
			source: "paths: 5\n",
			want:   "F:1: paths must be a mapping, got an integer",
		},
		{
			name:   "top level not a mapping",
			source: "- a\n- b\n",
			want:   "F: root value must be a mapping",
		},
		{
			name:   "flow mapping",
			source: "numbering: {pattern: x, start: 1.9}\n",
			want:   "F:1: numbering.start: expected an integer, got `1.9`",
		},
		{
			name:   "duplicate key",
			source: "paths:\n  issuer: i.yaml\npaths:\n  customers: c.yaml\n",
			want:   `F:3: duplicate key "paths" (first defined on line 1)`,
		},
		{
			name:   "recursive alias",
			source: "paths: &p\n  customers: *p\n",
			want:   "F:2: alias *p refers to a node that contains it",
		},
		{
			name:   "syntax error keeps the parser text",
			source: "archive:\n  dir: [unclosed\n",
			want:   "F: yaml: line 1: did not find expected ',' or ']'",
		},
		{
			name:   "indented top-level key",
			source: "# note\n  paths:\n    customers: c.yaml\n",
			want:   `F:2: top-level keys must not be indented; remove the leading whitespace before "paths"`,
		},
		{
			name:   "every problem in one pass, in file order",
			source: "paths:\n  customer: c.yaml\nnumbering:\n  start: x\nemail:\n  body: [x]\n",
			want: "F:2: unknown key \"customer\" in paths\n" +
				"F:4: numbering.start: expected an integer, got `x`\n" +
				"F:6: email.body: expected a string, got a list",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeConfig(t, tt.source)

			_, err := loadConfigFile(path)
			if err == nil {
				t.Fatalf("loadConfigFile returned nil error")
			}
			want := replaceF(tt.want, path)
			if err.Error() != want {
				t.Fatalf("loadConfigFile error =\n%s\nwant\n%s", err, want)
			}
			var cfgErr *billing.ConfigError
			if !errors.As(err, &cfgErr) {
				t.Fatalf("loadConfigFile error %#v is not a *billing.ConfigError", err)
			}
		})
	}
}

func replaceF(s, path string) string {
	out := ""
	for _, line := range splitLines(s) {
		if out != "" {
			out += "\n"
		}
		out += path + line[1:]
	}
	return out
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	return append(lines, s[start:])
}

func TestLoadDecodesSettings(t *testing.T) {
	template := filepath.Join(t.TempDir(), "t.tex")
	path := writeConfig(t, "# comment\npaths:\n  customers: ' sub/c.yaml '\n  issuer: ~/i.yaml\n  template: '"+template+"'\narchive:\n  dir: null\nnumbering:\n  pattern: ' {counter} '\n  start: \"7\"\nemail:\n  subject: 5\n")

	c, err := loadConfigFile(path)
	if err != nil {
		t.Fatalf("loadConfigFile returned error: %v", err)
	}
	home := filepath.Join(t.TempDir(), "ada")
	dir := filepath.Dir(path)
	for _, check := range []struct{ what, got, want string }{
		{"File", c.File, path},
		{"customers", c.Resolve(c.Paths.Customers, home), filepath.Join(dir, "sub", "c.yaml")},
		{"issuer", c.Resolve(c.Paths.Issuer, home), filepath.Join(home, "i.yaml")},
		{"template", c.Resolve(c.Paths.Template, home), template},
		{"defaults", c.Resolve(c.Paths.Defaults, home), ""},
		{"archive", c.Resolve(c.Archive.Dir, home), ""},
		{"pattern", string(c.Numbering.Pattern), "{counter}"},
		{"subject", string(c.Email.Subject), "5"},
		{"body", string(c.Email.Body), ""},
	} {
		if check.got != check.want {
			t.Errorf("%s = %q, want %q", check.what, check.got, check.want)
		}
	}
	if c.Numbering.Start == nil || *c.Numbering.Start != 7 {
		t.Errorf("Numbering.Start = %v, want 7", c.Numbering.Start)
	}
}

func TestLoadEmptyFileIsTheZeroConfig(t *testing.T) {
	for _, source := range []string{"", "# only a comment\n\n"} {
		path := writeConfig(t, source)
		c, err := loadConfigFile(path)
		if err != nil {
			t.Fatalf("loadConfigFile(%q) returned error: %v", source, err)
		}
		if *c != (config.Config{File: path}) {
			t.Fatalf("loadConfigFile(%q) = %+v, want the zero Config with File set", source, *c)
		}
	}
}

func TestLoadMissingFileIsNotExist(t *testing.T) {
	_, err := loadConfigFile(filepath.Join(t.TempDir(), "missing.yaml"))
	if !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("loadConfigFile error = %v, want fs.ErrNotExist", err)
	}
}
