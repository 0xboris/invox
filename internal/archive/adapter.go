package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

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
}

var _ billing.Archive = Archive{}

func (a Archive) store() (Store, error) {
	dir, err := a.Locate()
	if err != nil {
		return Store{}, err
	}
	return Store{Dir: dir}, nil
}

// Entries reads every archived invoice in the order of Walk.
func (a Archive) Entries() ([]billing.ArchiveEntry, error) {
	s, err := a.store()
	if err != nil {
		return nil, err
	}
	return s.entries(a.Read)
}

// Dir returns the archive directory.
func (a Archive) Dir() (string, error) {
	return a.Locate()
}

// Place says where archiving the invoice at src writes it.
func (a Archive) Place(src string, head billing.Head) (billing.Placement, error) {
	s, err := a.store()
	if err != nil {
		return billing.Placement{}, err
	}
	p := billing.Placement{Path: filepath.Join(s.Dir, filepath.Base(src)), HistoryDir: s.HistoryDir()}
	if head.WorkingCopy() {
		target, err := s.Resolve(head.ArchivePath)
		if err != nil {
			return billing.Placement{}, err
		}
		p.Path, p.Overwrite = target.Path, true
	} else if isFile(p.Path) {
		return billing.Placement{}, fmt.Errorf("%s already exists", p.Path)
	}
	if filepath.Clean(src) == p.Path {
		return billing.Placement{}, fmt.Errorf("%s is already in the archive directory", src)
	}
	if head.WorkingCopy() && head.ReplacePath != "" && head.ReplacePath != head.ArchivePath {
		target, err := s.Resolve(head.ReplacePath)
		if err != nil {
			return billing.Placement{}, err
		}
		if target.Path != p.Path {
			p.Remove = target.Path
		}
	}
	return p, nil
}

// Duplicate returns the archived invoice, in file name order, that has
// head's number, other than src and, for a working copy, the archived
// files it replaces.
func (a Archive) Duplicate(src string, head billing.Head) (string, error) {
	s, err := a.store()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(s.Dir) == "" || head.Number == "" {
		return "", nil
	}
	excluded := map[string]bool{filepath.Clean(src): true}
	// Only a working copy from `archive edit` may reuse the number of the
	// archived file it replaces.
	if head.WorkingCopy() {
		for _, name := range []string{head.ArchivePath, head.ReplacePath} {
			if name == "" {
				continue
			}
			target, err := s.Resolve(name)
			if err != nil {
				return "", err
			}
			excluded[target.Path] = true
		}
	}
	entries, err := s.List(a.Read)
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.Number == head.Number && !excluded[filepath.Clean(entry.Path)] {
			return filepath.Clean(entry.Path), nil
		}
	}
	return "", nil
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
	edit := target.Edit()
	return billing.Checkout{
		Archived: target.Path,
		Path:     filepath.Join(workDir, edit.Filename),
		Link:     billingLink(edit),
	}, nil
}

// Add writes the invoice at src to p.Path, after backing up the archived
// files it replaces, then removes p.Remove and src.
func (a Archive) Add(src string, p billing.Placement, opts billing.AddOptions) (billing.ArchiveResult, error) {
	var replaced []string
	if p.Overwrite {
		var err error
		if replaced, err = ExistingFiles(p.Path, p.Remove); err != nil {
			return billing.ArchiveResult{}, err
		}
	}
	if len(replaced) > 0 && !opts.Replace {
		return billing.ArchiveResult{}, &billing.ArchiveReplaceError{InvoicePath: src, Paths: replaced, HistoryDir: p.HistoryDir}
	}
	if opts.DryRun {
		result := billing.ArchiveResult{Path: p.Path, HistoryDir: p.HistoryDir}
		for _, path := range replaced {
			result.Replaced = append(result.Replaced, billing.Backup{Path: path})
		}
		return result, nil
	}
	s, err := a.store()
	if err != nil {
		return billing.ArchiveResult{}, err
	}
	if err := fsutil.MkdirAll(s.Dir, fsutil.Private); err != nil {
		return billing.ArchiveResult{}, err
	}
	backups, err := s.Backup(replaced, opts.Now)
	if err != nil {
		return billing.ArchiveResult{}, err
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return billing.ArchiveResult{}, err
	}
	if p.Overwrite {
		err = fsutil.WriteFile(p.Path, data, fsutil.Private)
	} else if err = fsutil.WriteNewFile(p.Path, data, fsutil.Private); errors.Is(err, os.ErrExist) {
		return billing.ArchiveResult{}, fmt.Errorf("%s already exists", p.Path)
	}
	if err != nil {
		return billing.ArchiveResult{}, err
	}
	if p.Remove != "" {
		if err := os.Remove(p.Remove); err != nil && !os.IsNotExist(err) {
			return billing.ArchiveResult{}, fmt.Errorf("remove %s: %w", p.Remove, err)
		}
	}
	if err := os.Remove(filepath.Clean(src)); err != nil {
		return billing.ArchiveResult{}, fmt.Errorf("remove %s: %w", filepath.Clean(src), err)
	}
	return billing.ArchiveResult{Path: p.Path, Replaced: backups, HistoryDir: p.HistoryDir}, nil
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
func (a Archive) Source(pdf string) (string, error) {
	base := strings.TrimSuffix(pdf, filepath.Ext(pdf))
	candidates := []string{base + ".yaml", base + ".yml"}
	for _, candidate := range candidates {
		if isFile(candidate) {
			return candidate, nil
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
func (a Archive) findFile(names ...string) (string, error) {
	s, err := a.store()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(s.Dir) == "" {
		return "", nil
	}
	for _, name := range names {
		if info, err := os.Stat(filepath.Join(s.Dir, name)); err == nil && !info.IsDir() {
			return filepath.Join(s.Dir, name), nil
		}
	}
	var matches []string
	err = s.Walk(func(path string) error {
		for _, name := range names {
			if filepath.Base(path) == name {
				matches = append(matches, path)
				break
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	switch len(matches) {
	case 0:
		return "", nil
	case 1:
		return matches[0], nil
	}
	sort.Strings(matches)
	return "", fmt.Errorf("%s: multiple archived invoice YAML files match %s; pass the YAML path explicitly: %s", s.Dir, names[0], strings.Join(matches, ", "))
}

func billingLink(e Edit) invoice.ArchiveLink {
	return invoice.ArchiveLink{ArchivePath: invoice.Text(e.Target), ArchiveReplacePath: invoice.Text(e.Replace)}
}
