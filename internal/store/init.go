package store

import (
	"embed"
	"errors"
	"strings"
	"text/template"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/fsutil"
)

//go:embed starter
var starterFiles embed.FS

// configStarter is the starter config.yaml, which names the archive
// directory and the email placeholders.
var configStarter = template.Must(template.ParseFS(starterFiles, "starter/config.yaml.tmpl"))

// initConfigDir creates the config directory and the starter files it
// lacks, and returns the directory.
func (h Host) initConfigDir() (string, []billing.InitFile, error) {
	configDir := h.ConfigDir()
	if strings.TrimSpace(configDir) == "" {
		return "", nil, errors.New("config directory is unavailable")
	}
	if err := fsutil.MkdirAll(configDir, fsutil.Private); err != nil {
		return "", nil, err
	}

	results := make([]billing.InitFile, 0, 5)

	created, err := ensureStarterFile(h.GlobalConfigPath(), []byte(h.ConfigTemplate()), fsutil.Public)
	if err != nil {
		return "", nil, err
	}
	results = append(results, billing.InitFile{Path: h.GlobalConfigPath(), Created: created})

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
		results = append(results, billing.InitFile{Path: file.path, Created: created})
	}

	return configDir, results, nil
}
