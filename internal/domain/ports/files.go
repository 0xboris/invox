// Package ports declares what the domain needs from outside the process.
// Adapters implement the ports, and the composition root passes them in.
package ports

import (
	"io/fs"
	"path/filepath"
)

// Perm is the mode of a file the writer creates and of the directories it
// creates for it. The process umask is cleared from it, except when Exact is
// set, as for Private.
type Perm struct {
	File, Dir fs.FileMode
	Exact     bool
}

var (
	// Private is for data that stays with its owner: issuer and customer
	// details, archived invoices and their backups, the config and archive
	// directories.
	Private = Perm{File: 0o600, Dir: 0o700, Exact: true}
	// Public is for rendered outputs, invoice drafts and other config files.
	Public = Perm{File: 0o644, Dir: 0o755}
)

// Files is the file system the domain reads and writes. Paths are OS paths.
// Errors keep the standard library's types, so errors.Is(err,
// fs.ErrNotExist) and errors.Is(err, fs.ErrExist) work on them.
type Files interface {
	ReadFile(path string) ([]byte, error)
	Stat(path string) (fs.FileInfo, error)
	Lstat(path string) (fs.FileInfo, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Walk(root string, fn filepath.WalkFunc) error
	WalkDir(root string, fn fs.WalkDirFunc) error
	EvalSymlinks(path string) (string, error)
	SameFile(a, b fs.FileInfo) bool
	Remove(path string) error
	RemoveAll(path string) error
	MkdirTemp(dir, pattern string) (string, error)
	// TempDir returns the directory for temporary files.
	TempDir() string

	// WriteFile atomically replaces or creates path with data. If path is a
	// symlink, the file it points to is written and the link is kept. An
	// existing file keeps its mode; a new one gets perm.File. Missing
	// directories above the file get perm.Dir.
	WriteFile(path string, data []byte, perm Perm) error
	// WriteNewFile creates path with data and fails with an error matching
	// fs.ErrExist when something is already there. Missing directories get
	// perm.Dir.
	WriteNewFile(path string, data []byte, perm Perm) error
	// MkdirAll creates dir with perm.Dir. Missing parents get perm.Dir too,
	// except under an Exact perm, where they get Public.Dir so that only dir
	// itself is private.
	MkdirAll(dir string, perm Perm) error
}
