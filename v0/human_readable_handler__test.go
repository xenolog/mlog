//nolint:goconst
package mlog_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	assert "github.com/stretchr/testify/require"
	"github.com/xenolog/mlog/v0"
)

const (
	attr0key = "zzz"
	attr1key = "aaa"
	attr2key = "bbb"
	attr3key = "ccc"
)

// -----------------------------------------------------------------------------
func Test__HrHandler__Simple(t *testing.T) {
	tt := assert.New(t)
	msg := "Just InfoMessage " + uuid.NewString()

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler))

	logger.Info(msg)

	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))
	nativeTimeStr := JqGetString(nativeData, ".time")
	tt.NotEqualValues("", nativeTimeStr)
	nativeTime, err := time.Parse(time.RFC3339Nano, nativeTimeStr)
	tt.NoError(err)

	svLogLineSplitted := strings.Fields(svWriter.String())
	tt.Greater(len(svLogLineSplitted), 2)
	svTime, err := time.Parse(time.RFC3339Nano, svLogLineSplitted[0])
	tt.NoError(err)

	timeDelta := svTime.Sub(nativeTime)
	tt.Zero(timeDelta.Truncate(time.Second)) // I suppose delta between 2 log lines less than 1 second

	tt.EqualValues("INFO", JqGetString(nativeData, ".level"))
	tt.EqualValues("I", svLogLineSplitted[1])

	tt.EqualValues(msg, strings.Join(svLogLineSplitted[3:], " "))

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__HrHandler__AddSource(t *testing.T) {
	tt := assert.New(t)
	msg := "Just InfoMessage " + uuid.NewString()

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, &slog.HandlerOptions{AddSource: true})
	svHandler := mlog.NewHumanReadableHandler(svWriter, &mlog.HumanReadableHandlerOptions{AddSource: true, AddSourceToAttrs: true})
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler))

	logger.Info(msg)

	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	srcFilePath := JqGetString(nativeData, ".source.file")
	srcFunction := JqGetString(nativeData, ".source.function")
	tt.NotZero(srcFilePath)
	tt.NotZero(srcFunction)

	svLogLineSplitted := strings.Fields(svWriter.String())
	tt.Greater(len(svLogLineSplitted), 2)
	sourceLineSplited := strings.Split(strings.Trim(svLogLineSplitted[2], "[]"), ":")
	tt.True(strings.HasSuffix(srcFilePath, sourceLineSplited[0]))

	tt.EqualValues(srcFilePath, JqGetString(svData, ".source.file"))
	tt.EqualValues(srcFunction, JqGetString(svData, ".source.function"))

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__HrHandler__InterceptSimpleLogger(t *testing.T) {
	tt := assert.New(t)
	msg := "Just InfoMessage " + uuid.NewString()

	svWriter := &bytes.Buffer{}
	hnldr := mlog.NewHumanReadableHandler(svWriter, nil)
	slog.New(hnldr)
	stdLog := slog.NewLogLogger(hnldr, slog.LevelInfo)
	now := time.Now()
	stdLog.Print(msg)

	svLogLineSplitted := strings.Fields(svWriter.String())
	tt.Greater(len(svLogLineSplitted), 2)
	svTime, err := time.Parse(time.RFC3339Nano, svLogLineSplitted[0])
	tt.NoError(err)

	timeDelta := svTime.Sub(now)
	tt.Zero(timeDelta.Truncate(time.Second)) // I suppose delta between 2 log lines less than 1 second

	tt.EqualValues("I", svLogLineSplitted[1])

	tt.EqualValues(msg, strings.Join(svLogLineSplitted[3:], " "))

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__HrHandler__Values(t *testing.T) {
	tt := assert.New(t)

	msg := "see values of " + uuid.NewString()

	valKeyInt := "intVal"
	valDataInt := 42 //revive:disable:add-constant

	valKeyBool := "boolVal"
	valDataBool := true

	valKeyString := "stringVal"
	valDataString := "a string"

	valKeyTime := "timeVal"
	valDataTime := time.Now()

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler))

	logger.Info(msg,
		valKeyInt, valDataInt,
		valKeyBool, valDataBool,
		valKeyString, valDataString,
		valKeyTime, valDataTime,
	)
	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	// tt.Zero(string(jsonBuf)) // enable if deep debug required
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	nativeBoolVal, err := JqGet(nativeData, "."+valKeyBool)
	tt.NoError(err)
	svBoolVal, err := JqGet(svData, "."+valKeyBool)
	tt.NoError(err)
	tt.EqualValues(valDataBool, svBoolVal)
	tt.EqualValues(nativeBoolVal, svBoolVal)

	nativeIntVal, err := JqGet(nativeData, "."+valKeyInt)
	tt.NoError(err)
	svIntVal, err := JqGet(svData, "."+valKeyInt)
	tt.NoError(err)
	tt.EqualValues(valDataInt, svIntVal)
	tt.EqualValues(nativeIntVal, svIntVal)

	nativeStringVal, err := JqGet(nativeData, "."+valKeyString)
	tt.NoError(err)
	svStringVal, err := JqGet(svData, "."+valKeyString)
	tt.NoError(err)
	tt.EqualValues(valDataString, svStringVal)
	tt.EqualValues(nativeStringVal, svStringVal)

	nativeTimeVal, err := JqGet(nativeData, "."+valKeyTime)
	tt.NoError(err)
	svTimeVal, err := JqGet(svData, "."+valKeyTime)
	tt.NoError(err)
	tt.EqualValues(valDataTime.Format(time.RFC3339Nano), svTimeVal)
	tt.EqualValues(nativeTimeVal, svTimeVal)

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__HrHandler__WithAttrs(t *testing.T) {
	tt := assert.New(t)

	msg := "Info Message" + uuid.NewString()

	attr1data := uuid.NewString()
	attr2data := true
	attr3data := uuid.NewString()

	attr1 := slog.Attr{Key: attr1key, Value: slog.StringValue(attr1data)}
	attr2 := slog.Attr{Key: attr2key, Value: slog.BoolValue(attr2data)}

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler).WithAttrs([]slog.Attr{attr1, attr2}))

	logger.Info(msg, attr3key, attr3data)
	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	// tt.Zero(string(jsonBuf)) // enable if deep debug required
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	nativeAttr1val, err := JqGet(nativeData, "."+attr1key)
	tt.NoError(err)
	svAttr1val, err := JqGet(svData, "."+attr1key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr1val, svAttr1val)

	nativeAttr2val, err := JqGet(nativeData, "."+attr2key)
	tt.NoError(err)
	svAttr2val, err := JqGet(svData, "."+attr2key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr2val, svAttr2val)

	nativeAttr3val, err := JqGet(nativeData, "."+attr3key)
	tt.NoError(err)
	svAttr3val, err := JqGet(svData, "."+attr3key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr3val, svAttr3val)

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__Logger__With(t *testing.T) {
	tt := assert.New(t)

	msg := "Info Message " + uuid.NewString()

	attr1data := uuid.NewString()
	attr2data := true
	attr3data := uuid.NewString()

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler)).With(attr1key, attr1data, attr2key, attr2data)

	logger.Info(msg, attr3key, attr3data)

	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	// tt.Zero(string(jsonBuf)) // enable if deep debug required
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	nativeAttr1val, err := JqGet(nativeData, "."+attr1key)
	tt.NoError(err)
	svAttr1val, err := JqGet(svData, "."+attr1key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr1val, svAttr1val)

	nativeAttr2val, err := JqGet(nativeData, "."+attr2key)
	tt.NoError(err)
	svAttr2val, err := JqGet(svData, "."+attr2key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr2val, svAttr2val)

	nativeAttr3val, err := JqGet(nativeData, "."+attr3key)
	tt.NoError(err)
	svAttr3val, err := JqGet(svData, "."+attr3key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr3val, svAttr3val)

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__Logger__With__DuplicateAttrs(t *testing.T) {
	tt := assert.New(t)

	msg := "Info Message " + uuid.NewString()

	attr1data := uuid.NewString()
	attr2data := uuid.NewString()
	attr3data := uuid.NewString()

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler))
	logger = logger.With(attr1key, attr1data, attr2key, attr3data) // fill Attr2 with attr3 data
	logger.Info(msg, attr2key, attr2data)                          // rewrite Attr2

	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	// tt.Zero(string(jsonBuf)) // enable if deep debug required
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	nativeAttr1val, err := JqGet(nativeData, "."+attr1key)
	tt.NoError(err)
	svAttr1val, err := JqGet(svData, "."+attr1key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr1val, svAttr1val)

	nativeAttr2val, err := JqGet(nativeData, "."+attr2key)
	tt.NoError(err)
	svAttr2val, err := JqGet(svData, "."+attr2key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr2val, svAttr2val)

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__Logger__With__DuplicateAttrsInTheDifferentGroup(t *testing.T) {
	tt := assert.New(t)

	msg := "Info Message " + uuid.NewString()

	attr1data := uuid.NewString()
	attr2data := uuid.NewString()
	attr3data := uuid.NewString()

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler))
	logger = logger.With(attr2key, attr3data).With(attr1key, attr1data) // fill Attr2 with attr3 data
	logger.Info(msg, attr2key, attr2data)                               // rewrite Attr2

	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	// tt.Zero(string(jsonBuf)) // enable if deep debug required
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	nativeAttr1val, err := JqGet(nativeData, "."+attr1key)
	tt.NoError(err)
	svAttr1val, err := JqGet(svData, "."+attr1key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr1val, svAttr1val)

	nativeAttr2val, err := JqGet(nativeData, "."+attr2key)
	tt.NoError(err)
	svAttr2val, err := JqGet(svData, "."+attr2key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr2val, svAttr2val)

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__Logger__WithGroup(t *testing.T) {
	tt := assert.New(t)

	msg := "Info Message " + uuid.NewString()

	group1name := "firstGroup"
	group2name := "secondGroup"

	attr0data := uuid.NewString() + "-0"
	attr1data := uuid.NewString() + "-1"
	attr2data := uuid.NewString() + "-2"

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler))
	logger = logger.With(attr0key, attr0data).WithGroup(group1name).With(attr1key, attr1data).WithGroup(group2name)

	logger.Info(msg, attr2key, attr2data)

	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	// tt.Zero(string(jsonBuf)) // enable if deep debug required
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	nativeAttr0val, err := JqGet(nativeData, "."+attr0key)
	tt.NoError(err)
	svAttr0val, err := JqGet(svData, "."+attr0key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr0val, svAttr0val)

	nativeAttr1val, err := JqGet(nativeData, "."+group1name+"."+attr1key)
	tt.NoError(err)
	svAttr1val, err := JqGet(svData, "."+group1name+"."+attr1key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr1val, svAttr1val)

	nativeAttr2val, err := JqGet(nativeData, "."+group1name+"."+group2name+"."+attr2key)
	tt.NoError(err)
	svAttr2val, err := JqGet(svData, "."+group1name+"."+group2name+"."+attr2key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr2val, svAttr2val)

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__Logger__WithGroup__GroupOverwriteExistingAttribute(t *testing.T) {
	tt := assert.New(t)

	msg := "Info Message " + uuid.NewString()

	group1name := "firstGroup"

	attr0data := uuid.NewString() + "-0"
	attr1data := uuid.NewString() + "-1"

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler)).With(attr0key, attr0data).WithGroup(attr0key)

	logger.Info(msg, attr2key, attr1data)

	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	// tt.Zero(string(jsonBuf)) // enable if deep debug required
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	nativeAttr0val, err := JqGet(nativeData, "."+attr0key)
	tt.NoError(err)
	svAttr0val, err := JqGet(svData, "."+attr0key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr0val, svAttr0val)

	nativeAttr1val, err := JqGet(nativeData, "."+group1name+"."+attr1key)
	tt.NoError(err)
	svAttr1val, err := JqGet(svData, "."+group1name+"."+attr1key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr1val, svAttr1val)

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}

