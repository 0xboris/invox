package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
)

// The test Factory's adapters run as on Linux, where the editor runs through
// $SHELL and documents open with xdg-open.
const (
	testGOOS   = "linux"
	testShell  = "/bin/sh"
	testOpener = "xdg-open"
)

func testFactory(t *testing.T) (*cmdutil.Factory, *run.Stub) {
	t.Helper()

	ios, _, _, _ := iostreams.Test()
	stub := run.NewStub(t)
	e := env.System()
	e.GOOS = testGOOS
	e.Getenv = func(key string) string {
		switch key {
		case "SHELL":
			return testShell
		case "XDG_CONFIG_HOME", "XDG_DATA_HOME", "APPDATA":
			// isolateUserDirs points these at a temporary directory.
			return os.Getenv(key)
		}
		return ""
	}
	return cmdutil.NewFactory(ios, stub, e), stub
}

func expectEditor(stub *run.Stub, err error) *string {
	opened := new(string)
	stub.Register(testShell, func(cmd run.Cmd) error {
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
