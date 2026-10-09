package cli_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/testfixture"
)

// testEnv returns an Env whose user directories are under root and whose
// working directory comes from getwd, independent of the test process.
func testEnv(root string, getwd func() (string, error)) env.Env {
	vars := map[string]string{
		"XDG_CONFIG_HOME": filepath.Join(root, "config"),
		"XDG_DATA_HOME":   filepath.Join(root, "data"),
		"APPDATA":         filepath.Join(root, "data"),
	}
	return env.Env{
		GOOS:    runtime.GOOS,
		Getenv:  func(key string) string { return vars[key] },
		HomeDir: func() (string, error) { return filepath.Join(root, "home"), nil },
		Getwd:   getwd,
		Now:     func() time.Time { return time.Date(2026, 3, 6, 12, 0, 0, 0, time.Local) },
	}
}

func runWithEnv(e env.Env, args ...string) (int, string, string) {
	ios, _, _, _ := iostreams.Test()
	exitCode := cli.Main(args, factory.New(ios, run.Exec{}, e))
	return exitCode, ios.Out.(*bytes.Buffer).String(), ios.ErrOut.(*bytes.Buffer).String()
}

func TestNewUsesTheInjectedWorkingDirectory(t *testing.T) {
	processDir := t.TempDir()
	t.Chdir(processDir)

	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	draft := testfixture.WriteDraft(t)
	for name, source := range map[string]string{"customers.yaml": draft.Customers, "issuer.yaml": draft.Issuer, "defaults.yaml": draft.Defaults} {
		content, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("ReadFile returned error: %v", err)
		}
		testfixture.WriteFile(t, filepath.Join(workDir, name), string(content))
	}
	testfixture.WriteFile(t, filepath.Join(root, "config", "invox", "config.yaml"), "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")

	e := testEnv(root, func() (string, error) { return workDir, nil })
	exitCode, stdout, stderr := runWithEnv(e, "new", "CUST-001", "-c", "customers.yaml", "-u", "issuer.yaml", "--defaults", "defaults.yaml")
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-002.yaml for CUST-001 (CUST-001-002)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if want := "CUST-001-002.yaml\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if _, err := os.Stat(filepath.Join(workDir, "CUST-001-002.yaml")); err != nil {
		t.Fatalf("invoice not created in the injected working directory: %v", err)
	}
	if entries, _ := os.ReadDir(processDir); len(entries) != 0 {
		t.Fatalf("process working directory has %d entries, want none", len(entries))
	}
}

func TestTemplateListDoesNotNeedTheWorkingDirectory(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	testfixture.WriteFile(t, filepath.Join(root, "config", "invox", "template.tex"), "\\documentclass{article}\n")

	e := testEnv(root, func() (string, error) { return "", errors.New("getwd: no such file or directory") })
	exitCode, stdout, stderr := runWithEnv(e, "template", "list", "--names")
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "template.tex\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}
