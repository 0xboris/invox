package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

// The test Factory's adapters run as on Linux with VISUAL and EDITOR unset,
// where the editor is vi and documents open with xdg-open.
const (
	testGOOS   = "linux"
	testEditor = "vi"
	testOpener = "xdg-open"
)

func testFactory(t *testing.T) (*cmdutil.Factory, *run.Stub) {
	t.Helper()
	return testFactoryEnv(t, nil)
}

// testFactoryEnv is testFactory with the variables in vars set.
func testFactoryEnv(t *testing.T, vars map[string]string) (*cmdutil.Factory, *run.Stub) {
	t.Helper()
	return testFactoryOn(t, testGOOS, vars)
}

// testFactoryOn is testFactoryEnv on the OS goos.
func testFactoryOn(t *testing.T, goos string, vars map[string]string) (*cmdutil.Factory, *run.Stub) {
	t.Helper()

	ios, _, _, _ := iostreams.Test()
	stub := run.NewStub(t)
	e := env.System()
	e.GOOS = goos
	e.Getenv = func(key string) string {
		if value, ok := vars[key]; ok {
			return value
		}
		switch key {
		case "XDG_CONFIG_HOME", "XDG_DATA_HOME", "APPDATA":
			// isolateUserDirs points these at a temporary directory.
			return os.Getenv(key)
		}
		return ""
	}
	return factory.New(ios, stub, e), stub
}

// expectEditor makes f's stdin and stderr terminals, so the editor may open,
// and expects one editor run that returns err.
func expectEditor(f *cmdutil.Factory, stub *run.Stub, err error) *string {
	f.IOStreams.SetStdinTTY(true)
	f.IOStreams.SetStderrTTY(true)
	opened := new(string)
	stub.Register(testEditor, func(cmd run.Cmd) error {
		*opened = cmd.Args[len(cmd.Args)-1]
		return err
	})
	return opened
}

func expectOpener(stub *run.Stub, err error) *string {
	opened := new(string)
	stub.Register(testOpener, func(cmd run.Cmd) error {
		*opened = cmd.Args[0]
		return err
	})
	return opened
}

func captureRunFactory(t *testing.T, f *cmdutil.Factory, args []string) (int, string, string) {
	t.Helper()

	isolateUserDirs(t)
	exitCode := Main(args, f)
	return exitCode, f.IOStreams.Out.(*bytes.Buffer).String(), f.IOStreams.ErrOut.(*bytes.Buffer).String()
}
