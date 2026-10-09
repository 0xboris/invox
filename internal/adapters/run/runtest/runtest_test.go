package runtest

import (
	"context"
	"fmt"
	"runtime"
	"testing"

	"github.com/0xboris/invox/internal/adapters/run"
)

func TestStubReturnsTheRegisteredResultsInOrder(t *testing.T) {
	stub := NewStub(t)
	var got []string
	stub.Register("open", func(cmd run.Cmd) error {
		got = append(got, cmd.Args[0])
		return nil
	})
	stub.Register("open", func(cmd run.Cmd) error {
		got = append(got, cmd.Args[0])
		return &run.ExecError{Name: "open", Code: 1}
	})

	first := stub.Run(context.Background(), run.Cmd{Name: "open", Args: []string{"a.eml"}})
	second := stub.Run(context.Background(), run.Cmd{Name: "open", Args: []string{"b.eml"}})

	if first != nil || second == nil || second.Error() != "exit status 1" {
		t.Fatalf("results = (%v, %v), want (nil, exit status 1)", first, second)
	}
	if want := []string{"a.eml", "b.eml"}; fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("args = %q, want %q", got, want)
	}
}

func TestStubPanicsOnUnregisteredCommand(t *testing.T) {
	stub := NewStub(t)
	stub.Register("open", func(run.Cmd) error { return nil })
	_ = stub.Run(context.Background(), run.Cmd{Name: "open"})

	defer func() {
		if got, want := fmt.Sprint(recover()), `runtest.Stub: unregistered command open ["x.eml"]`; got != want {
			t.Fatalf("panic = %q, want %q", got, want)
		}
	}()
	_ = stub.Run(context.Background(), run.Cmd{Name: "open", Args: []string{"x.eml"}})
}

type recordingT struct {
	cleanups []func()
	errors   []string
}

func (r *recordingT) Helper()                   {}
func (r *recordingT) Cleanup(fn func())         { r.cleanups = append(r.cleanups, fn) }
func (r *recordingT) Errorf(f string, a ...any) { r.errors = append(r.errors, fmt.Sprintf(f, a...)) }

func registerOsascript(stub *Stub) {
	stub.Register("osascript", func(run.Cmd) error { return nil })
}

func TestStubFailsWhenARegisteredCommandNeverRuns(t *testing.T) {
	rt := &recordingT{}
	stub := NewStub(rt)
	_, _, line, _ := runtime.Caller(0)
	registerOsascript(stub)
	stub.Register("tectonic", func(run.Cmd) error { return nil })
	_ = stub.Run(context.Background(), run.Cmd{Name: "tectonic"})

	for _, fn := range rt.cleanups {
		fn()
	}

	want := []string{fmt.Sprintf("runtest.Stub: osascript registered at runtest_test.go:%d was never run", line+1)}
	if fmt.Sprint(rt.errors) != fmt.Sprint(want) {
		t.Fatalf("errors = %q, want %q", rt.errors, want)
	}
}
