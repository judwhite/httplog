# httplog
Structured, leveled logging for your HTTP server. Returns structs as JSON and shuts down gracefully.

Wrapped up in a package so I don't need to copy this code over and over again.

Graceful shutdown packages:
- https://github.com/facebookgo/grace
- https://github.com/tylerb/graceful

Logging packages:
- https://github.com/Sirupsen/logrus
- https://github.com/natefinch/lumberjack
- https://github.com/judwhite/logrjack

# Example Log Output

```
GET http://localhost:5678/ping
GET http://localhost:5678/add?a=1&b=2
GET http://localhost:5678/wait?t=2

time="2016-03-04T18:17:07-06:00" level=info msg=OK bytes_sent=2 http_status=200 method=GET remote_addr="[::1]:55915" time_taken=0 uri="/ping"
time="2016-03-04T18:17:14-06:00" level=info msg=OK bytes_sent=37 http_status=200 method=GET remote_addr="[::1]:55915" req_a=1 req_b=2 result_sum=3 time_taken=2 uri="/add?a=1&b=2"
time="2016-03-04T18:17:30-06:00" level=info msg=OK bytes_sent=2 http_status=200 method=GET remote_addr="[::1]:55915" time_taken=2000 uri="/wait?t=2"

GET http://localhost:5678/add?a=1&b=badparam
GET http://localhost:5678/wait

time="2016-03-04T18:33:02-06:00" level=warning msg="strconv.ParseInt: parsing \"badparam\": invalid syntax" bytes_sent=31 callstack="httplog_test/main.go:89, httplog/server.go:102" http_status=400 method=GET remote_addr="[::1]:45908" req_a=1 req_b=badparam time_taken=0 uri="/add?a=1&b=badparam"
time="2016-03-04T18:33:08-06:00" level=warning msg="strconv.ParseInt: parsing \"\": invalid syntax" bytes_sent=23 callstack="httplog_test/main.go:128, httplog/server.go:102" http_status=400 method=GET remote_addr="[::1]:45908" time_taken=0 uri="/wait"

GET http://localhost:5678/panic

time="2016-03-04T18:21:21-06:00" level=error msg="runtime error: integer divide by zero" bytes_sent=0 callstack="httplog/server.go:92, runtime/panic.go:426, runtime/panic.go:27, runtime/signal_windows.go:166, httplog_test/main.go:112, httplog/server.go:102" http_status=500 method=GET remote_addr="[::1]:55915" time_taken=0 uri="/panic"
```

# Example

Note: In practice you'd probably want to use a mux package such as https://github.com/gorilla/mux.

```go
package main

import (
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/judwhite/httplog"
	log "github.com/judwhite/logrjack"
)

type webServer struct {
	httplog.Server
}

type fields map[string]interface{}

func main() {
	start := time.Now()

	log.Setup(log.Settings{
		MaxSizeMB:   100,
		MaxAgeDays:  30,
		WriteStdout: true,
	})

	svr := webServer{}
	svr.FormatJSON = true
	svr.ShutdownTimeout = 2 * time.Second
	go func() {
		svr.Start()
	}()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, os.Kill)
	<-signalChan

	log.Info("Shutting down...")
	svr.Shutdown()
	log.Infof("Shutdown complete. Total uptime: %v", time.Since(start))
}

func (svr *webServer) Start() {
	const addr = ":5678"

	http.HandleFunc("/add", svr.Handle(addHandler))
	http.HandleFunc("/ping", svr.Handle(pingHandler))
	http.HandleFunc("/panic", svr.Handle(panicHandler))
	http.HandleFunc("/wait", svr.Handle(waitHandler))
	http.HandleFunc("/", svr.Handle(catchAllHandler))

	log.Infof("Listening on %s...", addr)

	log.Fatal(http.ListenAndServe(addr, nil))
}

type addResponse struct {
	A      int `json:"a"`
	B      int `json:"b"`
	Result int `json:"result"`
}

func addHandler(r *http.Request, log httplog.Entry) (interface{}, int, error) {
	u, err := url.ParseRequestURI(r.RequestURI)
	if err != nil {
		log.AddCallstack()
		return nil, http.StatusBadRequest, err
	}
	m := u.Query()

	queryParamA := m.Get("a")
	queryParamB := m.Get("b")

	log.AddFields(fields{
		"req_a": queryParamA,
		"req_b": queryParamB,
	})

	a, err := strconv.Atoi(queryParamA)
	if err != nil {
		log.AddCallstack()
		return "specify ?a=[int]&b=[int] to add", http.StatusBadRequest, err
	}
	b, err := strconv.Atoi(queryParamB)
	if err != nil {
		log.AddCallstack()
		return "specify ?a=[int]&b=[int] to add", http.StatusBadRequest, err
	}

	sum := a + b
	log.AddField("result_sum", sum)

	resp := addResponse{
		A:      a,
		B:      b,
		Result: sum,
	}

	return resp, 200, nil
}

func pingHandler(r *http.Request, log httplog.Entry) (interface{}, int, error) {
	return "OK", 200, nil
}

func panicHandler(r *http.Request, requestLogger httplog.Entry) (interface{}, int, error) {
	var a int
	for i := 0; i < 1; i++ {
		a += 1 / i // cause a panic
	}
	return a, 200, nil
}

func waitHandler(r *http.Request, log httplog.Entry) (interface{}, int, error) {
	u, err := url.ParseRequestURI(r.RequestURI)
	if err != nil {
		log.AddCallstack()
		return nil, http.StatusBadRequest, err
	}
	m := u.Query()

	t := m.Get("t")
	secs, err := strconv.Atoi(t)
	if err != nil {
		log.AddCallstack()
		return "specific ?t= in seconds", http.StatusBadRequest, err
	}

	start := time.Now()
	time.Sleep(time.Duration(secs) * time.Second)
	return time.Since(start).String(), 200, nil
}

func catchAllHandler(r *http.Request, log httplog.Entry) (interface{}, int, error) {
	return "404", http.StatusNotFound, nil
}
```

# License

MIT
