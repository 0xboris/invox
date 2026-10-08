//go:build !unix

package fsutil

import "io/fs"

// umask is zero where the process has none.
var umask fs.FileMode
