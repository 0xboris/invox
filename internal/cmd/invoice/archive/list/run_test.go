package list

import (
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/factory/factorytest"
	"github.com/0xboris/invox/internal/iostreams"
)

func TestListRunEmptyHint(t *testing.T) {
	dataHome := t.TempDir()
	tests := []struct {
		name string
		in   factorytest.Options
		want string
	}{
		{name: "archive directory", in: factorytest.Options{NoHome: true, Vars: map[string]string{"XDG_DATA_HOME": dataHome}}, want: "No archived invoices found in " + filepath.Join(dataHome, "invox", "invoices") + "\n"},
		{name: "no archive directory", in: factorytest.Options{NoHome: true}, want: "No archived invoices found\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.in.ConfigDir = t.TempDir()
			f := factorytest.New(t, nil, tc.in)
			ios, _, out, errOut := iostreams.Test()
			ios.SetStdoutTTY(true)
			if err := listRun(&ListOptions{IO: ios, Service: f.Service}); err != nil {
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
