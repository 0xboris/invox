package invoice

import (
	"embed"
	"errors"
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

func InitializeConfigDir() (string, []InitFileResult, error) {
	configDir := ConfigDir()
	if strings.TrimSpace(configDir) == "" {
		return "", nil, errors.New("config directory is unavailable")
	}
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return "", nil, err
	}

	results := make([]InitFileResult, 0, 5)

	created, err := ensureStarterFile(GlobalConfigPath(), []byte(defaultConfigTemplate()))
	if err != nil {
		return "", nil, err
	}
	results = append(results, InitFileResult{Path: GlobalConfigPath(), Created: created})

	for _, file := range []struct {
		path string
		name string
	}{
		{path: GlobalCustomersPath(), name: "starter/customers.yaml"},
		{path: GlobalIssuerPath(), name: "starter/issuer.yaml"},
		{path: GlobalInvoiceDefaultsPath(), name: "starter/invoice_defaults.yaml"},
		{path: GlobalTemplatePath(), name: "starter/template.tex"},
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
