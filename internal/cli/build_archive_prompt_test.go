package cli

import (
	"path/filepath"
	"testing"
)

// When build --archive would replace an archived invoice and the user can't
// or won't confirm, the error says once that the PDF was built.
func TestBuildArchiveReplacePromptStderr(t *testing.T) {
	for _, tc := range []struct {
		name       string
		terminal   bool
		answer     string
		wantStderr func(archivedPath, historyDir string) string
	}{
		{
			name: "no terminal",
			wantStderr: func(archivedPath, _ string) string {
				return "error: built first.pdf but archiving first.yaml replaces archived invoice " + archivedPath +
					"; pass --yes to replace it (stdin is not a terminal)\nRun 'invox build --help' for usage.\n"
			},
		},
		{
			// tectonic shares stdin, so the prompt reads end of input,
			// which declines.
			name: "declined at end of input", terminal: true, answer: "n\n",
			wantStderr: func(archivedPath, historyDir string) string {
				return "Replace archived invoice " + archivedPath + "? The previous version is kept in " + historyDir + ". [y/N] \n" +
					"built first.pdf but not archived; the archive was not changed; pass --yes to replace without asking\n"
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			customersPath, issuerPath, fixturePath, templatePath := writeContextFixtures(t)
			installFakeTectonic(t, fakeTectonicWritePDF)
			archiveDir := t.TempDir()
			writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
			archivedPath := filepath.Join(archiveDir, "first.yaml")
			writeBuildableInvoice(t, fixturePath, archivedPath, "archived")
			chdirForTest(t, t.TempDir())
			if exitCode, _, stderr := captureRun(t, []string{"archive", "edit", "first.yaml"}); exitCode != 0 {
				t.Fatalf("archive edit: exit code %d, stderr %q", exitCode, stderr)
			}

			ios := promptStreams(tc.terminal, tc.answer)
			exitCode, stdout, stderr := captureRunStreams(t, ios, []string{"build", "first.yaml", "--archive", "-c", customersPath, "-u", issuerPath, "-t", templatePath})

			if exitCode != 2 || stdout != "" {
				t.Fatalf("exit code %d, stdout %q; want 2, empty", exitCode, stdout)
			}
			if want := tc.wantStderr(archivedPath, filepath.Join(archiveDir, ".history")); stderr != want {
				t.Fatalf("stderr = %q\nwant     %q", stderr, want)
			}
		})
	}
}
