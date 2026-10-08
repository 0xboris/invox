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
// config and the record of legacy files used.
type Host struct {
	goos       string
	home       string
	configBase string
	dataBase   string
	configDir  Resolved
	configFile string
	legacy     *legacyUse
	config     func() (*config.Config, error)
}

func NewHost(in HostInputs) Host {
	h := Host{goos: in.GOOS, home: in.Home, configFile: in.ConfigFile, legacy: &legacyUse{}}
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
	c, err := config.Load(file.Path)
	if file.Source == SourceExplicit && errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("config file %s does not exist", file.Path)
	}
	var configErr *config.Error
	if errors.As(err, &configErr) {
		return nil, &billing.ConfigError{Err: err}
	}
	return c, err
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

// LegacyConfigDir returns the deprecated invoice-tool directory, or "" when
// it is not read: the home directory is unknown or the config directory was
// chosen explicitly.
func (h Host) LegacyConfigDir() string {
	if h.configDir.Source != SourceDefault {
		return ""
	}
	return filepath.Join(h.configBase, legacyConfigDirName)
}

// findInConfigDir returns the first of names, as a file or as a directory,
// in the config directory. Each name missing there is looked up in the
// legacy directory next, and a hit is recorded for LegacyFilesUsed. An
// explicitly chosen config directory that does not exist is an error.
func (h Host) findInConfigDir(isDir bool, names ...string) (Resolved, error) {
	dir := h.configDir
	if dir.Source == SourceEnvDir {
		if info, err := os.Stat(dir.Path); err != nil || !info.IsDir() {
			return Resolved{}, &billing.ConfigDirNotFoundError{Dir: dir.Path}
		}
	}
	dirs := []Resolved{dir}
	if legacy := h.LegacyConfigDir(); legacy != "" {
		dirs = append(dirs, Resolved{Path: legacy, Source: SourceLegacy})
	}
	for _, d := range dirs {
		if d.Path == "" {
			continue
		}
		for _, name := range names {
			candidate := filepath.Join(d.Path, name)
			if !pathExists(candidate, isDir) {
				continue
			}
			if d.Source == SourceLegacy {
				h.legacy.record(candidate)
			}
			return Resolved{Path: candidate, Source: d.Source}, nil
		}
	}
	return Resolved{}, nil
}

// LegacyFilesUsed returns the files read from the legacy directory so far,
// in the order first used.
func (h Host) LegacyFilesUsed() []string {
	return h.legacy.files()
}

type legacyUse struct {
	mu    sync.Mutex
	paths []string
}

func (l *legacyUse) record(path string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, p := range l.paths {
		if p == path {
			return
		}
	}
	l.paths = append(l.paths, path)
}

func (l *legacyUse) files() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.paths...)
}

// Source says where a resolved path came from.
type Source = billing.Source

const (
	SourceNone     = billing.SourceNone
	SourceExplicit = billing.SourceExplicit
	SourceEnvDir   = billing.SourceEnvDir
	SourceDefault  = billing.SourceDefault
	SourceLegacy   = billing.SourceLegacy
	SourceProject  = billing.SourceProject
	SourceConfig   = billing.SourceConfig
)

type Resolved struct {
	Path   string // "" only with SourceNone
	Source Source
}

const (
	configDirName       = "invox"
	legacyConfigDirName = "invoice-tool"
)
