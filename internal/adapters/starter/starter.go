// Package starter holds the files `invox init` writes into a new config
// directory, embedded in the binary.
package starter

import (
	"embed"
	"io/fs"
)

//go:embed files/customers.yaml files/issuer.yaml files/invoice_defaults.yaml files/template.tex
var embedded embed.FS

// Files holds customers.yaml, issuer.yaml, invoice_defaults.yaml and
// template.tex at its root.
var Files = mustSub(embedded, "files")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
