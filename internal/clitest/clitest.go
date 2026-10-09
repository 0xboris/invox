// Package clitest runs invox commands in tests the way main does, through
// cli.Main, on a Factory from factorytest: the user directories are a
// testfixture.Host under temporary directories, programs run on a
// runtest.Stub, and the clock is factorytest's. A command's tests use it to
// pin what the command prints and the code it exits with.
package clitest

import (
	"bytes"
	"maps"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/run/runtest"
	"github.com/0xboris/invox/internal/cli"
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

// Chdir makes dir the working directory invox runs in.
func (x *Invox) Chdir(dir string) {
	x.getwd = func() (string, error) { return dir, nil }
}

// Setenv sets the environment variable key for every later Run.
func (x *Invox) Setenv(key, value string) {
	x.vars[key] = value
}

// Run runs invox with args and returns its exit code, stdout and stderr.
func (x *Invox) Run(args []string) (int, string, string) {
	x.t.Helper()
	ios := x.IO
	x.IO, _, _, _ = iostreams.Test()
	f := factorytest.New(x.t, ios, factorytest.Options{
		Home:   x.Host.Home,
		Vars:   maps.Clone(x.vars),
		Getwd:  x.getwd,
		Runner: x.Stub,
	})
	exitCode := cli.Main(args, f)
	return exitCode, ios.Out.(*bytes.Buffer).String(), ios.ErrOut.(*bytes.Buffer).String()
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
// the .tex file it compiles, as tectonic does.
func (x *Invox) ExpectTectonic() {
	x.Stub.Register("tectonic", func(cmd run.Cmd) error {
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
