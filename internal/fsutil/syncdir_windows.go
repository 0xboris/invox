package fsutil

// syncDir does nothing on Windows, which cannot sync a directory.
func syncDir(string) error {
	return nil
}
