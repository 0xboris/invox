// Package factorytest builds the Factory that main builds, on temporary
// directories, for tests of the commands.
package factorytest

import (
	"errors"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/store"
)

// Now is the clock of the Factory New returns, unless Options.Now is set.
var Now = time.Date(2026, 3, 6, 0, 0, 0, 0, time.UTC)

// Options say what New's environment has. Vars are the environment
// variables; GOOS defaults to linux and Home to a temporary directory.
type Options struct {
	GOOS string
	Home string
	// NoHome makes the home directory unknown.
	NoHome bool
	// ConfigDir is INVOX_CONFIG_DIR, unless Vars sets it.
	ConfigDir string
	Vars      map[string]string
	Getwd     func() (string, error)
	Runner    run.Runner
	// Now is the clock; the zero time means the package's Now.
	Now time.Time
}

// New returns a Factory on ios whose environment is opts. A nil Runner runs
// nothing: it is a run.Stub.
func New(t *testing.T, ios *iostreams.IOStreams, opts Options) *cmdutil.Factory {
	t.Helper()
	if ios == nil {
		ios, _, _, _ = iostreams.Test()
	}
	if opts.GOOS == "" {
		opts.GOOS = "linux"
	}
	if opts.Home == "" && !opts.NoHome {
		opts.Home = t.TempDir()
	}
	if opts.Runner == nil {
		opts.Runner = run.NewStub(t)
	}
	if opts.Now.IsZero() {
		opts.Now = Now
	}
	if opts.Getwd == nil {
		work := t.TempDir()
		opts.Getwd = func() (string, error) { return work, nil }
	}
	e := env.Env{
		GOOS: opts.GOOS,
		Getenv: func(key string) string {
			if value, ok := opts.Vars[key]; ok || key != "INVOX_CONFIG_DIR" {
				return value
			}
			return opts.ConfigDir
		},
		HomeDir: func() (string, error) {
			if opts.NoHome {
				return "", errors.New("home directory unknown")
			}
			return opts.Home, nil
		},
		Getwd: opts.Getwd,
		Now:   func() time.Time { return opts.Now },
	}
	return factory.New(ios, opts.Runner, e)
}

// SetStatus sets invoice.status in the invoice at path, as a build or an
// archive step would.
func SetStatus(path string, status invoice.Status) error {
	return (&store.Store{}).Update(path, func(inv *invoice.Invoice) error {
		inv.Header.Status = invoice.Text(status)
		return nil
	})
}
