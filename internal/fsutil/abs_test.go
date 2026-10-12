package fsutil

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestAbsResolvesAgainstBase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name, base, path, want string
		windowsOnly            bool
	}{
		{name: "relative", base: "/work", path: "customers.yaml", want: "/work/customers.yaml"},
		{name: "relative with dots", base: "/work/a", path: "../b/./c.yaml", want: "/work/b/c.yaml"},
		{name: "absolute", base: "/work", path: "/etc//x.yaml", want: "/etc/x.yaml"},
		{name: "empty stays empty", base: "/work", path: "", want: ""},
		{name: "windows absolute", base: `C:\work`, path: `D:\x.yaml`, want: `D:\x.yaml`, windowsOnly: true},
		// --config \x.yaml resolved to C:\work\x.yaml, while -c \x.yaml
		// resolved to C:\x.yaml as filepath.Abs does.
		{name: "windows rooted keeps the base drive", base: `C:\work`, path: `\x.yaml`, want: `C:\x.yaml`, windowsOnly: true},
		{name: "windows rooted keeps the base share", base: `\\server\share\work`, path: `\x.yaml`, want: `\\server\share\x.yaml`, windowsOnly: true},
	}
	for _, tc := range tests {
		if tc.windowsOnly && runtime.GOOS != "windows" {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			base, path, want := tc.base, tc.path, tc.want
			if !tc.windowsOnly {
				base, path, want = filepath.FromSlash(base), filepath.FromSlash(path), filepath.FromSlash(want)
			}
			if got := Abs(base, path); got != want {
				t.Fatalf("Abs(%q, %q) = %q, want %q", base, path, got, want)
			}
		})
	}
}
