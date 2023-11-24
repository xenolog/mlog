package mlog

import (
	"log/slog"
	"runtime"
)

// DecodeSource returns a new [slog.Source] which describes
// the location of a line of source code.
func DecodeSource(pc uintptr) *slog.Source {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return &slog.Source{
		Function: f.Function,
		File:     f.File,
		Line:     f.Line,
	}
}
