package store

import "os"

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func pathExists(path string, isDir bool) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir() == isDir
}
