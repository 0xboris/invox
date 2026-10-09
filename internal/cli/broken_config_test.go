package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

// brokenConfigSource is a config.yaml with a YAML syntax error on line 3.
const brokenConfigSource = "paths:\n  customers: customers.yaml\n  issuer: issuer.yaml: x\n"

// setupBrokenConfig writes a broken config.yaml into a fresh XDG_CONFIG_HOME
// and moves into an empty working directory, so nothing is found locally.
func setupBrokenConfig(t *testing.T, x *clitest.Invox) string {
	t.Helper()

	configPath := x.WriteConfig(brokenConfigSource)
	x.Chdir(t.TempDir())
	return configPath
}

func TestBrokenConfigDoesNotBlockCommandsThatDoNotNeedIt(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStdout string
	}{
		{name: "root help flag", args: []string{"--help"}, wantStdout: "Usage:"},
		{name: "help", args: []string{"help"}, wantStdout: "Usage:"},
		{name: "help config", args: []string{"help", "config"}, wantStdout: "config.yaml"},
		{name: "command help flag", args: []string{"validate", "--help"}, wantStdout: "Usage:"},
		{name: "completion", args: []string{"completion", "zsh"}, wantStdout: "#compdef invox"},
		{name: "version", args: []string{"version"}, wantStdout: "invox version "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := clitest.New(t)

			setupBrokenConfig(t, x)

			exitCode, stdout, stderr := x.Run(tt.args)
			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
			}
			if !strings.Contains(stdout, tt.wantStdout) {
				t.Fatalf("stdout = %q, want it to contain %q", stdout, tt.wantStdout)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestBrokenConfigInitSucceeds(t *testing.T) {
	x := clitest.New(t)

	configPath := setupBrokenConfig(t, x)

	exitCode, stdout, stderr := x.Run([]string{"init"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	for _, want := range []string{"Initialized ", "exists config.yaml", "created customers.yaml"} {
		if !strings.Contains(stderr, want) {
			t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
		}
	}

	source, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(config.yaml) returned error: %v", err)
	}
	if string(source) != brokenConfigSource {
		t.Fatalf("init changed config.yaml to %q", source)
	}
}

func TestBrokenConfigConfigOpensTheFile(t *testing.T) {
	x := clitest.New(t)

	configPath := setupBrokenConfig(t, x)
	openedPath := x.ExpectEditor(nil)

	exitCode, stdout, stderr := x.Run([]string{"config"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.HasPrefix(stderr, "Opened ") {
		t.Fatalf("stderr = %q, want it to start with %q", stderr, "Opened ")
	}
	if *openedPath != configPath {
		t.Fatalf("openedPath = %q, want %q", *openedPath, configPath)
	}
}

func TestBrokenConfigCustomerListWithExplicitPath(t *testing.T) {
	x := clitest.New(t)

	setupBrokenConfig(t, x)
	if err := os.WriteFile("customers.yaml", []byte("CUST-001:\n  legal_company_name: Appsters GmbH\n  status: active\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{"customer", "list", "-c", "./customers.yaml"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if stdout != "CUST-001\tAppsters GmbH\tactive\n" {
		t.Fatalf("stdout = %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("stderr = %q, want empty", stderr)
	}
}

func TestBrokenConfigUsageErrorsExitTwo(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantStderr string
	}{
		{name: "unknown flag", args: []string{"new", "--bogus"}, wantStderr: "unknown flag: --bogus"},
		{name: "missing input", args: []string{"validate"}, wantStderr: "missing required input: INVOICE.yaml or -i, --input"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := clitest.New(t)

			setupBrokenConfig(t, x)

			exitCode, stdout, stderr := x.Run(tt.args)
			if exitCode != 2 {
				t.Fatalf("exitCode = %d, want 2, stderr=%q", exitCode, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if !strings.Contains(stderr, tt.wantStderr) {
				t.Fatalf("stderr = %q, want it to contain %q", stderr, tt.wantStderr)
			}
			if strings.Contains(stderr, "yaml:") {
				t.Fatalf("stderr = %q, want no config error", stderr)
			}
		})
	}
}

func TestBrokenConfigNeededReportsFileLineAndHint(t *testing.T) {
	x := clitest.New(t)

	configPath := setupBrokenConfig(t, x)

	exitCode, stdout, stderr := x.Run([]string{"validate", "-i", filepath.Join("missing", "x.yaml")})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.HasPrefix(stderr, "error: "+configPath+": yaml: line 3: ") {
		t.Fatalf("stderr = %q, want it to start with %q", stderr, "error: "+configPath+": yaml: line 3: ")
	}
	if !strings.HasSuffix(stderr, "\nRun 'invox config' to open and fix the config file.\n") {
		t.Fatalf("stderr = %q, want the invox config hint", stderr)
	}
}
