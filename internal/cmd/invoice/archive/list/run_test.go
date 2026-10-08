package list

import (
	"testing"

	"github.com/0xboris/invox/internal/invoice"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestListRunEmptyHint(t *testing.T) {
	dataHome := t.TempDir()
	tests := []struct {
		name string
		in   invoice.HostInputs
		want string
	}{
		{name: "archive directory", in: invoice.HostInputs{GOOS: "linux", XDGDataHome: dataHome}, want: "No archived invoices found in " + invoice.NewHost(invoice.HostInputs{GOOS: "linux", XDGDataHome: dataHome}).DefaultArchiveDir() + "\n"},
		{name: "no archive directory", in: invoice.HostInputs{GOOS: "linux"}, want: "No archived invoices found\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.ConfigDir = t.TempDir()
			host := invoice.NewHost(tc.in)
			ios, _, out, errOut := iostreams.Test()
			ios.SetStdoutTTY(true)
			if err := listRun(&ListOptions{IO: ios, Host: func() invoice.Host { return host }}); err != nil {
				t.Fatalf("listRun returned error: %v", err)
			}
			if out.Len() != 0 {
				t.Errorf("stdout = %q, want empty", out.String())
			}
			if errOut.String() != tc.want {
				t.Errorf("stderr = %q, want %q", errOut.String(), tc.want)
			}
		})
	}
}
