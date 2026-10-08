package store

import (
	"os"
	"path/filepath"
	"strings"
)

// refuseArchivedOverwrite returns an *ArchivedOutputError when overwrite
// would replace an existing file inside the archive directory: an archived
// invoice is replaced only by re-archiving, which keeps a backup.
func (h Host) refuseArchivedOverwrite(outputPath string, overwrite bool) error {
	if !overwrite || !fileExists(outputPath) {
		return nil
	}
	store, err := h.archiveStore()
	if err != nil {
		return err
	}
	if strings.TrimSpace(store.Dir) == "" {
		return nil
	}
	archiveInfo, err := os.Stat(store.Dir)
	if err != nil {
		return nil //nolint:nilerr // no readable archive directory means no archived file to protect
	}
	if inDir(outputPath, archiveInfo) {
		return &ArchivedOutputError{Path: outputPath}
	}
	if resolved, err := filepath.EvalSymlinks(outputPath); err == nil && inDir(resolved, archiveInfo) {
		return &ArchivedOutputError{Path: outputPath}
	}
	return nil
}

// inDir reports whether one of path's parent directories is dir. It compares
// files rather than names, so a differently cased path on a case-insensitive
// file system, or a symlinked parent, still matches.
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

// refuseDirOutput returns an OutputIsDirError when path is a directory or a
// symlink to one.
func refuseDirOutput(path string) error {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return &OutputIsDirError{Path: path}
	}
	return nil
}
