package config

import "github.com/0xboris/invox/internal/sharedui/cmdutil"

// Topics returns the help topics of the config slice. The config topic is
// the help of the config command.
func Topics() []cmdutil.Topic {
	return []cmdutil.Topic{{Name: "config", Short: "config.yaml keys, precedence, and email placeholders"}}
}
