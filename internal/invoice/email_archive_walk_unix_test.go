//go:build unix

package invoice

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// A directory the archive walk cannot read fails the lookup, and the error
// names it below the resolved archive.dir, as archive list's does, even when
// archive.dir is reached through a symlinked ancestor.
func TestResolveEmailDraftPathsReportsUnreadableDirBelowResolvedArchiveDir(t *testing.T) {
	t.Parallel()
	if os.Geteuid() == 0 {
		t.Skip("root reads directories without permission")
	}

	realParent := t.TempDir()
	linkParent := filepath.Join(t.TempDir(), "parent-link")
	if err := os.Symlink(realParent, linkParent); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	archiveDir := filepath.Join(linkParent, "archive")
	h := writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")

	locked := filepath.Join(realParent, "archive", "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatalf("MkdirAll(locked) returned error: %v", err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatalf("Chmod(locked) returned error: %v", err)
	}
	t.Cleanup(func() { os.Chmod(locked, 0o755) })

	resolvedLocked, err := filepath.EvalSymlinks(filepath.Join(archiveDir, "locked"))
	if err != nil {
		t.Fatalf("EvalSymlinks returned error: %v", err)
	}

	_, err = h.ResolveEmailDraftPaths(filepath.Join(t.TempDir(), "BL00210001.pdf"), "", "")
	var pathErr *fs.PathError
	if !errors.As(err, &pathErr) || !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("ResolveEmailDraftPaths error = %v, want a permission error", err)
	}
	if pathErr.Path != resolvedLocked {
		t.Fatalf("error names %q, want the resolved %q", pathErr.Path, resolvedLocked)
	}
}
