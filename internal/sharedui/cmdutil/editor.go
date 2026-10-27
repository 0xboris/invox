package cmdutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xboris/invox/internal/utilities/iostreams"
	"github.com/0xboris/invox/internal/utilities/proc"
)

// OpenInEditor opens path with edit, a slice's use case that runs the user's
// editor and returns when it exits. Without a terminal, or with prompting
// disabled, it starts no editor and returns a *FlagError for command that ends
// with nextStep, what the user can do instead.
func OpenInEditor(ctx context.Context, ios *iostreams.IOStreams, edit func(ctx context.Context, path string) error, command, path, nextStep string) error {
	if reason := WhyNoPrompt(ios); reason != "" {
		return FlagErrorf(command, "cannot open an editor: %s; %s", reason, nextStep)
	}
	defer proc.HoldInterrupt(ctx)()
	err := edit(ctx, path)
	var execErr *proc.ExitError
	if errors.As(err, &execErr) {
		return &ExecError{Program: fmt.Sprintf("editor %q", execErr.Name), Code: execErr.Code, Err: err}
	}
	return err
}

// WhyNoPrompt returns why ios cannot prompt, or "" when it can.
func WhyNoPrompt(ios *iostreams.IOStreams) string {
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
