package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runConfig(ios *iostreams.IOStreams, args []string) int {
	spec := configSpec()

	if wantsHelp(args) {
		printConfigHelp(ios.Out)
		return 0
	}
	if len(args) > 0 {
		printCommandError(ios.ErrOut, spec, fmt.Sprintf("unexpected arguments: %s", strings.Join(args, " ")))
		return 2
	}

	configPath, err := invoice.EditableConfigPath()
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	if err := openTextFile(ios, configPath); err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	baseDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}
	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		fmt.Fprintln(ios.ErrOut, err)
		return 1
	}

	fmt.Fprintf(ios.Out, "Opened %s\n", invoice.DisplayPath(configPath, baseDir))
	return 0
}
