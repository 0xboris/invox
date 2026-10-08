package invoice

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/archive"
	"github.com/0xboris/invox/internal/fsutil"
)

// ArchiveInvoice moves a built invoice, or a working copy from `archive edit`,
// into the archive. Re-archiving a working copy over archived files returns
// an *ArchiveReplaceError, after all other checks passed and before anything
// is written, unless opts.Replace is set. With opts.Replace the replaced
// files are first copied to the archive's history directory.
func (h Host) ArchiveInvoice(now time.Time, invoicePath string, opts ArchiveOptions) (ArchiveResult, error) {
	document, err := loadYAMLDocument(invoicePath)
	if err != nil {
		return ArchiveResult{}, err
	}

	root, err := documentRootMapping(document, invoicePath)
	if err != nil {
		return ArchiveResult{}, err
	}

	invoiceNode, err := invoiceMapping(root, invoicePath)
	if err != nil {
		return ArchiveResult{}, err
	}

	status := strings.TrimSpace(nodeText(findMappingValue(invoiceNode, "status")))
	if opts.AssumeBuilt {
		status = StatusAfterBuild(status)
	}
	store, err := h.archiveStore()
	if err != nil {
		return ArchiveResult{}, err
	}
	if strings.TrimSpace(store.Dir) == "" {
		return ArchiveResult{}, fmt.Errorf("archive directory is unavailable")
	}

	archivePath := filepath.Join(store.Dir, filepath.Base(invoicePath))
	sourcePath := filepath.Clean(invoicePath)

	archiveTargetPath, archiveReplacePath := archiveMetadata(root)
	editingArchive := strings.TrimSpace(archiveTargetPath) != ""
	if editingArchive {
		switch status {
		case "editing", "built":
		default:
			return ArchiveResult{}, fmt.Errorf("%s: invoice.status must be `editing` or `built` before re-archiving, got `%s`", invoicePath, status)
		}
		target, err := store.Resolve(archiveTargetPath)
		if err != nil {
			return ArchiveResult{}, err
		}
		archivePath = target.Path
	} else {
		switch status {
		case "":
			return ArchiveResult{}, fmt.Errorf("%s: invoice.status: missing value", invoicePath)
		case "built":
		default:
			return ArchiveResult{}, fmt.Errorf("%s: invoice.status must be `built` before archiving, got `%s`", invoicePath, status)
		}
		if fileExists(archivePath) {
			return ArchiveResult{}, fmt.Errorf("%s already exists", archivePath)
		}
	}
	if sourcePath == archivePath {
		return ArchiveResult{}, fmt.Errorf("%s is already in the archive directory", invoicePath)
	}

	invoiceNumber := strings.TrimSpace(nodeText(findMappingValue(invoiceNode, "number")))
	if err := h.checkArchivedNumberUnique(invoicePath, invoiceNumber, store, root); err != nil {
		return ArchiveResult{}, err
	}

	var replacePath string
	if editingArchive && strings.TrimSpace(archiveReplacePath) != "" && archiveReplacePath != archiveTargetPath {
		target, err := store.Resolve(archiveReplacePath)
		if err != nil {
			return ArchiveResult{}, err
		}
		replacePath = target.Path
		if replacePath == archivePath {
			replacePath = ""
		}
	}
	var replaced []string
	if editingArchive {
		replaced, err = archive.ExistingFiles(archivePath, replacePath)
		if err != nil {
			return ArchiveResult{}, err
		}
	}
	if len(replaced) > 0 && !opts.Replace {
		return ArchiveResult{}, &ArchiveReplaceError{
			InvoicePath: invoicePath,
			Paths:       replaced,
			HistoryDir:  store.HistoryDir(),
		}
	}
	if opts.DryRun {
		result := ArchiveResult{Path: archivePath, HistoryDir: store.HistoryDir()}
		for _, path := range replaced {
			result.Replaced = append(result.Replaced, archive.Backup{Path: path})
		}
		return result, nil
	}
	if err := fsutil.MkdirAll(store.Dir, fsutil.Private); err != nil {
		return ArchiveResult{}, err
	}
	backups, err := store.Backup(replaced, now)
	if err != nil {
		return ArchiveResult{}, err
	}

	setMappingString(invoiceNode, "status", "archived")
	clearArchiveMetadata(root)
	data, err := encodeYAMLDocument(document)
	if err != nil {
		return ArchiveResult{}, err
	}
	if editingArchive {
		err = fsutil.WriteFile(archivePath, data, fsutil.Private)
	} else if err = fsutil.WriteNewFile(archivePath, data, fsutil.Private); errors.Is(err, fs.ErrExist) {
		return ArchiveResult{}, fmt.Errorf("%s already exists", archivePath)
	}
	if err != nil {
		return ArchiveResult{}, err
	}
	if replacePath != "" {
		if err := os.Remove(replacePath); err != nil && !os.IsNotExist(err) {
			return ArchiveResult{}, fmt.Errorf("remove %s: %w", replacePath, err)
		}
	}
	if err := os.Remove(sourcePath); err != nil {
		return ArchiveResult{}, fmt.Errorf("remove %s: %w", sourcePath, err)
	}
	return ArchiveResult{Path: archivePath, Replaced: backups, HistoryDir: store.HistoryDir()}, nil
}

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
