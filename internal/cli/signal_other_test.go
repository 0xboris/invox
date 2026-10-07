//go:build !unix

package cli

import "testing"

// assertProcessGone checks nothing on Windows, which has no signal-0 probe.
func assertProcessGone(t *testing.T, pid int) {}
