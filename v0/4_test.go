//revive:disable:add-constant
package mlog_test

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/itchyny/gojq"
)

const fakeSyslogSocket = "/fssock"

// -----------------------------------------------------------------------------
type FakeSyslog struct {
	buf       *bytes.Buffer
	sockPath  string
	keepAlive int
	connectNo int
	listener  net.Listener
}

func (s *FakeSyslog) Buffer() *bytes.Buffer {
	return s.buf
}

func (s *FakeSyslog) Run(ctx context.Context) (c chan struct{}, err error) {
	// create sock
	lc := &net.ListenConfig{
		KeepAlive: time.Duration(s.keepAlive) * time.Second,
	}
	s.listener, err = lc.Listen(ctx, "unix", s.sockPath)
	if err != nil {
		return nil, fmt.Errorf("Listen error: %w", err)
	}

	go func() {
		<-ctx.Done()
		s.Finish()
	}()

	log.Println("server started")
	go func() {
		for {
			// Wait for a connection.
			conn, err := s.listener.Accept()
			switch {
			case errors.Is(err, net.ErrClosed):
				return
			case err != nil:
				log.Printf("Accept error: %s", err)
				return
			default:
				go s.handleConnection(s.connectNo, conn)
				s.connectNo++
			}
		}
	}()
	c = make(chan struct{}, 1)
	c <- struct{}{}
	return c, nil
}

func (s *FakeSyslog) store(n int, str string) {
	_, err := s.buf.WriteString(str)
	if err != nil {
		log.Printf("ErrStore: %s", err)
	}
	log.Printf("%03d: stored: %s", n, str)
}

func (s *FakeSyslog) handleConnection(cNo int, c net.Conn) {
	log.Printf("%03d: handle connection on '%s' from '%s'", cNo, c.LocalAddr(), c.RemoteAddr())
	reader := bufio.NewReader(c)
exLoop:
	for {
		str, err := reader.ReadString('\n')
		switch {
		case errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF):
			s.store(cNo, str+"\n")
			// log.Printf("%03d: %s<EOF>", cNo, str)
			break exLoop
		case err != nil:
			log.Printf("%03d error while reading: %s", cNo, err)
			break exLoop
		default:
			s.store(cNo, str)
			// log.Printf("%03d: %s", cNo, strings.TrimRight(str, "\n"))
		}
	}
	// Shut down the connection.
	err := c.Close()
	if err != nil {
		log.Printf("Close connection error: %s", err)
	}
	log.Printf("%03d: connection closed.", cNo)
}

func (s *FakeSyslog) Finish() {
	log.Println("finishing server...")
	if err := s.listener.Close(); err != nil {
		log.Printf("Listener close error: %s", err)
	}
	if err := os.Remove(s.sockPath); err != nil && !os.IsNotExist(err) {
		log.Printf("unable to remove `%s`: %s", s.sockPath, err)
	}
}

func NewFakeSyslog(path string) *FakeSyslog {
	ss := &FakeSyslog{
		sockPath: path,
		buf:      &bytes.Buffer{},
	}
	return ss
}

// -----------------------------------------------------------------------------

func Jq(t *testing.T, data any, query string) bool {
	v, err := JqGet(data, query)
	if err != nil {
		t.Fatal(err)
	}

	res, ok := v.(bool)
	if !ok {
		t.Fatal(fmt.Errorf("JQ query result is not boolean"))
	}
	return res
}

func JqGet(data any, query string) (any, error) {
	jq, err := gojq.Parse(query)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}
	iter := jq.Run(data)
	v, ok := iter.Next()
	if !ok {
		return nil, fmt.Errorf("JQ query path not found")
	}
	if err, ok := v.(error); ok {
		return nil, err
	}
	return v, nil
}

func JqGetString(data any, query string) string { // note, no error returned to simplify test writing
	v, err := JqGet(data, query)
	if err != nil {
		return ""
	}
	str, ok := v.(string)
	if !ok {
		return ""
	}
	return str
}
