package cli

import (
	"testing"
)

// A support file named on the command line that does not exist is a
// runtime error worded like a missing --config file, not the operating
// system's open error, and not a usage error: no flag fixes it.
func TestNamedSupportFileThatDoesNotExist(t *testing.T) {
	for _, tc := range []struct {
		name string
		args func(customers, issuer, defaults, invoice string) []string
		want string
	}{
		{
			name: "new -c",
			args: func(_, issuer, defaults, _ string) []string {
				return []string{"new", "CUST-001", "-c", "nope.yaml", "-u", issuer, "--defaults", defaults}
			},
			want: "error: customers file nope.yaml does not exist\n",
		},
		{
			name: "new -u",
			args: func(customers, _, defaults, _ string) []string {
				return []string{"new", "CUST-001", "-c", customers, "-u", "nope.yaml", "--defaults", defaults}
			},
			want: "error: issuer file nope.yaml does not exist\n",
		},
		{
			name: "new --defaults",
			args: func(customers, issuer, _, _ string) []string {
				return []string{"new", "CUST-001", "-c", customers, "-u", issuer, "--defaults", "nope.yaml"}
			},
			want: "error: defaults file nope.yaml does not exist\n",
		},
		{
			name: "render -t",
			args: func(customers, issuer, _, invoice string) []string {
				return []string{"render", invoice, "-c", customers, "-u", issuer, "-t", "./nope.tex", "--dry-run"}
			},
			want: "error: template file nope.tex does not exist\n",
		},
		{
			name: "customer list -c",
			args: func(string, string, string, string) []string {
				return []string{"customer", "list", "-c", "nope.yaml"}
			},
			want: "error: customers file nope.yaml does not exist\n",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			customers, issuer, defaults := writeDraftFixtures(t)
			_, _, invoice, _ := writeContextFixtures(t)
			chdirForTest(t, t.TempDir())

			exitCode, stdout, stderr := captureRun(t, tc.args(customers, issuer, defaults, invoice))

			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr != tc.want {
				t.Errorf("stderr = %q, want %q", stderr, tc.want)
			}
		})
	}
}
