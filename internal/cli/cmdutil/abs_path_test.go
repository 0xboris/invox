package cmdutil

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestAbsPathResolvesAgainstBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, base, path, want string
		windowsOnly            bool
	}{
		{name: "relative", base: "/work", path: "customers.yaml", want: "/work/customers.yaml"},
		{name: "relative with dots", base: "/work/a", path: "../b/./c.yaml", want: "/work/b/c.yaml"},
		{name: "absolute", base: "/work", path: "/etc//x.yaml", want: "/etc/x.yaml"},
		{name: "windows absolute", base: `C:\work`, path: `D:\x.yaml`, want: `D:\x.yaml`, windowsOnly: true},
		{name: "windows rooted keeps the base drive", base: `C:\work`, path: `\x.yaml`, want: `C:\x.yaml`, windowsOnly: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.windowsOnly && runtime.GOOS != "windows" {
				t.Skip("Windows path syntax")
			}
			base, want := tc.base, tc.want
			if !tc.windowsOnly {
				base, want = filepath.FromSlash(base), filepath.FromSlash(want)
			}
			if got := AbsPath(base, filepath.FromSlash(tc.path)); got != want {
				t.Fatalf("AbsPath(%q, %q) = %q, want %q", base, tc.path, got, want)
			}
		})
	}
}
