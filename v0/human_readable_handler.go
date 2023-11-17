package mlog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"maps"
	"path/filepath"
	"sync"
)

type jsonTree map[string]any

type group struct {
	name  string
	attrs jsonTree
}

type HumanReadableHandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	AddSource        bool
	AddSourceToAttrs bool
	UseLocalTZ       bool

	// Level reports the minimum level to log.
	// Levels with lower levels are discarded.
	// If nil, the Handler uses [slog.LevelInfo].
	Level slog.Leveler
}

type HumanReadableHandler struct {
	opts   HumanReadableHandlerOptions
	groups []group
	// TODO: state for WithGroup and WithAttrs
	mu  *sync.Mutex
	out io.Writer
}

func NewHumanReadableHandler(out io.Writer, opts *HumanReadableHandlerOptions) *HumanReadableHandler {
	h := &HumanReadableHandler{
		out: out,
		mu:  &sync.Mutex{},
		groups: []group{{ // group[0] always exists, has no name and used
			attrs: jsonTree{}, // to store non-groupped attrs
		}},
	}
	if opts != nil {
		h.opts = *opts
	}
	if h.opts.Level == nil {
		h.opts.Level = slog.LevelInfo
	}
	return h
}

func (h *HumanReadableHandler) Copy() *HumanReadableHandler {
	rv := &HumanReadableHandler{
		opts:   h.opts,
		out:    h.out,
		mu:     h.mu,
		groups: make([]group, len(h.groups)),
	}
	for i := range h.groups {
		rv.groups[i].name = h.groups[i].name
		rv.groups[i].attrs = maps.Clone(h.groups[i].attrs)
	}
	return rv
}

func (h *HumanReadableHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.opts.Level.Level()
}

func (h *HumanReadableHandler) Handle(_ context.Context, r slog.Record) error { //nolint:gocritic
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
		source := DecodeSource(r.PC)
		buf = fmt.Appendf(buf, "[%s:%d]  ", filepath.Base(source.File), source.Line)
	} else {
		buf = fmt.Appendf(buf, "--  ")
	}

	buf = append(buf, r.Message...)

	attrs := jsonTree{}
	ptr := attrs // pointer to group (or tree root) to store record attributes
	for i := range h.groups {
		if h.groups[i].name != "" {
			ptr[h.groups[i].name] = jsonTree{}
			ptr = ptr[h.groups[i].name].(jsonTree) //revive:disable:unchecked-type-assertion // because created with right type in the previous line
		}
		maps.Copy(ptr, h.groups[i].attrs)
	}

	if h.opts.AddSourceToAttrs && r.PC != 0 {
		attrs[slog.SourceKey] = DecodeSource(r.PC)
	}

	// fill Attrs from records
	r.Attrs(func(a slog.Attr) bool {
		ptr[a.Key] = a.Value.Any()
		return true
	})

	// serialize and store JSON into buffer
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

func (h *HumanReadableHandler) WithAttrs(aa []slog.Attr) slog.Handler {
	hh := h.Copy()
	idx := len(hh.groups) - 1
	for k := range aa {
		hh.groups[idx].attrs[aa[k].Key] = aa[k].Value.Any()
	}
	return hh
}

func (h *HumanReadableHandler) WithGroup(name string) slog.Handler {
	var hh *HumanReadableHandler
	if name != "" {
		hh = h.Copy()
		hh.groups = append(hh.groups, group{
			name:  name,
			attrs: jsonTree{},
		})
	} else {
		hh = h
	}
	return hh
}
