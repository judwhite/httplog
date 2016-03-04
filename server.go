package httplog

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	log "github.com/judwhite/logrjack"
)

type Server struct {
	stopped         int32
	openConnections int32

	ShutdownTimeout time.Duration
	FormatJSON      bool
}

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

type LoggedHandler func(r *http.Request, requestLogger Entry) (interface{}, int, error)

func (svr *Server) Handle(handler LoggedHandler) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		bytesSent := 0
		status := 0
		start := time.Now()
		requestLogger := log.NewEntry()

		// stopped
		if atomic.LoadInt32(&svr.stopped) == 1 {
			w.WriteHeader(500)
			WriteHTTPLog(requestLogger, r, start, 500, bytesSent, errors.New("server shutting down"))
			return
		}

		atomic.AddInt32(&svr.openConnections, 1)

		defer func() {
			perr := recover()
			if perr != nil {
				w.WriteHeader(500)
				requestLogger.AddCallstack()
				perror, ok := perr.(error)
				if !ok {
					perror = errors.New(fmt.Sprintf("%v", perr))
				}
				WriteHTTPLog(requestLogger, r, start, 500, bytesSent, perror)
			}
			atomic.AddInt32(&svr.openConnections, -1)
		}()

		resp, status, err := handler(r, requestLogger)

		if resp != nil {
			var body []byte
			if respString, ok := resp.(string); ok {
				body = []byte(respString)
			} else if respBytes, ok := resp.([]byte); ok {
				body = respBytes
			} else {
				if svr.FormatJSON {
					body, err = json.MarshalIndent(resp, "", "  ")
				} else {
					body, err = json.Marshal(resp)
				}
				if err != nil {
					requestLogger.AddCallstack()
					status = 500
				}
			}

			w.WriteHeader(status)

			if body != nil {
				_, err = w.Write(body)
				if err != nil {
					requestLogger.AddCallstack()
				} else {
					bytesSent = len(body)
				}
			}
		} else {
			w.WriteHeader(status)
		}

		WriteHTTPLog(requestLogger, r, start, status, bytesSent, err)
	}
}

func (svr *Server) Shutdown() {
	logger := log.NewEntry()

	atomic.StoreInt32(&svr.stopped, 1)

	deadlineTimeout := svr.ShutdownTimeout
	if deadlineTimeout == 0 {
		deadlineTimeout = 5 * time.Second
	}

	deadline := time.After(deadlineTimeout)
	ticker := time.NewTicker(100 * time.Millisecond)
loop:
	for {
		select {
		case <-ticker.C:
			conns := atomic.LoadInt32(&svr.openConnections)
			if conns > 0 {
				logger.Warnf("waiting for %d connections to close", conns)
			} else {
				logger.Info("all connections closed")
				break loop
			}
		case <-deadline:
			logger.Errorf("stop deadline %v exceeded; aborting connections", deadlineTimeout)
			break loop
		}
	}
}

func WriteHTTPLog(entry Entry, r *http.Request, start time.Time, status int, bytesSent int, err error) {
	entry.AddFields(map[string]interface{}{
		"http_method":      r.Method,
		"http_path":        r.URL.Path,
		"http_remote_addr": r.RemoteAddr,
		"http_bytes_sent":  bytesSent,
		"http_status":      status,
		"time_taken":       int64(time.Since(start) / time.Millisecond),
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
