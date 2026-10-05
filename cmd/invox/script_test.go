package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"

	"github.com/0xboris/invox/internal/cli"
)

// The end-to-end suite in testdata/script pins what every command prints to
// stdout and stderr and the code it exits with. Run it with
//
//	go test ./cmd/invox -run TestScript
//
// and, after an intentional output change, rewrite the golden blocks with
//
//	go test ./cmd/invox -run TestScript -update
var updateScripts = flag.Bool("update", false, "rewrite cmp golden blocks in testdata/script with the actual output")

// Environment variables that make the fake programs fail.
const (
	fakeTectonicFailEnv = "FAKE_TECTONIC_FAIL"
	fakeEditorFailEnv   = "FAKE_EDITOR_FAIL"
	fakeOpenFailEnv     = "FAKE_OPEN_FAIL"
)

func TestMain(m *testing.M) {
	// Under -race every process sleeps a second on exit by default, and the
	// suite starts a few hundred. testscript passes GORACE on to them.
	if _, ok := os.LookupEnv("GORACE"); !ok {
		os.Setenv("GORACE", "atexit_sleep_ms=0")
	}

	// testscript copies the test binary into a bin directory on PATH once per
	// name below; running it under that name calls the matching function.
	// The external programs invox launches are faked the same way, so the
	// suite needs no shell scripts and runs on Windows.
	testscript.Main(m, map[string]func(){
		"invox":       func() { os.Exit(cli.Run(os.Args[1:])) },
		"tectonic":    func() { os.Exit(fakeTectonic(os.Args[1:])) },
		"fake-editor": func() { os.Exit(fakeEditor(os.Args[1:])) },
		"open":        func() { os.Exit(fakeOpen(os.Args[1:])) },
		"xdg-open":    func() { os.Exit(fakeOpen(os.Args[1:])) },
		"osascript":   func() { os.Exit(fakeOsascript(os.Args[1:])) },
		// On Unix invox runs the editor as `$SHELL -lc 'eval "$INVOX_EDITOR" ...'`.
		// A real login shell would source the host's /etc/profile, so $SHELL
		// points here instead.
		"fake-shell": func() { os.Exit(fakeShell(os.Args[1:])) },
		// On Windows invox runs the editor as `cmd /c EDITOR FILE` and opens
		// documents with `cmd /c start "" FILE`. PATH holds only the fakes, so
		// this `cmd` is the one it finds.
		"cmd": func() { os.Exit(fakeCmd(os.Args[1:])) },
	})
}

func TestScript(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir:                 filepath.Join("testdata", "script"),
		Setup:               setupSandbox,
		RequireExplicitExec: true,
		UpdateScripts:       *updateScripts,
		Cmds: map[string]func(ts *testscript.TestScript, neg bool, args []string){
			"exits":      cmdExits,
			"scrubpaths": cmdScrubPaths,
		},
	})
}

// setupSandbox points every directory invox reads from the environment into
// $WORK/home and leaves only the fake programs on PATH, so a script never
// sees the developer's config, archive or real tools.
//
//	config dir:  $WORK/home/.config/invox (XDG_CONFIG_HOME, on every OS)
//	archive dir: OS-specific under $WORK/home; scripts that print archive
//	             paths set archive.dir in config.yaml instead
func setupSandbox(env *testscript.Env) error {
	home := filepath.Join(env.WorkDir, "home")
	env.Setenv("HOME", home)
	env.Setenv("USERPROFILE", home)
	env.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	env.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	env.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))

	// The default archive directory lives in a different per-user data
	// directory on each OS. scrubpaths needs to know it to replace it.
	switch runtime.GOOS {
	case "darwin":
		env.Setenv("DATA_HOME", filepath.Join(home, "Library", "Application Support"))
		env.Setenv("DATA_HOME_TILDE", "~/Library/Application Support")
	case "windows":
		env.Setenv("DATA_HOME", filepath.Join(home, "AppData", "Roaming"))
		env.Setenv("DATA_HOME_TILDE", "~/AppData/Roaming")
	default:
		env.Setenv("DATA_HOME", filepath.Join(home, ".local", "share"))
		env.Setenv("DATA_HOME_TILDE", "~/.local/share")
	}

	invox, err := exec.LookPath("invox")
	if err != nil {
		return fmt.Errorf("locate the testscript bin directory: %w", err)
	}
	binDir := filepath.Dir(invox)
	env.Setenv("PATH", binDir)
	// $BIN lets a script run invox with a PATH that lacks the fakes.
	env.Setenv("BIN", binDir)

	env.Setenv("SHELL", filepath.Join(binDir, "fake-shell"))
	env.Setenv("VISUAL", "fake-editor")
	env.Setenv("EDITOR", "fake-editor")
	return nil
}

// cmdExits runs a program like exec but asserts its exact exit code:
//
//	exits CODE program [args...]
func cmdExits(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! exits")
	}
	if len(args) < 2 {
		ts.Fatalf("usage: exits CODE program [args...]")
	}
	want, err := strconv.Atoi(args[0])
	if err != nil {
		ts.Fatalf("exits: invalid exit code %q", args[0])
	}

	got := 0
	if err := ts.Exec(args[1], args[2:]...); err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			ts.Fatalf("exits: %v", err)
		}
		got = exitErr.ExitCode()
	}
	if got != want {
		ts.Fatalf("%s exited with code %d, want %d", args[1], got, want)
	}
}

