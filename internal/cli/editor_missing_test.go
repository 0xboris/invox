package cli

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
)

// An editor that is not installed is named with its setting, and the hint
// says which variables choose another, instead of the exec error.
func TestConfigReportsAnEditorThatIsNotInstalled(t *testing.T) {
	configHome := filepath.Join(t.TempDir(), "config-home")
	t.Setenv("XDG_CONFIG_HOME", configHome)
	f, stub := testFactory(t)
	expectEditor(f, stub, fmt.Errorf("exec: %w", run.ErrNotFound))

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{"config"})

	configPath := filepath.Join(configHome, "invox", "config.yaml")
	want := "error: failed to open " + configPath + ": editor \"vi\" not found in PATH\n" +
		"Set VISUAL or EDITOR to an installed editor, then rerun this command.\n"
	if exitCode != 1 {
		t.Errorf("exit code = %d, want 1", exitCode)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}
