package invoice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/config"
	"github.com/0xboris/invox/internal/fsutil"
)

const (
	configDirName       = "invox"
	legacyConfigDirName = "invoice-tool"
)

func (h Host) GlobalCustomersPath() string {
	return filepath.Join(h.ConfigDir(), "customers.yaml")
}

func (h Host) GlobalIssuerPath() string {
	return filepath.Join(h.ConfigDir(), "issuer.yaml")
}

func (h Host) GlobalTemplatePath() string {
	return filepath.Join(h.ConfigDir(), "template.tex")
}

func (h Host) GlobalConfigPath() string {
	return filepath.Join(h.ConfigDir(), "config.yaml")
}

// EditableConfigPath returns the config file to open in an editor, creating
// it from the template when it does not exist: the explicit config file, else
// config.yaml where Config reads it, else config.yaml in the config
// directory.
func (h Host) EditableConfigPath() (string, error) {
	path := h.configFile
	if path == "" {
		found, err := h.findInConfigDir(false, "config.yaml")
		var missingDir *ConfigDirNotFoundError
		if err != nil && !errors.As(err, &missingDir) {
			return "", err
		}
		path = found.Path
	}
	if path == "" {
		if h.ConfigDir() == "" {
			return "", errors.New("config directory is unavailable")
		}
		path = h.GlobalConfigPath()
	}
	if err := fsutil.MkdirAll(filepath.Dir(path), fsutil.Private); err != nil {
		return "", err
	}
	if err := h.ensureConfigTemplate(path); err != nil {
		return "", err
	}
	return path, nil
}

func (h Host) ensureConfigTemplate(path string) error {
	_, err := ensureStarterFile(path, []byte(h.defaultConfigTemplate()), fsutil.Public)
	return err
}

func (h Host) defaultConfigTemplate() string {
	defaultArchiveDir := h.configTemplatePath(h.DefaultArchiveDir())
	if strings.TrimSpace(defaultArchiveDir) == "" {
		defaultArchiveDir = "invoices"
	}

	return strings.TrimLeft(fmt.Sprintf(`
# Invox user configuration.
#
# Uncomment a setting and change it to override the default.
#
# Supported settings:
#   paths.customers
#   paths.issuer
#   paths.defaults
#   paths.template
#   numbering.pattern
#   numbering.start
#   archive.dir
#     Directory where archived invoice files are stored.
#   email.subject
#     Subject template for the email command.
#   email.body
#     Plain-text body template for the email command.
#     Supported placeholders for email.subject and email.body:
#       {customer_name}
#       {email_greeting}
#       {contact_person}
#       {customer_id}
#       {invoice_number}
#       {issue_date}
#       {due_date}
#       {total_amount}
#       {outstanding_amount}
#       {payment_terms_text}
#       {issuer_name}
#
# Notes:
# - Top-level keys must not be indented.
# - Relative paths are resolved relative to this file.
# - "~/" expands to your home directory.
# - Per-customer numbering overrides live in customers.yaml at:
#   <customer>.numbering.start
# - Support file resolution order is:
#   1. explicit CLI flag
#   2. upward project search
#   3. paths.* in this file
#   4. conventional files in this config directory
#
# paths:
#   customers: 'customers.yaml'
#   issuer: 'issuer.yaml'
#   defaults: 'invoice_defaults.yaml'
#   template: 'template.tex'
#
# numbering:
#   pattern: '{customer_code}-{counter:03}'
#   start: 1
#
# archive:
#   dir: '%s'
#
# email:
#   subject: 'Invoice {invoice_number}'
#   body: |
#     {email_greeting}
#     
#     Please find attached invoice {invoice_number}.
#     Issue date: {issue_date}
#     Due date: {due_date}
#     Outstanding amount: {outstanding_amount}
#     
#     Regards,
#     {issuer_name}
`, defaultArchiveDir), "\n")
}

func (h Host) ConfigTemplate() string {
	return h.defaultConfigTemplate()
}

func (h Host) configTemplatePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}

	if strings.TrimSpace(h.home) != "" {
		if path == h.home {
			path = "~"
		} else if strings.HasPrefix(path, h.home+string(os.PathSeparator)) {
			path = "~" + string(os.PathSeparator) + strings.TrimPrefix(path, h.home+string(os.PathSeparator))
		}
	}

	return filepath.ToSlash(path)
}

func (h Host) DefaultArchiveDir() string {
	baseDir := h.dataBase
	if baseDir == "" {
		return ""
	}
	return filepath.Join(baseDir, configDirName, "invoices")
}

func DisplayPath(path, baseDir string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

func (h Host) expandHomePath(path string) string {
	return config.ExpandHome(path, h.home)
}

func PDFPathForOutput(outputPath string) string {
	ext := filepath.Ext(outputPath)
	if ext == "" {
		return outputPath + ".pdf"
	}
	return strings.TrimSuffix(outputPath, ext) + ".pdf"
}

func firstExistingPath(paths ...string) string {
	for _, path := range paths {
		if path != "" && fileExists(path) {
			return path
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
