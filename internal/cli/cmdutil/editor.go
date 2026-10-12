package cmdutil

import (
	"context"

	"github.com/0xboris/invox/internal/adapters/editor"
	"github.com/0xboris/invox/internal/iostreams"
)

// OpenInEditor opens path in ed. Without a terminal, or with
// prompting disabled, it starts no editor and returns a *FlagError that ends
// with nextStep, what the user can do instead.
func OpenInEditor(ctx context.Context, ios *iostreams.IOStreams, ed *editor.Editor, path, nextStep string) error {
	if reason := WhyNoPrompt(ios); reason != "" {
		return FlagErrorf("cannot open an editor: %s; %s", reason, nextStep)
	}
	defer HoldInterrupt(ctx)()
	return ed.Edit(ctx, path)
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
