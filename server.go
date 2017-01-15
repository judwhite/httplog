package httplog

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"net"

	"strings"
	"sync"

	"github.com/pkg/errors"
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
	// formatted with MarshalIndent (when true) or Mashal (when false. The
	// default is false.
	FormatJSON bool
	// NewLogEntry is a "func() Entry" field. Set this property to specify
	// how new log entries are created. This field must be set to integrate
	// with an outside logging package.
	NewLogEntry func() Entry
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

// LoggedHandler is the signature of the handler passed to the Handle method.
type LoggedHandler func(r *http.Request, requestLogger Entry) (Response, error)

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

// Handle accepts a LoggedHandler and returns a function which
// can be passed to http.HandleFunc.
//
// If the Shutdown method has been called Handle responds with
// StatusServiceUnavailable (503).
//
// If the LoggedHandler panics it's recovered and the server responds with
// StatusInternalServerError (500). The callstack is also captured and added
// to the log.
//
// If the response from the LoggedHandler is a type other than string or
// []byte the object is serialized as JSON. See the FormatJSON field.
//
// Returning an error from LoggedHandler does not modify the status code. The
// error itself will be written to the log.
//
// After the response has been written to the client WriteHTTPLog is called.
func (svr *Server) Handle(handler LoggedHandler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		bytesSent := 0
		status := 0
		start := time.Now()
		requestLogger := svr.newEntry()

		// stopped
		if atomic.LoadInt32(&svr.stopped) == 1 {
			status = http.StatusServiceUnavailable
			w.WriteHeader(status)
			WriteHTTPLog(requestLogger, r, start, status, bytesSent, errors.New("server shutting down"))
			return
		}

		atomic.AddInt32(&svr.openConnections, 1)

		defer func() {
			if perr := recover(); perr != nil {
				status = http.StatusInternalServerError
				w.WriteHeader(status)

				perror, ok := perr.(error)
				if !ok {
					perror = fmt.Errorf("%v", perr)
				}

				perror = errors.WithStack(perror)

				WriteHTTPLog(requestLogger, r, start, status, bytesSent, perror)
			}
			atomic.AddInt32(&svr.openConnections, -1)
		}()

		httpResponse, err := handler(r, requestLogger)

		resp := httpResponse.Body
		status = httpResponse.Status
		headers := httpResponse.Headers

		if status == 0 {
			status = 200
		}

		for _, hdr := range headers {
			w.Header().Add(hdr.Name, hdr.Value)
		}

		if resp != nil {
			var body []byte
			if respString, ok := resp.(string); ok {
				body = []byte(respString)
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
					if err != nil {
						err = errors.Wrap(err, fmt.Sprintf("json.Marshal:%s", marshalErr))
					} else {
						err = errors.Wrap(marshalErr, "json.Marshal")
					}
					status = http.StatusInternalServerError
				} else {
					w.Header().Add("Content-Type", "application/json")
				}
			}

			w.WriteHeader(status)

			if body != nil {
				_, writeBodyErr := w.Write(body)
				if writeBodyErr != nil {
					if err != nil {
						err = errors.Wrap(err, fmt.Sprintf("http.ResponseWriter.Write:%s", writeBodyErr))
					} else {
						err = errors.Wrap(writeBodyErr, "http.ResponseWriter.Write")
					}
				} else {
					bytesSent = len(body)
				}
			}
		} else {
			w.WriteHeader(status)
		}

		go WriteHTTPLog(requestLogger, r, start, status, bytesSent, err)
	}
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
			entry.Errorf("stop deadline %v exceeded; aborting connections", deadlineTimeout)
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
//   http_status          The HTTP status code returned.
//   method               GET, POST, PUT, DELETE, etc
//   remote_addr          The remote IP address.
//   time_taken           The time taken to complete the request in milliseconds,
//                        including writing to the client.
//   uri                  The request URI.
//
// The log level is determined by the status code:
//
//   status < 400          Info
//   400 <= status < 500   Warning
//   status >= 500         Error
//
// This function is invoked by Server's Handle method.
func WriteHTTPLog(entry Entry, r *http.Request, start time.Time, status int, bytesSent int, err error) {
	timeTaken := int64(time.Since(start) / time.Millisecond)

	var host string
	ip, _, splitErr := net.SplitHostPort(r.RemoteAddr)
	if splitErr != nil {
		ip = r.RemoteAddr
		host = r.RemoteAddr
	} else {
		host = getHostFromIP(ip)
	}

	entry.AddFields(map[string]interface{}{
		"http_status": status,
		"method":      r.Method,
		"uri":         r.RequestURI,
		"bytes_sent":  bytesSent,
		"ip":          ip,
		"host":        host,
		"time_taken":  timeTaken,
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
