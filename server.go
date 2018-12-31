package httplog

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Server provides functionality for:
//
//   - Structured, leveled logging per request via the Handle method
//   - Error and panic handling
//   - Clean shutdown
//
// Server is intended to be embedded in another struct, though it
// can be used standalone.
//
// See the Handle method for behavior details.
type Server struct {
	stopped         int32
	openConnections int32

	// ShutdownTimeout defines the duration to wait for outstanding requests
	// to complete before the Shutdown method returns. The default is 30s.
	ShutdownTimeout time.Duration
	// FormatJSON determines whether non-byte and non-string responses are
	// formatted with MarshalIndent (when true) or Marshal (when false). The
	// default is false.
	FormatJSON bool
	// NewLogEntry is a "func() Entry" field. Set this property to specify
	// how new log entries are created. This field must be set to integrate
	// with an outside logging package.
	NewLogEntry func() Entry
}

const gzipMinLength = 1000
const gzipCompLevel = gzip.DefaultCompression

var gzipTypes = map[string]bool{
	"application/javascript": true,
	"application/json":       true,
	"application/xml":        true,
	"font/opentype":          true,
	"image/svg+xml":          true,
	"image/x-icon":           true,
	"text/css":               true,
	"text/html":              true,
	"text/plain":             true,
}

// Entry is implemented by a log entry.
type Entry interface {
	AddField(key string, value interface{})
	AddFields(fields map[string]interface{})
	AddError(err error)
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
}

// Handler contains the handler name and handler function.
type Handler struct {
	Name string
	Func loggedHandler
}

type loggedHandler func(r *http.Request, entry Entry) (Response, error)

// Response contains the body, status, and HTTP headers to return.
type Response struct {
	Body    interface{}
	Status  int
	Headers []Header
}

// Header contains the name/value pair of a response HTTP header.
type Header struct {
	Name  string
	Value string
}

// Handle accepts a Handler and returns a function which
// can be passed to http.HandleFunc.
//
// If the Shutdown method has been called Handle responds with
// StatusServiceUnavailable (503).
//
// If the Handler panics it's recovered and the server responds with
// StatusInternalServerError (500). The callstack is also captured and added
// to the log.
//
// If the response from Handler is a type other than string or
// []byte the object is serialized as JSON. See the FormatJSON field.
//
// Returning an error from Handler does not modify the status code. The
// error itself will be written to the log.
//
// After the response has been written to the client WriteHTTPLog is called.
func (svr *Server) Handle(handler Handler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		bytesSent := 0
		status := 0
		start := time.Now()
		logEntry := svr.newEntry()

		var decOpenConnections bool
		var err error

		defer func() {
			if perr := recover(); perr != nil {
				status = http.StatusInternalServerError
				w.WriteHeader(status)

				var ok bool
				var panicErr error
				if panicErr, ok = perr.(error); !ok {
					panicErr = fmt.Errorf("%v", perr)
				}
				panicErr = withStack(panicErr)
				if err == nil {
					err = panicErr
				} else {
					// TODO (judwhite): wipes stack trace. add method for adding multiple errors.
					err = fmt.Errorf("handler: %v\npanic: %v", err.Error(), panicErr.Error())
				}
			}

			duration := time.Since(start)
			go WriteHTTPLog(handler.Name, logEntry, r, duration, status, bytesSent, err)

			if decOpenConnections {
				atomic.AddInt32(&svr.openConnections, -1)
			}
		}()

		// stopped
		if atomic.LoadInt32(&svr.stopped) == 1 {
			status = http.StatusServiceUnavailable
			return
		}

		decOpenConnections = true
		atomic.AddInt32(&svr.openConnections, 1)

		httpResponse, err := handler.Func(r, logEntry)
		err = withStack(err)

		resp := httpResponse.Body
		status = httpResponse.Status
		headers := httpResponse.Headers

		if status == 0 {
			status = 200
		}

		for _, hdr := range headers {
			w.Header().Add(hdr.Name, hdr.Value)
		}

		if resp == nil {
			w.WriteHeader(status)
			return
		}

		var body []byte
		if respString, ok := resp.(string); ok {
			body = []byte(respString)
			if w.Header().Get("Content-Type") == "" {
				w.Header().Set("Content-Type", "text/plain")
			}
		} else if respBytes, ok := resp.([]byte); ok {
			body = respBytes
		} else {
			var marshalErr error
			if svr.FormatJSON {
				body, marshalErr = json.MarshalIndent(resp, "", "  ")
			} else {
				body, marshalErr = json.Marshal(resp)
			}
			if marshalErr != nil {
				panic(marshalErr)
			}
			w.Header().Set("Content-Type", "application/json")
		}

		if len(body) == 0 {
			w.WriteHeader(status)
			return
		}

		bodyHasGzipMagicHeader := len(body) > 1 && body[0] == 0x1f && body[1] == 0x8b

		writeBody := func() (int, error) {
			return w.Write(body)
		}

		gzipOK := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		if bodyHasGzipMagicHeader {
			if !gzipOK {
				w.Header().Del("Content-Encoding")

				buf := bytes.NewBuffer(body)
				reader, newReaderErr := gzip.NewReader(buf)
				if newReaderErr != nil {
					panic(newReaderErr)
				}
				writeBody = func() (int, error) {
					n, localErr := io.Copy(w, reader)
					closeErr := reader.Close()
					if localErr == nil && closeErr != nil {
						localErr = closeErr
					}
					return int(n), localErr
				}
			} else {
				w.Header().Set("Content-Encoding", "gzip")
			}
		} else if gzipOK && len(body) > gzipMinLength && gzipTypes[w.Header().Get("Content-Type")] {
			w.Header().Set("Content-Encoding", "gzip")

			wc := &writeCounter{writer: w}
			gzipWriter, newWriterErr := gzip.NewWriterLevel(wc, gzipCompLevel)
			if newWriterErr != nil {
				panic(newWriterErr)
			}
			writeBody = func() (int, error) {
				_, localErr := gzipWriter.Write(body)
				closeErr := gzipWriter.Close()
				if localErr == nil && closeErr != nil {
					localErr = closeErr
				}
				return wc.count, localErr
			}
		}

		w.WriteHeader(status)
		n, writeBodyErr := writeBody()
		bytesSent = n
		if writeBodyErr != nil {
			panic(writeBodyErr)
		}
	}
}

