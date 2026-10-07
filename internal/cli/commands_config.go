package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runConfig(ios *iostreams.IOStreams, args []string) error {
	spec := configSpec()

	if wantsHelp(args) {
		printConfigHelp(ios.Out)
		return nil
	}
	if len(args) > 0 {
		return cmdutil.FlagErrorf(spec.Name, "unexpected arguments: %s", strings.Join(args, " "))
	}

	configPath, err := invoice.EditableConfigPath()
	if err != nil {
		return err
	}

	if err := openTextFile(ios, configPath); err != nil {
		return fmt.Errorf("failed to open %s: %w", configPath, err)
	}

	baseDir, err := os.Getwd()
	if err != nil {
		return err
	}
	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		return err
	}

	fmt.Fprintf(ios.ErrOut, "Opened %s\n", invoice.DisplayPath(configPath, baseDir))
	return nil
}
