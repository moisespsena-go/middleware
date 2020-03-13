package middleware

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"time"

	"github.com/maruel/panicparse/stack"

	"github.com/go-chi/chi/middleware"
)

var (
	// LogEntryCtxKey is the context.Context key to store the request log entry.
	LogEntryCtxKey = &contextKey{"LogEntry"}
	// PanicEntryCtxKey is the context.Context key to store the request panic entry.
	PanicEntryCtxKey = &contextKey{"PanicEntry"}

	// DefaultLoggerExtensionsIgnore is the default request extensions to ignores
	DefaultLoggerExtensionsIgnore = StringsToExtensions("css", "js", "jpg", "png", "gif", "ico", "ttf", "woff2")

	DefaultRequestLogFormatter = &DefaultLogAndPanicFormatter{
		Logger:      log.New(os.Stdout, "", log.LstdFlags),
		PanicLogger: log.New(os.Stderr, "", log.LstdFlags),
	}

	// DefaultLogger is called by the Logger middleware handler to log each request.
	// Its made a package-level variable so that it can be reconfigured for custom
	// logging configurations.
	DefaultLogger = RequestLogger(DefaultRequestLogFormatter)
)

// Logger is a middleware that logs the start and end of each request, along
// with some useful data about what was requested, what the response status was,
// and how long it took to return. When standard output is a TTY, Logger will
// print in color, otherwise it will print in black and white. Logger prints a
// request BID if one is provided.
//
// Alternatively, look at https://github.com/pressly/lg and the `lg.RequestLogger`
// middleware pkg.
func Logger(next http.Handler) http.Handler {
	return DefaultLogger(next)
}

// RequestLogger returns a logger handler using a custom LogFormatter.
func RequestLogger(f LogFormatter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if f.Accept(r) {
				entry := f.NewLogEntry(r)
				ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

				t1 := time.Now()
				defer func() {
					entry.Write(ww.Status(), ww.BytesWritten(), time.Since(t1))
				}()
				next.ServeHTTP(ww, WithLogEntry(r, entry))
			} else {
				next.ServeHTTP(w, r)
			}
		}
		return http.HandlerFunc(fn)
	}
}

// PanicFormatter initiates the beginning of a new PanicEntry per request.
// See DefaultLogAndPanicFormatter for an example implementation.
type PanicFormatter interface {
	NewPanicEntry(r *http.Request) PanicEntry
}

// LogFormatter initiates the beginning of a new LogEntry per request.
// See DefaultLogAndPanicFormatter for an example implementation.
type LogFormatter interface {
	NewLogEntry(r *http.Request) LogEntry
	Accept(r *http.Request) bool
}

type LogAndPanicFormatter interface {
	LogFormatter
	PanicFormatter
}

// LogEntry records the final log when a request completes.
// See defaultLogEntry for an example implementation.
type LogEntry interface {
	Write(status, bytes int, elapsed time.Duration)
	WithLogger(logger LoggerInterface) LogEntry
}

// PanicEntry records the final log when a request failed.
// See defaultPanicEntry for an example implementation.
type PanicEntry interface {
	Write(v interface{}, stack []byte)
	WithLogger(logger LoggerInterface) PanicEntry
}

// GetLogEntry returns the in-context LogEntry for a request.
func GetLogEntry(r *http.Request) LogEntry {
	entry, _ := r.Context().Value(LogEntryCtxKey).(LogEntry)
	return entry
}

// GetLogEntry returns the in-context LogEntry for a request.
func GetPanicEntry(r *http.Request) PanicEntry {
	entry, _ := r.Context().Value(PanicEntryCtxKey).(PanicEntry)
	return entry
}

// WithLogEntry sets the in-context LogEntry for a request.
func WithLogEntry(r *http.Request, entry LogEntry) *http.Request {
	r = r.WithContext(context.WithValue(r.Context(), LogEntryCtxKey, entry))
	return r
}

