package cmdutil

import "io"

// Topic is a page of `invox help NAME`. Print is nil for a topic that is the
// help of the command with the same name. A slice declares the topics about
// its own files, and the composition root hands them to the root command.
type Topic struct {
	Name    string
	Aliases []string
	Short   string
	Print   func(w io.Writer)
}

// LookupTopic returns the topic in topics called name or one of its aliases.
func LookupTopic(topics []Topic, name string) (Topic, bool) {
	for _, topic := range topics {
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
