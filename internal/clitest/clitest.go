// Package clitest runs invox commands in tests the way main does, through
// cli.Main, on a Factory from factorytest: the user directories are a
// testfixture.Host under temporary directories, programs run on a
// runtest.Stub, and the clock is factorytest's. A command's tests use it to
// pin what the command prints and the code it exits with.
package clitest

import (
	"bytes"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/testfixture"
)

// The Factory runs as on Linux with VISUAL and EDITOR unset, where the
// editor is vi and documents open with xdg-open.
const (
	editor = "vi"
	opener = "xdg-open"
)

// Invox is one user's invox: their directories, environment and terminal.
// Each Run starts invox anew in it, as a shell would.
type Invox struct {
	t *testing.T
	// Host holds config.yaml and, unless it configures another, the archive.
	Host testfixture.Host
	// Stub runs the programs invox starts. A test registers each run it
	// expects; any other run panics.
	Stub *runtest.Stub
	// IO is the terminal of the next Run. A test sets stdin and the TTY
	// flags on it; after each Run it is a fresh one.
	IO *iostreams.IOStreams
	// GOOS is the OS invox runs as; "" is linux.
	GOOS string

	vars  map[string]string
	getwd func() (string, error)
}

// New returns an Invox whose directories are under temporary directories
// and whose working directory is a new temporary directory.
func New(t *testing.T) *Invox {
	t.Helper()
	ios, _, _, _ := iostreams.Test()
	work := t.TempDir()
	h := testfixture.NewHost(t)
	return &Invox{
		t:     t,
		Host:  h,
		Stub:  runtest.NewStub(t),
		IO:    ios,
		vars:  map[string]string{"XDG_CONFIG_HOME": h.ConfigHome},
		getwd: func() (string, error) { return work, nil },
	}
}

// WriteConfig writes source as the user's config.yaml and returns its path.
func (x *Invox) WriteConfig(source string) string {
	x.t.Helper()
	return x.Host.WriteConfig(x.t, source)
}

// Chdir makes dir the working directory of the test and of invox, so
// relative paths mean the same to both.
func (x *Invox) Chdir(dir string) {
	x.t.Chdir(dir)
	x.getwd = func() (string, error) { return dir, nil }
}

// Setenv sets the environment variable key for every later Run.
func (x *Invox) Setenv(key, value string) {
	x.vars[key] = value
}

// Factory returns the Factory a Run would run invox with, on x.IO.
func (x *Invox) Factory() *cmdutil.Factory {
	x.t.Helper()
	return factorytest.New(x.t, x.IO, factorytest.Options{
		GOOS:   x.GOOS,
		Home:   x.Host.Home,
		Vars:   maps.Clone(x.vars),
		Getwd:  x.getwd,
		Runner: x.Stub,
	})
}

// Run runs invox with args and returns its exit code, stdout and stderr.
func (x *Invox) Run(args []string) (int, string, string) {
	x.t.Helper()
	f := x.Factory()
	x.IO, _, _, _ = iostreams.Test()
	exitCode := cli.Main(args, f)
	return exitCode, f.IOStreams.Out.(*bytes.Buffer).String(), f.IOStreams.ErrOut.(*bytes.Buffer).String()
}

// Stdin makes the next Run read input from stdin. With terminal set, stdin
// and stderr are terminals, so invox may prompt.
func (x *Invox) Stdin(terminal bool, input string) {
	ios, in, _, _ := iostreams.Test()
	ios.SetStdinTTY(terminal)
	ios.SetStderrTTY(terminal)
	in.WriteString(input)
	x.IO = ios
}

// ExpectEditor makes the next Run's stdin and stderr terminals, so the
// editor may open, and expects one editor run that returns err. It returns
// where the path the editor opens is recorded.
func (x *Invox) ExpectEditor(err error) *string {
	x.IO.SetStdinTTY(true)
	x.IO.SetStderrTTY(true)
	opened := new(string)
	x.Stub.Register(editor, func(cmd run.Cmd) error {
		*opened = cmd.Args[len(cmd.Args)-1]
		return err
	})
	return opened
}

// ExpectOpener expects one run of the program that opens documents, which
// returns err. It returns where the path it opens is recorded.
func (x *Invox) ExpectOpener(err error) *string {
	opened := new(string)
	x.Stub.Register(opener, func(cmd run.Cmd) error {
		*opened = cmd.Args[0]
		return err
	})
	return opened
}

