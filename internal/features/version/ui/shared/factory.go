// Package shared holds what the version command uses to reach outside invox.
package shared

import "github.com/0xboris/invox/internal/utilities/iostreams"

// Factory holds what the version command uses to reach outside
// invox. The composition root builds it.
type Factory struct {
	IOStreams *iostreams.IOStreams
	// Version is the release version, such as "v1.2.3", or "DEV", and Date
	// its release date (YYYY-MM-DD) or "".
	Version, Date string
}
