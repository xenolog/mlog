/*
SyslogHandler provides message delivery to syslog functionality
to any [slog.Handler] compotible handler, which able to output to an [io.Writer].
Interaction with Syslog server should be setup before [slog.Logger]/[slog.Handler] setup:

	syslogPx := mlog.NewSyslogProxy(...)
	logHandler := slog.NewJSONHandler(syslogPx.Writer(), nil)
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
	"log"
	"log/slog"
	"net"
	netURL "net/url"
	"os"
	"slices"
	"sync"
	"time"
)

var allowedProto = []string{"tcp", "udp", "unix"} //nolint:gochecknoglobals

// -----------------------------------------------------------------------------
// SyslogProxy is a type that provides interaction with the syslog Proxyserver
type SyslogProxy struct {
	buf                    *bytes.Buffer
	lineProcessBuf         *bytes.Buffer
	facility               Facility
	tag                    string
	hostname               string
	url                    string
	conn                   net.Conn // connection, disconnected if nil
	useLocalTZ             bool
	timeFormat             string
	syslogLineHeaderFormat string
	timeout                time.Duration
	stderrLogger           *slog.Logger
	mu                     *sync.Mutex
}

// Writer returns a [io.Writer] which may be used
// to fill buffer, shich will be send to Syslog server later
func (s *SyslogProxy) Writer() io.Writer {
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
//
// if timeout is 0 the default timeout will be used
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
		timeout = defaultDialTimeout
	}

	// dial to the Syslog server
	c, err := net.DialTimeout(proto, addr, timeout)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSyslogConnection, err)
	}

	// Set successfully connected Syslog server as destination for messages
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Disconnect()
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

// ProcessLines process each line of LocalBuffer by given function.
// be carefully, strongly recommended wrap this call by mutex Lock()/Unlock.
func (s *SyslogProxy) ProcessLines(ts time.Time, level slog.Level, processFunc func([]byte) ([]byte, error)) (err error) {
	if !s.IsConnected() {
		return fmt.Errorf("%w: not connected", ErrSyslogConnection)
	}
	if err := s.conn.SetWriteDeadline(time.Now().Add(s.timeout)); err != nil {
		return fmt.Errorf("%w: %w", ErrSyslogConnection, err)
	}

	if s.useLocalTZ {
		ts = ts.Local() //nolint:gosmopolitan
	} else {
		ts = ts.UTC()
	}

	pid := os.Getpid()

	// todo(sv): Should be rewriten for async usage !!!
	// all processing should have ability to execute in separated goroutine, i.e. threadsafe
	s.mu.Lock()
	defer s.mu.Unlock()
	defer s.lineProcessBuf.Reset() // cleanup sensitive data
	scanner := bufio.NewScanner(s.buf)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			line, err := processFunc([]byte(line))
			if err != nil {
				return err // just return err from user provided function
			}
			// construct line coresponds to SysLog proto
			s.lineProcessBuf.Reset()
			prio := Severity(s.facility) | (level2severity[level.Level()] & severityMask) // combine Facility and slog.Level to one-byte syslog Priority
			timestamp := ts.Format(s.timeFormat)
			fmt.Fprintf(s.lineProcessBuf, s.syslogLineHeaderFormat, prio, timestamp, s.hostname, s.tag, pid)
			s.lineProcessBuf.Write(line)
			if line[len(line)-1] != '\n' { // add EOL if not present after processing by user function
				s.lineProcessBuf.WriteString("\n") // each line should leads by \n it is a Syslog protocol requirements
			}

			// write result to syslog output writer
			_, err = s.conn.Write(s.lineProcessBuf.Bytes())
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

// SyslogProxyOptions is options set to customize SyslogProxy while creation.
// Structure fields may be corrected while NewSyslogProxy(...) call to reflects
// actual values.
type SyslogProxyOptions struct {
	// Use Local timezone instead UTC
	UseLocalTZ bool
	// Date/Time format, RFC3339 used by default. Use time.Stamp or another from https://pkg.go.dev/time#pkg-constants
	TimeFormat string
	// Syslog Facility, LOG_USER will be used if not set
	Facility Facility
	// Hostname, os.Hostname() call will beused if not set
	Hostname string
	// strip hostname from Syslog stream
	StripHostname bool
	// Syslog tag, os.Args[0] will be used if not set
	Tag string
	// a syslog line header format, which will be passed to an fmt.Fprintf(...)
	// the "<%d>%s%s %s[%d]: " will be used, if not set, with `prio,timestamp,hostname,tag,pid` incoming values
	SyslogLineHeaderFormat string
	// buffer sizes, for experts only
	IOBufSize          uint
	LineProcessBufSize uint
}

// NewSyslog setup and return [Syslog] entity.
func NewSyslogProxy(opts *SyslogProxyOptions) (*SyslogProxy, error) {
	var err error

	if opts == nil {
		opts = &SyslogProxyOptions{}
	}

	if opts.IOBufSize == 0 {
		opts.IOBufSize = InitialIOBufSize
	}
	if opts.LineProcessBufSize == 0 {
		opts.LineProcessBufSize = InitialLineProcessBufSize
	}
	if opts.Facility == 0 {
		opts.Facility = defaultSyslogFacility
	} else {
		opts.Facility |= facilityMask
	}
	if opts.Tag == "" {
		opts.Tag = os.Args[0]
	}
	if opts.TimeFormat == "" {
		opts.TimeFormat = TimeOutputFormatRFC3339
	}
	if opts.StripHostname {
		opts.Hostname = ""
	} else {
		if opts.Hostname == "" {
			if opts.Hostname, err = os.Hostname(); err != nil {
				opts.Hostname = " os.hostname-error"
				log.Printf("Unable to detect hostname: %s\n", err)
			}
		}
		opts.Hostname = " " + opts.Hostname
	}
	if opts.SyslogLineHeaderFormat == "" {
		opts.SyslogLineHeaderFormat = DefaultSyslogLineHeaderFormat
	}

	s := &SyslogProxy{
		buf:                    bytes.NewBuffer(make([]byte, 0, opts.IOBufSize)),
		lineProcessBuf:         bytes.NewBuffer(make([]byte, 0, opts.LineProcessBufSize)),
		facility:               opts.Facility,
		hostname:               opts.Hostname,
		tag:                    opts.Tag,
		useLocalTZ:             opts.UseLocalTZ,
		timeFormat:             opts.TimeFormat,
		stderrLogger:           slog.New(NewHumanReadableHandler(os.Stderr, nil)),
		syslogLineHeaderFormat: opts.SyslogLineHeaderFormat,
		mu:                     &sync.Mutex{},
	}

	return s, nil
}

//-----------------------------------------------------------------------------

// SyslogHandler currently has no options,
// but this will change in the future and the type is reserved
// to maintain backward compatibility
type SyslogHandlerOptions struct {
	LineProcessFunc func(line []byte) ([]byte, error)
}

// SyslogHandler is a proxy Handler that ensures
// message delivery from any [slog.Handler] to the syslog server
// It should be used in conjunction with mlog.Syslog
type SyslogHandler struct {
	syslogPx        *SyslogProxy
	handler         slog.Handler
	level           slog.Level // should not be set manually. collected from uplevel slog handler
	lineProcessFunc func(line []byte) ([]byte, error)
}

func (h *SyslogHandler) Copy() *SyslogHandler {
	rv := &SyslogHandler{
		syslogPx:        h.syslogPx,
		handler:         h.handler,
		level:           h.level,
		lineProcessFunc: h.lineProcessFunc,
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

	if err := h.handler.Handle(ctx, r); err != nil {
		return fmt.Errorf("%w: %w", ErrSyslogHandle, err)
	}

	err := h.syslogPx.ProcessLines(r.Time, r.Level, h.lineProcessFunc)

	return err
}

func NewSyslogHandler(syslogPx *SyslogProxy, h slog.Handler, opts *SyslogHandlerOptions) *SyslogHandler {
	if opts == nil {
		opts = &SyslogHandlerOptions{}
	}
	if opts.LineProcessFunc == nil {
		opts.LineProcessFunc = func(line []byte) ([]byte, error) {
			rv, _ := TrimTimestamp(line)
			return rv, nil // `unable to trim timestamp` is not a global error
		}
	}
	sh := &SyslogHandler{
		syslogPx:        syslogPx,
		handler:         h,
		lineProcessFunc: opts.LineProcessFunc,
	}
	for _, logLevel := range allowedLevels {
		if h.Enabled(context.TODO(), logLevel) {
			sh.level = logLevel
			break
		}
	}
	return sh
}
