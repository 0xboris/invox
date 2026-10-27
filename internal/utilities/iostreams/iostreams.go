// Package iostreams owns the standard streams and answers every terminal
// question. Commands write to Out and ErrOut and read from In, never to the
// process streams directly.
package iostreams

import (
	"bytes"
	"io"
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

// New returns streams on in, out and errOut, with the given terminal flags.
func New(in io.ReadCloser, out, errOut io.Writer, stdinIsTTY, stdoutIsTTY, stderrIsTTY bool) *IOStreams {
	return &IOStreams{
		In:          in,
		Out:         out,
		ErrOut:      errOut,
		stdinIsTTY:  stdinIsTTY,
		stdoutIsTTY: stdoutIsTTY,
		stderrIsTTY: stderrIsTTY,
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

// PromptDisabled reports whether SetNeverPrompt turned prompting off.
func (s *IOStreams) PromptDisabled() bool { return s.neverPrompt }

// CanPrompt reports whether the user can answer a prompt: prompting is not
// disabled, answers come from a terminal and the question shows on one.
func (s *IOStreams) CanPrompt() bool {
	return !s.neverPrompt && s.stdinIsTTY && s.stderrIsTTY
}
