package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/config"
	yaml "gopkg.in/yaml.v3"
)

// HostInputs are the parts of the environment the user directories come
// from. Home is "" when the home directory is unknown. ConfigDir and
// ConfigFile are absolute or "": a config directory and a config file chosen
// by the user in place of the defaults.
type HostInputs struct {
	GOOS          string
	Home          string
	XDGConfigHome string
	XDGDataHome   string
	AppData       string
	ConfigDir     string
	ConfigFile    string
}

// Host holds the user directories invox reads and writes, resolved once, and
// the config file, read at most once. Copies of a Host share the loaded
// config.
type Host struct {
	goos       string
	home       string
	configBase string
	dataBase   string
	configDir  Resolved
	configFile string
	config     func() (*config.Config, error)
}

func NewHost(in HostInputs) Host {
	h := Host{goos: in.GOOS, home: in.Home, configFile: in.ConfigFile}
	homeKnown := strings.TrimSpace(in.Home) != ""

	if xdg := strings.TrimSpace(in.XDGConfigHome); xdg != "" {
		h.configBase = xdg
	} else if homeKnown {
		h.configBase = filepath.Join(in.Home, ".config")
	}

	switch in.GOOS {
	case "windows":
		if appData := strings.TrimSpace(in.AppData); appData != "" {
			h.dataBase = appData
		} else if homeKnown {
			h.dataBase = filepath.Join(in.Home, "AppData", "Roaming")
		}
	default:
		if xdg := strings.TrimSpace(in.XDGDataHome); xdg != "" {
			h.dataBase = xdg
		} else if in.GOOS == "darwin" && homeKnown {
			h.dataBase = filepath.Join(in.Home, "Library", "Application Support")
		} else if homeKnown {
			h.dataBase = filepath.Join(in.Home, ".local", "share")
		}
	}

	switch {
	case in.ConfigDir != "":
		h.configDir = Resolved{Path: in.ConfigDir, Source: SourceEnvDir}
	case h.configBase != "":
		h.configDir = Resolved{Path: filepath.Join(h.configBase, configDirName), Source: SourceDefault}
	}

	loader := h
	h.config = sync.OnceValues(loader.loadConfig)
	return h
}

// Config returns the parsed config file, reading it on the first call only.
// No config file gives the zero Config. It never returns nil with a nil
// error.
func (h Host) Config() (*config.Config, error) {
	return h.config()
}

func (h Host) loadConfig() (*config.Config, error) {
	file, err := h.configFileResolved()
	if err != nil {
		return nil, err
	}
	if file.Path == "" {
		return &config.Config{}, nil
	}
	c, err := loadConfigFile(file.Path)
	if file.Source == SourceExplicit && errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("config file %s does not exist", file.Path)
	}
	return c, err
}

// loadConfigFile reads and strictly decodes the config file at path. A file
// without a document, such as one with only comments, gives the zero Config
// with File set. A read failure is returned as is, so errors.Is(err,
// fs.ErrNotExist) tells a missing file apart; a problem in the file comes
// back as a *billing.ConfigError.
func loadConfigFile(path string) (*config.Config, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	c := &config.Config{File: path}
	if err := decodeConfig(source, path, c); err != nil {
		return nil, &billing.ConfigError{Err: err}
	}
	return c, nil
}

// decodeConfig decodes config.yaml like the support files, and also
// refuses an indented first key: uncommenting a line of the starter file
// leaves one behind.
func decodeConfig(source []byte, path string, c *config.Config) error {
	document, err := parseYAMLDocumentSource(source, path)
	if err != nil || len(document.Content) == 0 {
		return err
	}
	if root := document.Content[0]; root.Kind == yaml.MappingNode && root.Column > 1 && len(root.Content) > 0 {
		key := root.Content[0]
		return &billing.DecodeError{File: path, Line: key.Line, Problem: fmt.Sprintf("top-level keys must not be indented; remove the leading whitespace before %q", key.Value)}
	}
	return decodeYAMLDocument(document, path, c, true)
}

// configFileResolved returns the config file to read: the explicit one, or
// config.yaml from the config directories. Path is "" when there is none.
func (h Host) configFileResolved() (Resolved, error) {
	if h.configFile != "" {
		return Resolved{Path: h.configFile, Source: SourceExplicit}, nil
	}
	return h.findInConfigDir(false, "config.yaml")
}

func (h Host) ConfigDir() string {
	return h.configDir.Path
}

// findInConfigDir returns the first of names, as a file or as a directory,
// in the config directory. An explicitly chosen config directory that does
// not exist is an error.
func (h Host) findInConfigDir(isDir bool, names ...string) (Resolved, error) {
	dir := h.configDir
	if dir.Source == SourceEnvDir {
		if info, err := os.Stat(dir.Path); err != nil || !info.IsDir() {
			return Resolved{}, &billing.ConfigDirNotFoundError{Dir: dir.Path}
		}
	}
	if dir.Path == "" {
		return Resolved{}, nil
	}
	for _, name := range names {
		candidate := filepath.Join(dir.Path, name)
		if pathExists(candidate, isDir) {
			return Resolved{Path: candidate, Source: dir.Source}, nil
		}
	}
	return Resolved{}, nil
}

// Source says where a resolved path came from.
type Source = billing.Source

const (
	SourceNone     = billing.SourceNone
	SourceExplicit = billing.SourceExplicit
	SourceEnvDir   = billing.SourceEnvDir
	SourceDefault  = billing.SourceDefault
	SourceProject  = billing.SourceProject
	SourceConfig   = billing.SourceConfig
)

type Resolved struct {
	Path   string // "" only with SourceNone
	Source Source
}

const (
	configDirName = "invox"
)