// cmdScrubPaths makes output that contains sandbox paths comparable on every
// OS. It writes SOURCE (stdout, stderr or a file) to OUTFILE with the
// per-user data directory replaced by $DATA_HOME (or $DATA_HOME_TILDE, as
// config.yaml spells it) and the work directory by $WORK. On Windows it also
// turns the backslashes in those paths, and in paths relative to $WORK that
// start with home\, into slashes. Compare OUTFILE with cmp, which -update can
// rewrite:
//
//	scrubpaths SOURCE OUTFILE
func cmdScrubPaths(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! scrubpaths")
	}
	if len(args) != 2 {
		ts.Fatalf("usage: scrubpaths SOURCE OUTFILE")
	}
	text := ts.ReadFile(args[0])
	text = strings.ReplaceAll(text, ts.Getenv("DATA_HOME"), "$DATA_HOME")
	text = strings.ReplaceAll(text, ts.Getenv("DATA_HOME_TILDE"), "$DATA_HOME_TILDE")
	text = strings.ReplaceAll(text, ts.Getenv("WORK"), "$WORK")
	if runtime.GOOS == "windows" {
		text = windowsPathToken.ReplaceAllStringFunc(text, func(token string) string {
			return strings.ReplaceAll(token, `\`, "/")
		})
	}
	ts.Check(os.WriteFile(ts.MkAbs(args[1]), []byte(text), 0o666))
}

// windowsPathToken matches a sandbox path up to the next whitespace.
var windowsPathToken = regexp.MustCompile(`(\$WORK|\$DATA_HOME|\bhome)\\\S*`)

// fakeTectonic writes an empty PDF next to its single .tex argument, or fails
// when FAKE_TECTONIC_FAIL is set.
func fakeTectonic(args []string) int {
	if os.Getenv(fakeTectonicFailEnv) != "" {
		fmt.Fprintln(os.Stderr, "fake tectonic: forced failure")
		return 1
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "fake tectonic: want one input file, got %q\n", args)
		return 2
	}
	pdfPath := strings.TrimSuffix(args[0], filepath.Ext(args[0])) + ".pdf"
	if err := os.WriteFile(pdfPath, nil, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "fake tectonic: %v\n", err)
		return 1
	}
	return 0
}

// fakeEditor prints the name of the file it was asked to edit, or fails when
// FAKE_EDITOR_FAIL is set.
func fakeEditor(args []string) int {
	if os.Getenv(fakeEditorFailEnv) != "" {
		fmt.Fprintln(os.Stderr, "fake editor: forced failure")
		return 1
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "fake editor: want one file, got %q\n", args)
		return 2
	}
	fmt.Printf("fake editor: %s\n", filepath.Base(args[0]))
	return 0
}

// fakeOpen stands in for open (macOS), xdg-open (Linux) and `cmd /c start`
// (Windows). It prints the name of the file it was asked to open, or fails
// when FAKE_OPEN_FAIL is set.
func fakeOpen(args []string) int {
	if os.Getenv(fakeOpenFailEnv) != "" {
		fmt.Fprintln(os.Stderr, "fake open: forced failure")
		return 1
	}
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "fake open: want one file, got %q\n", args)
		return 2
	}
	fmt.Printf("fake open: %s\n", filepath.Base(args[0]))
	return 0
}

// fakeOsascript stands in for the Apple Mail AppleScript that `invox email`
// runs on macOS. invox passes the message after "--" as recipient, subject,
// body, attachment and sender.
func fakeOsascript(args []string) int {
	for i, arg := range args {
		if arg != "--" {
			continue
		}
		message := args[i+1:]
		if len(message) != 5 {
			break
		}
		fmt.Printf("fake osascript: to=%s subject=%q attachment=%s\n", message[0], message[1], filepath.Base(message[3]))
		return 0
	}
	fmt.Fprintf(os.Stderr, "fake osascript: unexpected arguments %q\n", args)
	return 2
}

// fakeShell stands in for the login shell invox runs the editor with:
//
//	$SHELL -lc 'eval "$INVOX_EDITOR" '"$1"'' invox FILE
//
// It runs $INVOX_EDITOR, split on spaces, with FILE appended.
func fakeShell(args []string) int {
	if len(args) != 4 || args[0] != "-lc" || args[2] != "invox" {
		fmt.Fprintf(os.Stderr, "fake shell: unexpected arguments %q\n", args)
		return 2
	}
	editor := strings.Fields(os.Getenv("INVOX_EDITOR"))
	if len(editor) == 0 {
		fmt.Fprintln(os.Stderr, "fake shell: INVOX_EDITOR is empty")
		return 2
	}
	return runPassthrough(editor[0], append(editor[1:], args[3])...)
}

// runPassthrough runs a program with this process's standard streams and
// returns its exit code.
func runPassthrough(name string, args ...string) int {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "%s: %v\n", filepath.Base(os.Args[0]), err)
		return 1
	}
	return 0
}

// fakeCmd handles the three ways invox uses cmd.exe on Windows:
//
//	cmd /c start "" FILE                  open a document
//	cmd /c "start \"\" /b cmd /c ..."      delete the email draft later
//	cmd /c EDITOR FILE                    run the editor
func fakeCmd(args []string) int {
	if len(args) < 2 || !strings.EqualFold(args[0], "/c") {
		fmt.Fprintf(os.Stderr, "fake cmd: unexpected arguments %q\n", args)
		return 2
	}
	switch {
	case strings.HasPrefix(args[1], "start "):
		// The delayed cleanup of the email draft: nothing to do.
		return 0
	case args[1] == "start":
		rest := args[2:]
		if len(rest) > 0 && rest[0] == "" {
			rest = rest[1:]
		}
		return fakeOpen(rest)
	}
	return runPassthrough(args[1], args[2:]...)
}
