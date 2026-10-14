package invoice

import (
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/0xboris/invox/internal/domain/ports"
	"github.com/0xboris/invox/internal/utilities/env"
)

// HostSource resolves the Host from the environment once, on first use, so
// that the config file chosen on the command line counts.
type HostSource struct {
	env     env.Env
	files   ports.Files
	starter fs.FS

	mu         sync.Mutex
	configFile string
	host       *Host
}

// NewHostSource returns a HostSource that reads the user directories from e
// and gives the Host files and starter.
func NewHostSource(e env.Env, files ports.Files, starter fs.FS) *HostSource {
	return &HostSource{env: e, files: files, starter: starter}
}

// SetConfigFile sets the config file as typed, "" for the default. It must
// be called before the first Host call to count.
func (s *HostSource) SetConfigFile(path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.configFile = path
}

// ConfigFile returns the config file as typed, "" when none was set.
func (s *HostSource) ConfigFile() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.configFile
}

// Host returns the user directories, resolved on the first call.
func (s *HostSource) Host() Host {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.host == nil {
		h := s.resolve()
		s.host = &h
	}
	return *s.host
}

// resolve resolves the user directories. Relative directory values are
// ignored, as the XDG spec asks, so every directory the Host holds is
// absolute. configFile and INVOX_CONFIG_DIR are made absolute against the
// working directory.
func (s *HostSource) resolve() Host {
	e := s.env
	home, err := e.HomeDir()
	if err != nil {
		home = ""
	}
	cwd, err := e.Getwd()
	if err != nil {
		cwd = ""
	}
	return NewHost(HostInputs{
		GOOS:          e.GOOS,
		Home:          absOnly(home),
		XDGConfigHome: absOnly(e.Getenv("XDG_CONFIG_HOME")),
		XDGDataHome:   absOnly(e.Getenv("XDG_DATA_HOME")),
		AppData:       absOnly(e.Getenv("APPDATA")),
		ConfigDir:     absAgainst(cwd, e.Getenv("INVOX_CONFIG_DIR")),
		ConfigFile:    absAgainst(cwd, s.configFile),
		Files:         s.files,
		Starter:       s.starter,
	})
}

func absOnly(path string) string {
	path = strings.TrimSpace(path)
	if !filepath.IsAbs(path) {
		return ""
	}
	return filepath.Clean(path)
}

func absAgainst(cwd, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}
