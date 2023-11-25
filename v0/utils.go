package mlog

import (
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type TimeFormat struct {
	format string
	words  int
}

//nolint:gochecknoglobals
var (
	allowedLevels = []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn, slog.LevelError}
	timeFormats   = []TimeFormat{{time.RFC3339Nano, 0}, {time.UnixDate, 0}, {time.RFC1123, 0}, {time.RFC1123Z, 0}, {time.DateTime, 0}, {time.StampNano, 0}, {time.Stamp, 0}}
	anySpaces     = regexp.MustCompile(`\s+`)
)

//nolint:gochecknoinits
func init() {
	for i := range timeFormats {
		timeFormats[i].words = len(strings.Fields(timeFormats[i].format))
	}
}

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

// TrimTimestamp got slice and detect whether it starts with timestamp.
// returns line with trimmed timestamp if detected
func TrimTimestamp(b []byte) []byte {
	for i := range timeFormats {
		fields := anySpaces.Split(strings.TrimSpace(string(b)), timeFormats[i].words+1)
		tss := strings.Join(fields[:len(fields)-1], " ")
		_, err := time.Parse(timeFormats[i].format, tss)
		if err == nil {
			// time parsed successfully
			return []byte(fields[len(fields)-1]) // last field
		}
	}
	return b // timestamp not recognized
}
