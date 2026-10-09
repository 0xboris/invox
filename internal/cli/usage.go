package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/cli/helptext"
)

// writeHelp writes the help page of cmd, all of it generated from the
// command tree: the Long text (or the Short one), then usage, subcommands,
// flags and examples. The root page also lists the help topics.
func writeHelp(w io.Writer, cmd *cobra.Command, l helptext.Locations) error {
	description := cmd.Long
	if description == "" {
		description = cmd.Short + "."
	}
	var page strings.Builder
	if err := helptext.Render(&page, description, l); err != nil {
		return fmt.Errorf("help for %s: %w", cmd.CommandPath(), err)
	}
	text := strings.TrimRight(page.String(), "\n") + "\n"
	section := func(title, body string) {
		if body != "" {
			text += "\n" + title + ":\n" + body
		}
	}

	section("Usage", usageLines(cmd))
	if cmd.HasParent() {
		section("Commands", commandList(cmd.Commands(), ""))
	} else {
		for _, group := range cmd.Groups() {
			section(group.Title, commandList(cmd.Commands(), group.ID))
		}
		section("Additional commands", commandList(cmd.Commands(), ""))
		section("Help topics", topicList())
	}
	if cmd.HasParent() {
		section("Flags", cmd.LocalFlags().FlagUsages())
		section("Global flags", cmd.InheritedFlags().FlagUsages())
	} else {
		section("Flags", cmd.LocalFlags().FlagUsages())
	}
	if fields, ok := cmd.Annotations[cmdutil.JSONFieldsAnnotation]; ok {
		section("JSON fields", "  "+strings.ReplaceAll(fields, ",", ", ")+"\n")
	}
	section("Examples", indent(cmd.Example))
	if !cmd.HasParent() {
		section("Learn more", "  Run `invox help <command>` for more information about a command.\n"+
			"  Run `invox help <topic>` to read a help topic.\n")
	}
	_, err := io.WriteString(w, text)
	return err
}

func usageLines(cmd *cobra.Command) string {
	if !cmd.HasParent() {
		return "  " + cmd.Name() + " <command> [flags]\n"
	}
	// Every command has flags, if only --help, but cmd.UseLine leaves out
	// [flags] until cobra has parsed them, which `invox help CMD` doesn't.
	lines := "  " + cmd.CommandPath() + strings.TrimPrefix(cmd.Use, cmd.Name()) + " [flags]\n"
	if cmd.HasAvailableSubCommands() && !strings.Contains(cmd.Use, "<subcommand>") {
		lines += "  " + cmd.CommandPath() + " <subcommand> [flags]\n"
	}
	return lines
}

// commandList lists the available commands in group, or those in no group
// when group is "".
func commandList(commands []*cobra.Command, group string) string {
	var available []*cobra.Command
	width := 0
	for _, c := range commands {
		if c.IsAvailableCommand() && c.GroupID == group {
			available = append(available, c)
			width = max(width, len(c.Name()))
		}
	}
	var list strings.Builder
	for _, c := range available {
		fmt.Fprintf(&list, "  %-*s  %s\n", width, c.Name(), c.Short)
	}
	return list.String()
}

func topicList() string {
	width := 0
	for _, topic := range helptext.Topics {
		width = max(width, len(topic.Name))
	}
	var list strings.Builder
	for _, topic := range helptext.Topics {
		fmt.Fprintf(&list, "  %-*s  %s\n", width, topic.Name, topic.Short)
	}
	return list.String()
}

func indent(text string) string {
	var out strings.Builder
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		if line != "" {
			out.WriteString("  " + line)
		}
		out.WriteString("\n")
	}
	if strings.TrimSpace(out.String()) == "" {
		return ""
	}
	return out.String()
}

// helpTopic writes the page `invox help ARGS` names: a topic, or the help of
// a command. Anything else, including `help` itself, is an unknown topic.
func helpTopic(w io.Writer, root *cobra.Command, l helptext.Locations, args []string) error {
	if len(args) == 0 {
		return writeHelp(w, root, l)
	}
	if topic, ok := helptext.LookupTopic(args[0]); ok && len(args) == 1 && topic.Print != nil {
		topic.Print(w, l)
		return nil
	}
	cmd, rest, err := root.Find(args)
	if err != nil || cmd == root || len(rest) > 0 || cmd.Parent() == root && cmd.Name() == "help" || !cmd.IsAvailableCommand() {
		return cmdutil.FlagErrorf("", "unknown help topic %q", strings.Join(args, " "))
	}
	return writeHelp(w, cmd, l)
}

// helpCompletions completes `invox help ARGS`: the subcommands of the
// command ARGS names, and the help topics after `help` alone.
func helpCompletions(root *cobra.Command, args []string) []cobra.Completion {
	cmd, rest, err := root.Find(args)
	if err != nil || len(rest) > 0 {
		return nil
	}
	var names []cobra.Completion
	for _, sub := range cmd.Commands() {
		if sub.IsAvailableCommand() {
			names = append(names, cobra.CompletionWithDesc(sub.Name(), sub.Short))
		}
	}
	if cmd == root {
		for _, topic := range helptext.Topics {
			if topic.Print != nil {
				names = append(names, cobra.CompletionWithDesc(topic.Name, topic.Short))
			}
		}
	}
	return names
}
