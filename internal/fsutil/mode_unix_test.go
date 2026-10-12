//go:build unix

package fsutil

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/0xboris/invox/internal/testfixture"
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

// setUmask sets the umask fsutil read at init for the rest of the test.
func setUmask(t *testing.T, mask fs.FileMode) {
	t.Helper()
	old := umask
	umask = mask
	t.Cleanup(func() { umask = old })
}

var writers = []struct {
	name  string
	write func(t *testing.T, path string, data []byte, perm Perm) error
}{
	{name: "WriteFile", write: func(_ *testing.T, path string, data []byte, perm Perm) error {
		return WriteFile(path, data, perm)
	}},
	{name: "WriteNewFile", write: func(_ *testing.T, path string, data []byte, perm Perm) error {
		return WriteNewFile(path, data, perm)
	}},
	{name: "WriteNewFile without hard links", write: func(t *testing.T, path string, data []byte, perm Perm) error {
		withoutHardLinks(t)
		return WriteNewFile(path, data, perm)
	}},
}

func TestNewFileModes(t *testing.T) {
	tests := []struct {
		name      string
		umask     fs.FileMode
		perm      Perm
		file, dir fs.FileMode
	}{
		{name: "private", umask: 0o022, perm: Private, file: 0o600, dir: 0o700},
		{name: "private", umask: 0o077, perm: Private, file: 0o600, dir: 0o700},
		{name: "private", umask: 0o777, perm: Private, file: 0o600, dir: 0o700},
		{name: "public", umask: 0o022, perm: Public, file: 0o644, dir: 0o755},
		{name: "public", umask: 0o077, perm: Public, file: 0o600, dir: 0o700},
		{name: "custom", umask: 0o022, perm: Perm{File: 0o640, Dir: 0o750}, file: 0o640, dir: 0o750},
		{name: "custom", umask: 0o077, perm: Perm{File: 0o640, Dir: 0o750}, file: 0o600, dir: 0o700},
	}
	for _, w := range writers {
		for _, tc := range tests {
			t.Run(fmt.Sprintf("%s %s umask %#o", w.name, tc.name, tc.umask), func(t *testing.T) {
				setUmask(t, tc.umask)
				root := t.TempDir()
				if err := os.Chmod(root, 0o711); err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(root, "outer", "inner", "file.yaml")
				if err := w.write(t, path, []byte("data"), tc.perm); err != nil {
					t.Fatalf("write returned error: %v", err)
				}
				assertMode(t, path, tc.file)
				assertMode(t, filepath.Join(root, "outer", "inner"), tc.dir)
				assertMode(t, filepath.Join(root, "outer"), tc.dir)
				assertMode(t, root, 0o711)
			})
		}
	}
}

func TestMkdirAllParentModes(t *testing.T) {
	tests := []struct {
		name          string
		umask         fs.FileMode
		perm          Perm
		leaf, parents fs.FileMode
	}{
		{name: "private", umask: 0o022, perm: Private, leaf: 0o700, parents: 0o755},
		{name: "private", umask: 0o077, perm: Private, leaf: 0o700, parents: 0o700},
		{name: "public", umask: 0o022, perm: Public, leaf: 0o755, parents: 0o755},
		{name: "public", umask: 0o027, perm: Public, leaf: 0o750, parents: 0o750},
		{name: "custom 0711", umask: 0o022, perm: Perm{Dir: 0o711}, leaf: 0o711, parents: 0o711},
		{name: "custom 0700", umask: 0o022, perm: Perm{Dir: 0o700}, leaf: 0o700, parents: 0o700},
	}
	for _, tc := range tests {
		t.Run(fmt.Sprintf("%s umask %#o", tc.name, tc.umask), func(t *testing.T) {
			setUmask(t, tc.umask)
			root := t.TempDir()
			leaf := filepath.Join(root, "a", "b", "leaf")
			if err := MkdirAll(leaf, tc.perm); err != nil {
				t.Fatalf("MkdirAll returned error: %v", err)
			}
			assertMode(t, leaf, tc.leaf)
			assertMode(t, filepath.Join(root, "a", "b"), tc.parents)
			assertMode(t, filepath.Join(root, "a"), tc.parents)
		})
	}
}

func TestReadUmask(t *testing.T) {
	old := syscall.Umask(0o027)
	t.Cleanup(func() { syscall.Umask(old) })
	if got := readUmask(); got != 0o027 {
		t.Fatalf("readUmask() = %#o, want 0o027", got)
	}
	if got := syscall.Umask(0o027); got != 0o027 {
		t.Fatalf("readUmask left the umask at %#o, want 0o027", got)
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
			testfixture.WriteFile(t, path, "old\n")
			if err := os.Chmod(path, tc.existing); err != nil {
				t.Fatal(err)
			}
			if err := WriteFile(path, []byte("new\n"), tc.perm); err != nil {
				t.Fatalf("WriteFile returned error: %v", err)
			}
			assertMode(t, path, tc.existing)
			if got := testfixture.ReadFile(t, path); got != "new\n" {
				t.Fatalf("content = %q, want %q", got, "new\n")
			}
		})
	}
}

func TestWriteFileThroughSymlinkKeepsTargetMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "invoice.yaml")
	testfixture.WriteFile(t, target, "old\n")
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
	if got := testfixture.ReadFile(t, target); got != "new\n" {
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
		{name: "MkdirAll", write: func(path string, _ []byte, perm Perm) error { return MkdirAll(filepath.Dir(path), perm) }},
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