type writeCounter struct {
	writer io.Writer
	count  int
}

func (c *writeCounter) Write(p []byte) (int, error) {
	n, err := c.writer.Write(p)
	c.count += n
	return n, err
}

// Shutdown attempts a graceful shutdown, waiting for outstanding connections
// to complete. See ShutdownTimeout.
func (svr *Server) Shutdown() {
	atomic.StoreInt32(&svr.stopped, 1)

	deadlineTimeout := svr.ShutdownTimeout
	if deadlineTimeout == 0 {
		deadlineTimeout = 30 * time.Second
	}

	deadline := time.After(deadlineTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
loop:
	for {
		entry := svr.newEntry()
		select {
		case <-ticker.C:
			conns := atomic.LoadInt32(&svr.openConnections)
			if conns > 0 {
				entry.Infof("waiting for %d connections to close", conns)
			} else {
				entry.Info("all connections closed")
				break loop
			}
		case <-deadline:
			conns := atomic.LoadInt32(&svr.openConnections)
			if conns > 0 {
				entry.Errorf("stop deadline %v exceeded; aborting %d connections", deadlineTimeout, conns)
			}
			break loop
		}
	}
}

func (svr *Server) newEntry() Entry {
	newEntryFunc := svr.NewLogEntry
	if newEntryFunc != nil {
		return newEntryFunc()
	}
	log.Print("*** WARNING *** Set Server.NewLogEntry implementation to use your logging framework. Using fallback logger.")
	svr.NewLogEntry = func() Entry { return &fallbackLogger{} }
	return svr.newEntry()
}

// WriteHTTPLog writes the following keys to the log entry:
//
//   bytes_sent           The number of bytes sent in the HTTP response body.
//   host                 The remote host name. If the host name cannot be resolved, IP is repeated here.
//   http_status          The HTTP status code returned.
//   ip                   The remote IP address.
//   method               GET, POST, PUT, DELETE, etc
//   time_taken           The time taken to complete the request in milliseconds, including writing to the client.
//   uri                  The request URI.
//
// The log level is determined by the status code:
//
//   status < 400          Info
//   400 <= status < 500   Warning
//   status >= 500         Error
//
// This function is invoked by Server's Handle method.
func WriteHTTPLog(handlerName string, entry Entry, r *http.Request, duration time.Duration, status int, bytesSent int, err error) {
	timeTakenSecs := float64(duration) / 1e9

	labelValues := []string{strconv.Itoa(status), handlerName, r.Method}
	httpRequestsTotal.WithLabelValues(labelValues...).Inc()
	httpRequestDurationCounter.WithLabelValues(labelValues...).Observe(timeTakenSecs)

	var host string

	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		forwardedFor := r.Header.Get("X-Forwarded-For")
		ip = strings.SplitN(forwardedFor, ",", 2)[0]
		if ip == "" {
			var splitErr error
			ip, _, splitErr = net.SplitHostPort(r.RemoteAddr)
			if splitErr != nil {
				ip = r.RemoteAddr
				host = r.RemoteAddr
			}
		}
	}

	if host == "" {
		host = getHostFromIP(ip)
	}

	entry.AddFields(map[string]interface{}{
		"bytes_sent":  bytesSent,
		"host":        host,
		"http_status": status,
		"ip":          ip,
		"method":      r.Method,
		"time_taken":  int64(timeTakenSecs * 1000),
		"uri":         r.RequestURI,
	})

	msg := http.StatusText(status)
	if err != nil {
		entry.AddError(err)
	}

	if status >= 400 && status < 500 {
		entry.Warn(msg)
	} else if status >= 500 {
		entry.Error(msg)
	} else {
		entry.Info(msg)
	}
}

var ipHost map[string]string
var ipHostMtx sync.RWMutex

func init() {
	ipHost = make(map[string]string)
}

// GetHostFromAddress gets a host name from an IPv4 address
func getHostFromIP(ip string) string {
	ipHostMtx.RLock()
	entry, ok := ipHost[ip]
	ipHostMtx.RUnlock()

	if !ok {
		names, lookupErr := net.LookupAddr(ip)
		if lookupErr != nil || len(names) == 0 {
			entry = ip
		} else {
			entry = strings.TrimSuffix(names[0], ".")
		}

		ipHostMtx.Lock()
		ipHost[ip] = entry
		ipHostMtx.Unlock()
	}

	return entry
}
