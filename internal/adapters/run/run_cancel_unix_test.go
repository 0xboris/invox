//go:build unix

package run

import (
	"errors"
	"syscall"
	"testing"
	"time"
)

func TestExecSendsSIGTERMOnCancel(t *testing.T) {
	r := startCancelChild(t, childTrapsSIGTERM)

	_, elapsed := r.cancelAndWait(t, waitDelay+5*time.Second)

	if got, want := r.stdout.String(), "got SIGTERM\n"; got != want {
		t.Fatalf("child stdout = %q, want %q", got, want)
	}
	if elapsed >= waitDelay {
		t.Fatalf("Run returned %s after the cancel, want under waitDelay (%s)", elapsed, waitDelay)
	}
}

func TestExecKillsAChildThatIgnoresSIGTERM(t *testing.T) {
	r := startCancelChild(t, childIgnoresTERM)

	err, elapsed := r.cancelAndWait(t, waitDelay+5*time.Second)

	if err == nil {
		t.Fatal("Run returned nil, want an error for the killed child")
	}
	if elapsed < waitDelay {
		t.Fatalf("Run returned %s after the cancel, want at least waitDelay (%s) before the kill", elapsed, waitDelay)
	}
	if err := syscall.Kill(r.pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("kill(%d, 0) = %v, want ESRCH: the child is still running", r.pid, err)
	}
}
