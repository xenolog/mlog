//revive:disable:add-constant
package mlog_test

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	assert "github.com/stretchr/testify/require"
	"github.com/xenolog/mlog/v0"
)

func Test__TrimTimestamp(t *testing.T) {
	testData := []string{
		"2006-01-02T15:04:05.999999999+07:00  xxx yyy zzz", // RFC3339
		"2006-01-02T15:04:05.999999999-07:00  xxx yyy zzz",
		"2006-01-02T15:04:05.999999999Z  xxx yyy zzz",
		"2006-01-02T15:04:05.999Z  xxx yyy zzz",
		"2006-01-02T15:04:05Z  xxx yyy zzz",
		"2006-01-02 15:04:05  xxx yyy zzz",      // DateTime
		"Jan 2 15:04:05.020000002  xxx yyy zzz", // Stamp
		"Jan  2 15:04:05.020000002  xxx yyy zzz",
		"Jan 12 15:04:05.020000002  xxx yyy zzz",
		"Jan 2 15:04:05.00003  xxx yyy zzz",
		"Jan 2 15:04:05  xxx yyy zzz",
		"Tue, 10 Nov 2009 18:00:00 EST  xxx yyy zzz",   // RFC1123
		"Tue, 10 Nov 2009 18:00:00 +0700  xxx yyy zzz", // RFC1123Z
		"Tue, 10 Nov 2009 18:00:00 -0500  xxx yyy zzz",
	}

	for i, line := range testData {
		t.Run(fmt.Sprintf("%02d-%s", i, line), func(t *testing.T) {
			res := mlog.TrimTimestamp([]byte(line))
			assert.EqualValues(t, "xxx yyy zzz", string(res))
		})
	}
}

func Test__SyslogProxy__Simple(t *testing.T) {
	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline.Add(-1*time.Second))
	defer cancel()

	tmpDir := t.TempDir()
	sockFile := tmpDir + fakeSyslogSocket

	tt := assert.New(t)
	msg1 := "First InfoMessage  " + uuid.NewString()
	msg2 := "Second InfoMessage  " + uuid.NewString()

	ss := NewFakeSyslog(sockFile)
	ready, err := ss.Run(ctx)
	tt.NoError(err)
	<-ready

	syslogPx := mlog.NewSyslogProxy(10, "xxx")
	tt.NoError(syslogPx.Connect("unix://"+sockFile, 0))

	buf := syslogPx.LocalWriter()

	_, err = buf.Write([]byte(msg1 + "\n")) // up-level handler send message to io.Writer, provided by setup
	tt.NoError(err)
	_, err = buf.Write([]byte(msg2)) // up-level handler send message to io.Writer, provided by setup
	tt.NoError(err)
	err = syslogPx.ProcessLines(func(b []byte) ([]byte, error) { //
		return b, nil
	})
	tt.NoError(err)

	syslogPx.Disconnect()
	time.Sleep(1 * time.Second) // may be required to fakeSyslogServer save all incoming messages

	firstMessage, err := ss.Buffer().ReadString('\n')
	tt.NoError(err)
	tt.EqualValues(msg1+"\n", firstMessage)

	secondMessage, err := ss.Buffer().ReadString('\n')
	tt.NoError(err)
	tt.EqualValues(msg2+"\n", secondMessage)
}

func Test__SyHandler__Simple(t *testing.T) {
	deadline, ok := t.Deadline()
	if !ok {
		deadline = time.Now().Add(5 * time.Second)
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline.Add(-1*time.Second))
	defer cancel()

	tmpDir := t.TempDir()
	sockFile := tmpDir + fakeSyslogSocket

	tt := assert.New(t)
	msg := "Just InfoMessage " + uuid.NewString()

	ss := NewFakeSyslog(sockFile)
	ready, err := ss.Run(ctx)
	tt.NoError(err)
	<-ready

	// setup mlog.SyslogHandler

	syslogPx := mlog.NewSyslogProxy(10, "xxx")
	tt.NoError(syslogPx.Connect("unix://"+sockFile, 0))
	defer syslogPx.Disconnect()

	logHandler := slog.NewTextHandler(syslogPx.LocalWriter(), nil)
	syslogHandler := mlog.NewSyslogHandler(syslogPx, logHandler, nil)
	logger := slog.New(syslogHandler)

	logger.Info(msg)
	time.Sleep(1 * time.Second) // may be required to fakeSyslogServer save all incoming messages

	firstMessage, err := ss.Buffer().ReadString('\n')
	tt.NoError(err)
	tt.Regexp(`msg="`+msg+`"`, firstMessage)
}
