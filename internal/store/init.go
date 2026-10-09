package store

import (
	"embed"
	"errors"
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
	if err := fsutil.MkdirAll(configDir, fsutil.Private); err != nil {
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
