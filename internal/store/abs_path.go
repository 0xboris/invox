package store

import (
	"os"
	"path/filepath"
)

// absPath is filepath.Abs with base in place of the process working
// directory.
func absPath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if path != "" && os.IsPathSeparator(path[0]) {
		// A rooted path without a volume, such as \x on Windows, stays on
		// base's drive, as filepath.Abs keeps it on the current drive.
		return filepath.Join(filepath.VolumeName(base), path)
	}
	return filepath.Join(base, path)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func pathExists(path string, isDir bool) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir() == isDir
}
