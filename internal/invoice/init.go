package invoice

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

//go:embed starter/customers.yaml starter/issuer.yaml starter/invoice_defaults.yaml starter/template.tex
var starterFiles embed.FS

type InitFileResult struct {
	Path    string
	Created bool
}

func (h Host) InitializeConfigDir() (string, []InitFileResult, error) {
	configDir := h.ConfigDir()
	if strings.TrimSpace(configDir) == "" {
		return "", nil, errors.New("config directory is unavailable")
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", nil, err
	}

	results := make([]InitFileResult, 0, 5)

	created, err := ensureStarterFile(h.GlobalConfigPath(), []byte(h.defaultConfigTemplate()))
	if err != nil {
		return "", nil, err
	}
	results = append(results, InitFileResult{Path: h.GlobalConfigPath(), Created: created})

	for _, file := range []struct {
		path string
		name string
	}{
		{path: h.GlobalCustomersPath(), name: "starter/customers.yaml"},
		{path: h.GlobalIssuerPath(), name: "starter/issuer.yaml"},
		{path: h.GlobalInvoiceDefaultsPath(), name: "starter/invoice_defaults.yaml"},
		{path: h.GlobalTemplatePath(), name: "starter/template.tex"},
	} {
		content, err := starterFiles.ReadFile(file.name)
		if err != nil {
			return "", nil, err
		}
		created, err := ensureStarterFile(file.path, content)
		if err != nil {
			return "", nil, err
		}
		results = append(results, InitFileResult{Path: file.path, Created: created})
	}

	return configDir, results, nil
}

func ensureStarterFile(path string, content []byte) (bool, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil && info.Size() > 0:
		return false, nil
	case err != nil && !errors.Is(err, os.ErrNotExist):
		return false, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

// LegacyFilesToCopy returns the files in the legacy config directory, relative
// to it, that the config directory does not have yet.
func (h Host) LegacyFilesToCopy() ([]string, error) {
	legacyDir := h.LegacyConfigDir()
	if legacyDir == "" {
		return nil, nil
	}
	var missing []string
	err := filepath.WalkDir(legacyDir, func(path string, entry fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) && path == legacyDir {
			return filepath.SkipDir
		}
		if err != nil || !entry.Type().IsRegular() {
			return err
		}
		rel, err := filepath.Rel(legacyDir, path)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(filepath.Join(h.ConfigDir(), rel)); errors.Is(err, fs.ErrNotExist) {
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
	missing, err := h.LegacyFilesToCopy()
	if err != nil {
		return nil, err
	}
	copied := make([]string, 0, len(missing))
	for _, rel := range missing {
		err := copyNewFile(filepath.Join(h.LegacyConfigDir(), rel), filepath.Join(h.ConfigDir(), rel))
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

// copyNewFile copies source to a dest that must not exist yet.
func copyNewFile(source, dest string) error {
	content, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	if _, err := file.Write(content); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}
