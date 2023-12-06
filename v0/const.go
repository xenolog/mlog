package mlog

import (
	"log/slog"
	"log/syslog"
	"time"
)

const (
	TimeOutputFormatRFC3339       = "2006-01-02T15:04:05.000000Z07"
	DefaultSyslogLineHeaderFormat = "<%d>%s%s %s[%d]: "

	LogLineBuffSize           = 1024
	InitialIOBufSize          = 1024 * 8
	InitialLineProcessBufSize = 1024
	AttrsJSONprefix           = "ATTRS="
	defaultDialTimeout        = 15 * time.Second
)

const ( // see log/syslog
	Syslog_KERN     = Facility(syslog.LOG_KERN)
	Syslog_USER     = Facility(syslog.LOG_USER)
	Syslog_MAIL     = Facility(syslog.LOG_MAIL)
	Syslog_DAEMON   = Facility(syslog.LOG_DAEMON)
	Syslog_AUTH     = Facility(syslog.LOG_AUTH)
	Syslog_SYSLOG   = Facility(syslog.LOG_SYSLOG)
	Syslog_LPR      = Facility(syslog.LOG_LPR)
	Syslog_NEWS     = Facility(syslog.LOG_NEWS)
	Syslog_UUCP     = Facility(syslog.LOG_UUCP)
	Syslog_CRON     = Facility(syslog.LOG_CRON)
	Syslog_AUTHPRIV = Facility(syslog.LOG_AUTHPRIV)
	Syslog_FTP      = Facility(syslog.LOG_FTP)
	Syslog_LOCAL0   = Facility(syslog.LOG_LOCAL0)
	Syslog_LOCAL1   = Facility(syslog.LOG_LOCAL1)
	Syslog_LOCAL2   = Facility(syslog.LOG_LOCAL2)
	Syslog_LOCAL3   = Facility(syslog.LOG_LOCAL3)
	Syslog_LOCAL4   = Facility(syslog.LOG_LOCAL4)
	Syslog_LOCAL5   = Facility(syslog.LOG_LOCAL5)
	Syslog_LOCAL6   = Facility(syslog.LOG_LOCAL6)
	Syslog_LOCAL7   = Facility(syslog.LOG_LOCAL7)

	defaultSyslogSeverity = Severity(syslog.LOG_NOTICE)
	defaultSyslogFacility = Syslog_USER

	severityMask = Severity(0x07)
	facilityMask = Facility(0xf8)
)

type (
	Facility uint8
	Severity uint8
)

//nolint:gochecknoglobals
var (
	level2Letter = map[slog.Level]string{
		slog.LevelDebug: " D ",
		slog.LevelInfo:  " I ",
		slog.LevelWarn:  " W ",
		slog.LevelError: " E ",
	}

	level2severity = map[slog.Level]Severity{
		slog.LevelDebug: Severity(syslog.LOG_DEBUG),
		slog.LevelInfo:  Severity(syslog.LOG_INFO),
		slog.LevelWarn:  Severity(syslog.LOG_WARNING),
		slog.LevelError: Severity(syslog.LOG_ERR),
	}
)
