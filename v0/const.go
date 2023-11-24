package mlog

import "log/slog"

const (
	TimeOutputFormatRFC3339 = "2006-01-02T15:04:05.000000Z07"
	LogLineBuffSize         = 1024
	AttrsJSONprefix         = "ATTRS="
)

var level2Letter = map[slog.Level]string{ //nolint:gochecknoglobals
	slog.LevelDebug: " D ",
	slog.LevelInfo:  " I ",
	slog.LevelWarn:  " W ",
	slog.LevelError: " E ",
}
