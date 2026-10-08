package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/tableprinter"
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
		fields := make([]string, len(row))
		for i, field := range row {
			fields[i] = tableprinter.EscapeTSVField(field)
		}
		b.WriteString(strings.Join(fields, "\t") + "\n")
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
			want := tsv([]string{"config-dir", filepath.Join(root, filepath.FromSlash(tt.wantDir)), tt.wantSource})
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
	opened := expectEditor(f, stub, nil)

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

func TestInitCopiesLegacyFiles(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		tty        bool
		answer     string
		wantCode   int
		wantCopied bool
		wantStderr func(l configLayout) string
	}{
		{
			name: "yes on a terminal", args: []string{"init"}, tty: true, answer: "y\n", wantCode: 0, wantCopied: true,
			wantStderr: func(l configLayout) string {
				return "Copy customers.yaml from " + l.legacyDir + " to " + l.invoxDir + "? [y/N] " + copiedLines(l)
			},
		},
		{
			name: "no on a terminal", args: []string{"init"}, tty: true, answer: "n\n", wantCode: 2,
			wantStderr: func(l configLayout) string {
				return "Copy customers.yaml from " + l.legacyDir + " to " + l.invoxDir + "? [y/N] not initialized; nothing was changed\n"
			},
		},
		{
			name: "--force without a terminal", args: []string{"init", "--force"}, wantCode: 0, wantCopied: true,
			wantStderr: copiedLines,
		},
		{
			name: "no terminal and no --force", args: []string{"init"}, wantCode: 2,
			wantStderr: func(l configLayout) string {
				return "error: the deprecated config directory " + l.legacyDir + " has files that " + l.invoxDir + " lacks; pass --force to copy them (no terminal to ask on)\nRun 'invox init --help' for usage.\n"
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := newConfigLayout(t)
			if err := os.Remove(l.invoxDir); err != nil {
				t.Fatalf("Remove returned error: %v", err)
			}
			writeTestFile(t, filepath.Join(l.legacyDir, "customers.yaml"), "legacy\n")
			ios, stdin, _, _ := iostreams.Test()
			stdin.WriteString(tt.answer)
			ios.SetStdinTTY(tt.tty)
			ios.SetStderrTTY(tt.tty)

			exitCode, stdout, stderr := captureRunStreams(t, ios, tt.args)
			if exitCode != tt.wantCode || stdout != "" || stderr != tt.wantStderr(l) {
				t.Fatalf("got exit %d, stdout %q, stderr\n%s\nwant exit %d, no stdout, stderr\n%s", exitCode, stdout, stderr, tt.wantCode, tt.wantStderr(l))
			}
			got, err := os.ReadFile(filepath.Join(l.invoxDir, "customers.yaml"))
			if copied := err == nil && string(got) == "legacy\n"; copied != tt.wantCopied {
				t.Fatalf("customers.yaml in the invox dir = %q, %v; want copied=%v", got, err, tt.wantCopied)
			}
			if _, err := os.Stat(filepath.Join(l.legacyDir, "customers.yaml")); err != nil {
				t.Fatalf("legacy customers.yaml is gone: %v", err)
			}
		})
	}
}

func copiedLines(l configLayout) string {
	return "copied customers.yaml from " + l.legacyDir + "\n" +
		"invox no longer reads " + l.legacyDir + " for these files; remove it once you are happy with " + l.invoxDir + "\n" +
		"Initialized " + l.invoxDir + "\n" +
		"created config.yaml\nexists customers.yaml\ncreated issuer.yaml\ncreated invoice_defaults.yaml\ncreated template.tex\n"
}

func TestLegacyWarningStopsAfterInit(t *testing.T) {
	l := newConfigLayout(t)
	writeTestFile(t, filepath.Join(l.legacyDir, "issuer.yaml"), "{}\n")

	_, _, before := captureRun(t, []string{"config", "paths"})
	if !strings.HasPrefix(before, "warning: using issuer.yaml from deprecated config directory ") {
		t.Fatalf("stderr before init = %q, want the legacy warning", before)
	}
	exitCode, _, stderr := captureRun(t, []string{"init", "--force"})
	if exitCode != 0 || strings.Contains(stderr, "warning:") {
		t.Fatalf("init: exit %d, stderr %q; want exit 0 and no warning", exitCode, stderr)
	}
	exitCode, _, stderr = captureRun(t, []string{"init"})
	if exitCode != 0 || strings.Contains(stderr, "Copy ") || strings.Contains(stderr, "copied") {
		t.Fatalf("second init without a terminal: exit %d, stderr %q; want exit 0 and nothing to copy", exitCode, stderr)
	}
	exitCode, stdout, stderr := captureRun(t, []string{"config", "paths"})
	if exitCode != 0 || stderr != "" || !strings.Contains(stdout, tsv([]string{"issuer", filepath.Join(l.invoxDir, "issuer.yaml"), "default"})) {
		t.Fatalf("config paths after init: exit %d, stdout %q, stderr %q; want issuer from the invox dir and no warning", exitCode, stdout, stderr)
	}
}

func TestInitWritesToInvoxConfigDir(t *testing.T) {
	l := newConfigLayout(t)
	writeTestFile(t, filepath.Join(l.legacyDir, "customers.yaml"), "legacy\n")
	envDir := filepath.Join(l.root, "env")
	t.Setenv("INVOX_CONFIG_DIR", envDir)

	exitCode, _, stderr := captureRun(t, []string{"init"})
	if want := "Initialized " + envDir + "\ncreated config.yaml\ncreated customers.yaml\n"; exitCode != 0 || !strings.HasPrefix(stderr, want) {
		t.Fatalf("init: exit %d, stderr %q; want exit 0, no legacy copy, and stderr starting with %q", exitCode, stderr, want)
	}
}

func TestConfigFlagErrorHintNamesTheFlag(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)
	writeTestFile(t, filepath.Join(dir, "acme.yaml"), "numbering:\n  patern: x\n")

	exitCode, stdout, stderr := captureRun(t, []string{"--config", "acme.yaml", "new", "CUST-001"})
	want := "error: acme.yaml:2: unknown key \"patern\" in numbering\nRun 'invox --config acme.yaml config' to open and fix the config file.\n"
	if exitCode != 1 || stdout != "" || stderr != want {
		t.Fatalf("exit %d, stdout %q, stderr %q; want exit 1 and stderr %q", exitCode, stdout, stderr, want)
	}
}
