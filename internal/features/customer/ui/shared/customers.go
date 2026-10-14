package shared

import (
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

// CustomersPath returns the absolute path of the customers file that
// command reads: flagValue made absolute against baseDir when it is set,
// else the file found from baseDir. Finding none is a usage error that names
// the ways to provide it.
func CustomersPath(h invoice.Host, command, flagValue, baseDir string) (string, error) {
	path := flagValue
	if strings.TrimSpace(path) == "" {
		found, err := h.ResolveSupportFile(invoice.Customers, baseDir)
		if err != nil {
			return "", err
		}
		path = found.Path
	}
	if strings.TrimSpace(path) == "" {
		return "", cmdutil.FlagErrorf(command, "customers file not found; pass -c/--customers, set paths.customers in config.yaml, or place customers.yaml at %s", h.GlobalCustomersPath())
	}
	return invoice.AbsPath(filepath.Clean(baseDir), path), nil
}
