// Package iostreams owns the standard streams and answers every terminal
// question. Commands write to Out and ErrOut and read from In, never to the
// process streams directly.
package iostreams

import (
	"bytes"
	"io"
	"os"
)

type IOStreams struct {
	In     io.ReadCloser
	Out    io.Writer
	ErrOut io.Writer

	stdinIsTTY  bool
	stdoutIsTTY bool
	stderrIsTTY bool
	neverPrompt bool
}

// System returns the process streams. INVOX_FORCE_TTY set to anything but
// the empty string makes stdout count as a terminal.
func System() *IOStreams {
	return &IOStreams{
		In:          os.Stdin,
		Out:         os.Stdout,
		ErrOut:      os.Stderr,
		stdinIsTTY:  isTerminal(os.Stdin),
		stdoutIsTTY: os.Getenv("INVOX_FORCE_TTY") != "" || isTerminal(os.Stdout),
		stderrIsTTY: isTerminal(os.Stderr),
	}
}

// Test returns streams backed by the returned buffers. No stream is a
// terminal until a Set*TTY call says so.
func Test() (*IOStreams, *bytes.Buffer, *bytes.Buffer, *bytes.Buffer) {
	in := &bytes.Buffer{}
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	return &IOStreams{In: io.NopCloser(in), Out: out, ErrOut: errOut}, in, out, errOut
}

func (s *IOStreams) IsStdinTTY() bool  { return s.stdinIsTTY }
func (s *IOStreams) IsStdoutTTY() bool { return s.stdoutIsTTY }
func (s *IOStreams) IsStderrTTY() bool { return s.stderrIsTTY }

func (s *IOStreams) SetStdinTTY(isTTY bool)  { s.stdinIsTTY = isTTY }
func (s *IOStreams) SetStdoutTTY(isTTY bool) { s.stdoutIsTTY = isTTY }
func (s *IOStreams) SetStderrTTY(isTTY bool) { s.stderrIsTTY = isTTY }

func (s *IOStreams) SetNeverPrompt(never bool) { s.neverPrompt = never }

// CanPrompt reports whether the user can answer a prompt: prompting is not
// disabled, answers come from a terminal and the question shows on one.
func (s *IOStreams) CanPrompt() bool {
	return !s.neverPrompt && s.stdinIsTTY && s.stderrIsTTY
}
