package fsutil

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/0xboris/invox/internal/testfixture"
)

func symlinkOrSkip(t *testing.T, target, link string) {
	t.Helper()
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
}

func assertNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir %s: %v", dir, err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".invox-") {
			t.Fatalf("temporary file %s left in %s", entry.Name(), dir)
		}
	}
}

func assertSymlinkTo(t *testing.T, link, want string) {
	t.Helper()
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("lstat %s: %v", link, err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("%s is no longer a symlink (mode %v)", link, info.Mode())
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatalf("readlink %s: %v", link, err)
	}
	if got != want {
		t.Fatalf("%s points to %q, want %q", link, got, want)
	}
}

func TestWriteFileCreatesAndReplaces(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "invoice.yaml")
	if err := WriteFile(path, []byte("first\n"), Public); err != nil {
		t.Fatalf("WriteFile(new) returned error: %v", err)
	}
	if got := testfixture.ReadFile(t, path); got != "first\n" {
		t.Fatalf("content = %q, want %q", got, "first\n")
	}
	if err := WriteFile(path, []byte("second\n"), Public); err != nil {
		t.Fatalf("WriteFile(existing) returned error: %v", err)
	}
	if got := testfixture.ReadFile(t, path); got != "second\n" {
		t.Fatalf("content = %q, want %q", got, "second\n")
	}
	assertNoTempFiles(t, filepath.Dir(path))
}

func TestWriteFileWritesThroughSymlinks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup returns the path to write, the link to check, what the link
		// must still point to, and the file that must receive the data.
		setup func(t *testing.T, dir string) (path, link, linkTarget, target string)
	}{
		{
			name: "absolute link",
			setup: func(t *testing.T, dir string) (string, string, string, string) {
				target := filepath.Join(dir, "real", "invoice.yaml")
				if err := os.Mkdir(filepath.Dir(target), 0o755); err != nil {
					t.Fatal(err)
				}
				testfixture.WriteFile(t, target, "old\n")
				link := filepath.Join(dir, "link.yaml")
				symlinkOrSkip(t, target, link)
				return link, link, target, target
			},
		},
		{
			name: "relative link",
			setup: func(t *testing.T, dir string) (string, string, string, string) {
				target := filepath.Join(dir, "real", "invoice.yaml")
				if err := os.Mkdir(filepath.Dir(target), 0o755); err != nil {
					t.Fatal(err)
				}
				testfixture.WriteFile(t, target, "old\n")
				link := filepath.Join(dir, "link.yaml")
				relative := filepath.Join("real", "invoice.yaml")
				symlinkOrSkip(t, relative, link)
				return link, link, relative, target
			},
		},
		{
			name: "dangling relative link",
			setup: func(t *testing.T, dir string) (string, string, string, string) {
				link := filepath.Join(dir, "link.yaml")
				relative := filepath.Join("missing", "invoice.yaml")
				symlinkOrSkip(t, relative, link)
				return link, link, relative, filepath.Join(dir, "missing", "invoice.yaml")
			},
		},
		{
			name: "chain of links",
			setup: func(t *testing.T, dir string) (string, string, string, string) {
				target := filepath.Join(dir, "invoice.yaml")
				testfixture.WriteFile(t, target, "old\n")
				middle := filepath.Join(dir, "middle.yaml")
				symlinkOrSkip(t, "invoice.yaml", middle)
				link := filepath.Join(dir, "link.yaml")
				symlinkOrSkip(t, "middle.yaml", link)
				return link, link, "middle.yaml", target
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path, link, linkTarget, target := tc.setup(t, dir)
			if err := WriteFile(path, []byte("new\n"), Public); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			assertSymlinkTo(t, link, linkTarget)
			if got := testfixture.ReadFile(t, target); got != "new\n" {
				t.Fatalf("target content = %q, want %q", got, "new\n")
			}
			assertNoTempFiles(t, dir)
			assertNoTempFiles(t, filepath.Dir(target))
		})
	}
}

func TestWriteFileRejectsSymlinkLoop(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	symlinkOrSkip(t, "b", a)
	symlinkOrSkip(t, "a", b)
	err := WriteFile(a, []byte("data"), Public)
	if !errors.Is(err, syscall.ELOOP) {
		t.Fatalf("WriteFile(loop) error = %v, want syscall.ELOOP", err)
	}
	assertSymlinkTo(t, a, "b")
	assertNoTempFiles(t, dir)
}

func TestWriteNewFileCreates(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "new", "draft.eml")
	if err := WriteNewFile(path, []byte("hello\n"), Public); err != nil {
		t.Fatalf("WriteNewFile returned error: %v", err)
	}
	if got := testfixture.ReadFile(t, path); got != "hello\n" {
		t.Fatalf("content = %q, want %q", got, "hello\n")
	}
	assertNoTempFiles(t, filepath.Dir(path))
}

