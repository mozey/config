package errors

import (
	"fmt"

	"github.com/pkg/errors"
)

// Base does not implement causer.
// It's the base for custom errors defined with this package
type Base struct {
	// msg is the error value
	msg string
}

func (e Base) Error() string {
	return e.msg
}

func NewCause(msg string) error {
	return Base{msg: msg}
}

// EmptyCause for matching errors created without a cause
var EmptyCause = NewCause("")

// Custom type for defining errors
type Custom struct {
	// code may also be used as the code string for msg
	code string
	// msg is the error value
	msg string
	// cause is an optional group for errors,
	// use empty value if not applicable
	cause string
}

// New creates a new error with msg
func New(msg string) error {
	return Newf(msg)
}

// New creates a new error with formatted msg
func Newf(format string, a ...any) error {
	return Custom{
		code:  format,
		msg:   fmt.Sprintf(format, a...),
		cause: "",
	}
}

// NewWithCause creates a new error with cause and msg
func NewWithCause(cause error, msg string) error {
	return NewWithCausef(cause, msg)
}

// NewType creates a new error with cause and formatted msg
func NewWithCausef(cause error, format string, a ...any) error {
	return Custom{
		code: format,
		msg:  fmt.Sprintf(format, a...),
		// Error message is the cause value
		cause: cause.Error(),
	}
}

func (e Custom) Error() string {
	return e.msg
}

// Is return true if, the type of the error is ErrExampleType,
// and the format matches the string passed into the constructor.
// Think of the format string as the unique error number of code.
// Do not call errors.Wrap or WithStack when creating errors below
func (e Custom) Is(err error) bool {
	v, ok := err.(Custom)
	if !ok {
		// Custom type must match for errors defined with this package
		return false
	}
	// Code must match for specific error
	return v.code == e.code
}

// Cause can be used to determine the cause of errors.
// Useful for grouping related errors,
// e.g. custom errors defined in the same package
func (e Custom) Cause() error {
	return NewCause(e.cause)
}

// Code is the format string used to create the message value
func (e Custom) Code() string {
	return e.code
}

// Is returns true for custom errors defined with this package,
// if the format code matches
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// WithStack may be used to annotate err with a stack trace,
// at the point where it is called.
// Don't use this when defining custom errors
func WithStack(err error) {
	errors.WithStack(err)
}

// Cause returns the underlying cause of the error
func Cause(err error) error {
	return errors.Cause(err)
}
