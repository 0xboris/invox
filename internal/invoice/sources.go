package invoice

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/config"
)

// Source says where a resolved path came from.
type Source int

const (
	SourceNone     Source = iota // nothing found
	SourceExplicit               // the config file the user named
	SourceEnvDir                 // the config directory the user chose, or a file in it
	SourceDefault                // the OS default directory, or a file in it
	SourceLegacy                 // the deprecated invoice-tool directory, or a file in it
	SourceProject                // found by the upward search from the working directory
	SourceConfig                 // a paths.* or archive.dir setting in the config file
)

type Resolved struct {
	Path   string // "" only with SourceNone
	Source Source
}

// SupportFile is one of the files a command reads besides the invoice.
type SupportFile int

const (
	Customers SupportFile = iota
	Issuer
	Defaults
	Template
)

// supportFiles is the one description of each support file: its name in
// `config paths`, the names the upward search and the config directory look
// for, and its paths.* setting.
var supportFiles = [...]struct {
	name        string
	localNames  []string
	globalNames []string
	configured  func(*config.Config) config.Path
}{
	Customers: {"customers", []string{"customers.yaml"}, []string{"customers.yaml"}, func(c *config.Config) config.Path { return c.Paths.Customers }},
	Issuer:    {"issuer", []string{"issuer.yaml"}, []string{"issuer.yaml"}, func(c *config.Config) config.Path { return c.Paths.Issuer }},
	Defaults:  {"defaults", []string{"invoice_defaults.yaml"}, []string{"invoice_defaults.yaml"}, func(c *config.Config) config.Path { return c.Paths.Defaults }},
	Template:  {"template", []string{"invoice_template.tex", "template.tex"}, []string{"template.tex", "invoice_template.tex"}, func(c *config.Config) config.Path { return c.Paths.Template }},
}

// ResolveSupportFile finds kind for a command run in start: the upward
// project search wins, then paths.* in the config file, then the config
// directory. Path is "" when nothing is found.
func (h Host) ResolveSupportFile(kind SupportFile, start string) (Resolved, error) {
	file := supportFiles[kind]
	for _, dir := range h.projectDirs(start) {
		for _, name := range file.localNames {
			if path := filepath.Join(dir, name); fileExists(path) {
				return Resolved{Path: path, Source: SourceProject}, nil
			}
		}
	}
	cfg, err := h.Config()
	if err != nil {
		return Resolved{}, err
	}
	if p := file.configured(cfg); p.IsSet() {
		return Resolved{Path: cfg.Resolve(p, h.home), Source: SourceConfig}, nil
	}
	return h.findInConfigDir(false, file.globalNames...)
}

func (h Host) ResolveArchiveDir() (string, error) {
	dir, err := h.archiveDir()
	return dir.Path, err
}

func (h Host) archiveDir() (Resolved, error) {
	cfg, err := h.Config()
	if err != nil {
		return Resolved{}, err
	}
	if cfg.Archive.Dir.IsSet() {
		return Resolved{Path: cfg.Resolve(cfg.Archive.Dir, h.home), Source: SourceConfig}, nil
	}
	if dir := h.DefaultArchiveDir(); dir != "" {
		return Resolved{Path: dir, Source: SourceDefault}, nil
	}
	return Resolved{}, nil
}

type PathReport struct {
	Name string
	Resolved
}

// Paths reports what each resolver returns for a command run in start, in a
// fixed order: config-dir, config, then the support files, then archive.
func (h Host) Paths(start string) ([]PathReport, error) {
	file, err := h.configFileResolved()
	if err != nil {
		return nil, err
	}
	if _, err := h.Config(); err != nil {
		return nil, err
	}
	reports := []PathReport{
		{Name: "config-dir", Resolved: h.configDir},
		{Name: "config", Resolved: file},
	}
	for kind := range supportFiles {
		got, err := h.ResolveSupportFile(SupportFile(kind), start)
		if err != nil {
			return nil, err
		}
		reports = append(reports, PathReport{Name: supportFiles[kind].name, Resolved: got})
	}
	archive, err := h.archiveDir()
	if err != nil {
		return nil, err
	}
	return append(reports, PathReport{Name: "archive", Resolved: archive}), nil
}

// projectMarkers end the upward search in the directory that holds one.
var projectMarkers = []string{".git", "invox.yaml", "invoice_defaults.yaml"}

// projectDirs lists the directories the upward search visits from start,
// nearest first. start is always searched. The search goes up to and
// including the nearest directory with a project marker, but never reaches
// the home directory unless start is home. Without a marker it covers the
// directories below home, or only start when start is not under home.
func (h Host) projectDirs(start string) []string {
	start = filepath.Clean(start)
	dirs := []string{start}
	if hasProjectMarker(start) {
		return dirs
	}
	underHome := h.isUnderHome(start)
	for dir := start; ; {
		parent := filepath.Dir(dir)
		if parent == dir || (underHome && h.samePath(parent, h.home)) {
			break
		}
		dir = parent
		dirs = append(dirs, dir)
		if hasProjectMarker(dir) {
			return dirs
		}
	}
	if underHome {
		return dirs
	}
	return dirs[:1]
}

func hasProjectMarker(dir string) bool {
	for _, marker := range projectMarkers {
		if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
			return true
		}
	}
	return false
}

// isUnderHome reports whether path is strictly inside the home directory.
// An unknown home counts as outside.
func (h Host) isUnderHome(path string) bool {
	home := strings.TrimSpace(h.home)
	if home == "" {
		return false
	}
	rel, err := filepath.Rel(h.fold(filepath.Clean(home)), h.fold(path))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (h Host) samePath(a, b string) bool {
	return h.fold(filepath.Clean(a)) == h.fold(filepath.Clean(b))
}

// fold lowercases path where the file system usually ignores case.
func (h Host) fold(path string) string {
	if h.goos == "windows" || h.goos == "darwin" {
		return strings.ToLower(path)
	}
	return path
}
