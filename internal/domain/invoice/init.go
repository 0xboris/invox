package invoice

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/domain/ports"
)

type InitFileResult struct {
	Path    string
	Created bool
}

func (h Host) InitializeConfigDir() (string, []InitFileResult, error) {
	files := h.files
	configDir := h.ConfigDir()
	if strings.TrimSpace(configDir) == "" {
		return "", nil, errors.New("config directory is unavailable")
	}
	if err := files.MkdirAll(configDir, ports.Private); err != nil {
		return "", nil, err
	}

	results := make([]InitFileResult, 0, 5)

	created, err := ensureStarterFile(files, h.GlobalConfigPath(), []byte(h.defaultConfigTemplate()), ports.Public)
	if err != nil {
		return "", nil, err
	}
	results = append(results, InitFileResult{Path: h.GlobalConfigPath(), Created: created})

	for _, file := range []struct {
		path string
		name string
		perm ports.Perm
	}{
		{path: h.GlobalCustomersPath(), name: "customers.yaml", perm: ports.Private},
		{path: h.GlobalIssuerPath(), name: "issuer.yaml", perm: ports.Private},
		{path: h.GlobalInvoiceDefaultsPath(), name: "invoice_defaults.yaml", perm: ports.Public},
		{path: h.GlobalTemplatePath(), name: "template.tex", perm: ports.Public},
	} {
		content, err := fs.ReadFile(h.starter, file.name)
		if err != nil {
			return "", nil, err
		}
		created, err := ensureStarterFile(files, file.path, content, file.perm)
		if err != nil {
			return "", nil, err
		}
		results = append(results, InitFileResult{Path: file.path, Created: created})
	}

	return configDir, results, nil
}

// ensureStarterFile writes content to path if path is missing or empty and
// reports whether it did. A dangling symlink at path counts as missing, and
// its target is written. A file created by someone else in the meantime is
// left alone.
func ensureStarterFile(files ports.Files, path string, content []byte, perm ports.Perm) (bool, error) {
	info, err := files.Stat(path)
	switch {
	case err == nil && info.Size() > 0:
		return false, nil
	case err == nil:
		if err := files.WriteFile(path, content, perm); err != nil {
			return false, err
		}
		return true, nil
	case !errors.Is(err, fs.ErrNotExist):
		return false, err
	}
	if info, err := files.Lstat(path); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		if err := files.WriteFile(path, content, perm); err != nil {
			return false, err
		}
		return true, nil
	}

	err = files.WriteNewFile(path, content, perm)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// LegacyFilesToCopy returns the files in the legacy config directory, relative
// to it, that the config directory does not have yet.
func (h Host) LegacyFilesToCopy() ([]string, error) {
	files := h.files
	legacyDir := h.LegacyConfigDir()
	if legacyDir == "" {
		return nil, nil
	}
	var missing []string
	err := files.WalkDir(legacyDir, func(path string, entry fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) && path == legacyDir {
			return filepath.SkipDir
		}
		if err != nil || entry.IsDir() {
			return err
		}
		if info, err := files.Stat(path); err != nil || !info.Mode().IsRegular() {
			return nil //nolint:nilerr // only regular files are copied; skip a link that does not resolve or cannot be read
		}
		rel, err := filepath.Rel(legacyDir, path)
		if err != nil {
			return err
		}
		if _, err := files.Lstat(filepath.Join(h.ConfigDir(), rel)); errors.Is(err, fs.ErrNotExist) {
			missing = append(missing, rel)
		}
		return nil
	})
	return missing, err
}

// CopyLegacyFiles copies the files LegacyFilesToCopy reports into the config
// directory and returns them. It never replaces a file, and it leaves the
// legacy directory as it is, so running it again copies only what is still
// missing.
func (h Host) CopyLegacyFiles() ([]string, error) {
	files := h.files
	missing, err := h.LegacyFilesToCopy()
	if err != nil {
		return nil, err
	}
	if len(missing) == 0 {
		return nil, nil
	}
	if err := files.MkdirAll(h.ConfigDir(), ports.Private); err != nil {
		return nil, err
	}
	copied := make([]string, 0, len(missing))
	for _, rel := range missing {
		content, err := files.ReadFile(filepath.Join(h.LegacyConfigDir(), rel))
		if err != nil {
			return copied, err
		}
		err = files.WriteNewFile(filepath.Join(h.ConfigDir(), rel), content, legacyFilePerm(rel))
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return copied, err
		}
		copied = append(copied, rel)
	}
	return copied, nil
}

// legacyFilePerm gives customers.yaml and issuer.yaml the same private mode
// that init gives their starter files.
func legacyFilePerm(rel string) ports.Perm {
	switch rel {
	case "customers.yaml", "issuer.yaml":
		return ports.Private
	}
	return ports.Public
}
