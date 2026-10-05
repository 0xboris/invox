package invoice

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// archiveHistoryDirName is the directory below archive.dir that keeps the
// previous version of every archived invoice that re-archiving replaced.
// It is not part of the archive: listing, numbering and the duplicate check
// skip it.
const archiveHistoryDirName = ".history"

// archiveBackupTimeFormat is the UTC timestamp in a backup's file name.
const archiveBackupTimeFormat = "20060102T150405Z"

// ArchiveOptions controls ArchiveInvoice.
type ArchiveOptions struct {
	// Replace allows re-archiving a working copy from `archive edit` over
	// the archived file it was opened from. The previous version is kept in
	// the archive's history directory.
	Replace bool
}

// ArchiveResult describes what ArchiveInvoice wrote.
type ArchiveResult struct {
	// Path is the archived invoice.
	Path string
	// Replaced lists the archived files the invoice replaced.
	Replaced []ArchiveBackup
}

// ArchiveBackup records an archived file that re-archiving replaced and
// where its previous version was kept.
type ArchiveBackup struct {
	Path       string
	BackupPath string
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

// isArchiveHistoryDir reports whether dir is the history directory directly
// below the archive root.
func isArchiveHistoryDir(root, dir string) bool {
	return filepath.Base(dir) == archiveHistoryDirName && filepath.Dir(dir) == filepath.Clean(root)
}

// existingArchivePaths returns the paths that exist, without duplicates.
func existingArchivePaths(paths ...string) ([]string, error) {
	var existing []string
	seen := make(map[string]bool)
	for _, path := range paths {
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if info.IsDir() {
			return nil, fmt.Errorf("%s: archived invoice must be a file", path)
		}
		existing = append(existing, path)
	}
	return existing, nil
}

// backupArchivedFiles copies each archived file to
// <archive.dir>/.history/<dir>/<name>.<UTC timestamp><ext>, keeping its
// directory below archive.dir.
func backupArchivedFiles(archiveDir string, paths []string) ([]ArchiveBackup, error) {
	absArchiveDir, err := filepath.Abs(archiveDir)
	if err != nil {
		return nil, err
	}
	stamp := currentDate().UTC().Format(archiveBackupTimeFormat)

	backups := make([]ArchiveBackup, 0, len(paths))
	for _, path := range paths {
		relativePath, err := filepath.Rel(absArchiveDir, path)
		if err != nil {
			return nil, err
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		backupPath := archiveBackupPath(filepath.Join(absArchiveDir, archiveHistoryDirName, relativePath), stamp)
		if err := writeFileAtomic(backupPath, source, 0o644); err != nil {
			return nil, fmt.Errorf("back up %s: %w", path, err)
		}
		backups = append(backups, ArchiveBackup{Path: path, BackupPath: backupPath})
	}
	return backups, nil
}

// archiveBackupPath inserts stamp before the extension of path and adds a
// counter when a backup with that name already exists.
func archiveBackupPath(path, stamp string) string {
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	candidate := stem + "." + stamp + ext
	for counter := 2; fileExists(candidate); counter++ {
		candidate = stem + "." + stamp + "-" + strconv.Itoa(counter) + ext
	}
	return candidate
}

// MarkInvoiceBuilt sets invoice.status to `built` after a successful PDF
// build. An archived invoice keeps `archived`: rebuilding its PDF does not
// take it out of the archive.
func MarkInvoiceBuilt(invoicePath string) error {
	value, err := loadYAML(invoicePath)
	if err != nil {
		return err
	}
	if root, ok := value.(map[string]any); ok {
		if invoice, ok := root["invoice"].(map[string]any); ok && strings.TrimSpace(asString(invoice["status"])) == "archived" {
			return nil
		}
	}
	return SetInvoiceStatus(invoicePath, "built")
}
