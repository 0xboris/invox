package cli

import (
	"fmt"
	"path/filepath"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runInit(ios *iostreams.IOStreams, args []string) int {
	spec := initSpec()

	_, _, exitCode, ok := parseCommand(ios, spec, args)
	if !ok {
		return exitCode
	}

	configDir, results, err := invoice.InitializeConfigDir()
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	configDir, err = filepath.Abs(configDir)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	fmt.Fprintf(ios.Out, "Initialized %s\n", configDir)
	for _, result := range results {
		status := "exists"
		if result.Created {
			status = "created"
		}
		fmt.Fprintf(ios.Out, "%s %s\n", status, invoice.DisplayPath(result.Path, configDir))
	}
	return 0
}
