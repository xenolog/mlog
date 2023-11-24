//revive:disable:add-constant
package mlog_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	assert "github.com/stretchr/testify/require"
	mlog "github.com/xenolog/mlog/v0"
)

func Test__MtHandler__Simple(t *testing.T) {
	tt := assert.New(t)
	msg := "Just InfoMessage " + uuid.NewString()

	firstWriter := &bytes.Buffer{}
	firstHandler := slog.NewJSONHandler(firstWriter, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})
	secondWriter := &bytes.Buffer{}
	secondHandler := mlog.NewHumanReadableHandler(secondWriter, &mlog.HumanReadableHandlerOptions{AddSourceToAttrs: true})

	mtHandler := mlog.NewMultipleHandler(nil, firstHandler, secondHandler)
	logger := slog.New(mtHandler)

	logger.Info(msg)

	firstData := map[string]any{}
	tt.NoError(json.Unmarshal(firstWriter.Bytes(), &firstData))
	firstTimeStr := JqGetString(firstData, ".time")
	tt.NotZero(firstTimeStr)
	firstTime, err := time.Parse(time.RFC3339Nano, firstTimeStr)
	tt.NoError(err)

	secondLogLineSplitted := strings.Fields(secondWriter.String())
	tt.Greater(len(secondLogLineSplitted), 2)
	secondTime, err := time.Parse(time.RFC3339Nano, secondLogLineSplitted[0])
	tt.NoError(err)

	tt.True(firstTime.UTC().Equal(secondTime))

	tt.EqualValues("INFO", JqGetString(firstData, ".level"))
	tt.EqualValues("I", secondLogLineSplitted[1])

	attrsPosition := slices.IndexFunc(secondLogLineSplitted, func(f string) bool {
		return strings.HasPrefix(f, mlog.AttrsJSONprefix)
	})
	tt.GreaterOrEqual(attrsPosition, 0)
	tt.EqualValues(msg, strings.Join(secondLogLineSplitted[3:attrsPosition], " "))
	tt.EqualValues(msg, JqGetString(firstData, ".msg"))

	// tt.EqualValues(firstWriter.String(), secondWriter.String()) // enable if deep debug required
}

func Test__MtHandler__LogLevel(t *testing.T) {
	tt := assert.New(t)

	debugMsg1 := "slog first message"
	debugMsg2 := "slog second message"
	infoMsg := "std log message"

	firstWriter := &bytes.Buffer{}
	firstHandler := slog.NewJSONHandler(firstWriter, &slog.HandlerOptions{AddSource: true, Level: slog.LevelDebug})
	secondWriter := &bytes.Buffer{}
	secondHandler := mlog.NewHumanReadableHandler(secondWriter, &mlog.HumanReadableHandlerOptions{}) // defaultLogLevel == INFO

	mtHandler := mlog.NewMultipleHandler(nil, firstHandler, secondHandler)
	logger := slog.New(mtHandler)
	stdLogger := slog.NewLogLogger(mtHandler, slog.LevelInfo)

	logger.Debug(debugMsg1)
	stdLogger.Print(infoMsg) // std goland log messages will be stored with INFO level
	logger.Debug(debugMsg2)

	firstLog := bytes.Split(firstWriter.Bytes(), []byte("\n"))
	tt.Len(firstLog, 4) // 3 lines + last EOL
	secondLog := bytes.Split(secondWriter.Bytes(), []byte("\n"))
	tt.Len(secondLog, 2) // 1 line + last EOL

	tt.GreaterOrEqual(0, slices.IndexFunc(firstLog, func(s []byte) bool {
		return string(s) == debugMsg1
	}))
	tt.GreaterOrEqual(0, slices.IndexFunc(firstLog, func(s []byte) bool {
		return string(s) == debugMsg2
	}))
	tt.GreaterOrEqual(0, slices.IndexFunc(firstLog, func(s []byte) bool {
		return string(s) == infoMsg
	}))

	tt.EqualValues(-1, slices.IndexFunc(secondLog, func(s []byte) bool {
		return string(s) == debugMsg1
	}))
	tt.EqualValues(-1, slices.IndexFunc(secondLog, func(s []byte) bool {
		return string(s) == debugMsg2
	}))
	tt.GreaterOrEqual(0, slices.IndexFunc(secondLog, func(s []byte) bool {
		return string(s) == infoMsg
	}))
}

// todo: Test__MtHandler__With()
// todo: Test__MtHandler__WithGroup()
