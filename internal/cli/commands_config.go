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
	h := userHost(e)
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

	if err := f.Editor.Edit(ctx, configPath); err != nil {
		return fmt.Errorf("failed to open %s: %w", configPath, err)
	}

	baseDir, err := e.Getwd()
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
