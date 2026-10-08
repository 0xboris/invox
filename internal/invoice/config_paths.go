package invoice

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/config"
	"github.com/0xboris/invox/internal/fsutil"
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

func (h Host) expandHomePath(path string) string {
	return config.ExpandHome(path, h.home)
}

func (h Host) GlobalInvoiceDefaultsPath() string {
	return filepath.Join(h.ConfigDir(), "invoice_defaults.yaml")
}

// ensureStarterFile writes content to path if path is missing or empty and
// reports whether it did. A dangling symlink at path counts as missing, and
// its target is written. A file created by someone else in the meantime is
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
	if info, err := os.Lstat(path); err == nil && info.Mode()&fs.ModeSymlink != 0 {
		if err := fsutil.WriteFile(path, content, perm); err != nil {
			return false, err
		}
		return true, nil
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
