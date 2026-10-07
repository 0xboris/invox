package invoice

import (
	"embed"
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/0xboris/invox/internal/fsutil"
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
	if err := fsutil.MkdirAll(configDir, fsutil.Private.Dir); err != nil {
		return "", nil, err
	}

	results := make([]InitFileResult, 0, 5)

	created, err := ensureStarterFile(h.GlobalConfigPath(), []byte(h.defaultConfigTemplate()), fsutil.Public)
	if err != nil {
		return "", nil, err
	}
	results = append(results, InitFileResult{Path: h.GlobalConfigPath(), Created: created})

	for _, file := range []struct {
		path string
		name string
		perm fsutil.Perm
	}{
		{path: h.GlobalCustomersPath(), name: "starter/customers.yaml", perm: fsutil.Private},
		{path: h.GlobalIssuerPath(), name: "starter/issuer.yaml", perm: fsutil.Private},
		{path: h.GlobalInvoiceDefaultsPath(), name: "starter/invoice_defaults.yaml", perm: fsutil.Public},
		{path: h.GlobalTemplatePath(), name: "starter/template.tex", perm: fsutil.Public},
	} {
		content, err := starterFiles.ReadFile(file.name)
		if err != nil {
			return "", nil, err
		}
		created, err := ensureStarterFile(file.path, content, file.perm)
		if err != nil {
			return "", nil, err
		}
		results = append(results, InitFileResult{Path: file.path, Created: created})
	}

	return configDir, results, nil
}

// ensureStarterFile writes content to path if path is missing or empty and
// reports whether it did. A file created by someone else in the meantime is
// left alone.
func ensureStarterFile(path string, content []byte, perm fsutil.Perm) (bool, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil && info.Size() > 0:
		return false, nil
	case err == nil:
		if err := fsutil.WriteFile(path, content, perm); err != nil {
			return false, err
		}
		return true, nil
	case !errors.Is(err, os.ErrNotExist):
		return false, err
	}

	err = fsutil.WriteNewFile(path, content, perm)
	if errors.Is(err, fs.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
