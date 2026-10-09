package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/tableprinter"
	"github.com/0xboris/invox/internal/testfixture"
)

// configLayout is a config home with an invox directory and a working
// directory outside the home directory, which the test moves into.
type configLayout struct {
	root, invoxDir, work string
}

func newConfigLayout(t *testing.T, x *clitest.Invox) configLayout {
	t.Helper()
	root := t.TempDir()
	l := configLayout{
		root:     root,
		invoxDir: filepath.Join(root, "cfg", "invox"),
		work:     filepath.Join(root, "work"),
	}
	for _, dir := range []string{l.invoxDir, l.work} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}
	x.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "cfg"))
	x.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	x.Setenv("APPDATA", filepath.Join(root, "data"))
	x.Chdir(l.work)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	l.work = cwd
	return l
}

func tsv(rows ...[]string) string {
	var b strings.Builder
	for _, row := range rows {
		fields := make([]string, len(row))
		for i, field := range row {
			fields[i] = tableprinter.EscapeTSVField(field)
		}
		b.WriteString(strings.Join(fields, "\t") + "\n")
	}
	return b.String()
}

func TestConfigPathsReportsEachSource(t *testing.T) {
	x := clitest.New(t)

	l := newConfigLayout(t, x)
	testfixture.WriteFile(t, filepath.Join(l.work, "customers.yaml"), "{}\n")
	testfixture.WriteFile(t, filepath.Join(l.invoxDir, "template.tex"), "x\n")
	explicit := filepath.Join(l.root, "explicit.yaml")
	testfixture.WriteFile(t, explicit, "paths:\n  defaults: 'd.yaml'\narchive:\n  dir: 'arch'\n")

	exitCode, stdout, stderr := x.Run([]string{"config", "--config", "ignored.yaml", "paths", "--config=" + explicit})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	want := tsv(
		[]string{"config-dir", l.invoxDir, "default"},
		[]string{"config", explicit, "flag"},
		[]string{"customers", filepath.Join(l.work, "customers.yaml"), "project"},
		[]string{"issuer", "", "none"},
		[]string{"defaults", filepath.Join(l.root, "d.yaml"), "config"},
		[]string{"template", filepath.Join(l.invoxDir, "template.tex"), "default"},
		[]string{"archive", filepath.Join(l.root, "arch"), "config"},
	)
	if stdout != want || stderr != "" {
		t.Fatalf("stdout =\n%s\nstderr = %q\nwant stdout\n%s\nand no stderr", stdout, stderr, want)
	}
}

