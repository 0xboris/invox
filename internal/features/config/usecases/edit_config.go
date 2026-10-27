// Package usecases holds the operations of the config commands. Each one
// reaches the outside of invox through the domain's ports, so the commands
// never name a port.
package usecases

import (
	"context"

	"github.com/0xboris/invox/internal/domain/ports"
)

// EditConfig opens config.yaml in the user's editor and returns when the
// editor exits.
type EditConfig struct {
	editor ports.Editor
}

func NewEditConfig(editor ports.Editor) EditConfig {
	return EditConfig{editor: editor}
}

func (u EditConfig) Run(ctx context.Context, configPath string) error {
	return u.editor.Edit(ctx, configPath)
}
