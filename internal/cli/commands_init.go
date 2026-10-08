package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func runInit(ctx context.Context, f *cmdutil.Factory, args []string) error {
	ios := f.IOStreams
	h := f.Host()
	spec := initSpec()

	opts, _, err := parseCommand(f, spec, args)
	if err != nil {
		return err
	}
	if err := copyLegacyFiles(ctx, ios, h, spec, opts.OverwriteOutput); err != nil {
		return err
	}

	configDir, results, err := h.InitializeConfigDir()
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

// copyLegacyFiles copies the files of the deprecated config directory that
// the config directory lacks, after asking, unless force is set.
func copyLegacyFiles(ctx context.Context, ios *iostreams.IOStreams, h invoice.Host, spec commandSpec, force bool) error {
	missing, err := h.LegacyFilesToCopy()
	if err != nil || len(missing) == 0 {
		return err
	}
	legacyDir, configDir := h.LegacyConfigDir(), h.ConfigDir()
	if !force {
		if !ios.CanPrompt() {
			return cmdutil.FlagErrorf(spec.Name, "the deprecated config directory %s has files that %s lacks; pass --force to copy them (no terminal to ask on)", legacyDir, configDir)
		}
		confirmed, err := confirm(ctx, ios, fmt.Sprintf("Copy %s from %s to %s?", strings.Join(missing, ", "), legacyDir, configDir))
		if err != nil {
			return err
		}
		if !confirmed {
			fmt.Fprintf(ios.ErrOut, "not initialized; nothing was changed\n")
			return cmdutil.CancelError
		}
	}

	copied, err := h.CopyLegacyFiles()
	for _, rel := range copied {
		fmt.Fprintf(ios.ErrOut, "copied %s from %s\n", rel, legacyDir)
	}
	if err != nil {
		return err
	}
	fmt.Fprintf(ios.ErrOut, "invox no longer reads %s for these files; remove it once you are happy with %s\n", legacyDir, configDir)
	return nil
}
