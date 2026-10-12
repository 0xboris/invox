package cli_test

import (
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

// An invoice argument that does not exist gets one wording for every
// verb that reads it, not the operating system's open error. It is a
// runtime error: the argument was well formed.
func TestInvoiceArgumentThatDoesNotExist(t *testing.T) {
	for _, tc := range []struct {
		name string
		args func(customers, issuer, template string) []string
	}{
		{"validate", func(c, u, _ string) []string { return []string{"validate", "nope.yaml", "-c", c, "-u", u} }},
		{"render", func(c, u, t string) []string {
			return []string{"render", "nope.yaml", "-c", c, "-u", u, "-t", t, "--dry-run"}
		}},
		{"build -i", func(c, u, t string) []string { return []string{"build", "-i", "nope.yaml", "-c", c, "-u", u, "-t", t} }},
		{"email", func(c, u, _ string) []string { return []string{"email", "nope.yaml", "-c", c, "-u", u, "-o", "x.eml"} }},
		{"increment", func(c, u, _ string) []string { return []string{"increment", "nope.yaml", "-c", c} }},
		{"archive add", func(string, string, string) []string { return []string{"archive", "add", "nope.yaml"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			fx := testfixture.WriteContext(t)
			x.Chdir(t.TempDir())

			exitCode, stdout, stderr := x.Run(tc.args(fx.Customers, fx.Issuer, fx.Template))

			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if want := "error: invoice file nope.yaml does not exist\n"; stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}
		})
	}
}
