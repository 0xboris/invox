// Package config reads config.yaml into a typed Config. It knows the schema
// and nothing about where the file lives.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	yaml "gopkg.in/yaml.v3"
)

// Config is config.yaml. The zero value means no config file: every setting
// unset.
type Config struct {
	// File is the absolute path the config was loaded from, "" for the zero
	// value. Relative paths in the file resolve against its directory.
	File      string    `yaml:"-"`
	Paths     Paths     `yaml:"paths"`
	Archive   Archive   `yaml:"archive"`
	Numbering Numbering `yaml:"numbering"`
	Email     Email     `yaml:"email"`
}

type Paths struct {
	Customers Path `yaml:"customers"`
	Issuer    Path `yaml:"issuer"`
	Defaults  Path `yaml:"defaults"`
	Template  Path `yaml:"template"`
}

type Archive struct {
	Dir Path `yaml:"dir"`
}

type Numbering struct {
	Pattern Text `yaml:"pattern"`
	// Start is nil when unset.
	Start *Int `yaml:"start"`
}

type Email struct {
	Subject Text `yaml:"subject"`
	Body    Text `yaml:"body"`
}

// Text is a setting written as a YAML string. yaml.v3 would otherwise decode
// 5 or true into a string field silently. Surrounding space is trimmed.
type Text string

func (t *Text) UnmarshalYAML(n *yaml.Node) error {
	s, err := scalar(n, "!!str", "a string")
	if err != nil {
		return err
	}
	*t = Text(strings.TrimSpace(s))
	return nil
}

// Int is a setting written as a YAML integer: "3" and 1.9 are refused.
type Int int64

func (i *Int) UnmarshalYAML(n *yaml.Node) error {
	if _, err := scalar(n, "!!int", "an integer"); err != nil {
		return err
	}
	var v int64
	if err := n.Decode(&v); err != nil {
		return typeError(n, "expected an integer, got %s", describe(n))
	}
	*i = Int(v)
	return nil
}

// Path is a file or directory setting. Its text is private so that no caller
// can use a relative value without resolving it against the config file.
type Path struct{ raw string }

func (p *Path) UnmarshalYAML(n *yaml.Node) error {
	s, err := scalar(n, "!!str", "a string")
	if err != nil {
		return err
	}
	p.raw = strings.TrimSpace(s)
	return nil
}

func (p Path) IsSet() bool { return p.raw != "" }

// Resolve returns p as an absolute path: "~" and "~/x" expand to home, and a
// relative path joins the directory of c.File. It returns "" when p is unset.
func (c *Config) Resolve(p Path, home string) string {
	if !p.IsSet() {
		return ""
	}
	path := ExpandHome(p.raw, home)
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(c.File), path)
	}
	return filepath.Clean(path)
}

// ExpandHome replaces a leading "~" or "~/" with home. It leaves path alone
// when home is unknown.
func ExpandHome(path, home string) string {
	if strings.TrimSpace(home) == "" {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}

func scalar(n *yaml.Node, tag, want string) (string, error) {
	if n.Kind != yaml.ScalarNode || n.ShortTag() != tag {
		return "", typeError(n, "expected %s, got %s", want, describe(n))
	}
	return n.Value, nil
}

// typeError returns a *yaml.TypeError, which makes yaml.v3 record the problem
// and keep decoding, so one Load reports every bad value.
func typeError(n *yaml.Node, format string, args ...any) error {
	return &yaml.TypeError{Errors: []string{fmt.Sprintf("line %d: %s", n.Line, fmt.Sprintf(format, args...))}}
}

func describe(n *yaml.Node) string {
	switch n.Kind {
	case yaml.MappingNode:
		return "a mapping"
	case yaml.SequenceNode:
		return "a list"
	}
	switch n.ShortTag() {
	case "!!str":
		return fmt.Sprintf("string %q", n.Value)
	case "!!int":
		return "integer " + n.Value
	case "!!float":
		return "number " + n.Value
	case "!!bool":
		return "boolean " + n.Value
	}
	return n.Value
}

// Load reads and strictly decodes the config file at path. An empty file
// gives the zero Config with File set. A read failure is returned as is, so
// errors.Is(err, fs.ErrNotExist) tells a missing file apart. Problems in the
// file come back as *Error values, several joined with errors.Join in file
// order.
func Load(path string) (*Config, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if line, text := indentedTopLevelKey(src); line > 0 {
		return nil, &Error{File: path, Line: line, Err: fmt.Errorf("top-level keys must not be indented; remove the leading whitespace before %q", text)}
	}

	c := &Config{}
	dec := yaml.NewDecoder(bytes.NewReader(src))
	dec.KnownFields(true)
	err = dec.Decode(c)
	var typeErr *yaml.TypeError
	switch {
	case errors.Is(err, io.EOF), err == nil:
	case errors.As(err, &typeErr):
		keys := keysByLine(src)
		problems := make([]error, 0, len(typeErr.Errors))
		for _, msg := range typeErr.Errors {
			problems = append(problems, decodeProblem(path, msg, keys))
		}
		return nil, errors.Join(problems...)
	default:
		return nil, &Error{File: path, Err: err}
	}
	c.File = path
	return c, nil
}

func indentedTopLevelKey(src []byte) (int, string) {
	for i, line := range strings.Split(string(src), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			return i + 1, trimmed
		}
		return 0, ""
	}
	return 0, ""
}

