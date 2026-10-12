package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/config"
)

// supportFiles is the one description of each support file, the files a
// command reads besides the invoice: the names the upward search and the
// config directory look for, and its paths.* setting. Its name in `config
// paths` is the billing.File's.
var supportFiles = [...]struct {
	localNames  []string
	globalNames []string
	configured  func(*config.Config) config.Path
}{
	billing.CustomersFile: {[]string{"customers.yaml"}, []string{"customers.yaml"}, func(c *config.Config) config.Path { return c.Paths.Customers }},
	billing.IssuerFile:    {[]string{"issuer.yaml"}, []string{"issuer.yaml"}, func(c *config.Config) config.Path { return c.Paths.Issuer }},
	billing.DefaultsFile:  {[]string{"invoice_defaults.yaml"}, []string{"invoice_defaults.yaml"}, func(c *config.Config) config.Path { return c.Paths.Defaults }},
	billing.TemplateFile:  {[]string{"invoice_template.tex", "template.tex"}, []string{"template.tex", "invoice_template.tex"}, func(c *config.Config) config.Path { return c.Paths.Template }},
}

// resolveSupportFile finds the support file kind for a command run in
// start: the upward project search wins, then paths.* in the config file,
// then the config directory. Path is "" when nothing is found.
func (h Host) resolveSupportFile(kind billing.File, start string) (resolved, error) {
	if kind < 0 || int(kind) >= len(supportFiles) {
		return resolved{}, fmt.Errorf("%s is not a support file", kind)
	}
	file := supportFiles[kind]
	for _, dir := range h.projectDirs(start) {
		for _, name := range file.localNames {
			if path := filepath.Join(dir, name); fileExists(path) {
				return resolved{Path: path, Source: billing.SourceProject}, nil
			}
		}
	}
	cfg, err := h.config()
	if err != nil {
		return resolved{}, err
	}
	if p := file.configured(cfg); p.IsSet() {
		return resolved{Path: cfg.Resolve(p, h.home), Source: billing.SourceConfig}, nil
	}
	return h.findInConfigDir(false, file.globalNames...)
}

func (h Host) ResolveArchiveDir() (string, error) {
	dir, err := h.archiveDir()
	return dir.Path, err
}

func (h Host) archiveDir() (resolved, error) {
	cfg, err := h.config()
	if err != nil {
		return resolved{}, err
	}
	if cfg.Archive.Dir.IsSet() {
		return resolved{Path: cfg.Resolve(cfg.Archive.Dir, h.home), Source: billing.SourceConfig}, nil
	}
	if dir := h.DefaultArchiveDir(); dir != "" {
		return resolved{Path: dir, Source: billing.SourceDefault}, nil
	}
	return resolved{}, nil
}

func (r resolved) report(name string) billing.PathReport {
	return billing.PathReport{Name: name, Path: r.Path, Source: r.Source}
}

// paths reports what each resolver returns for a command run in start, in a
// fixed order: config-dir, config, then the support files, then archive.
func (h Host) paths(start string) ([]billing.PathReport, error) {
	file, err := h.configFileResolved()
	if err != nil {
		return nil, err
	}
	if _, err := h.config(); err != nil {
		return nil, err
	}
	reports := []billing.PathReport{h.configDir.report("config-dir"), file.report("config")}
	for kind := range supportFiles {
		got, err := h.resolveSupportFile(billing.File(kind), start)
		if err != nil {
			return nil, err
		}
		reports = append(reports, got.report(billing.File(kind).String()))
	}
	archive, err := h.archiveDir()
	if err != nil {
		return nil, err
	}
	return append(reports, archive.report("archive")), nil
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
	homes := h.homeForms()
	underHome := false
	for _, home := range homes {
		underHome = underHome || h.isUnder(start, home)
	}
	for dir := start; ; {
		parent := filepath.Dir(dir)
		if parent == dir || (underHome && h.isHome(parent, homes)) {
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

// homeForms returns the home directory as given and, when it differs, with
// its symlinks resolved, because the working directory usually comes back in
// the resolved form. An unknown home gives none.
func (h Host) homeForms() []string {
	home := strings.TrimSpace(h.home)
	if home == "" {
		return nil
	}
	home = filepath.Clean(home)
	forms := []string{home}
	if real, err := filepath.EvalSymlinks(home); err == nil && !h.samePath(real, home) {
		forms = append(forms, real)
	}
	return forms
}

// isUnder reports whether path is strictly inside dir.
func (h Host) isUnder(path, dir string) bool {
	rel, err := filepath.Rel(h.fold(dir), h.fold(path))
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func (h Host) isHome(path string, homes []string) bool {
	for _, home := range homes {
		if h.samePath(path, home) {
			return true
		}
	}
	return false
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
