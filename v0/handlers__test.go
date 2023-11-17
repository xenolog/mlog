package main_test

import (
	"log/slog"
	"testing"

	assert "github.com/stretchr/testify/require"
	svLog "github.com/xenolog/slog/v0"
)

type TestWriter struct {
	Buf []byte
}

func NewTestWriter() *TestWriter {
	buf := make([]byte, 0, 8192)
	return &TestWriter{Buf: buf}
}

func (r *TestWriter) Write(p []byte) (int, error) {
	n := len(p)
	r.Buf = append(r.Buf, p...)
	return n, nil
}

func (r *TestWriter) String() string {
	return string(r.Buf)
}

//-----------------------------------------------------------------------------

func Test__Handler__Simple(t *testing.T) {
	tt := assert.New(t)
	msg := "Just InfoMessage"

	nativeWriter := NewTestWriter()
	nativeLogger := slog.New(slog.NewTextHandler(nativeWriter, &slog.HandlerOptions{AddSource: true}))
	nativeLogger.Info(msg)

	svWriter := NewTestWriter()
	svLogger := slog.New(svLog.NewHandler(svWriter, &svLog.HandlerOptions{AddSource: true}))
	svLogger.Info(msg)

	tt.EqualValues(nativeWriter.String(), svWriter.String())
}
