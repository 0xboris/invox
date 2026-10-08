package run

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"testing"
	"time"
)

// cancelChildEnv makes the test binary act as a long-running child for the
// cancellation tests. The child writes its pid to the file named by
// cancelChildPidfileEnv once it is ready for a signal.
const (
	cancelChildEnv        = "RUN_TEST_CANCEL_CHILD"
	cancelChildPidfileEnv = "RUN_TEST_CANCEL_PIDFILE"
)

const (
	childSleeps       = "sleep"
	childTrapsSIGTERM = "trap-sigterm"
	childIgnoresTERM  = "ignore-sigterm"
)

func runCancelChild(mode string) int {
	terminated := make(chan os.Signal, 1)
	switch mode {
	case childTrapsSIGTERM:
		signal.Notify(terminated, syscall.SIGTERM)
	case childIgnoresTERM:
		signal.Ignore(syscall.SIGTERM)
	}
	pidfile := os.Getenv(cancelChildPidfileEnv)
	if err := os.WriteFile(pidfile+".tmp", []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := os.Rename(pidfile+".tmp", pidfile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	select {
	case <-terminated:
		fmt.Println("got SIGTERM")
		return 0
	case <-time.After(time.Minute):
		return 0
	}
}

type cancelRun struct {
	cancel context.CancelFunc
	done   chan error
	stdout *bytes.Buffer
	pid    int
}

// startCancelChild runs a child in mode under a cancellable context and
// returns once the child has written its pid.
func startCancelChild(t *testing.T, mode string) *cancelRun {
	t.Helper()

	pidfile := filepath.Join(t.TempDir(), "pid")
	t.Setenv(cancelChildEnv, mode)
	t.Setenv(cancelChildPidfileEnv, pidfile)
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	r := &cancelRun{cancel: cancel, done: make(chan error, 1), stdout: new(bytes.Buffer)}
	go func() {
		r.done <- Exec{}.Run(ctx, Cmd{Name: testExecutable(t), Stdout: r.stdout})
	}()

	deadline := time.Now().Add(30 * time.Second)
	for {
		data, err := os.ReadFile(pidfile)
		if err == nil {
			r.pid, err = strconv.Atoi(string(data))
			if err != nil {
				t.Fatalf("pidfile %q: %v", data, err)
			}
			return r
		}
		select {
		case err := <-r.done:
			t.Fatalf("child exited before it was ready: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("child did not write its pidfile within 30s")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// cancelAndWait cancels the run and returns Run's error and how long Run took
// to return after the cancel. It fails the test if Run does not return within
// limit.
func (r *cancelRun) cancelAndWait(t *testing.T, limit time.Duration) (error, time.Duration) {
	t.Helper()

	start := time.Now()
	r.cancel()
	select {
	case err := <-r.done:
		return err, time.Since(start)
	case <-time.After(limit):
		t.Fatalf("Run did not return within %s of the cancel", limit)
		return nil, 0
	}
}

func TestExecStopsASleepingChildOnCancel(t *testing.T) {
	r := startCancelChild(t, childSleeps)

	err, elapsed := r.cancelAndWait(t, waitDelay+5*time.Second)

	if err == nil {
		t.Fatal("Run returned nil, want an error for the stopped child")
	}
	if elapsed >= waitDelay {
		t.Fatalf("Run returned %s after the cancel, want under waitDelay (%s)", elapsed, waitDelay)
	}
}
