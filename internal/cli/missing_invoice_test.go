package cli

import (
	"testing"
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
			customers, issuer, _, template := writeContextFixtures(t)
			chdirForTest(t, t.TempDir())

			exitCode, stdout, stderr := captureRun(t, tc.args(customers, issuer, template))

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
