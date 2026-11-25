package errs

import (
	"errors"
	"fmt"
	"runtime"
)

const maxStackDepth = 32

// New creates a new Error with the given code and message.
// Stack trace is eagerly captured at the call site.
func New(code int, message string) *Error {
	return &Error{
		code:    code,
		message: message,
		stack:   captureStack(1),
	}
}

// Wrap creates a new Error wrapping a cause with the given code and message.
// Stack trace is eagerly captured at the call site.
func Wrap(code int, message string, cause error) *Error {
	return &Error{
		code:    code,
		message: message,
		cause:   cause,
		stack:   captureStack(1),
	}
}

// AsError extracts an *Error from an error chain.
// If err is nil, returns nil. If err wraps a non-nil *Error, returns that
// *Error. A typed-nil *Error is ignored. If err is a plain error, wraps it
// with code 0 (unknown).
func AsError(err error) *Error {
	if err == nil {
		return nil
	}
	if e := firstError(err); e != nil {
		return e
	}
	message := fmt.Sprint(err)
	cause := err
	if e, ok := err.(*Error); ok && e == nil {
		message = "nil *errs.Error"
		cause = nil
	}
	return &Error{
		code:    0,
		message: message,
		cause:   cause,
		stack:   captureStack(1),
	}
}

// firstError returns the first non-nil *Error in err's chain.
func firstError(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok && e == nil {
		return nil
	}
	e, ok := errors.AsType[*Error](err)
	if !ok {
		return nil
	}
	if e != nil {
		return e
	}
	// A typed-nil *Error matched. Walk past it to find a non-nil one, which
	// errors.AsType can hide in an errors.Join.
	switch x := err.(type) {
	case interface{ Unwrap() []error }:
		for _, child := range x.Unwrap() {
			if e := firstError(child); e != nil {
				return e
			}
		}
	case interface{ Unwrap() error }:
		return firstError(x.Unwrap())
	}
	return nil
}

func captureStack(skip int) []uintptr {
	var pcs [maxStackDepth]uintptr
	n := runtime.Callers(skip+1, pcs[:])
	return pcs[:n]
}
