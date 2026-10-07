package cli

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/iostreams"
)

// openInEditor opens path in the user's editor for command. Without a
// terminal, or with prompting disabled, it starts no editor and returns a
// *cmdutil.FlagError that ends with nextStep, what the user can do instead.
func openInEditor(ctx context.Context, f *cmdutil.Factory, command, path, nextStep string) error {
	if reason := whyNoPrompt(f.IOStreams); reason != "" {
		return cmdutil.FlagErrorf(command, "cannot open an editor: %s; %s", reason, nextStep)
	}
	release := holdInterrupt(ctx)
	err := f.Editor.Edit(ctx, path)
	release()
	var execErr *run.ExecError
	if errors.As(err, &execErr) {
		return &cmdutil.ExecError{Program: fmt.Sprintf("editor %q", execErr.Name), Code: execErr.Code, Err: err}
	}
	return err
}

// whyNoPrompt returns why ios cannot prompt, or "" when it can.
func whyNoPrompt(ios *iostreams.IOStreams) string {
	switch {
	case ios.CanPrompt():
		return ""
	case ios.PromptDisabled():
		return "prompts are disabled (--no-input or INVOX_PROMPT_DISABLED)"
	case !ios.IsStdinTTY():
		return "stdin is not a terminal"
	default:
		return "stderr is not a terminal"
	}
}
