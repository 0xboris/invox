//go:build !windows

package fsutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func assertMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %#o, want %#o", path, got, want)
	}
}

type modeCase struct {
	name  string
	write func(path string, data []byte, perm Perm) error
	perm  Perm
	file  fs.FileMode
	dir   fs.FileMode
}

var modeCases = []modeCase{
	{name: "WriteFile private", write: WriteFile, perm: Private, file: 0o600, dir: 0o700},
	{name: "WriteFile public", write: WriteFile, perm: Public, file: 0o644, dir: 0o755},
	{name: "WriteNewFile private", write: WriteNewFile, perm: Private, file: 0o600, dir: 0o700},
	{name: "WriteNewFile public", write: WriteNewFile, perm: Public, file: 0o644, dir: 0o755},
	{name: "WriteFile custom", write: WriteFile, perm: Perm{File: 0o640, Dir: 0o750}, file: 0o640, dir: 0o750},
}

// runModeCases checks the mode of a new file and of each directory created
// for it, under whatever umask is in effect.
func runModeCases(t *testing.T) {
	t.Helper()
	for _, tc := range modeCases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.Chmod(root, 0o711); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "outer", "inner", "file.yaml")
			if err := tc.write(path, []byte("data"), tc.perm); err != nil {
				t.Fatalf("write returned error: %v", err)
			}
			assertMode(t, path, tc.file)
			assertMode(t, filepath.Join(root, "outer", "inner"), tc.dir)
			assertMode(t, filepath.Join(root, "outer"), tc.dir)
			assertMode(t, root, 0o711)
		})
	}
}

func TestNewFileModes(t *testing.T) {
	runModeCases(t)
}

func TestNewFileModesIgnoreUmask(t *testing.T) {
	for _, umask := range []int{0o077, 0o022} {
		t.Run(fmt.Sprintf("umask %#o", umask), func(t *testing.T) {
			old := syscall.Umask(umask)
			t.Cleanup(func() { syscall.Umask(old) })
			runModeCases(t)
		})
	}
}

func TestWriteFileKeepsExistingMode(t *testing.T) {
	tests := []struct {
		name     string
		existing fs.FileMode
		perm     Perm
	}{
		{name: "private file written as public", existing: 0o600, perm: Public},
		{name: "public file written as private", existing: 0o644, perm: Private},
		{name: "group-readable file", existing: 0o640, perm: Public},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "invoice.yaml")
			writeTestFile(t, path, "old\n")
			if err := os.Chmod(path, tc.existing); err != nil {
				t.Fatal(err)
			}
			if err := WriteFile(path, []byte("new\n"), tc.perm); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			assertMode(t, path, tc.existing)
			if got := readFile(t, path); got != "new\n" {
				t.Fatalf("content = %q, want %q", got, "new\n")
			}
		})
	}
}

func TestWriteFileThroughSymlinkKeepsTargetMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "invoice.yaml")
	writeTestFile(t, target, "old\n")
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "link.yaml")
	symlinkOrSkip(t, "invoice.yaml", link)

	if err := WriteFile(link, []byte("new\n"), Public); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}
	assertSymlinkTo(t, link, "invoice.yaml")
	assertMode(t, target, 0o600)
	if got := readFile(t, target); got != "new\n" {
		t.Fatalf("content = %q, want %q", got, "new\n")
	}
}

func TestExistingDirectoriesKeepTheirMode(t *testing.T) {
	tests := []struct {
		name  string
		write func(path string, data []byte, perm Perm) error
	}{
		{name: "WriteFile", write: WriteFile},
		{name: "WriteNewFile", write: WriteNewFile},
		{name: "MkdirAll", write: func(path string, _ []byte, perm Perm) error { return MkdirAll(filepath.Dir(path), perm.Dir) }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			existing := filepath.Join(root, "existing")
			if err := os.Mkdir(existing, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(existing, 0o775); err != nil {
				t.Fatal(err)
			}
			if err := tc.write(filepath.Join(existing, "new", "file.yaml"), []byte("data"), Private); err != nil {
				t.Fatalf("write returned error: %v", err)
			}
			assertMode(t, existing, 0o775)
			assertMode(t, filepath.Join(existing, "new"), 0o700)
		})
	}
}
