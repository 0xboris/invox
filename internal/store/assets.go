package store

import "path/filepath"

// findAsset looks for relPath next to the template, then in the config
// directories.
func (h Host) findAsset(templatePath, relPath string, isDir bool) string {
	if candidate := filepath.Join(filepath.Dir(templatePath), relPath); pathExists(candidate, isDir) {
		return candidate
	}
	found, _ := h.findInConfigDir(isDir, relPath)
	return found.Path
}
