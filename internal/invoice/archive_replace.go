package invoice

import (
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/archive"
)

// ArchiveOptions controls ArchiveInvoice.
type ArchiveOptions struct {
	// Replace allows re-archiving a working copy from `archive edit` over
	// the archived file it was opened from. The previous version is kept in
	// the archive's history directory.
	Replace bool
	// DryRun runs every check and returns the result without writing. The
	// result's backups have no BackupPath.
	DryRun bool
	// AssumeBuilt checks the invoice as if `build` had already marked it
	// built, for `build --archive --dry-run`, which does not mark it.
	AssumeBuilt bool
}

// ArchiveResult describes what ArchiveInvoice wrote.
type ArchiveResult struct {
	// Path is the archived invoice.
	Path string
	// Replaced lists the archived files the invoice replaced.
	Replaced []archive.Backup
	// HistoryDir is where the replaced files' previous versions are kept.
	HistoryDir string
}

// ArchiveReplaceError reports that archiving the invoice would replace
// archived files and ArchiveOptions.Replace was not set. Nothing was written.
type ArchiveReplaceError struct {
	InvoicePath string
	// Paths are the archived files that would be replaced.
	Paths []string
	// HistoryDir is where their previous versions would be kept.
	HistoryDir string
}

func (e *ArchiveReplaceError) Error() string {
	return fmt.Sprintf("%s: archiving replaces archived invoice %s", e.InvoicePath, strings.Join(e.Paths, ", "))
}

// MarkInvoiceBuilt sets invoice.status to `built` after a successful PDF
// build. An archived invoice keeps `archived`: rebuilding its PDF does not
// take it out of the archive.
func MarkInvoiceBuilt(invoicePath string) error {
	var identity invoiceIdentity
	if err := decodeYAMLFile(invoicePath, &identity, false); err != nil && !isDecodeError(err) {
		return err
	}
	status := ""
	if identity.Invoice != nil {
		status = identity.Invoice.Status.Trim()
	}
	if StatusAfterBuild(status) == "archived" {
		return nil
	}
	return SetInvoiceStatus(invoicePath, "built")
}

// StatusAfterBuild is the invoice.status a successful build leaves: `built`,
// except that an archived invoice stays `archived`.
func StatusAfterBuild(status string) string {
	if status == "archived" {
		return status
	}
	return "built"
}
