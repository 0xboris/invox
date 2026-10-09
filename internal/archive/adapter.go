package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/fsutil"
	"github.com/0xboris/invox/internal/invoice"
)

// Archive is the configured archive directory as billing uses it.
type Archive struct {
	// Locate returns archive.dir, "" when none is configured.
	Locate func() (string, error)
	// Read reads what an archived invoice says about itself.
	Read Reader
	// Rewrite returns the invoice at path with change applied, keeping its
	// comments and layout.
	Rewrite func(path string, change func(*invoice.Invoice) error) ([]byte, error)
	// Now stamps the backups of replaced files.
	Now func() time.Time
}

var _ billing.Archive = Archive{}

func (a Archive) store() (Store, error) {
	dir, err := a.Locate()
	if err != nil {
		return Store{}, err
	}
	return Store{Dir: dir}, nil
}

// Entries reads every archived invoice, sorted by Filename.
func (a Archive) Entries() ([]billing.ArchiveEntry, billing.Unread, error) {
	s, err := a.store()
	if err != nil {
		return nil, billing.Unread{}, err
	}
	return s.List(a.Read)
}

// Duplicate returns the archived invoice, in file name order, that has
// inv's number, other than src and, for a working copy, the archived file
// it replaces.
func (a Archive) Duplicate(src string, inv invoice.Invoice) (string, billing.Unread, error) {
	s, err := a.store()
	if err != nil {
		return "", billing.Unread{}, err
	}
	var number string
	if inv.Header != nil {
		number = inv.Header.Number.Trim()
	}
	if strings.TrimSpace(s.Dir) == "" || number == "" {
		return "", billing.Unread{}, nil
	}
	excluded := map[string]bool{filepath.Clean(src): true}
	// Only a working copy from `archive edit` may reuse the number of the
	// archived file it replaces.
	if link := linkedPath(inv); link != "" {
		target, err := s.Resolve(link)
		if err != nil {
			return "", billing.Unread{}, err
		}
		excluded[target.Path] = true
	}
	entries, unread, err := s.List(a.Read)
	if err != nil {
		return "", unread, err
	}
	for _, entry := range entries {
		if entry.Number == number && !excluded[filepath.Clean(entry.Path)] {
			return filepath.Clean(entry.Path), unread, nil
		}
	}
	return "", unread, nil
}

// linkedPath is the archived file a working copy from `archive edit`
// replaces, relative to the archive, or "" for any other invoice. The link
// is written with forward slashes on every OS.
func linkedPath(inv invoice.Invoice) string {
	if inv.Archive == nil {
		return ""
	}
	return filepath.FromSlash(inv.Archive.ArchivePath.Trim())
}

// Checkout resolves ref and says where its working copy in workDir goes.
func (a Archive) Checkout(ref, workDir string) (billing.Checkout, error) {
	s, err := a.store()
	if err != nil {
		return billing.Checkout{}, err
	}
	if strings.TrimSpace(s.Dir) == "" {
		return billing.Checkout{}, errors.New("archive directory is unavailable")
	}
	target, err := s.Find(ref)
	if err != nil {
		return billing.Checkout{}, err
	}
	return billing.Checkout{
		Archived: target.Path,
		Path:     filepath.Join(workDir, filepath.Base(target.Rel)),
		Link:     invoice.ArchiveLink{ArchivePath: invoice.Text(target.Rel)},
	}, nil
}