func Test__Logger__WithGroup__AttributeWithSameNameWithGroup(t *testing.T) {
	tt := assert.New(t)

	msg := "Info Message " + uuid.NewString()

	group1name := "firstGroup"

	attr0data := uuid.NewString() + "-0"
	attr1data := uuid.NewString() + "-1"

	nativeWriter := &bytes.Buffer{}
	svWriter := &bytes.Buffer{}
	nativeHandler := slog.NewJSONHandler(nativeWriter, nil)
	svHandler := mlog.NewHumanReadableHandler(svWriter, nil)
	logger := slog.New(mlog.NewMultipleHandler(nil, nativeHandler, svHandler)).WithGroup(attr0key).With(attr0key, attr0data)

	logger.Info(msg, attr2key, attr1data)

	nativeData := map[string]any{}
	tt.NoError(json.Unmarshal(nativeWriter.Bytes(), &nativeData))

	pos := bytes.Index(svWriter.Bytes(), []byte(mlog.AttrsJSONprefix))
	jsonBuf := svWriter.Bytes()[pos+len(mlog.AttrsJSONprefix):]
	// tt.Zero(string(jsonBuf)) // enable if deep debug required
	svData := map[string]any{}
	tt.NoError(json.Unmarshal(jsonBuf, &svData))

	nativeAttr0val, err := JqGet(nativeData, "."+attr0key+"."+attr0key)
	tt.NoError(err)
	svAttr0val, err := JqGet(svData, "."+attr0key+"."+attr0key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr0val, svAttr0val)

	nativeAttr1val, err := JqGet(nativeData, "."+group1name+"."+attr1key)
	tt.NoError(err)
	svAttr1val, err := JqGet(svData, "."+group1name+"."+attr1key)
	tt.NoError(err)
	tt.EqualValues(nativeAttr1val, svAttr1val)

	// tt.EqualValues(nativeWriter.String(), svWriter.String()) // enable if deep debug required
}
