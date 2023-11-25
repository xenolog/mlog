//revive:disable:add-constant
package mlog_test

import (
	"fmt"
	"testing"

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

func Test__SyHandler__Simple(t *testing.T) {
	tmpDir := t.TempDir()
	sockFile := tmpDir + fakeSyslogSocket

	tt := assert.New(t)
	msg := "Just InfoMessage " + uuid.NewString()

	ss := NewFakeSyslog(sockFile)
	defer ss.Destroy()

	// small TCP or unix socket server should be inplemented to store
	// all incoming lines into bytes.Buffer

	tt.NotZero(msg)
}
