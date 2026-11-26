// Package apperr marks errors the user can act on, so the API can show them as-is, while any
// other error is treated as a server fault: logged in full, shown to the client as a generic message.
package apperr

import (
	"errors"
	"fmt"
)

// Kinds. Match with errors.Is.
var (
	ErrInvalid     = errors.New("invalid input")            // 400: the request can't be done as asked
	ErrNotFound    = errors.New("not found")                // 404
	ErrConflict    = errors.New("conflict")                 // 409: clashes with current state
	ErrUnavailable = errors.New("temporarily unavailable") // 429: try again later (e.g. quota paused)
)

// userError carries a message written for the user and the kind it belongs to.
type userError struct {
	kind error
	msg  string
}

func (e *userError) Error() string { return e.msg }
func (e *userError) Unwrap() error { return e.kind }

func newf(kind error, format string, args ...any) error {
	return &userError{kind: kind, msg: fmt.Sprintf(format, args...)}
}

// Invalid returns a user-facing "invalid input" error.
func Invalid(format string, args ...any) error { return newf(ErrInvalid, format, args...) }

// NotFound returns a user-facing "not found" error.
func NotFound(format string, args ...any) error { return newf(ErrNotFound, format, args...) }

// Conflict returns a user-facing "conflict" error.
func Conflict(format string, args ...any) error { return newf(ErrConflict, format, args...) }

// Unavailable returns a user-facing "try again later" error.
func Unavailable(format string, args ...any) error { return newf(ErrUnavailable, format, args...) }

// IsUserFacing reports whether err (or anything it wraps) is one of the kinds above.
func IsUserFacing(err error) bool {
	return errors.Is(err, ErrInvalid) || errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrConflict) || errors.Is(err, ErrUnavailable)
}
