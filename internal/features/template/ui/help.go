package template

import (
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/domain/schema"
	"github.com/0xboris/invox/internal/sharedui/cmdutil"
)

// Topics returns the help topics of the template slice. The template topic
// is the help of the template command.
func Topics() []cmdutil.Topic {
	return []cmdutil.Topic{{Name: "template", Short: "template placeholders and authoring rules"}}
}

// templatePlaceholderReference is the table of template placeholders.
func templatePlaceholderReference() string {
	var w strings.Builder
	fmt.Fprintf(&w, "Available .tex placeholders:\n")
	for _, group := range schema.TemplatePlaceholders {
		fmt.Fprintf(&w, "  %s:\n", group.Title)
		for _, placeholder := range group.Placeholders {
			fmt.Fprintf(&w, "    %-32s %s\n", placeholder.Token, placeholder.Description)
		}
	}
	fmt.Fprintf(&w, "\n")
	return w.String()
}
