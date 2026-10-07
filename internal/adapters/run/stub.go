package run

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// T is the part of testing.TB a Stub uses.
type T interface {
	Helper()
	Cleanup(func())
	Errorf(format string, args ...any)
}

// Stub is a Runner for tests. Each Register expects one call to the named
// program. Run panics on a call nothing is registered for, and the test fails
// at cleanup when a registered call never happened.
type Stub struct {
	mu    sync.Mutex
	calls []*stubCall
}

type stubCall struct {
	name string
	at   string
	fn   func(Cmd) error
	done bool
}

// NewStub returns a Stub that checks its registrations when t ends.
func NewStub(t T) *Stub {
	s := &Stub{}
	t.Cleanup(func() {
		t.Helper()
		s.mu.Lock()
		defer s.mu.Unlock()
		for _, call := range s.calls {
			if !call.done {
				t.Errorf("run.Stub: %s registered at %s was never run", call.name, call.at)
			}
		}
	})
	return s
}

// Register expects one run of the program name. fn receives the Cmd and
// returns the result of the run.
func (s *Stub) Register(name string, fn func(Cmd) error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, &stubCall{name: name, at: testCaller(), fn: fn})
}

// testCaller returns the file:line in the Test function that led to the
// Register call, or Register's direct caller when no Test function is on the
// stack, so a helper that registers points at the test that used it.
func testCaller() string {
	pcs := make([]uintptr, 32)
	frames := runtime.CallersFrames(pcs[:runtime.Callers(3, pcs)])
	first := ""
	for {
		frame, more := frames.Next()
		at := fmt.Sprintf("%s:%d", filepath.Base(frame.File), frame.Line)
		if first == "" {
			first = at
		}
		name := frame.Function[strings.LastIndex(frame.Function, "/")+1:]
		if strings.HasPrefix(name[strings.Index(name, ".")+1:], "Test") {
			return at
		}
		if !more {
			return first
		}
	}
}

func (s *Stub) Run(_ context.Context, cmd Cmd) error {
	s.mu.Lock()
	var match *stubCall
	for _, call := range s.calls {
		if !call.done && call.name == cmd.Name {
			call.done = true
			match = call
			break
		}
	}
	s.mu.Unlock()
	if match == nil {
		panic(fmt.Sprintf("run.Stub: unregistered command %s %q", cmd.Name, cmd.Args))
	}
	return match.fn(cmd)
}
