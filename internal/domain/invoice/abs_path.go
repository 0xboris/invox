package invoice

import "path/filepath"

// AbsPath is filepath.Abs with base in place of the process working
// directory.
func AbsPath(base, path string) string {
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	if path != "" && (path[0] == '/' || path[0] == filepath.Separator) {
		// A rooted path without a volume, such as \x on Windows, stays on
		// base's drive, as filepath.Abs keeps it on the current drive.
		return filepath.Join(filepath.VolumeName(base), path)
	}
	return filepath.Join(base, path)
}
