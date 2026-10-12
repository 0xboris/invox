package store

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/billing"
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

// editableConfigPath returns the config file to open in an editor, creating
// it from the template when it does not exist: the explicit config file, else
// config.yaml where Host reads it, else config.yaml in the config
// directory.
func (h Host) editableConfigPath() (string, error) {
	path := h.configFile
	if path == "" {
		found, err := h.findInConfigDir(false, "config.yaml")
		var missingDir *billing.ConfigDirNotFoundError
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
	if _, err := ensureStarterFile(path, []byte(h.ConfigTemplate()), fsutil.Public); err != nil {
		return "", err
	}
	return path, nil
}

// ConfigTemplate returns the starter config.yaml: each setting, commented
// out, with its default.
func (h Host) ConfigTemplate() string {
	archiveDir := h.configTemplatePath(h.DefaultArchiveDir())
	if strings.TrimSpace(archiveDir) == "" {
		archiveDir = "invoices"
	}
	var b strings.Builder
	err := configStarter.Execute(&b, struct {
		ArchiveDir        string
		EmailPlaceholders []billing.EmailPlaceholder
	}{archiveDir, billing.EmailPlaceholders()})
	if err != nil {
		panic(fmt.Sprintf("store: starter config.yaml: %v", err))
	}
	return b.String()
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
