package cmdconfig

import (
	"io"
	"io/fs"
	"os"

	"github.com/pkg/errors"
)

// Copy the src file to dst.
// Return an error if dst exists.
// Inspired by https://stackoverflow.com/a/21061062/639133
func Copy(src, dst string) error {
	// Open src
	in, err := os.Open(src)
	if err != nil {
		return errors.WithStack(err)
	}
	defer in.Close()

	_, err = os.Stat(dst)
	if err != nil {
		if os.IsNotExist(err) {
			// Continue, dst does not exist
		} else {
			// Could not stat file info, abort
			return errors.WithStack(err)
		}
	} else {
		// The dst already exists
		return errors.WithStack(fs.ErrExist)
	}

	// Create dst
	out, err := os.Create(dst)
	if err != nil {
		return errors.WithStack(err)
	}
	defer out.Close()

	// Copy
	_, err = io.Copy(out, in)
	if err != nil {
		return errors.WithStack(err)
	}
	return out.Close()
}
