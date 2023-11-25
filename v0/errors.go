package mlog

import (
	"errors"
	"fmt"
)

var (
	Err = errors.New("mLog")

	ErrSyslog                    = fmt.Errorf("%w syslog", Err)
	ErrSyslogConnection          = fmt.Errorf("%w connection error", ErrSyslog)
	ErrSyslogURLparse            = fmt.Errorf("%w URL parse error", ErrSyslog)
	ErrSyslogHandle              = fmt.Errorf("%w handle error", ErrSyslog)
	ErrSyslogProcessHandleResult = fmt.Errorf("%w handle result process error", ErrSyslog)
	ErrSyslogWrite               = fmt.Errorf("%w write error", ErrSyslog)
)
