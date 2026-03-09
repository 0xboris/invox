package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func chdirForTest(t *testing.T, dir string) {
	t.Helper()

	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() returned error: %v", err)
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) returned error: %v", dir, err)
	}
	if err := os.Chdir(absDir); err != nil {
		t.Fatalf("os.Chdir(%q) returned error: %v", absDir, err)
	}
	oldPWD, hadPWD := os.LookupEnv("PWD")
	if err := os.Setenv("PWD", absDir); err != nil {
		t.Fatalf("os.Setenv(PWD) returned error: %v", err)
	}

	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Errorf("restore working directory to %q: %v", oldDir, err)
		}
		if hadPWD {
			if err := os.Setenv("PWD", oldPWD); err != nil {
				t.Errorf("restore PWD to %q: %v", oldPWD, err)
			}
			return
		}
		if err := os.Unsetenv("PWD"); err != nil {
			t.Errorf("unset PWD: %v", err)
		}
	})
}
