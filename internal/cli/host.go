package cli

import (
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/invoice"
)

func userHost(e env.Env) invoice.Host {
	home, err := e.HomeDir()
	if err != nil {
		home = ""
	}
	return invoice.NewHost(invoice.HostInputs{
		GOOS:          e.GOOS,
		Home:          home,
		XDGConfigHome: e.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:   e.Getenv("XDG_DATA_HOME"),
		AppData:       e.Getenv("APPDATA"),
	})
}
