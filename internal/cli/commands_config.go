package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
)

func runConfig(ctx context.Context, f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	e := f.Env
	h := f.Host()
	spec := configSpec()

	if wantsHelp(args) {
		printConfigHelp(ios.Out, h)
		return nil
	}
	if len(args) > 0 {
		return cmdutil.FlagErrorf(spec.Name, "unexpected arguments: %s", strings.Join(args, " "))
	}

	configPath, err := h.EditableConfigPath()
	if err != nil {
		return err
	}

	baseDir, err := e.Getwd()
	if err != nil {
		return err
	}
	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		return err
	}
	displayPath := invoice.DisplayPath(configPath, baseDir)

	if err := openInEditor(ctx, f, spec.Name, configPath, "edit "+displayPath+" directly"); err != nil {
		return fmt.Errorf("failed to open %s: %w", configPath, err)
	}

	fmt.Fprintf(ios.ErrOut, "Opened %s\n", displayPath)
	return nil
}
