package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
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
	exitCode := Main(args, cmdutil.NewFactory(ios, run.Exec{}, e))
	return exitCode, ios.Out.(*bytes.Buffer).String(), ios.ErrOut.(*bytes.Buffer).String()
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
}

func TestNewUsesTheInjectedWorkingDirectory(t *testing.T) {
	processDir := t.TempDir()
	chdirForTest(t, processDir)

	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
	for name, source := range map[string]string{"customers.yaml": customersPath, "issuer.yaml": issuerPath, "defaults.yaml": defaultsPath} {
		content, err := os.ReadFile(source)
		if err != nil {
			t.Fatalf("ReadFile returned error: %v", err)
		}
		writeTestFile(t, filepath.Join(workDir, name), string(content))
	}
	writeTestFile(t, filepath.Join(root, "config", "invox", "config.yaml"), "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")

	e := testEnv(root, func() (string, error) { return workDir, nil })
	exitCode, stdout, stderr := runWithEnv(e, "new", "CUST-001", "-c", "customers.yaml", "-u", "issuer.yaml", "-s", "defaults.yaml")
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
	writeTestFile(t, filepath.Join(root, "config", "invox", "template.tex"), "\\documentclass{article}\n")

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
