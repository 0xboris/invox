package cli

import (
	"fmt"
	"path/filepath"

	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runInit(ios *iostreams.IOStreams, e env.Env, args []string) error {
	h := userHost(e)
	spec := initSpec()

	_, _, err := parseCommand(ios, e, spec, args)
	if err != nil {
		return err
	}

	configDir, results, err := h.InitializeConfigDir()
	if err != nil {
		return err
	}

	configDir, err = filepath.Abs(configDir)
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.ErrOut, "Initialized %s\n", configDir)
	for _, result := range results {
		status := "exists"
		if result.Created {
			status = "created"
		}
		fmt.Fprintf(ios.ErrOut, "%s %s\n", status, invoice.DisplayPath(result.Path, configDir))
	}
	return nil
}
