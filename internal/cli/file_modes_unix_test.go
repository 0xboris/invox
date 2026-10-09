//go:build !windows

package cli_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

// setUmask sets the process umask for the rest of the test. invox reads the
// umask once at start, so this only steers modes it does not set itself.
func setUmask(t *testing.T, umask int) {
	t.Helper()
	old := syscall.Umask(umask)
	t.Cleanup(func() { syscall.Umask(old) })
}

// processUmask returns the umask the test process started with, which invox
// clears from the modes of files and directories that are not private.
func processUmask(t *testing.T) fs.FileMode {
	t.Helper()
	umask := syscall.Umask(0)
	syscall.Umask(umask)
	return fs.FileMode(umask)
}

func assertFileMode(t *testing.T, path string, want fs.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("%s mode = %#o, want %#o", path, got, want)
	}
}

func TestEmailWritesPublicDraft(t *testing.T) {
	x := clitest.New(t)

	fx := testfixture.WriteBuiltContext(t)
	x.ExpectOpener(nil)
	outputPath := filepath.Join(t.TempDir(), "draft.eml")

	exitCode, stdout, stderr := x.Run([]string{
		"email", fx.Invoice, "-o", outputPath, "-c", fx.Customers, "-u", fx.Issuer,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Opened email draft for CUST-001 (CUST-001-001) to office@appsters.example\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != outputPath+"\n" {
		t.Fatalf("stdout = %q, want %q", stdout, outputPath+"\n")
	}
	assertFileMode(t, outputPath, 0o644&^processUmask(t))
}
