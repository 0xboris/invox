package cmdutil

import (
	"context"
	"errors"
	"fmt"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/iostreams"
)

// OpenInEditor opens path in ed for command. Without a terminal, or with
// prompting disabled, it starts no editor and returns a *FlagError that ends
// with nextStep, what the user can do instead.
func OpenInEditor(ctx context.Context, ios *iostreams.IOStreams, ed *editor.Editor, command, path, nextStep string) error {
	if reason := WhyNoPrompt(ios); reason != "" {
		return FlagErrorf(command, "cannot open an editor: %s; %s", reason, nextStep)
	}
	defer HoldInterrupt(ctx)()
	err := ed.Edit(ctx, path)
	var execErr *run.ExecError
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