// Add writes inv, the invoice at src, into the archive with opts.Change
// applied: over the archived file a working copy names, else under src's
// name. It backs up the archived file it replaces, then removes src. The
// rewrite runs before a dry run returns, so a dry run checks the invoice
// too.
func (a Archive) Add(src string, inv invoice.Invoice, opts billing.AddOptions) (billing.ArchiveResult, error) {
	s, err := a.store()
	if err != nil {
		return billing.ArchiveResult{}, err
	}
	if strings.TrimSpace(s.Dir) == "" {
		return billing.ArchiveResult{}, errors.New("archive directory is unavailable")
	}
	path, overwrite := filepath.Join(s.Dir, filepath.Base(src)), false
	if link := linkedPath(inv); link != "" {
		target, err := s.Resolve(link)
		if err != nil {
			return billing.ArchiveResult{}, err
		}
		path, overwrite = target.Path, true
	} else if isFile(path) {
		return billing.ArchiveResult{}, fmt.Errorf("%s already exists", path)
	}
	if filepath.Clean(src) == path {
		return billing.ArchiveResult{}, fmt.Errorf("%s is already in the archive directory", src)
	}
	var replaced []string
	if overwrite {
		if replaced, err = ExistingFiles(path); err != nil {
			return billing.ArchiveResult{}, err
		}
	}
	if len(replaced) > 0 && !opts.Replace {
		return billing.ArchiveResult{}, &billing.ArchiveReplaceError{InvoicePath: src, Paths: replaced, HistoryDir: s.HistoryDir()}
	}
	data, err := a.Rewrite(src, opts.Change)
	if err != nil {
		return billing.ArchiveResult{}, err
	}
	if opts.DryRun {
		result := billing.ArchiveResult{Path: path, HistoryDir: s.HistoryDir()}
		for _, p := range replaced {
			result.Replaced = append(result.Replaced, billing.Backup{Path: p})
		}
		return result, nil
	}
	if err := fsutil.MkdirAll(s.Dir, fsutil.Private); err != nil {
		return billing.ArchiveResult{}, err
	}
	backups, err := s.Backup(replaced, a.Now())
	if err != nil {
		return billing.ArchiveResult{}, err
	}
	if overwrite {
		err = fsutil.WriteFile(path, data, fsutil.Private)
	} else if err = fsutil.WriteNewFile(path, data, fsutil.Private); errors.Is(err, os.ErrExist) {
		return billing.ArchiveResult{}, fmt.Errorf("%s already exists", path)
	}
	if err != nil {
		return billing.ArchiveResult{}, err
	}
	if err := os.Remove(filepath.Clean(src)); err != nil {
		return billing.ArchiveResult{}, fmt.Errorf("remove %s: %w", filepath.Clean(src), err)
	}
	return billing.ArchiveResult{Path: path, Replaced: backups, HistoryDir: s.HistoryDir()}, nil
}

// Protects reports whether path is an existing file inside the archive
// directory. It compares files rather than names, so a differently cased
// path on a case-insensitive file system, or a symlinked parent, still
// matches.
func (a Archive) Protects(path string) (bool, error) {
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		return false, nil //nolint:nilerr // no file there means nothing to protect
	}
	s, err := a.store()
	if err != nil {
		return false, err
	}
	if strings.TrimSpace(s.Dir) == "" {
		return false, nil
	}
	archiveInfo, err := os.Stat(s.Dir)
	if err != nil {
		return false, nil //nolint:nilerr // no readable archive directory means no archived file to protect
	}
	if inDir(path, archiveInfo) {
		return true, nil
	}
	if resolved, err := filepath.EvalSymlinks(path); err == nil && inDir(resolved, archiveInfo) {
		return true, nil
	}
	return false, nil
}

// inDir reports whether one of path's parent directories is dir.
func inDir(path string, dir os.FileInfo) bool {
	for parent := filepath.Dir(filepath.Clean(path)); ; parent = filepath.Dir(parent) {
		if info, err := os.Stat(parent); err == nil && os.SameFile(info, dir) {
			return true
		}
		if filepath.Dir(parent) == parent {
			return false
		}
	}
}

// Source returns the invoice YAML file the PDF at pdf was built from: next
// to it, else in the archive.
func (a Archive) Source(pdf string) (string, billing.Unread, error) {
	base := strings.TrimSuffix(pdf, filepath.Ext(pdf))
	candidates := []string{base + ".yaml", base + ".yml"}
	for _, candidate := range candidates {
		if isFile(candidate) {
			return candidate, billing.Unread{}, nil
		}
	}
	return a.findFile(filepath.Base(candidates[0]), filepath.Base(candidates[1]))
}

// isFile reports whether path is a file.
func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// findFile returns the archived file named one of names: one directly in
// the archive directory, else the only one below it. It returns "" when
// there is none or no archive directory, and an error when several match.
func (a Archive) findFile(names ...string) (string, billing.Unread, error) {
	s, err := a.store()
	if err != nil {
		return "", billing.Unread{}, err
	}
	if strings.TrimSpace(s.Dir) == "" {
		return "", billing.Unread{}, nil
	}
	for _, name := range names {
		if info, err := os.Stat(filepath.Join(s.Dir, name)); err == nil && !info.IsDir() {
			return filepath.Join(s.Dir, name), billing.Unread{}, nil
		}
	}
	var matches []string
	markdown, err := s.Walk(func(path string) error {
		for _, name := range names {
			if filepath.Base(path) == name {
				matches = append(matches, path)
				break
			}
		}
		return nil
	})
	unread := billing.Unread{Dir: s.Dir, Markdown: markdown}
	if err != nil {
		return "", unread, err
	}
	switch len(matches) {
	case 0:
		return "", unread, nil
	case 1:
		return matches[0], unread, nil
	}
	sort.Strings(matches)
	return "", unread, fmt.Errorf("%s: multiple archived invoice YAML files match %s; pass the YAML path explicitly: %s", s.Dir, names[0], strings.Join(matches, ", "))
}
