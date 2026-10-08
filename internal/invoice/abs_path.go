package invoice

import (
	"os"
	"path/filepath"
	"strings"
)

// AbsPath is filepath.Abs with base in place of the process working
// directory.
func AbsPath(base, path string) string {
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

func DisplayPath(path, baseDir string) string {
	rel, err := filepath.Rel(baseDir, path)
	if err == nil && !strings.HasPrefix(rel, "..") {
		return rel
	}
	return path
}

// ReplaceExt returns path with its extension replaced by ext, or with ext
// added when it has none. It returns "" for an empty path.
func ReplaceExt(path, ext string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	return strings.TrimSuffix(path, filepath.Ext(path)) + ext
}

func firstExistingPath(paths ...string) string {
	for _, path := range paths {
		if path != "" && fileExists(path) {
			return path
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func pathExists(path string, isDir bool) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir() == isDir
}