func TestWriteNewFileNeverClobbers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup creates the existing entry at path and returns what must
		// still be readable through path afterwards, or "" for a dangling link.
		setup func(t *testing.T, dir, path string) string
	}{
		{
			name: "existing file",
			setup: func(t *testing.T, dir, path string) string {
				testfixture.WriteFile(t, path, "keep\n")
				return "keep\n"
			},
		},
		{
			name: "symlink to a file",
			setup: func(t *testing.T, dir, path string) string {
				testfixture.WriteFile(t, filepath.Join(dir, "other.yaml"), "other\n")
				symlinkOrSkip(t, "other.yaml", path)
				return "other\n"
			},
		},
		{
			name: "dangling symlink",
			setup: func(t *testing.T, dir, path string) string {
				symlinkOrSkip(t, "missing.yaml", path)
				return ""
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			path := filepath.Join(dir, "invoice.yaml")
			want := tc.setup(t, dir, path)
			before, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}

			err = WriteNewFile(path, []byte("clobbered\n"), Public)
			if !errors.Is(err, fs.ErrExist) {
				t.Fatalf("WriteNewFile error = %v, want fs.ErrExist", err)
			}
			after, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if after.Mode().Type() != before.Mode().Type() {
				t.Fatalf("entry type changed from %v to %v", before.Mode().Type(), after.Mode().Type())
			}
			if want != "" {
				if got := testfixture.ReadFile(t, path); got != want {
					t.Fatalf("content = %q, want %q", got, want)
				}
			} else if _, err := os.Stat(filepath.Join(dir, "missing.yaml")); !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("dangling link target was created: %v", err)
			}
			assertNoTempFiles(t, dir)
		})
	}
}

func TestWriteNewFileLosesRaceWithoutClobbering(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.yaml")
	testHookBeforeCommit = func(target string) {
		testfixture.WriteFile(t, target, "racer\n")
	}
	t.Cleanup(func() { testHookBeforeCommit = nil })

	err := WriteNewFile(path, []byte("loser\n"), Public)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("WriteNewFile error = %v, want fs.ErrExist", err)
	}
	if got := testfixture.ReadFile(t, path); got != "racer\n" {
		t.Fatalf("content = %q, want the racing writer's %q", got, "racer\n")
	}
	assertNoTempFiles(t, dir)
}

// withoutHardLinks makes WriteNewFile act as on a file system that has no
// hard links, for the rest of the test.
func withoutHardLinks(t *testing.T) {
	t.Helper()
	link = func(oldname, newname string) error {
		return &os.LinkError{Op: "link", Old: oldname, New: newname, Err: syscall.EPERM}
	}
	t.Cleanup(func() { link = os.Link })
}

func TestWriteNewFileWithoutHardLinks(t *testing.T) {
	withoutHardLinks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "draft.eml")

	if err := WriteNewFile(path, []byte("hello\n"), Public); err != nil {
		t.Fatalf("WriteNewFile returned error: %v", err)
	}
	if got := testfixture.ReadFile(t, path); got != "hello\n" {
		t.Fatalf("content = %q, want %q", got, "hello\n")
	}
	assertNoTempFiles(t, dir)

	err := WriteNewFile(path, []byte("clobbered\n"), Public)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("WriteNewFile(existing) error = %v, want fs.ErrExist", err)
	}
	if got := testfixture.ReadFile(t, path); got != "hello\n" {
		t.Fatalf("content = %q, want %q", got, "hello\n")
	}
	assertNoTempFiles(t, dir)
}

func TestWriteNewFileWithoutHardLinksLosesRaceWithoutClobbering(t *testing.T) {
	withoutHardLinks(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.yaml")
	testHookBeforeCommit = func(target string) {
		testfixture.WriteFile(t, target, "racer\n")
	}
	t.Cleanup(func() { testHookBeforeCommit = nil })

	err := WriteNewFile(path, []byte("loser\n"), Public)
	if !errors.Is(err, fs.ErrExist) {
		t.Fatalf("WriteNewFile error = %v, want fs.ErrExist", err)
	}
	if got := testfixture.ReadFile(t, path); got != "racer\n" {
		t.Fatalf("content = %q, want the racing writer's %q", got, "racer\n")
	}
	assertNoTempFiles(t, dir)
}

func TestWriteFileReplacesRacingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invoice.yaml")
	testHookBeforeCommit = func(target string) {
		testfixture.WriteFile(t, target, "racer\n")
	}
	t.Cleanup(func() { testHookBeforeCommit = nil })

	if err := WriteFile(path, []byte("winner\n"), Public); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	if got := testfixture.ReadFile(t, path); got != "winner\n" {
		t.Fatalf("content = %q, want %q", got, "winner\n")
	}
	assertNoTempFiles(t, dir)
}

func TestMkdirAllRejectsFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	file := filepath.Join(dir, "file")
	testfixture.WriteFile(t, file, "")
	for _, target := range []string{file, filepath.Join(file, "child")} {
		if err := MkdirAll(target, Private); err == nil {
			t.Fatalf("MkdirAll(%s) returned nil, want an error", target)
		}
	}
}
