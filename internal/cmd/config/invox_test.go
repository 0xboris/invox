package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/clitest"
)

func TestConfigOpensConfigFile(t *testing.T) {
	x := clitest.New(t)

	configHome := filepath.Join(t.TempDir(), "config-home")
	x.Setenv("XDG_CONFIG_HOME", configHome)
	openedPath := x.ExpectEditor(nil)

	exitCode, stdout, stderr := x.Run([]string{"config"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}

	wantPath := filepath.Join(configHome, "invox", "config.yaml")
	if *openedPath != wantPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}
	source, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("ReadFile(config.yaml) returned error: %v", err)
	}
	for _, want := range []string{
		"# Invox user configuration.",
		"#   paths.customers",
		"#   paths.template",
		"#   numbering.pattern",
		"#   numbering.start",
		"#   archive.dir",
	} {
		if !strings.Contains(string(source), want) {
			t.Fatalf("config template %q does not contain %q", string(source), want)
		}
	}
	if want := "Opened " + wantPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestConfigOpensMalformedConfigFileForEditing(t *testing.T) {
	x := clitest.New(t)

	configPath := x.WriteConfig(" numbering:\n  pattern: '{customer_id}-{counter:03}'\n")
	openedPath := x.ExpectEditor(nil)

	exitCode, stdout, stderr := x.Run([]string{"config"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if *openedPath != configPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, configPath)
	}
	if want := "Opened " + configPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
}

func TestConfigHelpShowsUsage(t *testing.T) {
	x := clitest.New(t)

	exitCode, stdout, stderr := x.Run([]string{"config", "-h"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0", exitCode)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
	for _, want := range []string{
		"Open config.yaml in your editor.",
		"invox help config",
		"Formatting:",
		"Top-level keys must start at column 1 with no leading spaces.",
		"Supported settings:",
		"paths.customers",
		"paths.template",
		"numbering.pattern",
		"numbering.start",
		"archive.dir",
		"email.subject",
		"email.body",
		"email template placeholders:",
		"{email_greeting}",
		"{contact_person}",
		"{outstanding_amount}",
		"Customer overrides:",
		"customers.<CUSTOMER_ID>.numbering.start",
		"Support file precedence:",
		"Template:",
		"# archive:",
		"invox config",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("stdout %q does not contain %q", stdout, want)
		}
	}
}

// An editor that is not installed is named with its setting, and the hint
// says which variables choose another, instead of the exec error.
func TestConfigReportsAnEditorThatIsNotInstalled(t *testing.T) {
	x := clitest.New(t)

	configHome := filepath.Join(t.TempDir(), "config-home")
	x.Setenv("XDG_CONFIG_HOME", configHome)
	x.ExpectEditor(fmt.Errorf("exec: %w", run.ErrNotFound))

	exitCode, stdout, stderr := x.Run([]string{"config"})

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