func TestConfigPathsWithInvoxConfigDir(t *testing.T) {
	x := clitest.New(t)

	l := newConfigLayout(t, x)
	envDir := filepath.Join(l.root, "env")
	testfixture.WriteFile(t, filepath.Join(envDir, "config.yaml"), "")
	testfixture.WriteFile(t, filepath.Join(envDir, "customers.yaml"), "{}\n")
	x.Setenv("INVOX_CONFIG_DIR", envDir)

	exitCode, stdout, stderr := x.Run([]string{"config", "paths"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	want := tsv(
		[]string{"config-dir", envDir, "env"},
		[]string{"config", filepath.Join(envDir, "config.yaml"), "env"},
		[]string{"customers", filepath.Join(envDir, "customers.yaml"), "env"},
		[]string{"issuer", "", "none"},
		[]string{"defaults", "", "none"},
		[]string{"template", "", "none"},
		[]string{"archive", filepath.Join(l.root, "data", "invox", "invoices"), "default"},
	)
	if stdout != want || stderr != "" {
		t.Fatalf("stdout =\n%s\nstderr = %q\nwant stdout\n%s\nand no stderr", stdout, stderr, want)
	}
}

func TestConfigLocationErrors(t *testing.T) {
	tests := []struct {
		name       string
		env        string
		args       func(l configLayout) []string
		wantCode   int
		wantStderr func(l configLayout) string
	}{
		{
			name: "missing --config file",
			args: func(l configLayout) []string {
				return []string{"--config", filepath.Join(l.root, "missing.yaml"), "config", "paths"}
			},
			wantCode: 1,
			wantStderr: func(l configLayout) string {
				return "error: config file " + filepath.Join(l.root, "missing.yaml") + " does not exist\n"
			},
		},
		{
			name:     "--config without a value",
			args:     func(configLayout) []string { return []string{"validate", "--config"} },
			wantCode: 2,
			wantStderr: func(configLayout) string {
				return "error: flag needs an argument: --config\nRun 'invox --help' for usage.\n"
			},
		},
		{
			name:     "--config= with an empty value",
			args:     func(configLayout) []string { return []string{"--config=", "config", "paths"} },
			wantCode: 2,
			wantStderr: func(configLayout) string {
				return "error: flag needs an argument: --config\nRun 'invox --help' for usage.\n"
			},
		},
		{
			name:     "missing INVOX_CONFIG_DIR",
			env:      "no-such-dir",
			args:     func(configLayout) []string { return []string{"new", "CUST-001"} },
			wantCode: 1,
			wantStderr: func(l configLayout) string {
				return "error: config directory " + filepath.Join(l.root, "no-such-dir") + " does not exist\n"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			x := clitest.New(t)

			l := newConfigLayout(t, x)
			if tt.env != "" {
				x.Setenv("INVOX_CONFIG_DIR", filepath.Join(l.root, tt.env))
			}

			exitCode, stdout, stderr := x.Run(tt.args(l))
			if exitCode != tt.wantCode || stdout != "" || stderr != tt.wantStderr(l) {
				t.Fatalf("got exit %d, stdout %q, stderr %q; want exit %d, no stdout, stderr %q", exitCode, stdout, stderr, tt.wantCode, tt.wantStderr(l))
			}
		})
	}
}

func TestConfigUnknownKeyNamesFileAndLine(t *testing.T) {
	x := clitest.New(t)

	configPath := x.WriteConfig("numbering:\n  start: 2\n  patern: x\n")
	x.Chdir(t.TempDir())

	exitCode, stdout, stderr := x.Run([]string{"new", "CUST-001"})
	want := "error: " + configPath + ":3: unknown key \"patern\" in numbering\nRun 'invox config' to open and fix the config file.\n"
	if exitCode != 1 || stdout != "" || stderr != want {
		t.Fatalf("got exit %d, stdout %q, stderr %q; want exit 1, no stdout, stderr %q", exitCode, stdout, stderr, want)
	}
}

func TestDirectoryVariablesBecomeAbsolute(t *testing.T) {
	tests := []struct {
		name, key, value, wantDir, wantSource string
	}{
		{name: "relative XDG_CONFIG_HOME is ignored", key: "XDG_CONFIG_HOME", value: "rel-config", wantDir: "home/.config/invox", wantSource: "default"},
		{name: "relative INVOX_CONFIG_DIR joins the working directory", key: "INVOX_CONFIG_DIR", value: "env-dir", wantDir: "work/env-dir", wantSource: "env"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			work := filepath.Join(root, "work")
			if err := os.MkdirAll(filepath.Join(work, "env-dir"), 0o755); err != nil {
				t.Fatalf("MkdirAll returned error: %v", err)
			}
			e := testEnv(root, func() (string, error) { return work, nil })
			getenv := e.Getenv
			e.Getenv = func(key string) string {
				if key == tt.key {
					return tt.value
				}
				return getenv(key)
			}

			exitCode, stdout, stderr := runWithEnv(e, "config", "paths")
			want := tsv([]string{"config-dir", filepath.Join(root, filepath.FromSlash(tt.wantDir)), tt.wantSource})
			if exitCode != 0 || !strings.HasPrefix(stdout, want) {
				t.Fatalf("exit %d, stdout %q, stderr %q; want stdout to start with %q", exitCode, stdout, stderr, want)
			}
		})
	}
}

func TestConfigOpensTheConfigFlagFile(t *testing.T) {
	x := clitest.New(t)

	x.Chdir(t.TempDir())
	path := filepath.Join(t.TempDir(), "acme.yaml")
	opened := x.ExpectEditor(nil)

	exitCode, stdout, stderr := x.Run([]string{"config", "--config", path})
	if exitCode != 0 || stdout != "" || stderr != "Opened "+path+"\n" {
		t.Fatalf("got exit %d, stdout %q, stderr %q; want exit 0 and %q", exitCode, stdout, stderr, "Opened "+path+"\n")
	}
	if *opened != path {
		t.Fatalf("editor opened %q, want %q", *opened, path)
	}
	source, err := os.ReadFile(path)
	if err != nil || !strings.HasPrefix(string(source), "# Invox user configuration.") {
		t.Fatalf("ReadFile(%s) = %q, %v; want the config template", path, source, err)
	}
}

func TestInitWritesToInvoxConfigDir(t *testing.T) {
	x := clitest.New(t)

	l := newConfigLayout(t, x)
	envDir := filepath.Join(l.root, "env")
	x.Setenv("INVOX_CONFIG_DIR", envDir)

	exitCode, _, stderr := x.Run([]string{"init"})
	if want := "Initialized " + envDir + "\ncreated config.yaml\ncreated customers.yaml\n"; exitCode != 0 || !strings.HasPrefix(stderr, want) {
		t.Fatalf("init: exit %d, stderr %q; want exit 0 and stderr starting with %q", exitCode, stderr, want)
	}
}

func TestConfigFlagErrorHintNamesTheFlag(t *testing.T) {
	x := clitest.New(t)

	dir := t.TempDir()
	x.Chdir(dir)
	testfixture.WriteFile(t, filepath.Join(dir, "acme.yaml"), "numbering:\n  patern: x\n")

	exitCode, stdout, stderr := x.Run([]string{"--config", "acme.yaml", "new", "CUST-001"})
	want := "error: acme.yaml:2: unknown key \"patern\" in numbering\nRun 'invox --config acme.yaml config' to open and fix the config file.\n"
	if exitCode != 1 || stdout != "" || stderr != want {
		t.Fatalf("exit %d, stdout %q, stderr %q; want exit 1 and stderr %q", exitCode, stdout, stderr, want)
	}
}
