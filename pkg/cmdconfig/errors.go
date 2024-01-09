package cmdconfig

import "github.com/mozey/errors"

var ErrCmdConfig = errors.NewCause("cmdconfig")

var ErrDuplicateKey = func(key string) error {
	return errors.NewWithCausef(ErrCmdConfig, "duplicate key %s", key)
}

var ErrNotImplemented = errors.NewWithCausef(ErrCmdConfig, "not implemented")

var ErrParentNotFound = errors.NewWithCausef(
	ErrCmdConfig, "parent config not found")
