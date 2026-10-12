package initcmd_test

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestInitCreatesStarterFilesAndAllowsNewWithGlobalDefaults(t *testing.T) {
	x := clitest.New(t)

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	x.Setenv("XDG_CONFIG_HOME", configHome)

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

	for _, file := range []struct {
		name string
		want string
	}{
		{name: "config.yaml", want: "# Invox user configuration."},
		{name: "customers.yaml", want: "CUST-001:"},
		{name: "issuer.yaml", want: "vat_label: VAT"},
		{name: "invoice_defaults.yaml", want: "status: draft"},
		{name: "template.tex", want: "\\documentclass"},
	} {
		path := filepath.Join(configDir, file.name)
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s) returned error: %v", path, err)
		}
		if !strings.Contains(string(source), file.want) {
			t.Fatalf("%s does not contain %q:\n%s", path, file.want, string(source))
		}
	}

	workDir := t.TempDir()
	x.Chdir(workDir)

	exitCode, stdout, stderr = x.Run([]string{"new", "CUST-001"})
	if exitCode != 0 {
		t.Fatalf("new exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Created CUST-001-001.yaml for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "CUST-001-001.yaml\n" {
		t.Fatalf("new stdout = %q, want %q", stdout, "CUST-001-001.yaml\n")
	}
	if _, err := os.Stat(filepath.Join(workDir, "CUST-001-001.yaml")); err != nil {
		t.Fatalf("starter config should allow invoice creation: %v", err)
	}
}

func TestInitDoesNotOverwriteExistingSupportFiles(t *testing.T) {
	x := clitest.New(t)

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	x.Setenv("XDG_CONFIG_HOME", configHome)
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("MkdirAll(configDir) returned error: %v", err)
	}

	customCustomers := "CUSTOM:\n  name: Custom Customer\n"
	customersPath := filepath.Join(configDir, "customers.yaml")
	if err := os.WriteFile(customersPath, []byte(customCustomers), 0o644); err != nil {
		t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
	}

	exitCode, stdout, stderr := x.Run([]string{"init"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "Initialized " + configDir + "\ncreated config.yaml\nexists customers.yaml\ncreated issuer.yaml\ncreated invoice_defaults.yaml\ncreated template.tex\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}

	source, err := os.ReadFile(customersPath)
	if err != nil {
		t.Fatalf("ReadFile(customers.yaml) returned error: %v", err)
	}
	if string(source) != customCustomers {
		t.Fatalf("customers.yaml was overwritten:\n%s", string(source))
	}
}

func TestInitWritesThroughDanglingStarterSymlink(t *testing.T) {
	x := clitest.New(t)

	configHome := filepath.Join(t.TempDir(), "config-home")
	configDir := filepath.Join(configHome, "invox")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	x.Setenv("XDG_CONFIG_HOME", configHome)
	targetPath := filepath.Join(t.TempDir(), "issuer.yaml")
	linkPath := filepath.Join(configDir, "issuer.yaml")
	if err := os.Symlink(targetPath, linkPath); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

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

	info, err := os.Lstat(linkPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&fs.ModeSymlink == 0 {
		t.Fatalf("%s was replaced by a regular file", linkPath)
	}
	if got, err := os.Readlink(linkPath); err != nil || got != targetPath {
		t.Fatalf("Readlink(%s) = %q, %v, want %q", linkPath, got, err, targetPath)
	}
	created, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("link target was not created: %v", err)
	}
	starter, err := os.ReadFile(filepath.Join("..", "..", "store", "starter", "issuer.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(created) != string(starter) {
		t.Fatalf("link target content = %q, want the starter issuer.yaml %q", created, starter)
	}
}
