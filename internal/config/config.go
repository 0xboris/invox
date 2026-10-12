// Package config holds the settings of config.yaml as typed values. It
// knows neither where the file lives nor how it is decoded; store does both.
package config

import (
	"path/filepath"
	"strings"
)

// Config is config.yaml. The zero value means no config file: every setting
// unset.
type Config struct {
	// File is the absolute path the config was loaded from, "" for the zero
	// value. Relative paths in the file resolve against its directory.
	File      string
	Paths     Paths
	Archive   Archive
	Numbering Numbering
	Email     Email
}

type Paths struct {
	Customers Path
	Issuer    Path
	Defaults  Path
	Template  Path
}

type Archive struct {
	Dir Path
}

type Numbering struct {
	Pattern Text
	// Start is nil when unset.
	Start *Int
}

type Email struct {
	Subject Text
	Body    Text
}

// Text is a setting written as text, without surrounding space.
type Text string

// Int is a setting written as an integer.
type Int int64

// Path is a file or directory setting. Its text is private so that no caller
// can use a relative value without resolving it against the config file.
type Path struct{ raw string }

// NewPath returns the setting written as text, without surrounding space.
func NewPath(text string) Path { return Path{raw: strings.TrimSpace(text)} }

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
