// Package fsutil writes files so that a reader never sees a partial file:
// data goes to a temporary file next to the target, is synced, and then
// replaces or claims the target in one step.
package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
)

// Perm is the mode of a file the writer creates and of the directories it
// creates for it. The process umask is cleared from it, except for Private,
// which is used as is.
type Perm struct {
	File, Dir fs.FileMode
	exact     bool
}

var (
	// Private is for data that stays with its owner: issuer and customer
	// details, archived invoices and their backups, the config and archive
	// directories.
	Private = Perm{File: 0o600, Dir: 0o700, exact: true}
	// Public is for rendered outputs, invoice drafts and other config files.
	Public = Perm{File: 0o644, Dir: 0o755}
)

func (p Perm) fileMode() fs.FileMode {
	if p.exact {
		return p.File
	}
	return p.File &^ umask
}

func (p Perm) dirMode() fs.FileMode {
	if p.exact {
		return p.Dir
	}
	return p.Dir &^ umask
}

const maxSymlinkHops = 40

// link is os.Link, swapped in tests to act like a file system without hard
// links.
var link = os.Link

// testHookBeforeCommit, when set, runs right before the temporary file
// replaces or claims target.
var testHookBeforeCommit func(target string)

// WriteFile atomically replaces or creates path with data. If path is a
// symlink, the file it points to is written and the link is kept. An
// existing file keeps its mode; a new one gets perm.File. Missing directories
// above the file get perm.Dir, so a caller writing Private files creates its
// private root with MkdirAll first.
func WriteFile(path string, data []byte, perm Perm) error {
	target, err := resolve(path)
	if err != nil {
		return err
	}
	mode := perm.fileMode()
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(target)
	if err := mkdirAll(dir, perm.dirMode(), perm.dirMode()); err != nil {
		return err
	}
	tempPath, err := writeTemp(dir, data, mode)
	if err != nil {
		return err
	}
	if testHookBeforeCommit != nil {
		testHookBeforeCommit(target)
	}
	if err := os.Rename(tempPath, target); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return syncDir(dir)
}

// WriteNewFile atomically creates path with data and mode perm.File. If
// anything already exists at path, a symlink included, it is left untouched
// and the error matches fs.ErrExist. Missing directories above the file get
// perm.Dir, so a caller writing Private files creates its private root with
// MkdirAll first. On file systems without hard links it falls back to
// O_EXCL, which never clobbers but can leave a partial file if the process
// dies mid-write.
func WriteNewFile(path string, data []byte, perm Perm) error {
	if _, err := os.Lstat(path); err == nil {
		return existsError(path)
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(path)
	if err := mkdirAll(dir, perm.dirMode(), perm.dirMode()); err != nil {
		return err
	}
	tempPath, err := writeTemp(dir, data, perm.fileMode())
	if err != nil {
		return err
	}
	defer os.Remove(tempPath)
	if testHookBeforeCommit != nil {
		testHookBeforeCommit(path)
	}
	// A hard link fails if path exists, so nothing created since the check
	// above is ever replaced.
	if err := link(tempPath, path); err != nil {
		if errors.Is(err, fs.ErrExist) {
			return existsError(path)
		}
		if err := writeExclusive(path, data, perm.fileMode()); err != nil {
			return err
		}
	}
	return syncDir(dir)
}

// MkdirAll creates dir with mode perm.Dir and any missing parents with
// Public's. Directories that already exist keep their mode.
func MkdirAll(dir string, perm Perm) error {
	return mkdirAll(dir, perm.dirMode(), Public.dirMode())
}

// mkdirAll creates dir with mode leaf and any missing parents with mode
// parents, set explicitly so that the umask does not decide them.
func mkdirAll(dir string, leaf, parents fs.FileMode) error {
	var missing []string
	for current := dir; ; {
		info, err := os.Stat(current)
		if err == nil {
			if !info.IsDir() {
				return &fs.PathError{Op: "mkdir", Path: current, Err: syscall.ENOTDIR}
			}
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}
	for i := len(missing) - 1; i >= 0; i-- {
		current := missing[i]
		perm := parents
		if i == 0 {
			perm = leaf
		}
		if err := os.Mkdir(current, perm); err != nil {
			if info, statErr := os.Stat(current); statErr == nil && info.IsDir() {
				continue
			}
			return err
		}
		if err := os.Chmod(current, perm); err != nil {
			return err
		}
	}
	return nil
}

// resolve returns the file path refers to, following a symlink in its last
// component even when the link dangles. A path that does not exist is
// returned as is.
func resolve(path string) (string, error) {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved, nil
	}
	current := path
	for range maxSymlinkHops {
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			return current, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			return current, nil
		}
		target, err := os.Readlink(current)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(current), target)
		}
		current = target
	}
	return "", &fs.PathError{Op: "write", Path: path, Err: syscall.ELOOP}
}

// writeTemp writes data to a new temporary file in dir with the given mode,
// synced to disk, and returns its path.
func writeTemp(dir string, data []byte, mode fs.FileMode) (string, error) {
	file, err := os.CreateTemp(dir, ".invox-*")
	if err != nil {
		return "", err
	}
	if err := writeAndClose(file, data, mode); err != nil {
		_ = os.Remove(file.Name())
		return "", err
	}
	return file.Name(), nil
}

// writeExclusive creates path with O_EXCL, for file systems without hard
// links.
func writeExclusive(path string, data []byte, mode fs.FileMode) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if errors.Is(err, fs.ErrExist) {
		return existsError(path)
	}
	if err != nil {
		return err
	}
	if err := writeAndClose(file, data, mode); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

// writeAndClose sets the mode explicitly so that the umask does not decide
// it, then syncs the data to disk.
func writeAndClose(file *os.File, data []byte, mode fs.FileMode) error {
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Chmod(mode); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func existsError(path string) error {
	return &fs.PathError{Op: "write", Path: path, Err: fs.ErrExist}
}
