package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"invox/internal/invoice"
)

func runConfig(args []string) int {
	spec := configSpec()

	if wantsHelp(args) {
		printConfigHelp(os.Stdout)
		return 0
	}
	if len(args) > 0 {
		printCommandError(os.Stderr, spec, fmt.Sprintf("unexpected arguments: %s", strings.Join(args, " ")))
		return 2
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

	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Printf("Opened %s\n", invoice.DisplayPath(configPath, baseDir))
	return 0
}
