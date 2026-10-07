package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
)

// editorCase is one command that opens an editor. setup creates the files it
// needs with dir as the working directory and returns its arguments and the
// file it opens. gate is its error message when no editor may open, with %s
// for the reason.
type editorCase struct {
	name  string
	setup func(t *testing.T, dir string) (args []string, path string)
	gate  string
	usage string
}

var editorCases = []editorCase{
	{
		name: "new -e",
		setup: func(t *testing.T, dir string) ([]string, string) {
			customersPath, issuerPath, defaultsPath := writeDraftFixtures(t)
			writeConfigFile(t, "numbering:\n  pattern: '{customer_id}-{counter:03}'\n  start: 2\n")
			return []string{"new", "CUST-001", "-e", "-c", customersPath, "-u", issuerPath, "-s", defaultsPath},
				filepath.Join(dir, "CUST-001-002.yaml")
		},
		gate:  "created CUST-001-002.yaml but cannot open an editor: %s; edit it and run 'invox validate -i CUST-001-002.yaml'",
		usage: "Run 'invox new --help' for usage.",
	},
	{
		name: "config",
		setup: func(t *testing.T, dir string) ([]string, string) {
			t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config-home"))
			return []string{"config"}, filepath.Join(dir, "config-home", "invox", "config.yaml")
		},
		gate:  "cannot open an editor: %s; edit " + filepath.Join("config-home", "invox", "config.yaml") + " directly",
		usage: "Run 'invox config --help' for usage.",
	},
	{
		name: "customer config",
		setup: func(t *testing.T, dir string) ([]string, string) {
			path := filepath.Join(dir, "customers.yaml")
			if err := os.WriteFile(path, []byte("CUST-001: {}\n"), 0o644); err != nil {
				t.Fatalf("WriteFile(customers.yaml) returned error: %v", err)
			}
			return []string{"customer", "config", "-c", "customers.yaml"}, path
		},
		gate:  "cannot open an editor: %s; edit customers.yaml directly",
		usage: "Run 'invox customer config --help' for usage.",
	},
}

func gateStderr(c editorCase, reason string) string {
	return "error: " + fmt.Sprintf(c.gate, reason) + "\n" + c.usage + "\n"
}

func TestEditorNeedsATerminal(t *testing.T) {
	reasons := []struct {
		name      string
		stdinTTY  bool
		stderrTTY bool
		reason    string
	}{
		{name: "no terminal", reason: "stdin is not a terminal"},
		{name: "stderr piped", stdinTTY: true, reason: "stderr is not a terminal"},
	}
	for _, c := range editorCases {
		for _, r := range reasons {
			t.Run(c.name+"/"+r.name, func(t *testing.T) {
				dir := t.TempDir()
				chdirForTest(t, dir)
				args, path := c.setup(t, dir)
				f, _ := testFactory(t)
				f.IOStreams.SetStdinTTY(r.stdinTTY)
				f.IOStreams.SetStderrTTY(r.stderrTTY)

				exitCode, stdout, stderr := captureRunFactory(t, f, args)

				if exitCode != 2 || stdout != "" || stderr != gateStderr(c, r.reason) {
					t.Fatalf("got (%d, %q, %q), want (2, \"\", %q)", exitCode, stdout, stderr, gateStderr(c, r.reason))
				}
				if _, err := os.Stat(path); err != nil {
					t.Fatalf("%s does not exist after the gate: %v", path, err)
				}
			})
		}
	}
}

func TestEditorRespectsDisabledPrompts(t *testing.T) {
	const reason = "prompts are disabled (--no-input or INVOX_PROMPT_DISABLED)"
	ways := []struct {
		name string
		vars map[string]string
		args func([]string) []string
	}{
		{name: "--no-input first", args: func(args []string) []string { return append([]string{"--no-input"}, args...) }},
		{name: "--no-input last", args: func(args []string) []string { return append(slices.Clone(args), "--no-input") }},
		{name: "INVOX_PROMPT_DISABLED", vars: map[string]string{"INVOX_PROMPT_DISABLED": "1"}, args: func(args []string) []string { return args }},
	}
	for _, c := range editorCases {
		for _, w := range ways {
			t.Run(c.name+"/"+w.name, func(t *testing.T) {
				dir := t.TempDir()
				chdirForTest(t, dir)
				args, _ := c.setup(t, dir)
				f, _ := testFactoryEnv(t, w.vars)
				f.IOStreams.SetStdinTTY(true)
				f.IOStreams.SetStderrTTY(true)

				exitCode, stdout, stderr := captureRunFactory(t, f, w.args(args))

				if exitCode != 2 || stdout != "" || stderr != gateStderr(c, reason) {
					t.Fatalf("got (%d, %q, %q), want (2, \"\", %q)", exitCode, stdout, stderr, gateStderr(c, reason))
				}
			})
		}
	}
}

func TestEditorRunsTheEditorSettingOnATerminal(t *testing.T) {
	for _, c := range editorCases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			chdirForTest(t, dir)
			args, path := c.setup(t, dir)
			f, stub := testFactoryEnv(t, map[string]string{"EDITOR": "code -w"})
			f.IOStreams.SetStdinTTY(true)
			f.IOStreams.SetStderrTTY(true)
			var got run.Cmd
			stub.Register("code", func(cmd run.Cmd) error {
				got = cmd
				return nil
			})

			exitCode, _, stderr := captureRunFactory(t, f, args)

			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
			}
			if want := []string{"-w", path}; !slices.Equal(got.Args, want) {
				t.Fatalf("code ran with %q, want %q", got.Args, want)
			}
		})
	}
}

func TestEditorFailureNamesTheEditor(t *testing.T) {
	dir := t.TempDir()
	chdirForTest(t, dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, "config-home"))
	f, stub := testFactoryEnv(t, map[string]string{"EDITOR": "code -w"})
	f.IOStreams.SetStdinTTY(true)
	f.IOStreams.SetStderrTTY(true)
	stub.Register("code", func(run.Cmd) error { return &run.ExecError{Name: "code", Code: 3} })

	exitCode, stdout, stderr := captureRunFactory(t, f, []string{"config"})

	want := "error: failed to open " + filepath.Join("config-home", "invox", "config.yaml") + ": editor \"code -w\" exited with status 3\n"
	if exitCode != 1 || stdout != "" || stderr != want {
		t.Fatalf("got (%d, %q, %q), want (1, \"\", %q)", exitCode, stdout, stderr, want)
	}
}
