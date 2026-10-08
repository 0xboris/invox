package cli

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/build"
)

func setBuildInfo(t *testing.T, version, date string) {
	t.Helper()

	oldVersion, oldDate := build.Version, build.Date
	build.Version, build.Date = version, date
	t.Cleanup(func() {
		build.Version, build.Date = oldVersion, oldDate
	})
}

func TestVersionPrintsVersion(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		version string
		date    string
		want    string
	}{
		{name: "subcommand", args: []string{"version"}, version: "v1.2.3", date: "2026-10-05", want: "invox version 1.2.3 (2026-10-05)\n"},
		{name: "flag", args: []string{"--version"}, version: "v1.2.3", date: "2026-10-05", want: "invox version 1.2.3 (2026-10-05)\n"},
		{name: "no date", args: []string{"version"}, version: "DEV", want: "invox version DEV\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setBuildInfo(t, tt.version, tt.date)

			exitCode, stdout, stderr := captureRun(t, tt.args)
			if exitCode != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr %q)", exitCode, stderr)
			}
			if stdout != tt.want {
				t.Fatalf("stdout = %q, want %q", stdout, tt.want)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
		})
	}
}

func TestVersionWorksWithBrokenConfig(t *testing.T) {
	setBuildInfo(t, "v1.2.3", "")
	writeConfigFile(t, "paths: [unterminated\n")

	for _, args := range [][]string{{"version"}, {"--version"}} {
		exitCode, stdout, stderr := captureRun(t, args)
		if exitCode != 0 || stdout != "invox version 1.2.3\n" || stderr != "" {
			t.Fatalf("Main(%q) = (%d, %q, %q), want (0, %q, \"\")", args, exitCode, stdout, stderr, "invox version 1.2.3\n")
		}
	}
}

func TestVersionHelp(t *testing.T) {
	for _, args := range [][]string{{"version", "--help"}, {"help", "version"}} {
		exitCode, stdout, stderr := captureRun(t, args)
		if exitCode != 0 {
			t.Fatalf("Main(%q) exit code = %d, want 0", args, exitCode)
		}
		if !strings.HasPrefix(stdout, "Show the invox version.") {
			t.Fatalf("Main(%q) stdout = %q, want version help", args, stdout)
		}
		if stderr != "" {
			t.Fatalf("Main(%q) stderr = %q, want empty", args, stderr)
		}
	}
}

func TestVersionRejectsArguments(t *testing.T) {
	exitCode, stdout, stderr := captureRun(t, []string{"version", "extra"})
	if exitCode != 2 {
		t.Fatalf("exit code = %d, want 2", exitCode)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "unexpected arguments for version: extra") {
		t.Fatalf("stderr %q does not mention the unexpected argument", stderr)
	}
}

func TestRootHelpListsVersion(t *testing.T) {
	_, stdout, _ := captureRun(t, []string{"--help"})
	if !strings.Contains(stdout, "  version     Show the invox version\n") {
		t.Fatalf("root help does not list the version subcommand:\n%s", stdout)
	}
	if !strings.Contains(stdout, "      --version         Show the invox version\n") {
		t.Fatalf("root help does not list --version:\n%s", stdout)
	}
}
