package store

import "github.com/0xboris/invox/internal/testfixture"

// testHost returns the Linux Host of h: its config directory is under
// h.ConfigHome and its archive directory under h.Home/.local/share.
func testHost(h testfixture.Host) Host {
	return NewHost(HostInputs{GOOS: "linux", Home: h.Home, XDGConfigHome: h.ConfigHome})
}
