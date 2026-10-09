package shared

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/0xboris/invox/internal/cli/cmdutil"
)

func TestTakeInput(t *testing.T) {
	work := filepath.Join(t.TempDir(), "work")
	getwd := func() (string, error) { return work, nil }
	tests := []struct {
		name    string
		input   string
		args    []string
		want    string
		wantErr string
	}{
		{name: "no argument", input: "x.yaml", want: "x.yaml"},
		{name: "argument only", args: []string{"x.yaml"}, want: "x.yaml"},
		{name: "same relative path", input: "./x.yaml", args: []string{"x.yaml"}, want: "./x.yaml"},
		{name: "relative argument, absolute input", input: filepath.Join(work, "x.yaml"), args: []string{"x.yaml"}, want: filepath.Join(work, "x.yaml")},
		{name: "absolute argument, relative input", input: "x.yaml", args: []string{filepath.Join(work, "x.yaml")}, want: "x.yaml"},
		{name: "different files", input: "y.yaml", args: []string{"x.yaml"}, wantErr: "the INVOICE argument x.yaml and -i, --input y.yaml name different files; pass only one"},
		{name: "same name in another directory", input: filepath.Join(work, "sub", "x.yaml"), args: []string{"x.yaml"}, wantErr: "the INVOICE argument x.yaml and -i, --input " + filepath.Join(work, "sub", "x.yaml") + " name different files; pass only one"},
		{name: "extra argument", args: []string{"x.yaml", "y.yaml"}, wantErr: "unexpected arguments: y.yaml"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := tc.input
			err := TakeInput(getwd, &path, tc.args)
			if tc.wantErr != "" {
				var flagErr *cmdutil.FlagError
				if !errors.As(err, &flagErr) || err.Error() != tc.wantErr {
					t.Fatalf("TakeInput error = %#v, want FlagError %q", err, tc.wantErr)
				}
				return
			}
			if err != nil || path != tc.want {
				t.Fatalf("TakeInput = %v, invoice %q; want nil, %q", err, path, tc.want)
			}
		})
	}
}
