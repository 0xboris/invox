package cli_test

import (
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
	"github.com/0xboris/invox/internal/testfixture"
)

// A paths.* setting in config.yaml that names a missing file is a problem
// with config.yaml: the error names the file and the setting, and the hint
// opens the config file, the --config one when it was given.
func TestConfiguredSupportFileThatDoesNotExist(t *testing.T) {
	for _, tc := range []struct {
		name   string
		key    string
		config bool
		args   func(customers, issuer, invoice string) []string
	}{
		{
			name: "customer list",
			key:  "customers",
			args: func(string, string, string) []string {
				return []string{"customer", "list"}
			},
		},
		{
			name: "validate",
			key:  "issuer",
			args: func(customers, _, invoice string) []string {
				return []string{"validate", invoice, "-c", customers}
			},
		},
		{
			name: "new",
			key:  "defaults",
			args: func(customers, issuer, _ string) []string {
				return []string{"new", "CUST-001", "-c", customers, "-u", issuer, "--dry-run"}
			},
		},
		{
			name: "render",
			key:  "template",
			args: func(customers, issuer, invoice string) []string {
				return []string{"render", invoice, "-c", customers, "-u", issuer, "--dry-run"}
			},
		},
		{
			name: "template list --json",
			key:  "template",
			args: func(string, string, string) []string {
				return []string{"template", "list", "--json", "name,default"}
			},
		},
		{
			name:   "render --config",
			key:    "template",
			config: true,
			args: func(customers, issuer, invoice string) []string {
				return []string{"render", invoice, "-c", customers, "-u", issuer, "--dry-run"}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			x := clitest.New(t)

			fx := testfixture.WriteContext(t)
			configPath := x.WriteConfig("paths:\n  " + tc.key + ": nope.file\n")
			missing := filepath.Join(filepath.Dir(configPath), "nope.file")
			x.Chdir(t.TempDir())
			args := tc.args(fx.Customers, fx.Issuer, fx.Invoice)
			hint := "Run 'invox config' to open and fix the config file.\n"
			if tc.config {
				args = append([]string{"--config", configPath}, args...)
				hint = "Run 'invox --config " + configPath + " config' to open and fix the config file.\n"
			}

			exitCode, stdout, stderr := x.Run(args)

			want := "error: " + tc.key + " file " + missing + " does not exist; paths." + tc.key + " in " + configPath + " sets it\n" + hint
			if exitCode != 1 {
				t.Errorf("exit code = %d, want 1", exitCode)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
			}
		})
	}
}
