package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/0xboris/invox/internal/testfixture"
)

const childEnv = "RUN_TEST_CHILD"

func TestMain(m *testing.M) {
	if mode := os.Getenv(cancelChildEnv); mode != "" {
		os.Exit(runCancelChild(mode))
	}
	if os.Getenv(childEnv) != "" {
		cwd, _ := os.Getwd()
		fmt.Printf("dir=%s value=%s args=%q\n", filepath.Base(cwd), os.Getenv("RUN_TEST_VALUE"), os.Args[1:])
		fmt.Fprintln(os.Stderr, "to stderr")
		code, _ := strconv.Atoi(os.Args[1])
		os.Exit(code)
	}
	os.Exit(m.Run())
}

func TestExecRunsInDirWithAddedEnv(t *testing.T) {
	t.Setenv(childEnv, "1")
	dir := filepath.Join(t.TempDir(), "work")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer

	err := Exec{}.Run(context.Background(), Cmd{
		Dir:    dir,
		Name:   testfixture.Executable(t),
		Args:   []string{"0", "a b"},
		Env:    []string{"RUN_TEST_VALUE=added"},
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got, want := stdout.String(), "dir=work value=added args=[\"0\" \"a b\"]\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if got, want := stderr.String(), "to stderr\n"; got != want {
		t.Fatalf("stderr = %q, want %q", got, want)
	}
}

func TestExecReturnsExecErrorWithExitCode(t *testing.T) {
	t.Setenv(childEnv, "1")
	name := testfixture.Executable(t)

	err := Exec{}.Run(context.Background(), Cmd{Name: name, Args: []string{"3"}})

	var execErr *ExecError
	if !errors.As(err, &execErr) {
		t.Fatalf("Run error = %v (%T), want *ExecError", err, err)
	}
	if execErr.Name != name || execErr.Code != 3 {
		t.Fatalf("ExecError = {Name: %q, Code: %d}, want {Name: %q, Code: 3}", execErr.Name, execErr.Code, name)
	}
	if got, want := err.Error(), "exit status 3"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}

func TestExecReportsMissingProgram(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	err := Exec{}.Run(context.Background(), Cmd{Name: "invox-no-such-program"})

	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Run error = %v, want ErrNotFound", err)
	}
}

func TestExecErrorWithoutErrPrintsTheCode(t *testing.T) {
	if got, want := (&ExecError{Name: "tectonic", Code: 2}).Error(), "exit status 2"; got != want {
		t.Fatalf("Error() = %q, want %q", got, want)
	}
}
