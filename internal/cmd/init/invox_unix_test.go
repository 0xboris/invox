//go:build !windows

package initcmd_test

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

func TestInitCreatesPrivateConfigDirAndSecrets(t *testing.T) {
	x := clitest.New(t)

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	x.Setenv("XDG_CONFIG_HOME", configHome)
	umask := testfixture.ProcessUmask(t)

	exitCode, stdout, stderr := x.Run([]string{"init"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Initialized " + configDir + "\ncreated config.yaml\ncreated customers.yaml\ncreated issuer.yaml\ncreated invoice_defaults.yaml\ncreated template.tex\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	testfixture.AssertFileMode(t, configDir, 0o700)
	for name, want := range map[string]fs.FileMode{
		"issuer.yaml":           0o600,
		"customers.yaml":        0o600,
		"config.yaml":           0o644 &^ umask,
		"invoice_defaults.yaml": 0o644 &^ umask,
		"template.tex":          0o644 &^ umask,
	} {
		testfixture.AssertFileMode(t, filepath.Join(configDir, name), want)
	}
}

func TestInitCreatesMissingConfigParentsPublic(t *testing.T) {
	x := clitest.New(t)

	root := t.TempDir()
	configHome := filepath.Join(root, "missing", "cfg")
	configDir := filepath.Join(configHome, "invox")
	x.Setenv("XDG_CONFIG_HOME", configHome)
	umask := testfixture.ProcessUmask(t)

	exitCode, stdout, stderr := x.Run([]string{"init"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Initialized " + configDir + "\ncreated config.yaml\ncreated customers.yaml\ncreated issuer.yaml\ncreated invoice_defaults.yaml\ncreated template.tex\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	testfixture.AssertFileMode(t, filepath.Join(root, "missing"), 0o755&^umask)
	testfixture.AssertFileMode(t, configHome, 0o755&^umask)
	testfixture.AssertFileMode(t, configDir, 0o700)
}