// ExpectTectonic expects one tectonic run, which writes an empty PDF next to
// the .tex file it compiles, as tectonic does. Like a real child process it
// is handed invox's stdin, so input the test gave is gone after it.
func (x *Invox) ExpectTectonic() {
	x.Stub.Register("tectonic", func(cmd run.Cmd) error {
		if cmd.Stdin != nil {
			_, _ = io.Copy(io.Discard, cmd.Stdin)
		}
		pdf := strings.TrimSuffix(cmd.Args[0], filepath.Ext(cmd.Args[0])) + ".pdf"
		testfixture.WriteFile(x.t, filepath.Join(cmd.Dir, pdf), "")
		return nil
	})
}

// ExpectTectonicFailure expects one tectonic run, which prints an error and
// exits with code.
func (x *Invox) ExpectTectonicFailure(code int) {
	x.Stub.Register("tectonic", func(cmd run.Cmd) error {
		_, _ = cmd.Stderr.Write([]byte("fake tectonic: forced failure\n"))
		return &run.ExecError{Name: "tectonic", Code: code}
	})
}

// EditedArchive is the archived invoice first.yaml, checked out with
// archive edit into the working directory and changed there.
type EditedArchive struct {
	ArchiveDir   string
	ArchivedPath string
	// Original is the archived file's content before any replacement.
	Original    string
	WorkingCopy string
}

// EditArchive configures an archive holding first.yaml, CUST-001-001,
// checks it out with archive edit into a new working directory and raises
// its unit price there, so archiving the working copy replaces it.
func (x *Invox) EditArchive() EditedArchive {
	x.t.Helper()
	archiveDir := x.t.TempDir()
	x.WriteConfig("archive:\n  dir: " + testfixture.QuoteYAML(archiveDir) + "\n")
	archivedPath := testfixture.WriteNumberedInvoice(x.t, archiveDir, "first.yaml", "CUST-001-001", "archived")

	workDir := x.t.TempDir()
	x.Chdir(workDir)
	if exitCode, _, stderr := x.Run([]string{"archive", "edit", "first.yaml"}); exitCode != 0 {
		x.t.Fatalf("archive edit: exitCode = %d, want 0, stderr=%q", exitCode, stderr)
	}
	workingCopy := filepath.Join(workDir, "first.yaml")
	testfixture.WriteFile(x.t, workingCopy, strings.Replace(testfixture.ReadFile(x.t, workingCopy), "unit_price: 100", "unit_price: 250", 1))
	return EditedArchive{
		ArchiveDir:   archiveDir,
		ArchivedPath: archivedPath,
		Original:     testfixture.ReadFile(x.t, archivedPath),
		WorkingCopy:  workingCopy,
	}
}

// AssertUnchanged fails t unless the archive and the working copy are as
// EditArchive left them, with no backup written.
func (e EditedArchive) AssertUnchanged(t *testing.T) {
	t.Helper()
	if got := testfixture.ReadFile(t, e.ArchivedPath); got != e.Original {
		t.Fatalf("archived invoice changed:\n%s", got)
	}
	if _, err := os.Stat(e.WorkingCopy); err != nil {
		t.Fatalf("working copy should stay in place: %v", err)
	}
	if _, err := os.Stat(filepath.Join(e.ArchiveDir, ".history")); !os.IsNotExist(err) {
		t.Fatalf("no backup should be written, Stat err = %v", err)
	}
}

// AssertReplaced fails t unless the working copy replaced the archived
// invoice and the previous version is kept in .history, and returns the
// backup's path.
func (e EditedArchive) AssertReplaced(t *testing.T) string {
	t.Helper()
	if !strings.Contains(testfixture.ReadFile(t, e.ArchivedPath), "unit_price: 250") {
		t.Fatalf("archived invoice was not replaced:\n%s", testfixture.ReadFile(t, e.ArchivedPath))
	}
	if _, err := os.Stat(e.WorkingCopy); !os.IsNotExist(err) {
		t.Fatalf("working copy should be removed, Stat err = %v", err)
	}
	backups, err := filepath.Glob(filepath.Join(e.ArchiveDir, ".history", "first.*.yaml"))
	if err != nil {
		t.Fatalf("Glob returned error: %v", err)
	}
	if len(backups) != 1 {
		t.Fatalf("backups = %q, want exactly one", backups)
	}
	if got := testfixture.ReadFile(t, backups[0]); got != e.Original {
		t.Fatalf("backup = %q, want the previous version %q", got, e.Original)
	}
	return backups[0]
}

// ReplacedNotice is what invox prints when it replaced the archived invoice
// and kept the previous version at backupPath.
func (e EditedArchive) ReplacedNotice(backupPath string) string {
	return "Replaced archived invoice " + e.ArchivedPath + "; previous version kept at " + backupPath + "\n"
}
