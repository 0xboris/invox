package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeTectonicEnv makes the test binary act as tectonic instead of running
// tests. installFakeTectonic sets it and puts a copy of the test binary on
// PATH, so the build tests need no shell and run on every OS.
const fakeTectonicEnv = "INVOX_TEST_FAKE_TECTONIC"

const (
	fakeTectonicWritePDF = "write-pdf"
	fakeTectonicFail     = "fail"
	fakeTectonicExit2    = "exit-2"
	fakeTectonicChatter  = "chatter"
)

func TestMain(m *testing.M) {
	if mode := os.Getenv(fakeTectonicEnv); mode != "" {
		os.Exit(runFakeTectonic(mode, os.Args[1:]))
	}
	if os.Getenv(fakeChildEnv) != "" {
		os.Exit(runFakeChild(os.Args[1:]))
	}
	os.Exit(m.Run())
}

// runFakeTectonic writes an empty PDF next to the single .tex argument, or
// fails with exit code 1 (fakeTectonicFail) or 2 (fakeTectonicExit2).
// fakeTectonicChatter also prints progress to stdout, as tectonic does.
func runFakeTectonic(mode string, args []string) int {
	switch mode {
	case fakeTectonicFail:
		fmt.Fprintln(os.Stderr, "fake tectonic: forced failure")
		return 1
	case fakeTectonicExit2:
		fmt.Fprintln(os.Stderr, "fake tectonic: forced failure")
		return 2
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "fake tectonic: want one input file, got %q\n", args)
		return 2
	}

	if mode == fakeTectonicChatter {
		fmt.Println("note: running TeX ...")
		fmt.Println("note: writing `invoice.pdf`")
	}
	pdfPath := strings.TrimSuffix(args[0], filepath.Ext(args[0])) + ".pdf"
	if err := os.WriteFile(pdfPath, nil, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "fake tectonic: %v\n", err)
		return 1
	}
	return 0
}

func installFakeTectonic(t *testing.T, mode string) {
	t.Helper()

	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() returned error: %v", err)
	}

	name := "tectonic"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binDir := t.TempDir()
	copyExecutable(t, self, filepath.Join(binDir, name))

	t.Setenv(fakeTectonicEnv, mode)
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func copyExecutable(t *testing.T, src, dst string) {
	t.Helper()

	in, err := os.Open(src)
	if err != nil {
		t.Fatalf("open %s: %v", src, err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		t.Fatalf("create %s: %v", dst, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		t.Fatalf("copy %s to %s: %v", src, dst, err)
	}
	if err := out.Close(); err != nil {
		t.Fatalf("close %s: %v", dst, err)
	}
}
