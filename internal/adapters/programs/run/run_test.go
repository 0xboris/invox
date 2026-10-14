package run

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
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

func testExecutable(t *testing.T) string {
	t.Helper()
	self, err := os.Executable()
	if err != nil {
		t.Fatalf("os.Executable() returned error: %v", err)
	}
	return self
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
		Name:   testExecutable(t),
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
	name := testExecutable(t)

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

func TestStubReturnsTheRegisteredResultsInOrder(t *testing.T) {
	stub := NewStub(t)
	var got []string
	stub.Register("open", func(cmd Cmd) error {
		got = append(got, cmd.Args[0])
		return nil
	})
	stub.Register("open", func(cmd Cmd) error {
		got = append(got, cmd.Args[0])
		return &ExecError{Name: "open", Code: 1}
	})

	first := stub.Run(context.Background(), Cmd{Name: "open", Args: []string{"a.eml"}})
	second := stub.Run(context.Background(), Cmd{Name: "open", Args: []string{"b.eml"}})

	if first != nil || second == nil || second.Error() != "exit status 1" {
		t.Fatalf("results = (%v, %v), want (nil, exit status 1)", first, second)
	}
	if want := []string{"a.eml", "b.eml"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestStubPanicsOnUnregisteredCommand(t *testing.T) {
	stub := NewStub(t)
	stub.Register("open", func(Cmd) error { return nil })
	_ = stub.Run(context.Background(), Cmd{Name: "open"})

	defer func() {
		if got, want := fmt.Sprint(recover()), `run.Stub: unregistered command open ["x.eml"]`; got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	_ = stub.Run(context.Background(), Cmd{Name: "open", Args: []string{"x.eml"}})
}

type recordingT struct {
	cleanups []func()
	errors   []string
}

func (r *recordingT) Helper()                   {}
func (r *recordingT) Cleanup(fn func())         { r.cleanups = append(r.cleanups, fn) }
func (r *recordingT) Errorf(f string, a ...any) { r.errors = append(r.errors, fmt.Sprintf(f, a...)) }

func registerOsascript(stub *Stub) {
	stub.Register("osascript", func(Cmd) error { return nil })
}

func TestStubFailsWhenARegisteredCommandNeverRuns(t *testing.T) {
	rt := &recordingT{}
	stub := NewStub(rt)
	_, _, line, _ := runtime.Caller(0)
	registerOsascript(stub)
	stub.Register("tectonic", func(Cmd) error { return nil })
	_ = stub.Run(context.Background(), Cmd{Name: "tectonic"})

	for _, fn := range rt.cleanups {
		fn()
	}

	want := []string{fmt.Sprintf("run.Stub: osascript registered at run_test.go:%d was never run", line+1)}
	if fmt.Sprint(rt.errors) != fmt.Sprint(want) {
		t.Fatalf("errors = %q, want %q", rt.errors, want)
	}
}
