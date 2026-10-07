package invoice

import (
	"path/filepath"
	"strings"
)

// HostInputs are the parts of the environment the user directories come
// from. Home is "" when the home directory is unknown.
type HostInputs struct {
	GOOS          string
	Home          string
	XDGConfigHome string
	XDGDataHome   string
	AppData       string
}

// Host holds the user directories invox reads and writes, resolved once.
type Host struct {
	home       string
	configBase string
	dataBase   string
}

func NewHost(in HostInputs) Host {
	h := Host{home: in.Home}
	homeKnown := strings.TrimSpace(in.Home) != ""

	if xdg := strings.TrimSpace(in.XDGConfigHome); xdg != "" {
		h.configBase = xdg
	} else if homeKnown {
		h.configBase = filepath.Join(in.Home, ".config")
	}

	switch in.GOOS {
	case "darwin":
		if homeKnown {
			h.dataBase = filepath.Join(in.Home, "Library", "Application Support")
		}
	case "windows":
		if appData := strings.TrimSpace(in.AppData); appData != "" {
			h.dataBase = appData
		} else if homeKnown {
			h.dataBase = filepath.Join(in.Home, "AppData", "Roaming")
		}
	default:
		if xdg := strings.TrimSpace(in.XDGDataHome); xdg != "" {
			h.dataBase = xdg
		} else if homeKnown {
			h.dataBase = filepath.Join(in.Home, ".local", "share")
		}
	}
	return h
}
