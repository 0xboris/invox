package invoice

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

// isInArchiveHistory reports whether relativePath, relative to archive.dir,
// is inside the history directory. The comparison ignores case, matching
// file systems that do.
func isInArchiveHistory(relativePath string) bool {
	first, _, _ := strings.Cut(filepath.ToSlash(filepath.Clean(relativePath)), "/")
	return strings.EqualFold(first, archiveHistoryDirName)
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
func backupArchivedFiles(archiveDir string, paths []string, now time.Time) ([]ArchiveBackup, error) {
	absArchiveDir := filepath.Clean(archiveDir)
	stamp := now.UTC().Format(archiveBackupTimeFormat)

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
		backupPath, err := writeArchiveBackup(filepath.Join(absArchiveDir, archiveHistoryDirName, relativePath), stamp, source)
		if err != nil {
			return nil, fmt.Errorf("back up %s: %w", path, err)
		}
		backups = append(backups, ArchiveBackup{Path: path, BackupPath: backupPath})
	}
	return backups, nil
}

// writeArchiveBackup writes data to path with stamp inserted before the
// extension, adding a counter when a backup with that name already exists.
// The name is claimed with O_EXCL, so an existing backup is never replaced,
// and the data is synced to disk before it returns the backup's path.
func writeArchiveBackup(path, stamp string, data []byte) (string, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for counter := 1; ; counter++ {
		candidate := stem + "." + stamp + ext
		if counter > 1 {
			candidate = stem + "." + stamp + "-" + strconv.Itoa(counter) + ext
		}
		file, err := os.OpenFile(candidate, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if err := writeAndClose(file, data); err != nil {
			_ = os.Remove(candidate)
			return "", err
		}
		return candidate, syncDir(dir)
	}
}

// writeAndClose writes data to file, syncs it to disk and closes it.
func writeAndClose(file *os.File, data []byte) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
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
