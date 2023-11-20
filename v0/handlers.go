package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"runtime"
	"sync"
)

const (
	TimeInputFormatRFC3339  = "2006-01-02T15:04:05.99999Z07"
	TimeOutputFormatRFC3339 = "2006-01-02T15:04:05.00000Z07"
	LogLineBuffSize         = 1024
	AttrsJSONprefix         = "ATTRS="
)

var level2Letter = map[slog.Level]string{ //nolint:gochecknoglobals
	slog.LevelDebug: " D ",
	slog.LevelInfo:  " I ",
	slog.LevelWarn:  " W ",
	slog.LevelError: " E ",
}

type HandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource bool

	UseLocalTZ bool

	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
}

type MixedHandler struct {
	opts              HandlerOptions
	preCollectedAttrs map[string]any
	// TODO: state for WithGroup and WithAttrs
	mu  *sync.Mutex
	out io.Writer
}

func NewHandler(out io.Writer, opts *HandlerOptions) *MixedHandler {
	h := &MixedHandler{
		out: out,
		mu:  &sync.Mutex{},
	}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	return h
}

func (h *MixedHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *MixedHandler) Handle(_ context.Context, r slog.Record) error { //nolint:gocritic
	buf := make([]byte, 0, LogLineBuffSize)
	if !r.Time.IsZero() {
		if h.opts.UseLocalTZ {
			buf = r.Time.AppendFormat(buf, TimeOutputFormatRFC3339)
		} else {
			buf = r.Time.UTC().AppendFormat(buf, TimeOutputFormatRFC3339)
		}
	}
	buf = append(buf, level2Letter[r.Level]...)

	if h.opts.AddSource && r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		buf = fmt.Appendf(buf, "[%s:%d]  ", filepath.Base(f.File), f.Line)
	} else {
		buf = fmt.Appendf(buf, "--  ")
	}

	buf = append(buf, r.Message...)

	attrs := map[string]any{}
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})
	if len(h.preCollectedAttrs) != 0 {
		for k := range h.preCollectedAttrs {
			attrs[k] = h.preCollectedAttrs[k]
		}
	}
	if len(attrs) != 0 {
		attrsJSON, err := json.Marshal(attrs)
		if err != nil {
			buf = append(buf, "slogERR: "+err.Error()...)
		} else {
			buf = append(buf, "  "+AttrsJSONprefix...)
			buf = append(buf, attrsJSON...)
		}
	}

	buf = append(buf, "\n"...)
	h.mu.Lock()
	defer h.mu.Unlock()
	_, err := h.out.Write(buf)
	if err != nil {
		err = errors.Join(Error, err)
	}
	return err
}

func (h *MixedHandler) WithAttrs(aa []slog.Attr) slog.Handler {
	hh := &MixedHandler{
		opts:              h.opts,
		out:               h.out,
		mu:                h.mu,
		preCollectedAttrs: map[string]any{},
	}
	for k := range aa {
		hh.preCollectedAttrs[aa[k].Key] = aa[k].Value.Any()
	}

	return hh
}

func (h *MixedHandler) WithGroup(name string) slog.Handler {
	// TODO: investigate and implement WithGroup functionality
	return h
}
