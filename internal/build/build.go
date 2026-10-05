// Package build holds version information stamped in at link time:
//
//	go build -ldflags "-X github.com/0xboris/invox/internal/build.Version=v1.2.3 -X github.com/0xboris/invox/internal/build.Date=2026-10-05"
package build

import "runtime/debug"

var (
	// Version is the release version, such as "v1.2.3", or "DEV".
	Version = "DEV"
	// Date is the release date (YYYY-MM-DD), or empty.
	Date = ""
)

func init() {
	if Version != "DEV" {
		return
	}
	// `go install github.com/0xboris/invox/cmd/invox@<version>` records the
	// module version in the binary even without ldflags.
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
	}
}
