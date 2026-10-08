package cmdutil

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/tectonic"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

// Factory holds what commands use to reach outside invox: the standard
// streams, the process environment and the external programs.
type Factory struct {
	IOStreams *iostreams.IOStreams
	Env       env.Env
	Compiler  *tectonic.Compiler
	Editor    *editor.Editor
	Opener    *opener.Opener
	// Mailer is nil where Apple Mail is not available.
	Mailer *applemail.Composer
	// ConfigFile is the --config value as typed, "" when not given. Main
	// sets it before anything calls Host.
	ConfigFile string

	host func() invoice.Host
}

// NewFactory returns a Factory whose adapters run programs with runner on the
// OS e.GOOS and read the editor settings with e.Getenv.
func NewFactory(ios *iostreams.IOStreams, runner run.Runner, e env.Env) *Factory {
	f := &Factory{
		IOStreams: ios,
		Env:       e,
		Compiler:  tectonic.New(runner, ios, e.GOOS),
		Editor:    editor.New(runner, ios, e.GOOS, e.Getenv),
		Opener:    opener.New(runner, ios, e.GOOS),
	}
	f.host = sync.OnceValue(func() invoice.Host { return newHost(e, f.ConfigFile) })
	if e.GOOS == "darwin" {
		f.Mailer = applemail.New(runner, ios)
	}
	return f
}

// Host returns the user directories, resolved from Env on the first call so
// that a test can set up its environment after building the Factory.
func (f *Factory) Host() invoice.Host {
	return f.host()
}

// newHost resolves the user directories. Relative directory values are
// ignored, as the XDG spec asks, so every directory the Host holds is
// absolute. configFile and INVOX_CONFIG_DIR are made absolute against the
// working directory.
func newHost(e env.Env, configFile string) invoice.Host {
	home, err := e.HomeDir()
	if err != nil {
		home = ""
	}
	cwd, err := e.Getwd()
	if err != nil {
		cwd = ""
	}
	return invoice.NewHost(invoice.HostInputs{
		GOOS:          e.GOOS,
		Home:          absOnly(home),
		XDGConfigHome: absOnly(e.Getenv("XDG_CONFIG_HOME")),
		XDGDataHome:   absOnly(e.Getenv("XDG_DATA_HOME")),
		AppData:       absOnly(e.Getenv("APPDATA")),
		ConfigDir:     absAgainst(cwd, e.Getenv("INVOX_CONFIG_DIR")),
		ConfigFile:    absAgainst(cwd, configFile),
	})
}

func absOnly(path string) string {
	path = strings.TrimSpace(path)
	if !filepath.IsAbs(path) {
		return ""
	}
	return filepath.Clean(path)
}

func absAgainst(cwd, path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return filepath.Clean(path)
	}
	return filepath.Join(cwd, path)
}
