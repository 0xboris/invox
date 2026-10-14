//go:build unix

package invoice

import (
	"os"
	"path/filepath"
	"testing"
)

// Private modes are exact whatever the umask, so this holds under 022 and 077
// alike; Public files follow the umask fsutil reads at start-up.
func TestCopyLegacyFilesUsesPrivateModes(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	legacyDir := filepath.Join(configHome, "invoice-tool")
	writeFile(t, filepath.Join(legacyDir, "customers.yaml"), "customers\n")
	writeFile(t, filepath.Join(legacyDir, "invoice_defaults.yaml"), "defaults\n")
	h := NewHost(HostInputs{GOOS: "linux", Home: t.TempDir(), XDGConfigHome: configHome})

	if _, err := h.CopyLegacyFiles(); err != nil {
		t.Fatalf("CopyLegacyFiles returned error: %v", err)
	}
	invoxDir := filepath.Join(configHome, "invox")
	for path, want := range map[string]os.FileMode{
		invoxDir: 0o700,
		filepath.Join(invoxDir, "customers.yaml"): 0o600,
	} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		if got := info.Mode().Perm(); got != want {
			t.Errorf("%s mode = %#o, want %#o", path, got, want)
		}
	}
}
