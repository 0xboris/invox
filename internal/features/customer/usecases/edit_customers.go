package usecases

import (
	"context"

	"github.com/0xboris/invox/internal/domain/ports"
)

// EditCustomers opens customers.yaml in the user's editor and returns when
// the editor exits.
type EditCustomers struct {
	editor ports.Editor
}

func NewEditCustomers(editor ports.Editor) EditCustomers {
	return EditCustomers{editor: editor}
}

func (u EditCustomers) Run(ctx context.Context, customersPath string) error {
	return u.editor.Edit(ctx, customersPath)
}
