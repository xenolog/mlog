//revive:disable:add-constant
package mlog_test

import (
	"testing"

	"github.com/google/uuid"
	assert "github.com/stretchr/testify/require"
)

func Test__SyHandler__Simple(t *testing.T) {
	tmpDir := t.TempDir()
	sockFile := tmpDir + fakeSyslogSocket

	tt := assert.New(t)
	msg := "Just InfoMessage " + uuid.NewString()

	ss := NewFakeSyslog(sockFile)
	defer ss.Destroy()

	// small TCP or unix socket server should be inplemented to store
	// all incoming lines into bytes.Buffer
}
