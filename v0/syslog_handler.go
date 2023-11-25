/*
SyslogHandler provides message delivery to syslog functionality
to any [slog.Handler] compotible handler, which able to output to an [io.Writer].
Interaction with Syslog server should be setup before [slog.Logger]/[slog.Handler] setup:

	syslogPx := mlog.NewSyslogProxy(...)
	logHandler := slog.NewJSONHandler(syslogPx.LocalWriter(), nil)
	syslogHandler := mlog.NewSyslogHandler(syslogPx, logHandler, &SyslogHandlerOptions{...})

	if err := syslogPx.Connect("udp://1.2.3.4:514"); err != nil {
		// Handle dial to syslog server error
	}
	logger := slog.New(syslogHandler)
	logger.Info("very important message")
*/
package mlog

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	logSyslog "log/syslog"
	"net"
	netURL "net/url"
	"os"
	"slices"
	"sync"
	"time"
)

const (
	severityMask = 0x07 // see log/syslog code
	facilityMask = 0xf8 // see log/syslog code

	defaultTimeout = 15 * time.Second
)

var allowedProto = []string{"tcp", "udp", "unix"} //nolint:gochecknoglobals

// -----------------------------------------------------------------------------
// SyslogProxy is a type that provides interaction with the syslog Proxyserver
type SyslogProxy struct {
	buf            *bytes.Buffer
	lineProcessBuf []byte
	priority       logSyslog.Priority
	tag            string
	hostname       string
	url            string
	conn           net.Conn // connection, disconnected if nil
	timeout        time.Duration
	stderrLogger   *slog.Logger
	mu             *sync.Mutex
}

// LocalWriter returns a [io.Writer] which may be used
// to fill buffer, shich will be send to Syslog server later
func (s *SyslogProxy) LocalWriter() io.Writer {
	return s.buf
}

// func (s *SyslogProxy) LocalBuffer() bufio.ReadWriter {
// 	return s.buf
// }

func (s *SyslogProxy) Lock() {
	s.mu.Lock()
}

func (s *SyslogProxy) Unlock() {
	s.mu.Unlock()
}

// Connect (or re-connect) to the given syslog server or socket
// url should be in one of following format:
//
//	tcp://1.2.3.4:514
//	udp://1.2.3.4:514
//	unix:///var/run/syslog
func (s *SyslogProxy) Connect(url string, timeout time.Duration) error {
	var addr, proto string

	if url == "" {
		// use previous url
		url = s.url
	}

	// parse URL
	u, err := netURL.Parse(url)
	if err != nil {
		return fmt.Errorf("%w: URL `%s` is wrong: %w", ErrSyslogURLparse, url, err)
	}
	if slices.Index(allowedProto, u.Scheme) == -1 {
		return fmt.Errorf("%w: URL `%s` is wrong: unsupported proto '%s', allowed only %v", ErrSyslogURLparse, url, u.Scheme, allowedProto)
	}
	proto = u.Scheme
	if proto == "unix" {
		addr = u.Path
	} else {
		addr = u.Host
	}
	s.url = fmt.Sprintf("%s://%s", proto, addr)

	if timeout == 0 {
		timeout = defaultTimeout
	}

	// dial to the Syslog server
	s.mu.Lock()
	defer s.mu.Unlock() // todo

	s.Disconnect()

	c, err := net.DialTimeout(proto, addr, timeout)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSyslogConnection, err)
	}
	s.conn = c
	s.timeout = timeout
	return nil
}

func (s *SyslogProxy) IsConnected() bool {
	return s.conn != nil
}

func (s *SyslogProxy) Disconnect() {
	if s.conn != nil {
		s.conn.Close() // revive:disable:unhandled-error
		s.conn = nil
	}
}

// // CleanLocalBuffer LocalBuffer safe
// func (s *SyslogProxy) CleanLocalBuffer() {
// 	s.mu.Lock()
// 	defer s.mu.Unlock()
// 	s.buf.Reset()
// }
// // Writer returns a [io.Writer] which may be used
// // to send something to Syslog server
// // returns nil if not connected
// func (s *SyslogProxy) Writer() io.Writer {
// 	return s.conn
// }

