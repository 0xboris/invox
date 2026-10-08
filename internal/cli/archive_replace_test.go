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

func TestArchiveReplaceWithoutTerminalRequiresYes(t *testing.T) {
	e := setupEditedArchive(t)
	ios := promptStreams(false, "y\n")

	exitCode, stdout, stderr := captureRunStreams(t, ios, []string{"archive", "add", "first.yaml"})
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2, stderr=%q", exitCode, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	want := "error: archiving first.yaml replaces archived invoice " + e.archivedPath + "; pass --yes to replace it (stdin is not a terminal)\n" +
		"Run 'invox archive add --help' for usage.\n"
	if stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	e.assertUnchanged(t)
}

func TestArchiveReplaceOnTerminalDeclined(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
	}{
		{name: "no", input: "n\n"},
		{name: "empty answer", input: "\n"},
		{name: "end of input", input: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := setupEditedArchive(t)
			ios := promptStreams(true, tc.input)

			exitCode, stdout, stderr := captureRunStreams(t, ios, []string{"archive", "add", "first.yaml"})
			if exitCode != 2 {
				t.Fatalf("exitCode = %d, want 2, stderr=%q", exitCode, stderr)
			}
			if stdout != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			prompt := "Replace archived invoice " + e.archivedPath + "? The previous version is kept in " +
				filepath.Join(e.archiveDir, ".history") + ". [y/N] "
			if !strings.HasPrefix(stderr, prompt) {
				t.Fatalf("stderr = %q, want it to start with prompt %q", stderr, prompt)
			}
			if !strings.HasSuffix(stderr, "not archived; the archive was not changed; pass --yes to replace without asking\n") {
				t.Fatalf("stderr = %q, want abort notice", stderr)
			}
			e.assertUnchanged(t)
		})
	}
}

