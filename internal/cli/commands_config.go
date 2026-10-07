package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
)

func runConfig(ctx context.Context, f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	e := f.Env
	h := f.Host()
	spec := configSpec()

	if len(args) > 0 && args[0] == "paths" {
		return runConfigPaths(f, args[1:])
	}
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

	fmt.Fprintf(ios.ErrOut, "Opened %s\n", invoice.DisplayPath(configPath, baseDir))
	return nil
}

func runConfigPaths(f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	spec := configPathsSpec()
	if wantsHelp(args) {
		printConfigPathsHelp(ios.Out)
		return nil
	}
	if len(args) > 0 {
		return cmdutil.FlagErrorf(spec.Name, "unexpected arguments: %s", strings.Join(args, " "))
	}

	cwd, err := f.Env.Getwd()
	if err != nil {
		return err
	}
	reports, err := f.Host().Paths(cwd)
	if err != nil {
		return err
	}
	t := table{columns: []column{{header: "NAME"}, {header: "PATH"}, {header: "SOURCE"}}}
	for _, r := range reports {
		t.addRow(r.Name, r.Path, sourceWords[r.Source])
	}
	t.print(ios)
	return nil
}

// sourceWords are the SOURCE column of `config paths`.
var sourceWords = map[invoice.Source]string{
	invoice.SourceNone:     "none",
	invoice.SourceExplicit: "flag",
	invoice.SourceEnvDir:   "env",
	invoice.SourceDefault:  "default",
	invoice.SourceLegacy:   "legacy",
	invoice.SourceProject:  "project",
	invoice.SourceConfig:   "config",
}
