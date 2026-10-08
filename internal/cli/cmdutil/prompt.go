package cmdutil

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/0xboris/invox/internal/iostreams"
)

// Confirm asks question on stderr and reads a yes/no answer. Anything but
// y or yes, including end of input, is a no. It returns CancelError
// when ctx is cancelled before an answer arrives.
func Confirm(ctx context.Context, ios *iostreams.IOStreams, question string) (bool, error) {
	fmt.Fprintf(ios.ErrOut, "%s [y/N] ", question)
	type reply struct {
		answer string
		err    error
	}
	replies := make(chan reply, 1)
	go func() {
		answer, err := bufio.NewReader(ios.In).ReadString('\n')
		replies <- reply{answer, err}
	}()
	var r reply
	select {
	case <-ctx.Done():
		fmt.Fprintln(ios.ErrOut)
		return false, CancelError
	case r = <-replies:
	}
	if r.err != nil && r.answer == "" {
		fmt.Fprintln(ios.ErrOut)
		return false, nil
	}
	switch strings.ToLower(strings.TrimSpace(r.answer)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