func NewLogEntry(r *http.Request) LogEntry {
	return DefaultRequestLogFormatter.NewLogEntry(r)
}

func NewPanicEntry(r *http.Request) PanicEntry {
	return DefaultRequestLogFormatter.NewPanicEntry(r)
}

func Accept(r *http.Request) bool {
	return DefaultRequestLogFormatter.Accept(r)
}

// LoggerInterface accepts printing to stdlib logger or compatible logger.
type LoggerInterface interface {
	Print(v ...interface{})
}

// DefaultLogAndPanicFormatter is a simple logger that implements a LogFormatter.
type DefaultLogAndPanicFormatter struct {
	Logger, PanicLogger LoggerInterface
	NoColor             bool
	IgnoreExtensions    Extensions
	TruncateUri         int
	NoColorTtyCheck     bool
}

func (l *DefaultLogAndPanicFormatter) Accept(r *http.Request) bool {
	if l.IgnoreExtensions != nil {
		if ext := path.Ext(r.URL.Path); ext != "" {
			return !l.IgnoreExtensions[ext[1:]]
		}
	}
	return true
}

// NewLogEntry creates a new LogEntry for the request.
func (l *DefaultLogAndPanicFormatter) NewLogEntry(r *http.Request) LogEntry {
	useColor := !l.NoColor
	entry := &defaultLogEntry{
		baseLogEntry{
			DefaultLogAndPanicFormatter: l,
			logger:                      l.Logger,
			request:                     r,
			buf:                         &bytes.Buffer{},
			useColor:                    useColor,
		},
	}

	reqID := middleware.GetReqID(r.Context())

	var cW = ColorWriteTtyCheck
	if l.NoColorTtyCheck {
		cW = ColorWrite
	}

	if reqID != "" {
		cW(entry.buf, useColor, nYellow, "[%s] ", reqID)
	}
	cW(entry.buf, useColor, nCyan, "\"")
	cW(entry.buf, useColor, bMagenta, "%s ", r.Method)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	uri := r.RequestURI
	if l.TruncateUri > 0 && len(uri) > l.TruncateUri+4 {
		uri = uri[0:l.TruncateUri] + " ..."
	}
	cW(entry.buf, useColor, nCyan, "%s://%s%s %s\" ", scheme, r.Host, uri, r.Proto)

	entry.buf.WriteString("from ")
	entry.buf.WriteString(r.RemoteAddr)
	entry.buf.WriteString(" - ")

	return entry
}

// NewPanicEntry creates a new LogEntry for the request panic.
func (l *DefaultLogAndPanicFormatter) NewPanicEntry(r *http.Request) PanicEntry {
	useColor := !l.NoColor
	entry := &defaultPanicEntry{
		baseLogEntry{
			DefaultLogAndPanicFormatter: l,
			logger:                      l.PanicLogger,
			request:                     r,
			buf:                         &bytes.Buffer{},
			useColor:                    useColor,
		},
	}

	reqID := middleware.GetReqID(r.Context())

	var cW = ColorWriteTtyCheck
	if l.NoColorTtyCheck {
		cW = ColorWrite
	}

	if reqID != "" {
		cW(entry.buf, useColor, nYellow, "[%s] ", reqID)
	}
	cW(entry.buf, useColor, nCyan, "\"")
	cW(entry.buf, useColor, bMagenta, "%s ", r.Method)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	cW(entry.buf, useColor, nCyan, "%s://%s%s %s\" ", scheme, r.Host, r.RequestURI, r.Proto)

	entry.buf.WriteString("from ")
	entry.buf.WriteString(r.RemoteAddr)
	entry.buf.WriteString(" - ")

	return entry
}

type baseLogEntry struct {
	*DefaultLogAndPanicFormatter
	logger                    LoggerInterface
	request                   *http.Request
	buf                       *bytes.Buffer
	useColor, fullUrl, panics bool
}

