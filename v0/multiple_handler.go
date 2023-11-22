package mlog

import (
	"context"
	"log/slog"
	"slices"
)

type MultipleHandlerOptions struct { // todo(sv): reserved to the future to prevend broke interface
}

type MultipleHandler struct {
	level    slog.Level
	handlers []slog.Handler
}

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

func (h *MultipleHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *MultipleHandler) WithAttrs(aa []slog.Attr) slog.Handler {
	rv := h.Copy()
	for i := range h.handlers {
		rv.handlers[i] = h.handlers[i].WithAttrs(aa)
	}
	return rv
}

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
