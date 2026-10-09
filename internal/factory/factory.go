// Package factory builds invox: the store, the adapters and the use cases,
// wired into the cmdutil.Factory the commands use.
package factory

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/0xboris/invox/internal/adapters/applemail"
	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/opener"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/adapters/tectonic"
	"github.com/0xboris/invox/internal/archive"
	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/email"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/render/latex"
	"github.com/0xboris/invox/internal/store"
)

// New returns the Factory of invox: its adapters run programs with runner
// on the OS e.GOOS, read the editor settings with e.Getenv, and keep the
// user's files where e says. It is the composition root: the only place
// that picks the implementation of each billing port.
func New(ios *iostreams.IOStreams, runner run.Runner, e env.Env) *cmdutil.Factory {
	f := &cmdutil.Factory{
		IOStreams: ios,
		Env:       e,
		Editor:    editor.New(runner, ios, e.GOOS, e.Getenv),
		Opener:    opener.New(runner, ios, e.GOOS),
	}
	// The user directories are resolved on the first call, so that a test
	// can set up its environment after building the Factory.
	host := sync.OnceValue(func() store.Host { return newHost(e, f.ConfigFile) })
	compiler := tectonic.New(runner, ios, e.GOOS)
	var mailApp *applemail.Composer
	if e.GOOS == "darwin" {
		mailApp = applemail.New(runner, ios)
	}
	f.Service = func(files cmdutil.Files) *billing.Service {
		h := host()
		var mailer billing.Mailer = email.Mailer{}
		if mailApp != nil && !files.EmailOutput {
			mailer = mailApp
		}
		st := &store.Store{Host: h, Getwd: e.Getwd, Files: store.Files{Customers: files.Customers, Issuer: files.Issuer, Defaults: files.Defaults}}
		return &billing.Service{
			Invoices:  st,
			Directory: st,
			Archives:  archive.Archive{Locate: h.ResolveArchiveDir, Read: store.ReadArchived, Rewrite: st.Rewrite},
			Renderer:  latex.Renderer{},
			Compiler:  compiler,
			Mailer:    mailer,
			Settings:  h.Settings,
			Now:       e.Now,
		}
	}
	return f
}

// newHost resolves the user directories. Relative directory values are
// ignored, as the XDG spec asks, so every directory the Host holds is
// absolute. configFile and INVOX_CONFIG_DIR are made absolute against the
// working directory.
func newHost(e env.Env, configFile string) store.Host {
	home, err := e.HomeDir()
	if err != nil {
		home = ""
	}
	cwd, err := e.Getwd()
	if err != nil {
		cwd = ""
	}
	return store.NewHost(store.HostInputs{
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
