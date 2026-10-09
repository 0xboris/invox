package store

// testHost returns a Linux Host whose config directory is under configHome
// and whose archive directory is under home/.local/share.
func testHost(configHome, home string) Host {
	return NewHost(HostInputs{GOOS: "linux", Home: home, XDGConfigHome: configHome})
}
