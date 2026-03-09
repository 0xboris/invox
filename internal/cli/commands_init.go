package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"invox/internal/invoice"
)

func runInit(args []string) int {
	spec := initSpec()

	_, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	configDir, results, err := invoice.InitializeConfigDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	configDir, err = filepath.Abs(configDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("Initialized %s\n", configDir)
	for _, result := range results {
		status := "exists"
		if result.Created {
			status = "created"
		}
		fmt.Printf("%s %s\n", status, invoice.DisplayPath(result.Path, configDir))
	}
	return 0
}
