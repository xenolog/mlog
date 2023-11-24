package mlog

import (
	"context"
	"log/slog"
	"slices"
)

type MultipleHandlerOptions struct { // todo(sv): reserved to the future to prevend broke interface
}

// MultipleHandler is a [slog.Handler] that multiply Records to each given handler as is.
type MultipleHandler struct {
	level    slog.Level
	handlers []slog.Handler
}

// NewMultipleHandler creates a MultipleHandler that multiply each incoming message to each given handler,
// using the given options. If opts is nil, the default options are used.
func NewMultipleHandler(handlerSet []slog.Handler, _ *MultipleHandlerOptions) *MultipleHandler {
	h := &MultipleHandler{
		handlers: handlerSet,
		level:    (^slog.Level(0) >> 1), // maximum value of slog.Level type
	}
exLoop:
	for _, logLevel := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError} {
		for _, hh := range h.handlers {
			if hh.Enabled(context.TODO(), logLevel) {
				h.level = logLevel
				break exLoop
			}
		}
	}
	return h
}

func (h *MultipleHandler) Copy() *MultipleHandler {
	rv := &MultipleHandler{
		level:    h.level,
		handlers: slices.Clone(h.handlers),
	}
	return rv
}

// Enabled reports whether the handler handles records at the given level. The handler ignores records whose level is lower.
// Implements [slog.Handler] interface.
func (h *MultipleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// WithAttrs returns a new HumanReadableHandler whose attributes consists of h's attributes followed by attrs.
// Implements [slog.Handler] interface.
func (h *MultipleHandler) WithAttrs(aa []slog.Attr) slog.Handler {
	rv := h.Copy()
	for i := range h.handlers {
		rv.handlers[i] = h.handlers[i].WithAttrs(aa)
	}
	return rv
}

// WithGroup returns a new HumanReadableHandler with the given group appended to the receiver's existing groups.
// Implements [slog.Handler] interface.
func (h *MultipleHandler) WithGroup(name string) slog.Handler {
	var rv *MultipleHandler
	if name != "" {
		rv = h.Copy()
		for i := range h.handlers {
			rv.handlers[i] = h.handlers[i].WithGroup(name)
		}
	} else {
		rv = h
	}
	return rv
}

// Handle handles the Record.
// It will only be called when Enabled returns true.
// Implements [slog.Handler] interface.
func (h *MultipleHandler) Handle(ctx context.Context, r slog.Record) error { //nolint:gocritic
	var firstErr error
	for i := range h.handlers {
		if h.handlers[i].Enabled(ctx, r.Level) {
			if err := h.handlers[i].Handle(ctx, r); err != nil {
				if firstErr != nil {
					firstErr = err
				}
			}
		}
	}
	return firstErr
}
