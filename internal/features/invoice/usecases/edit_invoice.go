package usecases

import (
	"context"

	"github.com/0xboris/invox/internal/domain/ports"
)

// EditInvoice opens an invoice file in the user's editor and returns when the
// editor exits.
type EditInvoice struct {
	editor ports.Editor
}

func NewEditInvoice(editor ports.Editor) EditInvoice {
	return EditInvoice{editor: editor}
}

func (u EditInvoice) Run(ctx context.Context, path string) error {
	return u.editor.Edit(ctx, path)
}