func (l *baseLogEntry) ColorWriter() ColorWriterFunc {
	var cW = ColorWriteTtyCheck
	if l.NoColorTtyCheck {
		cW = ColorWrite
	}
	return cW
}

type defaultLogEntry struct {
	baseLogEntry
}

func (l *defaultLogEntry) Write(status, bytes int, elapsed time.Duration) {
	var cW = l.ColorWriter()

	switch {
	case status < 200:
		cW(l.buf, l.useColor, bBlue, "%03d", status)
	case status < 300:
		cW(l.buf, l.useColor, bGreen, "%03d", status)
	case status < 400:
		cW(l.buf, l.useColor, bCyan, "%03d", status)
	case status < 500:
		cW(l.buf, l.useColor, bYellow, "%03d", status)
	default:
		cW(l.buf, l.useColor, bRed, "%03d", status)
	}

	cW(l.buf, l.useColor, bBlue, " %dB", bytes)

	l.buf.WriteString(" in ")
	if elapsed < 500*time.Millisecond {
		cW(l.buf, l.useColor, nGreen, "%s", elapsed)
	} else if elapsed < 5*time.Second {
		cW(l.buf, l.useColor, nYellow, "%s", elapsed)
	} else {
		cW(l.buf, l.useColor, nRed, "%s", elapsed)
	}

	l.Logger.Print(time.Now().Format(time.RFC3339Nano), l.buf.String())
}

type defaultPanicEntry struct {
	baseLogEntry
}

func (l *defaultPanicEntry) Write(v interface{}, stackb []byte) {
	panicEntry := l.DefaultLogAndPanicFormatter.NewPanicEntry(l.request).(*defaultPanicEntry)
	panicEntry.fullUrl = true
	l.ColorWriter()(panicEntry.buf, l.useColor, bRed, "panic: %+v", v)
	lgr := l.PanicLogger
	if lgr == nil {
		lgr = l.Logger
	}
	lgr.Print(time.Now().Format(time.RFC3339Nano), panicEntry.buf.String())
	var out bytes.Buffer
	c, err := stack.ParseDump(bytes.NewReader(stackb), &out, false)
	if err != nil {
		lgr.Print(string(stackb))
	} else {
		buckets := stack.Aggregate(c.Goroutines, stack.AnyValue)
		if err := StackWriteToConsole(&out, &defaultStackPalette, buckets, false, true, nil, nil); err == nil {
			lgr.Print(out.String())
		} else {
			lgr.Print(string(stackb))
		}
	}
}

func (this defaultPanicEntry) WithLogger(logger LoggerInterface) PanicEntry {
	fmtr := *this.DefaultLogAndPanicFormatter
	fmtr.Logger = logger
	this.DefaultLogAndPanicFormatter = &fmtr
	return &this
}

func StackWriteToConsole(out io.Writer, p *StackPalette, buckets []*stack.Bucket, fullPath, needsEnv bool, filter, match *regexp.Regexp) error {
	if needsEnv {
		_, _ = io.WriteString(out, "\nTo see all goroutines, visit https://github.com/maruel/panicparse#gotraceback\n\n")
	}
	srcLen, pkgLen := CalcLengths(buckets, fullPath)
	for _, bucket := range buckets {
		header := p.BucketHeader(bucket, fullPath, len(buckets) > 1)
		if filter != nil && filter.MatchString(header) {
			continue
		}
		if match != nil && !match.MatchString(header) {
			continue
		}
		_, _ = io.WriteString(out, header)
		_, _ = io.WriteString(out, p.StackLines(&bucket.Signature, srcLen, pkgLen, fullPath))
	}
	return nil
}

func (l defaultLogEntry) WithLogger(logger LoggerInterface) LogEntry {
	fmtr := *l.DefaultLogAndPanicFormatter
	fmtr.Logger = logger
	l.DefaultLogAndPanicFormatter = &fmtr
	return &l
}
