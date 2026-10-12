package cli_test

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"testing"
	"time"

	"github.com/0xboris/invox/internal/adapters/run"
	"github.com/0xboris/invox/internal/cli"
	"github.com/0xboris/invox/internal/env"
	"github.com/0xboris/invox/internal/factory"
	"github.com/0xboris/invox/internal/iostreams"
	"github.com/0xboris/invox/internal/testfixture"
)

// fakeTectonicEnv makes the test binary act as tectonic instead of running
// tests. installFakeTectonic sets it and puts a copy of the test binary on
// PATH, so the signal tests have a real child process on every OS.
const fakeTectonicEnv = "INVOX_TEST_FAKE_TECTONIC"

// fakeTectonicPidfileEnv names the file the fake tectonic writes its pid to.
const fakeTectonicPidfileEnv = "INVOX_TEST_FAKE_TECTONIC_PIDFILE"

// runMainEnv makes the test binary act as invox, so tests can send it real
// signals.
const runMainEnv = "INVOX_TEST_RUN_MAIN"

const (
	fakeTectonicSleep = "sleep"
	// fakeTectonicSleepIgnoreTerm sleeps like fakeTectonicSleep but ignores
	// SIGTERM, so invox keeps waiting for it after the first signal.
	fakeTectonicSleepIgnoreTerm = "sleep-ignore-sigterm"
)

func TestMain(m *testing.M) {
	if os.Getenv(runMainEnv) != "" {
		// Unset so the programs invox runs, such as the fake tectonic, are not
		// invox too.
		_ = os.Unsetenv(runMainEnv)
		os.Exit(cli.Main(os.Args[1:], factory.New(iostreams.System(), run.Exec{}, env.System())))
	}
	if mode := os.Getenv(fakeTectonicEnv); mode != "" {
		os.Exit(runFakeTectonic(mode))
	}
	if os.Getenv(fakeChildEnv) != "" {
		os.Exit(runFakeChild(os.Args[1:]))
	}
	os.Exit(m.Run())
}

// runFakeTectonic writes its pid to the file named by fakeTectonicPidfileEnv
// and sleeps for a minute, for the signal tests to stop it.
func runFakeTectonic(mode string) int {
	if mode == fakeTectonicSleepIgnoreTerm {
		signal.Ignore(syscall.SIGTERM)
	}
	pidfile := os.Getenv(fakeTectonicPidfileEnv)
	if err := os.WriteFile(pidfile+".tmp", []byte(strconv.Itoa(os.Getpid())), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "fake tectonic: %v\n", err)
		return 1
	}
	if err := os.Rename(pidfile+".tmp", pidfile); err != nil {
		fmt.Fprintf(os.Stderr, "fake tectonic: %v\n", err)
		return 1
	}
	time.Sleep(time.Minute)
	return 0
}

func installFakeTectonic(t *testing.T, mode string) {
	t.Helper()

	name := "tectonic"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binDir := t.TempDir()
	copyExecutable(t, testfixture.Executable(t), filepath.Join(binDir, name))

	t.Setenv(fakeTectonicEnv, mode)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func copyExecutable(t *testing.T, src, dst string) {
	t.Helper()

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close() //nolint:errcheck // read-only source; the copy already succeeded or failed the test

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		t.Fatalf("copy %s to %s: %v", src, dst, err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", dst, err)
	}
}
