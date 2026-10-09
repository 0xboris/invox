package cmdutil

import (
	"errors"
	"fmt"

	"github.com/0xboris/invox/internal/billing"
)

// OutputChoice is what OutputError offers in place of an -o path that
// cannot be written.
const OutputChoice = "a different -o/--output path"

// OutputError words err when the file a command writes already exists
// (*billing.OutputExistsError) or is a directory
// (*billing.OutputIsDirError), offering choice, such as OutputChoice, in its
// place. Any other err comes back as it is.
func OutputError(err error, choice string) error {
	var exists *billing.OutputExistsError
	if errors.As(err, &exists) {
		return fmt.Errorf("%s; pass --force to replace it or choose %s", exists, choice)
	}
	var isDir *billing.OutputIsDirError
	if errors.As(err, &isDir) {
		return fmt.Errorf("%s; choose %s", isDir, choice)
	}
	return err
}
