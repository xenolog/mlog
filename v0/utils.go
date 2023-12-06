package mlog

import (
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"
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
func TrimTimestamp(b []byte) ([]byte, error) {
	for i := range timeFormats {
		fields := anySpaces.Split(strings.TrimSpace(string(b)), timeFormats[i].words+1)
		tss := strings.Join(fields[:len(fields)-1], " ")
		if _, err := time.Parse(timeFormats[i].format, tss); err == nil {
			// time parsed successfully
			return []byte(fields[len(fields)-1]), nil // last field
		}
	}
	return b, &time.ParseError{ // timestamp not recognized
		Message: "Unable to parse timestamp",
	}
}

// ScanJSONobject is a split function for a Scanner that returns each JSON object {...} of input stream.
// Any any surrounding text skipped. The returned JSON object may be empty.
func ScanJSONobject(data []byte, _ bool) (advance int, token []byte, err error) {
	var (
		r                rune
		rWidth           int
		pos              int
		bracketCount     int
		process          bool
		startIDX, endIDX int
	)

	// log.Printf("%v:'%s'\n", atEOF, string(data))
	// processLoop:
	for pos = 0; pos < len(data); pos += rWidth {
		r, rWidth = utf8.DecodeRune(data[pos:])
		if rWidth > 1 { // unicode rune, do nothing
			continue
		}
		switch r {
		case '"':
			if bracketCount > 0 { // json object started
				process = !process
			}
		case '{':
			if process {
				bracketCount++
				break // switch
			}
			if bracketCount == 0 { // start of JSON object
				process = true
				bracketCount = 1
				startIDX = pos
			}
		case '}':
			if process {
				bracketCount--
				break // switch
			}
		}

		if process && bracketCount == 0 { // finish `}` at JSON object achieved
			endIDX = pos
			break // processLoop
		}
	}

	if endIDX != 0 {
		return endIDX, data[startIDX : endIDX+1], nil
	}
	return 0, nil, nil
}