func TestArchiveReplaceOnTerminalConfirmed(t *testing.T) {
	e := setupEditedArchive(t)
	ios := promptStreams(true, "y\n")

	exitCode, stdout, stderr := captureRunStreams(t, ios, []string{"archive", "add", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := e.archivedPath + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	backupPath := e.assertReplaced(t)
	prompt := "Replace archived invoice " + e.archivedPath + "? The previous version is kept in " +
		filepath.Join(e.archiveDir, ".history") + ". [y/N] "
	if want := prompt + e.replacedNotice(backupPath) + "Archived first.yaml -> " + e.archivedPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
}

func TestArchiveReplaceWithYesKeepsBackup(t *testing.T) {
	e := setupEditedArchive(t)
	// --yes answers the question, so the declining input is never read.
	ios := promptStreams(true, "n\n")

	exitCode, stdout, stderr := captureRunStreams(t, ios, []string{"archive", "add", "first.yaml", "--yes"})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := e.archivedPath + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	backupPath := e.assertReplaced(t)
	if want := e.replacedNotice(backupPath) + "Archived first.yaml -> " + e.archivedPath + "\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}

	exitCode, stdout, stderr = captureRun(t, []string{"archive", "list"})
	if exitCode != 0 {
		t.Fatalf("archive list: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "first.yaml\tCUST-001\t2026-03-06\tarchived\n"; stdout != want {
		t.Fatalf("archive list: stdout = %q, want %q (backups must not be listed)", stdout, want)
	}
}

func TestArchiveReplaceWithYesStillValidates(t *testing.T) {
	e := setupEditedArchive(t)
	writeNumberedInvoice(t, e.archiveDir, "second.yaml", "CUST-001-002", "archived")
	renumbered := strings.Replace(readFileForTest(t, e.workingCopy), "number: CUST-001-001", "number: CUST-001-002", 1)
	if err := os.WriteFile(e.workingCopy, []byte(renumbered), 0o644); err != nil {
		t.Fatalf("WriteFile(working copy) returned error: %v", err)
	}

	exitCode, stdout, stderr := captureRun(t, []string{"archive", "add", "first.yaml", "--yes"})
	if exitCode != 1 {
		t.Fatalf("exitCode = %d, want 1, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if !strings.Contains(stderr, "invoice number CUST-001-002 is already used by archived invoice "+filepath.Join(e.archiveDir, "second.yaml")) {
		t.Fatalf("stderr = %q, want duplicate-number error", stderr)
	}
	e.assertUnchanged(t)
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

func TestBuildKeepsArchivedStatus(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicWritePDF)
	writeBuildableInvoice(t, invoicePath, invoicePath, "archived")
	original := readFileForTest(t, invoicePath)

	exitCode, stdout, stderr := captureRun(t, []string{
		"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath,
	})
	if exitCode != 0 {
		t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	pdfPath := strings.TrimSuffix(invoicePath, ".yaml") + ".pdf"
	if want := "Built " + pdfPath + " for CUST-001 (CUST-001-001)\n"; stderr != want {
		t.Fatalf("stderr = %q, want %q", stderr, want)
	}
	if want := pdfPath + "\n"; stdout != want {
		t.Fatalf("stdout = %q, want %q", stdout, want)
	}
	if got := readFileForTest(t, invoicePath); got != original {
		t.Fatalf("build changed the archived invoice:\n%s", got)
	}
}

func TestBuildArchiveReplacingArchivedInvoiceNeedsYes(t *testing.T) {
	customersPath, issuerPath, fixturePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicWritePDF)
	archiveDir := t.TempDir()
	writeConfigFile(t, "archive:\n  dir: "+quoteYAMLString(archiveDir)+"\n")
	archivedPath := filepath.Join(archiveDir, "first.yaml")
	writeBuildableInvoice(t, fixturePath, archivedPath, "archived")
	original := readFileForTest(t, archivedPath)

	workDir := t.TempDir()
	chdirForTest(t, workDir)
	exitCode, _, stderr := captureRun(t, []string{"archive", "edit", "first.yaml"})
	if exitCode != 0 {
		t.Fatalf("archive edit: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	buildArgs := []string{"build", "first.yaml", "--archive", "-c", customersPath, "-u", issuerPath, "-t", templatePath}

	exitCode, stdout, stderr := captureRun(t, buildArgs)
	if exitCode != 2 {
		t.Fatalf("exitCode = %d, want 2, stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if stdout != "" {
		t.Fatalf("stdout = %q, want empty", stdout)
	}
	if want := "error: built first.pdf but archiving first.yaml replaces archived invoice " + archivedPath + "; pass --yes to replace it"; !strings.Contains(stderr, want) {
		t.Fatalf("stderr = %q, want it to contain %q", stderr, want)
	}
	if got := readFileForTest(t, archivedPath); got != original {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(archiveDir, ".history")); !os.IsNotExist(err) {
		t.Fatalf("no backup should be written, Stat err = %v", err)
	}

	exitCode, stdout, stderr = captureRun(t, append(buildArgs, "--yes"))
	if exitCode != 0 {
		t.Fatalf("--yes: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	if want := "first.pdf\n"; stdout != want {
		t.Fatalf("--yes: stdout = %q, want %q", stdout, want)
	}
	backups, err := filepath.Glob(filepath.Join(archiveDir, ".history", "first.*.yaml"))
	if err != nil || len(backups) != 1 {
		t.Fatalf("backups = %q (err %v), want exactly one", backups, err)
	}
	if want := "Replaced archived invoice " + archivedPath + "; previous version kept at " + backups[0] + "\n" +
		"Built first.pdf for CUST-001 (CUST-001-001)\nArchived first.yaml -> " + archivedPath + "\n"; stderr != want {
		t.Fatalf("--yes: stderr = %q, want %q", stderr, want)
	}
	if got := readFileForTest(t, backups[0]); got != original {
		t.Fatalf("backup = %q, want %q", got, original)
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
