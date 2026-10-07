package cli

import (
	"fmt"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"path/filepath"

	"github.com/0xboris/invox/internal/invoice"
)

func runInit(f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	h := f.Host()
	spec := initSpec()

	_, _, err := parseCommand(f, spec, args)
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
