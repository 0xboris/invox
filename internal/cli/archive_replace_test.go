package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/iostreams"
)

// promptStreams returns test streams on which confirmation prompts see a
// terminal (or not) and read their answer from input.
func promptStreams(terminal bool, input string) *iostreams.IOStreams {
	ios, in, _, _ := iostreams.Test()
	ios.SetStdinTTY(terminal)
	ios.SetStderrTTY(terminal)
	in.WriteString(input)
	return ios
}

type editedArchive struct {
	archiveDir   string
	archivedPath string
	original     string
	workingCopy  string
}

// setupEditedArchive archives first.yaml, opens it with `archive edit` in a
// fresh working directory and changes the working copy.
func setupEditedArchive(t *testing.T) editedArchive {
	t.Helper()

	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := writeNumberedInvoice(t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	exitCode, _, stderr := captureRun(t, []string{"archive", "edit", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("archive edit: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	workingCopy := filepath.Join(workDir, "first.yaml")
	edited := strings.Replace(readFileForTest(t, workingCopy), "unit_price: 100", "unit_price: 250", 1)
	if err := os.WriteFile(workingCopy, []byte(edited), 0o644); err != nil {
		t.Fatalf("WriteFile(working copy) returned error: %v", err)
	}
	return editedArchive{
		archiveDir:   archiveDir,
		archivedPath: archivedPath,
		original:     readFileForTest(t, archivedPath),
		workingCopy:  workingCopy,
	}
}

func (e editedArchive) assertUnchanged(t *testing.T) {
	t.Helper()

	if got := readFileForTest(t, e.archivedPath); got != e.original {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(e.workingCopy); err != nil {
		t.Fatalf("working copy should stay in place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.archiveDir, ".history")); !os.IsNotExist(err) {
		t.Fatalf("no backup should be written, Stat err = %v", err)
	}
}

// assertReplaced checks that the edited working copy replaced the archived
// invoice and that the previous version is kept in .history.
func (e editedArchive) assertReplaced(t *testing.T) string {
	t.Helper()

	if !strings.Contains(readFileForTest(t, e.archivedPath), "unit_price: 250") {
		t.Fatalf("archived invoice was not replaced:\n%s", readFileForTest(t, e.archivedPath))
	}
	if _, err := os.Stat(e.workingCopy); !os.IsNotExist(err) {
		t.Fatalf("working copy should be removed, Stat err = %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(e.archiveDir, ".history", "first.*.yaml"))
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backups = %q, want exactly one", backups)
	}
	if got := readFileForTest(t, backups[0]); got != e.original {
		t.Fatalf("backup = %q, want the previous version %q", got, e.original)
	}
	return backups[0]
}

func (e editedArchive) replacedNotice(backupPath string) string {
	return "Replaced archived invoice " + e.archivedPath + "; previous version kept at " + backupPath + "\n"
}

// writeBuildableInvoice rewrites the context fixture invoice at path with
// the given status.
func writeBuildableInvoice(t *testing.T, fixturePath, path, status string) {
	t.Helper()

	source := strings.Replace(readFileForTest(t, fixturePath), "  paid_amount: 0\n", "  paid_amount: 0\n  status: "+status+"\n", 1)
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) returned error: %v", path, err)
	}
}

func TestYesFlagIsDocumented(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{
			args: []string{"archive", "add", "-h"},
			want: []string{"      --yes            Replace an archived invoice without asking\n", "archive.dir/.history/<path>.<UTC timestamp>.<ext>"},
		},
		{
			args: []string{"build", "-h"},
			want: []string{"      --yes                Replace an archived invoice without asking\n", "keeps that status when its PDF is rebuilt"},
		},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			exitCode, stdout, stderr := captureRun(t, tc.args)
			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
				}
			}
		})
	}
}
