//go:build unix

package cli

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func assertProcessGone(t *testing.T, pid int) {
	t.Helper()

	if err := syscall.Kill(pid, 0); !errors.Is(err, syscall.ESRCH) {
		t.Fatalf("kill(%d, 0) = %v, want ESRCH: tectonic is still running", pid, err)
	}
}

func TestBuildExitsWithSignalCodeOnRealSignal(t *testing.T) {
	for _, tc := range signalCases {
		t.Run(tc.name, func(t *testing.T) {
			customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
			pidfile := installSleepingTectonic(t)
			tempDir := isolateTempDir(t)
			isolateUserDirs(t)
			self, err := os.Executable()
			if err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(self, "build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath)
			cmd.Env = append(os.Environ(), runMainEnv+"=1")
			// Files, not buffers: Wait would also wait for a leftover tectonic
			// to close a pipe, and hide that invox itself had exited.
			stdoutPath := filepath.Join(t.TempDir(), "stdout")
			stderrPath := filepath.Join(t.TempDir(), "stderr")
			cmd.Stdout = createFile(t, stdoutPath)
			cmd.Stderr = createFile(t, stderrPath)
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = cmd.Process.Kill() })
			var waitErr error
			done := make(chan struct{})
			go func() {
				defer close(done)
				waitErr = cmd.Wait()
			}()

			pid := waitForPid(t, pidfile, done)
			if err := cmd.Process.Signal(tc.signal); err != nil {
				t.Fatal(err)
			}
			select {
			case <-done:
			case <-time.After(30 * time.Second):
				t.Fatal("invox did not exit within 30s of the signal")
			}

			stdout, stderr := readFileForTest(t, stdoutPath), readFileForTest(t, stderrPath)
			var exitErr *exec.ExitError
			if !errors.As(waitErr, &exitErr) || exitErr.ExitCode() != tc.exitCode {
				t.Fatalf("invox exited with %v, want exit status %d, stderr=%q", waitErr, tc.exitCode, stderr)
			}
			if stdout != "" || stderr != "" {
				t.Fatalf("stdout = %q, stderr = %q, want both empty", stdout, stderr)
			}
			assertNoBuildDir(t, tempDir)
			assertProcessGone(t, pid)
		})
	}
}

func TestSecondSignalEndsInvoxAtOnce(t *testing.T) {
	customersPath, issuerPath, invoicePath, templatePath := writeContextFixtures(t)
	installFakeTectonic(t, fakeTectonicSleepIgnoreTerm)
	pidfile := filepath.Join(t.TempDir(), "tectonic.pid")
	t.Setenv(fakeTectonicPidfileEnv, pidfile)
	isolateTempDir(t)
	isolateUserDirs(t)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(self, "build", invoicePath, "-c", customersPath, "-u", issuerPath, "-t", templatePath)
	cmd.Env = append(os.Environ(), runMainEnv+"=1")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })
	var waitErr error
	done := make(chan struct{})
	go func() {
		defer close(done)
		waitErr = cmd.Wait()
	}()
	pid := waitForPid(t, pidfile, done)
	// The tectonic ignores SIGTERM and outlives invox here; kill it.
	t.Cleanup(func() { _ = syscall.Kill(pid, syscall.SIGKILL) })

	// After the first SIGINT invox waits waitDelay (3s) for the tectonic. A
	// SIGINT that arrives after invox stopped listening must end it at once.
	// Repeat the signal, because one sent before then is absorbed.
	start := time.Now()
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	deadline := time.After(30 * time.Second)
	for waiting := true; waiting; {
		if err := cmd.Process.Signal(syscall.SIGINT); err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Fatal(err)
		}
		select {
		case <-done:
			waiting = false
		case <-ticker.C:
		case <-deadline:
			t.Fatal("invox did not exit within 30s of the signals")
		}
	}

	var exitErr *exec.ExitError
	if !errors.As(waitErr, &exitErr) {
		t.Fatalf("invox exited with %v, want it ended by SIGINT", waitErr)
	}
	status := exitErr.Sys().(syscall.WaitStatus)
	if !status.Signaled() || status.Signal() != syscall.SIGINT {
		t.Fatalf("invox exited with %v, want it ended by SIGINT", waitErr)
	}
	if elapsed := time.Since(start); elapsed >= 2*time.Second {
		t.Fatalf("invox exited %s after the first signal, want well under the 3s wait for tectonic", elapsed)
	}
}

func createFile(t *testing.T, path string) *os.File {
	t.Helper()

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return f
}
