package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// configLayout is a config home with an invox and a legacy directory and a
// working directory outside the home directory, which the test moves into.
type configLayout struct {
	root, invoxDir, legacyDir, work string
}

func newConfigLayout(t *testing.T) configLayout {
	t.Helper()
	root := t.TempDir()
	l := configLayout{
		root:      root,
		invoxDir:  filepath.Join(root, "cfg", "invox"),
		legacyDir: filepath.Join(root, "cfg", "invoice-tool"),
		work:      filepath.Join(root, "work"),
	}
	for _, dir := range []string{l.invoxDir, l.legacyDir, l.work} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("MkdirAll returned error: %v", err)
		}
	}
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(root, "cfg"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(root, "data"))
	t.Setenv("APPDATA", filepath.Join(root, "data"))
	chdirForTest(t, l.work)
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
		b.WriteString(strings.Join(row, "\t") + "\n")
	}
	return b.String()
}

func TestConfigPathsReportsEachSource(t *testing.T) {
	l := newConfigLayout(t)
	writeTestFile(t, filepath.Join(l.work, "customers.yaml"), "{}\n")
	writeTestFile(t, filepath.Join(l.legacyDir, "issuer.yaml"), "{}\n")
	writeTestFile(t, filepath.Join(l.invoxDir, "template.tex"), "x\n")
	explicit := filepath.Join(l.root, "explicit.yaml")
	writeTestFile(t, explicit, "paths:\n  defaults: 'd.yaml'\narchive:\n  dir: 'arch'\n")

	exitCode, stdout, stderr := captureRun(t, []string{"config", "--config", "ignored.yaml", "paths", "--config=" + explicit})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	want := tsv(
		[]string{"config-dir", l.invoxDir, "default"},
		[]string{"config", explicit, "flag"},
		[]string{"customers", filepath.Join(l.work, "customers.yaml"), "project"},
		[]string{"issuer", filepath.Join(l.legacyDir, "issuer.yaml"), "legacy"},
		[]string{"defaults", filepath.Join(l.root, "d.yaml"), "config"},
		[]string{"template", filepath.Join(l.invoxDir, "template.tex"), "default"},
		[]string{"archive", filepath.Join(l.root, "arch"), "config"},
	)
	if stdout != want {
		t.Fatalf("stdout =\n%s\nwant\n%s", stdout, want)
	}
	wantStderr := "warning: using issuer.yaml from deprecated config directory " + l.legacyDir + "; run 'invox init' to copy it to " + l.invoxDir + "\n"
	if stderr != wantStderr {
		t.Fatalf("stderr = %q, want %q", stderr, wantStderr)
	}
}

func TestConfigPathsWithInvoxConfigDir(t *testing.T) {
	l := newConfigLayout(t)
	writeTestFile(t, filepath.Join(l.legacyDir, "issuer.yaml"), "{}\n")
	envDir := filepath.Join(l.root, "env")
	writeTestFile(t, filepath.Join(envDir, "config.yaml"), "")
	writeTestFile(t, filepath.Join(envDir, "customers.yaml"), "{}\n")
	t.Setenv("INVOX_CONFIG_DIR", envDir)

	exitCode, stdout, stderr := captureRun(t, []string{"config", "paths"})
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
			l := newConfigLayout(t)
			if tt.env != "" {
				t.Setenv("INVOX_CONFIG_DIR", filepath.Join(l.root, tt.env))
			}

			exitCode, stdout, stderr := captureRun(t, tt.args(l))
			if exitCode != tt.wantCode || stdout != "" || stderr != tt.wantStderr(l) {
				t.Fatalf("got exit %d, stdout %q, stderr %q; want exit %d, no stdout, stderr %q", exitCode, stdout, stderr, tt.wantCode, tt.wantStderr(l))
			}
		})
	}
}

func TestConfigUnknownKeyNamesFileAndLine(t *testing.T) {
	configPath := writeConfigFile(t, "numbering:\n  start: 2\n  patern: x\n")
	chdirForTest(t, t.TempDir())

	exitCode, stdout, stderr := captureRun(t, []string{"new", "CUST-001"})
	want := "error: " + configPath + ":3: unknown key \"patern\" in numbering\nRun 'invox config' to open and fix the config file.\n"
	if exitCode != 1 || stdout != "" || stderr != want {
		t.Fatalf("got exit %d, stdout %q, stderr %q; want exit 1, no stdout, stderr %q", exitCode, stdout, stderr, want)
	}
}

func TestLegacyFileWarningComesBeforeTheError(t *testing.T) {
	l := newConfigLayout(t)
	writeTestFile(t, filepath.Join(l.legacyDir, "config.yaml"), "numbering:\n  start: 3\n")
	writeTestFile(t, filepath.Join(l.legacyDir, "customers.yaml"), "{}\n")

	exitCode, stdout, stderr := captureRun(t, []string{"validate", "-i", "missing.yaml"})
	want := "warning: using config.yaml and customers.yaml from deprecated config directory " + l.legacyDir + "; run 'invox init' to copy them to " + l.invoxDir + "\n" +
		"error: issuer file not found; pass -u/--issuer, set paths.issuer in config.yaml, or place issuer.yaml at " + filepath.Join(l.invoxDir, "issuer.yaml") + "\n" +
		"Run 'invox validate --help' for usage.\n"
	if exitCode != 2 || stdout != "" || stderr != want {
		t.Fatalf("got exit %d, stdout %q, stderr\n%s\nwant exit 2, no stdout, stderr\n%s", exitCode, stdout, stderr, want)
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
			want := "config-dir\t" + filepath.Join(root, filepath.FromSlash(tt.wantDir)) + "\t" + tt.wantSource + "\n"
			if exitCode != 0 || !strings.HasPrefix(stdout, want) {
				t.Fatalf("exit %d, stdout %q, stderr %q; want stdout to start with %q", exitCode, stdout, stderr, want)
			}
		})
	}
}

func TestConfigOpensTheConfigFlagFile(t *testing.T) {
	chdirForTest(t, t.TempDir())
	path := filepath.Join(t.TempDir(), "acme.yaml")
	f, stub := testFactory(t)
	opened := expectEditor(stub, nil)

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{"config", "--config", path})
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