// ProcessLines process each line of  LocalBuffer by given function.
// be carefully, strongly recommended wrap this call by mutex Lock()/Unlock.
func (s *SyslogProxy) ProcessLines(f func([]byte) ([]byte, error)) (err error) {
	if !s.IsConnected() {
		return fmt.Errorf("%w: not connected", ErrSyslogConnection)
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(s.timeout)); err != nil {
		return fmt.Errorf("%w: %w", ErrSyslogConnection, err)
	}

	// for { // todo(sv): Should be rewriten for async usage !!!
	// 	line, err := s.bufReader.ReadBytes('\n') // or .ReadSlice ???
	// 	switch err {
	// 		case
	// 	}
	// }
	scanner := bufio.NewScanner(s.buf)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			line, err := f([]byte(line))
			if err != nil {
				return err // just return err from user provided function
			}
			_, err = s.conn.Write(line)
			if err != nil {
				return fmt.Errorf("%w: %w", ErrSyslogWrite, err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%w: %w", ErrSyslogProcessHandleResult, err)
	}
	return nil
}

// NewSyslog setup and return [Syslog] entity.
func NewSyslogProxy(priority logSyslog.Priority, tag string) *SyslogProxy {
	if priority < 0 || priority > logSyslog.LOG_LOCAL7|logSyslog.LOG_DEBUG {
		priority = logSyslog.LOG_INFO | logSyslog.LOG_USER
	}

	if tag == "" {
		tag = os.Args[0]
	}
	hostname, _ := os.Hostname()

	buf := make([]byte, 0, InitialBufSize)

	s := &SyslogProxy{
		buf:            bytes.NewBuffer(buf),
		lineProcessBuf: make([]byte, 256),
		priority:       priority,
		tag:            tag,
		hostname:       hostname,
		stderrLogger:   slog.New(NewHumanReadableHandler(os.Stderr, nil)),
		mu:             &sync.Mutex{},
	}
	return s
}

//-----------------------------------------------------------------------------

// SyslogHandler currently has no options,
// but this will change in the future and the type is reserved
// to maintain backward compatibility
type SyslogHandlerOptions struct{}

// SyslogHandler is a proxy Handler that ensures
// message delivery from any [slog.Handler] to the syslog server
// It should be used in conjunction with mlog.Syslog
type SyslogHandler struct {
	syslogPx *SyslogProxy
	handler  slog.Handler
	level    slog.Level
}

func (h *SyslogHandler) Copy() *SyslogHandler {
	rv := &SyslogHandler{
		syslogPx: h.syslogPx,
		handler:  h.handler,
	}
	return rv
}

// Enabled reports whether the handler handles records at the given level. The handler ignores records whose level is lower.
// Implements [slog.Handler] interface.
func (h *SyslogHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

// WithAttrs returns a new HumanReadableHandler whose attributes consists of h's attributes followed by attrs.
// Implements [slog.Handler] interface.
func (h *SyslogHandler) WithAttrs(aa []slog.Attr) slog.Handler {
	hh := h.Copy()
	hh.handler = h.handler.WithAttrs(aa)
	return hh
}

// WithGroup returns a new HumanReadableHandler with the given group appended to the receiver's existing groups.
// Implements [slog.Handler] interface.
func (h *SyslogHandler) WithGroup(name string) slog.Handler {
	var rv *SyslogHandler
	if name != "" {
		rv = h.Copy()
		rv.handler = h.handler.WithGroup(name)
	} else {
		rv = h
	}
	return rv
}

// Handle handles the Record.
// It will only be called when Enabled(...) returns true.
// Implements [slog.Handler] interface.
func (h *SyslogHandler) Handle(ctx context.Context, r slog.Record) error { //nolint:gocritic
	if !h.syslogPx.IsConnected() {
		return fmt.Errorf("%w: not connected", ErrSyslogConnection)
	}

	h.syslogPx.Lock() // should be locked before call chield's Handle() because  LocalWriter used by one
	defer h.syslogPx.Unlock()
	if err := h.handler.Handle(ctx, r); err != nil {
		return fmt.Errorf("%w: %w", ErrSyslogHandle, err)
	}

	err := h.syslogPx.ProcessLines(func(line []byte) ([]byte, error) {
		return TrimTimestamp(line), nil
	})

	return err
}

func NewSyslogHandler(syslogPx *SyslogProxy, h slog.Handler, _ *SyslogHandlerOptions) *SyslogHandler {
	sh := &SyslogHandler{
		syslogPx: syslogPx,
		handler:  h,
	}
	for _, logLevel := range allowedLevels {
		if h.Enabled(context.TODO(), logLevel) {
			sh.level = logLevel
			break
		}
	}
	return sh
}
