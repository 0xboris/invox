package app

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/adapters/programs/applemail"
	"github.com/0xboris/invox/internal/adapters/programs/run"
	"github.com/0xboris/invox/internal/domain/invoice"
	"github.com/0xboris/invox/internal/domain/ports"
	"github.com/0xboris/invox/internal/utilities/env"
	"github.com/0xboris/invox/internal/utilities/iostreams"
)

// mailComposer returns Apple Mail on macOS and nil elsewhere.
func mailComposer(runner run.Runner, ios *iostreams.IOStreams, goos string) ports.MailComposer {
	if goos != "darwin" {
		return nil
	}
	return applemail.New(runner, ios)
}

// hostResolver returns the func that resolves the user directories from e
// when a command first needs them, so that a test can set up its
// environment after building the Dependencies. Relative directory values are
// ignored, as the XDG spec asks, so every directory the Host holds is
// absolute. configFile and INVOX_CONFIG_DIR are made absolute against the
// working directory.
func hostResolver(e env.Env, files ports.Files, starter fs.FS) func(configFile string) invoice.Host {
	return func(configFile string) invoice.Host {
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
			Files:         files,
			Starter:       starter,
		})
	}
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
