package cli

import (
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
)

var signalCases = []struct {
	name     string
	signal   syscall.Signal
	exitCode int
}{
	{name: "SIGINT", signal: syscall.SIGINT, exitCode: 130},
	{name: "SIGTERM", signal: syscall.SIGTERM, exitCode: 143},
}

// isolateTempDir points the temporary directory at a fresh directory on every
// OS and returns it.
func isolateTempDir(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for _, key := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(key, dir)
	}
	return dir
}

// installSleepingTectonic puts a tectonic on PATH that writes its pid and
// sleeps, and returns the pidfile path.
func installSleepingTectonic(t *testing.T) string {
	t.Helper()

	installFakeTectonic(t, fakeTectonicSleep)
	pidfile := filepath.Join(t.TempDir(), "tectonic.pid")
	t.Setenv(fakeTectonicPidfileEnv, pidfile)
	return pidfile
}

// waitForPid returns the pid in pidfile once it exists. It fails the test if
// done is closed first or the pidfile does not appear within 30 seconds.
func waitForPid(t *testing.T, pidfile string, done <-chan struct{}) int {
	t.Helper()

	deadline := time.Now().Add(30 * time.Second)
	for {
		if data, err := os.ReadFile(pidfile); err == nil {
			pid, err := strconv.Atoi(string(data))
			if err != nil {
				t.Fatalf("pidfile %q: %v", data, err)
			}
			return pid
		}
		select {
		case <-done:
			t.Fatal("invox exited before tectonic started")
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("tectonic did not write its pidfile within 30s")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func assertNoBuildDir(t *testing.T, tempDir string) {
	t.Helper()

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "invox-build-") {
			t.Fatalf("temporary build directory %s was left behind", entry.Name())
		}
	}
}

func TestBuildCancelledBySignalStopsTectonicAndCleansUp(t *testing.T) {
	for _, tc := range signalCases {
		t.Run(tc.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			pidfile := installSleepingTectonic(t)
			tempDir := isolateTempDir(t)
			isolateUserDirs(t)
			ios, _, stdout, stderr := iostreams.Test()
			f := factory.New(ios, run.Exec{}, env.System())
			ctx, cancel := context.WithCancelCause(context.Background())
			t.Cleanup(func() { cancel(nil) })

			var exitCode int
			done := make(chan struct{})
			go func() {
				defer close(done)
				exitCode = mainContext(ctx, []string{
					"build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath,
				}, f)
			}()
			pid := waitForPid(t, pidfile, done)
			start := time.Now()
			cancel(&SignalError{Signal: tc.signal})
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				t.Fatal("invox did not return within 30s of the signal")
			}

			if elapsed := time.Since(start); elapsed > 10*time.Second {
				t.Fatalf("invox returned %s after the signal, want it to stop tectonic at once", elapsed)
			}
			if exitCode != tc.exitCode {
				t.Fatalf("exitCode = %d, want %d, stderr=%q", exitCode, tc.exitCode, stderr)
			}
			if stdout.String() != "" {
				t.Fatalf("stdout = %q, want empty", stdout)
			}
			if stderr.String() != "" {
				t.Fatalf("stderr = %q, want empty", stderr)
			}
			assertNoBuildDir(t, tempDir)
			assertProcessGone(t, pid)
			if _, err := os.Stat(strings.TrimSuffix(invoicePath, ".yaml") + ".pdf"); !os.IsNotExist(err) {
				t.Fatalf("no PDF should be written, Stat err = %v", err)
			}
		})
	}
}
