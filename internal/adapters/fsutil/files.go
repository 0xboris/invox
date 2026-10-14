package fsutil

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/0xboris/invox/internal/domain/ports"
)

// Files is ports.Files on the real file system.
type Files struct{}

var _ ports.Files = Files{}

func (Files) ReadFile(path string) ([]byte, error)          { return os.ReadFile(path) }
func (Files) Stat(path string) (fs.FileInfo, error)         { return os.Stat(path) }
func (Files) Lstat(path string) (fs.FileInfo, error)        { return os.Lstat(path) }
func (Files) ReadDir(path string) ([]fs.DirEntry, error)    { return os.ReadDir(path) }
func (Files) Walk(root string, fn filepath.WalkFunc) error  { return filepath.Walk(root, fn) }
func (Files) WalkDir(root string, fn fs.WalkDirFunc) error  { return filepath.WalkDir(root, fn) }
func (Files) EvalSymlinks(path string) (string, error)      { return filepath.EvalSymlinks(path) }
func (Files) SameFile(a, b fs.FileInfo) bool                { return os.SameFile(a, b) }
func (Files) Remove(path string) error                      { return os.Remove(path) }
func (Files) RemoveAll(path string) error                   { return os.RemoveAll(path) }
func (Files) MkdirTemp(dir, pattern string) (string, error) { return os.MkdirTemp(dir, pattern) }

func (Files) WriteFile(path string, data []byte, perm Perm) error {
	return WriteFile(path, data, perm)
}

func (Files) WriteNewFile(path string, data []byte, perm Perm) error {
	return WriteNewFile(path, data, perm)
}

func (Files) MkdirAll(dir string, perm Perm) error { return MkdirAll(dir, perm) }
