package paths

import (
	"errors"
	"io"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestNewCmdPathsParsing(t *testing.T) {
	ios, _, _, _ := iostreams.Test()
	f := factory.New(ios, run.Exec{}, env.System())

	ran := false
	cmd := NewCmdPaths(f, func(*PathsOptions) error { ran = true; return nil })
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err != nil || !ran {
		t.Fatalf("Execute() = %v, runF ran = %v; want nil, true", err, ran)
	}

	ran = false
	cmd = NewCmdPaths(f, func(*PathsOptions) error { ran = true; return nil })
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"extra"})
	err := cmd.Execute()
	var flagErr *cmdutil.FlagError
	if !errors.As(err, &flagErr) || flagErr.Command != "config paths" || err.Error() != "unexpected arguments: extra" || ran {
		t.Fatalf("Execute(extra) = %#v, runF ran = %v; want FlagError for config paths and no run", err, ran)
	}
}
