package shared

import (
	"errors"

	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

// CustomersPath returns the absolute path of the customers file that
// command reads: flagValue made absolute against baseDir when it is set,
// else the file found from baseDir. Finding none is a usage error that names
// the ways to provide it.
func CustomersPath(h invoice.Host, command, flagValue, baseDir string) (string, error) {
	path, err := h.SupportPath(invoice.Customers, flagValue, baseDir)
	var notFound *invoice.SupportFileNotFoundError
	if errors.As(err, &notFound) {
		return "", cmdutil.FlagErrorf(command, "%s; pass -c/--customers, %s", notFound, notFound.Remedy())
	}
	return path, err
}
