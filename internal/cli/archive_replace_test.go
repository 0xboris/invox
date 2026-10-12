package cli_test

import (
	"strings"
	"testing"

	"github.com/0xboris/invox/internal/clitest"
)

func TestYesFlagIsDocumented(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{
			args: []string{"archive", "add", "-h"},
			want: []string{"      --yes            Replace an archived invoice without asking\n", "archive.dir/.history/<path>.<UTC timestamp>.<ext>"},
		},
		{
			args: []string{"build", "-h"},
			want: []string{"      --yes                Replace an archived invoice without asking\n", "keeps that status when its PDF is rebuilt"},
		},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			x := clitest.New(t)

			exitCode, stdout, stderr := x.Run(tc.args)
			if exitCode != 0 {
				t.Fatalf("exitCode = %d, want 0, stderr=%q", exitCode, stderr)
			}
			if stderr != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			for _, want := range tc.want {
				if !strings.Contains(stdout, want) {
					t.Fatalf("stdout does not contain %q:\n%s", want, stdout)
				}
			}
		})
	}
}
