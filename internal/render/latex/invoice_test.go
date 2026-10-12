package latex_test

import (
	"testing"

	"github.com/0xboris/invox/internal/billing"
	"github.com/0xboris/invox/internal/cli/cmdutil"
	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/render/latex"
	"github.com/0xboris/invox/internal/testfixture"
)

// service returns the use cases as invox wires them for h, with the
// support files named as on a command line.
func service(t *testing.T, h testfixture.Host, files cmdutil.Files) *billing.Service {
	t.Helper()
	return factorytest.New(t, nil, factorytest.Options{
		Home: h.Home,
		Vars: map[string]string{"XDG_CONFIG_HOME": h.ConfigHome},
	}).Service(files)
}

// renderer is the renderer factory builds, which finds assets next to the
// template and in h's config directory.
func renderer(t *testing.T, h testfixture.Host) latex.Renderer {
	t.Helper()
	return service(t, h, cmdutil.Files{}).Renderer.(latex.Renderer)
}

// template resolves the template at path as render and build do.
func template(t *testing.T, h testfixture.Host, path string) billing.Template {
	t.Helper()
	tmpl, err := service(t, h, cmdutil.Files{}).Directory.Template(path)
	if err != nil {
		t.Fatalf("Template(%s) returned error: %v", path, err)
	}
	return tmpl
}

// renderInvoice fills the template at templatePath with ctx and writes it,
// with the assets it uses, to outputPath, as render does.
func renderInvoice(t *testing.T, h testfixture.Host, templatePath, outputPath string, ctx *invoice.Context) error {
	t.Helper()
	tmpl := template(t, h, templatePath)
	source, err := renderer(t, h).Render(tmpl, ctx, billing.EPCFor(ctx))
	if err != nil {
		return err
	}
	return renderer(t, h).Write(tmpl, source, outputPath)
}