var (
	problemLine  = regexp.MustCompile(`^line (\d+): (.*)$`)
	unknownField = regexp.MustCompile(`^field (.+) not found in type config\.(\w+)$`)
	notAMapping  = regexp.MustCompile("^cannot unmarshal (!!\\w+)(?: `.*`)? into config\\.(\\w+)$")
)

// sections names the YAML key of each schema struct, for messages that must
// not show Go type names.
var sections = map[string]string{
	"Config":    "",
	"Paths":     "paths",
	"Archive":   "archive",
	"Numbering": "numbering",
	"Email":     "email",
}

// keysByLine maps the line of each setting's value to its dotted key, so a
// wrong-type message can name the key. A line that holds more than one value,
// as in a flow mapping, maps to no key.
func keysByLine(src []byte) map[int]string {
	var doc yaml.Node
	if yaml.Unmarshal(src, &doc) != nil || len(doc.Content) == 0 {
		return nil
	}
	keys := map[int]string{}
	ambiguous := map[int]bool{}
	var walk func(n *yaml.Node, prefix string)
	walk = func(n *yaml.Node, prefix string) {
		if n.Kind != yaml.MappingNode {
			return
		}
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, value := prefix+n.Content[i].Value, n.Content[i+1]
			if value.Kind == yaml.MappingNode {
				walk(value, key+".")
				continue
			}
			if _, seen := keys[value.Line]; seen {
				ambiguous[value.Line] = true
			}
			keys[value.Line] = key
		}
	}
	walk(doc.Content[0], "")
	for line := range ambiguous {
		delete(keys, line)
	}
	return keys
}

func decodeProblem(path, msg string, keys map[int]string) error {
	m := problemLine.FindStringSubmatch(msg)
	if m == nil {
		return &Error{File: path, Err: errors.New(msg)}
	}
	// problemLine captures only digits, so Atoi cannot fail.
	line, _ := strconv.Atoi(m[1])
	text := m[2]
	if u := unknownField.FindStringSubmatch(text); u != nil {
		text = fmt.Sprintf("unknown key %q", u[1])
		if section := sections[u[2]]; section != "" {
			text += " in " + section
		}
	} else if u := notAMapping.FindStringSubmatch(text); u != nil {
		if section, ok := sections[u[2]]; ok {
			if section == "" {
				section = "the top level"
			}
			text = fmt.Sprintf("%s must be a mapping, got %s", section, tagName(u[1]))
		}
	} else if key := keys[line]; key != "" && strings.HasPrefix(text, "expected ") {
		text = key + ": " + text
	}
	return &Error{File: path, Line: line, Err: errors.New(text)}
}

func tagName(tag string) string {
	switch tag {
	case "!!str":
		return "a string"
	case "!!int":
		return "an integer"
	case "!!float":
		return "a number"
	case "!!bool":
		return "a boolean"
	case "!!seq":
		return "a list"
	}
	return tag
}

// Error is one problem in one config file. Line is 0 when the problem has no
// line of its own.
type Error struct {
	File string
	Line int
	Err  error
}

func (e *Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%s:%d: %v", e.File, e.Line, e.Err)
	}
	return fmt.Sprintf("%s: %v", e.File, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }
