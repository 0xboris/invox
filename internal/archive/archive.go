// Package archive holds the rules of the archive directory: which files
// below it are archived invoices, how a name is resolved to one, where the
// previous version of a replaced file is kept, and which working copy
// `archive edit` makes. It reads no YAML. The caller says what an archived
// invoice says about itself through a Reader.
package archive

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/fsutil"
)

// historyDirName is the directory below the archive root that keeps the
// previous version of every archived invoice that re-archiving replaced.
// It is not part of the archive: Walk, List and Resolve skip it.
const historyDirName = ".history"

// backupTimeFormat is the UTC timestamp in a backup's file name.
const backupTimeFormat = "20060102T150405Z"

// Store is the archive directory as configured (archive.dir). Dir may be a
// symlink, and every path the Store reports stays below Dir as configured,
// not below the symlink's target. An empty Dir is an archive with no files.
type Store struct {
	Dir string
}

// HistoryDir is where Backup keeps previous versions.
func (s Store) HistoryDir() string {
	return filepath.Join(s.Dir, historyDirName)
}

// Walk calls visit for every archived invoice file below the root, in the
// lexical order of filepath.WalkDir. A missing or empty Dir has no files. A
// Dir that is not a directory is an error. Backups in the history directory
// are not archived invoices and are skipped.
func (s Store) Walk(visit func(path string) error) error {
	return s.walk(func(path, _ string) error { return visit(path) })
}

func (s Store) walk(visit func(path, rel string) error) error {
	if strings.TrimSpace(s.Dir) == "" {
		return nil
	}
	info, err := os.Stat(s.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s: archive.dir must point to a directory", s.Dir)
	}

	root, err := filepath.EvalSymlinks(s.Dir)
	if err != nil {
		root = s.Dir
	}
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && isHistoryDir(root, path) {
			return filepath.SkipDir
		}
		if entry.IsDir() || !isInvoiceFile(path) {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return visit(filepath.Join(s.Dir, rel), rel)
	})
}

// Identity is what the archive lists of an archived invoice.
type Identity struct {
	CustomerID    string
	IssueDate     string
	Status        string
	InvoiceNumber string
}

// Reader reads the Identity of the archived invoice at path. ok is false
// for a file that is not an invoice, which List leaves out.
type Reader func(path string) (Identity, bool, error)

// Entry is an archived invoice: where it is and what it says about itself.
type Entry struct {
	// Path is the file, below Store.Dir as configured.
	Path string
	// Filename is Path relative to the root, as `archive list` prints it.
	Filename string
	Identity
}

// List reads every archived invoice with read and returns them sorted by
// Filename.
func (s Store) List(read Reader) ([]Entry, error) {
	entries := make([]Entry, 0)
	err := s.walk(func(path, rel string) error {
		identity, ok, err := read(path)
		if err != nil || !ok {
			return err
		}
		entries = append(entries, Entry{Path: path, Filename: rel, Identity: identity})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Filename < entries[j].Filename
	})
	return entries, nil
}

// Newer reports whether e was issued after other: by issue date, where a
// date that parses beats one that does not, then by invoice number,
// Filename and Path.
func (e Entry) Newer(other Entry) bool {
	leftDate, leftOK := parseIssueDate(e.IssueDate)
	rightDate, rightOK := parseIssueDate(other.IssueDate)

	switch {
	case leftOK && !rightOK:
		return true
	case !leftOK && rightOK:
		return false
	case leftOK && rightOK && !leftDate.Equal(rightDate):
		return leftDate.After(rightDate)
	}

	switch {
	case e.InvoiceNumber != other.InvoiceNumber:
		return e.InvoiceNumber > other.InvoiceNumber
	case e.Filename != other.Filename:
		return e.Filename > other.Filename
	default:
		return e.Path > other.Path
	}
}

func parseIssueDate(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false
	}
	return parsed, true
}

// Target is a file inside the archive, named relative to the root.
type Target struct {
	Path string
	Rel  string
}

// Resolve turns a name relative to the root into a Target. The name must
// not be empty, absolute, outside the root or inside the history directory.
// The file need not exist.
func (s Store) Resolve(name string) (Target, error) {
	dir := filepath.Clean(s.Dir)
	cleanName := filepath.Clean(strings.TrimSpace(name))
	if cleanName == "" || cleanName == "." {
		return Target{}, fmt.Errorf("archive filename must not be empty")
	}
	if filepath.IsAbs(cleanName) {
		return Target{}, fmt.Errorf("archive filename must be relative to archive.dir, got %s", cleanName)
	}

	path := filepath.Join(dir, cleanName)
	rel, err := filepath.Rel(dir, path)
	if err != nil {
		return Target{}, err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return Target{}, fmt.Errorf("%s must stay within %s", cleanName, dir)
	}
	if isInHistory(rel) {
		return Target{}, fmt.Errorf("%s is a backup in %s, not an archived invoice", cleanName, historyDirName)
	}
	return Target{Path: path, Rel: filepath.Clean(rel)}, nil
}

// Find is Resolve for a name that must exist as a file.
func (s Store) Find(name string) (Target, error) {
	target, err := s.Resolve(name)
	if err != nil {
		return Target{}, err
	}
	info, err := os.Stat(target.Path)
	if errors.Is(err, os.ErrNotExist) {
		return Target{}, fmt.Errorf("%s does not exist in %s", target.Rel, s.Dir)
	}
	if err != nil {
		return Target{}, err
	}
	if info.IsDir() {
		return Target{}, fmt.Errorf("%s: archived invoice must be a file", target.Path)
	}
	return target, nil
}

// Edit describes the working copy `archive edit` makes of a Target.
type Edit struct {
	// Filename is the working copy's file name.
	Filename string
	// Target is where re-archiving the working copy writes it, relative to
	// the root.
	Target string
	// Replace is the archived file the working copy supersedes, relative to
	// the root, or "" when that is Target itself.
	Replace string
}

// Edit returns the working copy of t. A Markdown archived invoice is edited
// as YAML and re-archived as YAML, replacing the Markdown original.
func (t Target) Edit() Edit {
	rel := filepath.Clean(t.Rel)
	if isMarkdown(rel) {
		yamlRel := strings.TrimSuffix(rel, filepath.Ext(rel)) + ".yaml"
		return Edit{Filename: filepath.Base(yamlRel), Target: yamlRel, Replace: rel}
	}
	return Edit{Filename: filepath.Base(rel), Target: rel}
}

// Backup records an archived file that re-archiving replaced and where its
// previous version was kept.
type Backup struct {
	Path       string
	BackupPath string
}

// Backup copies each archived file to
// <Dir>/.history/<dir>/<name>.<UTC timestamp><ext>, keeping its directory
// below the root. An existing backup is never replaced: a second backup in
// the same second gets a counter.
func (s Store) Backup(paths []string, now time.Time) ([]Backup, error) {
	dir := filepath.Clean(s.Dir)
	stamp := now.UTC().Format(backupTimeFormat)

	backups := make([]Backup, 0, len(paths))
	for _, path := range paths {
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil, err
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		backupPath, err := writeBackup(filepath.Join(dir, historyDirName, rel), stamp, source)
		if err != nil {
			return nil, fmt.Errorf("back up %s: %w", path, err)
		}
		backups = append(backups, Backup{Path: path, BackupPath: backupPath})
	}
	return backups, nil
}

// writeBackup writes data to path with stamp inserted before the extension,
// adding a counter when a backup with that name already exists. The data is
// synced to disk before it returns the backup's path.
func writeBackup(path, stamp string, data []byte) (string, error) {
	ext := filepath.Ext(path)
	stem := strings.TrimSuffix(path, ext)
	for counter := 1; ; counter++ {
		candidate := stem + "." + stamp + ext
		if counter > 1 {
			candidate = stem + "." + stamp + "-" + strconv.Itoa(counter) + ext
		}
		err := fsutil.WriteNewFile(candidate, data, fsutil.Private)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		return candidate, nil
	}
}

// ExistingFiles returns the paths that exist, without duplicates. A
// directory is an error.
func ExistingFiles(paths ...string) ([]string, error) {
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

// isHistoryDir reports whether dir is the history directory directly below
// root.
func isHistoryDir(root, dir string) bool {
	return filepath.Base(dir) == historyDirName && filepath.Dir(dir) == filepath.Clean(root)
}

// isInHistory reports whether rel, relative to the root, is inside the
// history directory. The comparison ignores case, matching file systems
// that do.
func isInHistory(rel string) bool {
	first, _, _ := strings.Cut(filepath.ToSlash(filepath.Clean(rel)), "/")
	return strings.EqualFold(first, historyDirName)
}

// isInvoiceFile reports whether path has the extension of an archived
// invoice: YAML, or Markdown with the invoice as front matter.
func isInvoiceFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".yaml", ".yml":
		return true
	default:
		return isMarkdown(path)
	}
}

func isMarkdown(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".md", ".markdown":
		return true
	default:
		return false
	}
}
