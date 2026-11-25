package errs

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"strings"
)

// LogValue implements slog.LogValuer, producing structured key-value pairs.
// Fields: code, message, debug (if set), meta (if set), cause (if set), stack (if set).
func (e *Error) LogValue() slog.Value {
	if e == nil {
		return slog.GroupValue(slog.String("message", "nil *errs.Error"))
	}
	attrs := []slog.Attr{
		slog.Int("code", e.code),
		slog.String("message", e.message),
	}

	if debug := e.Debug(); debug != "" {
		attrs = append(attrs, slog.String("debug", debug))
	}

	meta := maps.Clone(e.meta)
	delete(meta, "debug")
	if len(meta) > 0 {
		attrs = append(attrs, slog.Any("meta", meta))
	}

	if e.cause != nil {
		if inner, ok := e.cause.(*Error); ok {
			if inner == nil {
				attrs = append(attrs, slog.String("cause", "nil *errs.Error"))
			} else {
				attrs = append(attrs, slog.Any("cause", inner.LogValue()))
			}
		} else {
			attrs = append(attrs, slog.String("cause", fmt.Sprint(e.cause)))
		}
	}

	if frames := e.StackFrames(); len(frames) > 0 {
		attrs = append(attrs, slog.String("stack", strings.Join(frames, "\n")))
	}

	return slog.GroupValue(attrs...)
}

// SlogAttr returns an slog Attr that logs err using LogValue() for structured output.
// It finds the first non-nil *Error in err's chain, so wrapped errors keep the
// structured output. A typed-nil *Error is logged as text. Errors without an
// *Error in their chain are logged with slog.Any.
// Only the found *Error's own fields are logged, so text added by outer
// wrappers, such as "handler: " in fmt.Errorf("handler: %w", e), is not
// included; put that context in the log message instead.
// Usage: slog.Error("msg", errs.SlogAttr(err)).
func SlogAttr(err error) slog.Attr {
	if e := firstError(err); e != nil {
		return slog.Attr{Key: "err", Value: e.LogValue()}
	}
	if e, ok := errors.AsType[*Error](err); ok && e == nil {
		return slog.String("err", errorText(err))
	}
	return slog.Any("err", err)
}

// errorText returns a panic-free description of err.
func errorText(err error) string {
	if e, ok := err.(*Error); ok && e == nil {
		return "nil *errs.Error"
	}
	return fmt.Sprint(err)
}
