package main_test

import (
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	assert "github.com/stretchr/testify/require"
	svLog "github.com/xenolog/slog/v0"
)

// -----------------------------------------------------------------------------
func Test__Handler__Simple(t *testing.T) {
	tt := assert.New(t)
	msg := "Just InfoMessage"

	nativeWriter := NewTestWriter()
	nativeLogger := slog.New(slog.NewJSONHandler(nativeWriter, &slog.HandlerOptions{AddSource: true}))
	nativeLogger.Info(msg)

	nativeData := map[string]any{}
	err := json.Unmarshal(nativeWriter.Buf, &nativeData)
	tt.NoError(err)
	nativeTimeStr := JqGetString(nativeData, ".time")
	tt.NotEqualValues("", nativeTimeStr)
	nativeTime, err := time.Parse(time.RFC3339Nano, nativeTimeStr)
	tt.NoError(err)

	svWriter := NewTestWriter()
	svLogger := slog.New(svLog.NewHandler(svWriter, &svLog.HandlerOptions{AddSource: true}))
	svLogger.Info(msg)
	svLogLineSplitted := strings.Split(string(svWriter.Buf), " ")
	tt.Greater(len(svLogLineSplitted), 2)
	svTime, err := time.Parse(time.RFC3339Nano, svLogLineSplitted[0])
	tt.NoError(err)

	timeDelta := svTime.Sub(nativeTime)
	tt.Zero(timeDelta.Truncate(time.Second)) // I suppose delta between 2 log lines less than 1 second

	tt.EqualValues("INFO", JqGetString(nativeData, ".level"))
	tt.EqualValues("I", svLogLineSplitted[1])

	sourceLineSplited := strings.Split(strings.Trim(svLogLineSplitted[2], "[]"), ":")
	tt.EqualValues("handlers__test.go", sourceLineSplited[0])

	// tt.EqualValues(nativeWriter.String(), svWriter.String())
}
