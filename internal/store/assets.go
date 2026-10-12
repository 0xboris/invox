package store

import "path/filepath"

// FindAsset looks for relPath, a file or with isDir a directory, next to
// the template, then in the config directory. It returns "" when there is
// none.
func (h Host) FindAsset(templatePath, relPath string, isDir bool) string {
	if candidate := filepath.Join(filepath.Dir(templatePath), relPath); pathExists(candidate, isDir) {
		return candidate
	}
	found, _ := h.findInConfigDir(isDir, relPath)
	return found.Path
}
