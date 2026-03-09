package cli

import (
	"fmt"
	"os"

	"invox/internal/invoice"
)

func runConfig(args []string) int {
	spec := configSpec()

	opts, _, exitCode, ok := parseCommand(spec, args)
	if !ok {
		return exitCode
	}

	configPath, err := invoice.EditableConfigPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	if err := openTextFile(configPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("Opened %s\n", invoice.DisplayPath(configPath, opts.BaseDir))
	return 0
}
