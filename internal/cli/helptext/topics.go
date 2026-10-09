package helptext

import "embed"

// Topic is a page of `invox help NAME`. Its text is topics/NAME.tmpl, which
// Render fills in like a command's Long. A topic without that file is the
// help of the command with the same name.
type Topic struct {
	Name    string
	Aliases []string
	Short   string
}

// Topics lists the help topics in the order the root help shows them.
var Topics = []Topic{
	{Name: "config", Short: "config.yaml keys, precedence, and email placeholders"},
	{Name: "customers", Short: "customers.yaml fields, aliases, and example"},
	{Name: "issuer", Short: "issuer.yaml fields, validation rules, and example"},
	{Name: "defaults", Aliases: []string{"invoice-defaults", "invoice_defaults"}, Short: "invoice_defaults.yaml shape and new-command behavior"},
	{Name: "template", Short: "template placeholders and authoring rules"},
	{Name: "environment", Short: "environment variables, default directories, and precedence"},
	{Name: "exit-codes", Short: "what each exit status means"},
}

//go:embed topics/*.tmpl
var pages embed.FS

// Page returns the text of the topic's own page, and false when the topic is
// the help of a command.
func (t Topic) Page() (string, bool) {
	text, err := pages.ReadFile("topics/" + t.Name + ".tmpl")
	return string(text), err == nil
}

// LookupTopic returns the topic called name or one of its aliases.
func LookupTopic(name string) (Topic, bool) {
	for _, topic := range Topics {
		if topic.Name == name {
			return topic, true
		}
		for _, alias := range topic.Aliases {
			if alias == name {
				return topic, true
			}
		}
	}
	return Topic{}, false
}
