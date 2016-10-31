// Package httplog provides common functionality for a web server, including:
//
//   - Structured logging
//   - Panic handling
//   - JSON marshalling
//   - Waiting for processing requests to complete before shutdown
//   - Optional Gzip compression
package httplog

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	// formatted with MarshalIndent (when true) or Mashal (when false. The
	// default is false.
	FormatJSON bool
	// NewLogEntry is a "func() Entry" field. Set this property to specify
	// how new log entries are created. This field must be set to integrate
	// with an outside logging package.
	NewLogEntry func() Entry
	DisableGzip bool
}

// Entry is implemented by a log entry.
type Entry interface {
	AddField(key string, value interface{})
	AddFields(fields map[string]interface{})
	AddCallstack()
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
			perr := recover()
			if perr != nil {
				status = http.StatusInternalServerError
				w.WriteHeader(status)
				requestLogger.AddCallstack()
				perror, ok := perr.(error)
				if !ok {
					perror = fmt.Errorf("%v", perr)
				}
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
					err = fmt.Errorf("%s - json.Marshal:%s", err, marshalErr)
					requestLogger.AddCallstack()
					status = http.StatusInternalServerError
				}
				if w.Header().Get("Content-Type") == "" {
					w.Header().Add("Content-Type", "application/json")
				}
			}

			writeBodyErr := svr.writeHeaderBody(w, r, status, body)

			if writeBodyErr != nil {
				err = fmt.Errorf("%s - w.Write:%s", err, writeBodyErr)
				requestLogger.AddCallstack()
			} else {
				bytesSent = len(body)
			}
		} else {
			w.WriteHeader(status)
		}

		WriteHTTPLog(requestLogger, r, start, status, bytesSent, err)
	}
}

// Shutdown attempst a graceful shutdown, waiting for outstanding connections
// to complete. See ShutdownTimeout.
func (svr *Server) Shutdown() {
	atomic.StoreInt32(&svr.stopped, 1)

	deadlineTimeout := svr.ShutdownTimeout
	if deadlineTimeout == 0 {
		deadlineTimeout = 30 * time.Second
	}

	deadline := time.After(deadlineTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
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
//   status <= 200         Info
//   400 <= status < 500   Warning
//   status > 500          Error
//
// This function is invoked by Server's Handle method.
func WriteHTTPLog(entry Entry, r *http.Request, start time.Time, status int, bytesSent int, err error) {
	entry.AddFields(map[string]interface{}{
		"bytes_sent":  bytesSent,
		"http_status": status,
		"method":      r.Method,
		"remote_addr": r.RemoteAddr,
		"time_taken":  int64(time.Since(start) / time.Millisecond),
		"uri":         r.RequestURI,
	})

	var msg string
	if err != nil {
		msg = err.Error()
	} else {
		msg = "OK"
	}

	if status >= 400 && status < 500 {
		entry.Warn(msg)
	} else if status >= 500 || err != nil {
		entry.Error(msg)
	} else {
		entry.Info(msg)
	}
}

func (svr *Server) writeHeaderBody(w http.ResponseWriter, r *http.Request, status int, body []byte) error {
	var err error

	contentEncoding := w.Header().Get("Content-Encoding")
	acceptEncoding := r.Header.Get("Accept-Encoding")
	if len(body) < 150 || svr.DisableGzip || contentEncoding != "" || !acceptsGzip(acceptEncoding) {
		w.WriteHeader(status)
		if len(body) > 0 {
			_, err = w.Write(body)
		}
		return err
	}

	gw, err := gzip.NewWriterLevel(w, gzip.DefaultCompression)
	if err != nil {
		return err
	}
	_, err = gw.Write(body)
	return err
}

func acceptsGzip(acceptEncoding string) bool {
	acceptEncoding = strings.TrimSpace(acceptEncoding)

	// fast path for common Accept-Encoding strings
	if len(acceptEncoding) == 0 {
		return false
	}
	// examples:
	//   gzip
	//   deflate, gzip
	//   gzip, deflate, sdch, br
	//   gzip, deflate, br
	//   gzip, deflate
	/*if strings.Contains(acceptEncoding, "gzip,") || strings.HasSuffix(acceptEncoding, "gzip") {
		return true
	}
	if !strings.Contains(acceptEncoding, "gzip") {
		return false
	}*/

	j := 0
	for i := 0; i < len(acceptEncoding); i++ {
		if acceptEncoding[i] == ',' || i == len(acceptEncoding)-1 {
			if i == len(acceptEncoding)-1 {
				i++
			}
			k := j
			skipSpaces := func() {
				for ; k < i && acceptEncoding[k] == ' '; k++ {
				}
			}
			skipSpaces()
			if k+4 > i {
				return false
			}
			j = i + 1
			if acceptEncoding[k:k+4] != "gzip" {
				//log.Printf("! %q\n", acceptEncoding[k:k+4])
				continue
			}
			if k+4 == i {
				//log.Printf("OK! %q %d %d\n", acceptEncoding, k, i)
				return true
			}
			k += 4
			skipSpaces()
			if acceptEncoding[k] == ',' {
				return true
			}
			if acceptEncoding[k] != ';' {
				return false
			}
			k++
			skipSpaces()
			if k+2 >= i {
				return false
			}
			if acceptEncoding[k] != 'q' || acceptEncoding[k+1] != '=' {
				return false
			}
			k += 2
			for ; k < i; k++ {
				if acceptEncoding[k] != '0' && acceptEncoding[k] != '.' {
					return true
				}
			}
			return false
		}
	}
	return false

	/*loop:
	for _, val := range strings.Split(acceptEncoding, ",") {
		for i, part := range strings.Split(val, ";") {
			part = strings.TrimSpace(part)
			if i == 0 {
				if part != "gzip" {
					continue loop
				}
			} else {
				if strings.HasPrefix(part, "q=") {
					for j := 2; j < len(part); j++ {
						if part[j] != '0' && part[j] != '.' {
							return true
						}
					}
					return false
				}
			}
		}
		return true
	}
	return false*/
}
