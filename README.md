# httplog
Structured, leveled logging for your HTTP server

# Log Output

```
time="2016-03-04T05:34:10-06:00" level=info msg=OK http_bytes_sent=2 http_method=GET http_path="/ping" http_remote_addr="[::1]:36295" http_status=200 time_taken=0
time="2016-03-04T05:34:42-06:00" level=info msg=OK http_bytes_sent=24 http_method=GET http_path="/add" http_remote_addr="[::1]:36305" http_status=200 req_a=1 req_b=2 result_sum=3 time_taken=1
```

```
time="2016-03-04T05:37:37-06:00" level=warning msg="strconv.ParseInt: parsing \"2i\": invalid syntax" callstack="httplog_test/main.go:90, httplog/server.go:66,http/server.go:1618, http/server.go:1910, http/server.go:2081, http/server.go:1472" http_bytes_sent=0 http_method=GET http_path="/add" http_remote_addr="[::1]:36461" http_status=400 req_a=1 req_b=2i time_taken=0
```

```
time="2016-03-04T05:39:21-06:00" level=error msg="runtime error: integer divideby zero" callstack="httplog/server.go:56, runtime/panic.go:426, runtime/panic.go:27, runtime/signal_windows.go:166, httplog_test/main.go:113, httplog/server.go:66, http/server.go:1618, http/server.go:1910, http/server.go:2081, http/server.go:1472" http_bytes_sent=0 http_method=GET http_path="/panic" http_remote_addr="[::1]:36461" http_status=500 time_taken=0
```

# Example

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

	svr.ShutdownTimeout = 2 * time.Second

	http.HandleFunc("/add", svr.Handle(addHandler))
	http.HandleFunc("/ping", svr.Handle(pingHandler))
	http.HandleFunc("/panic", svr.Handle(panicHandler))
	http.HandleFunc("/slow", svr.Handle(slowHandler))
	http.HandleFunc("/slower", svr.Handle(slowerHandler))
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

	qpa := m.Get("a")
	qpb := m.Get("b")

	log.AddFields(fields{
		"req_a": qpa,
		"req_b": qpb,
	})

	a, err := strconv.Atoi(qpa)
	if err != nil {
		log.AddCallstack()
		return nil, http.StatusBadRequest, err
	}
	b, err := strconv.Atoi(qpb)
	if err != nil {
		log.AddCallstack()
		return nil, http.StatusBadRequest, err
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

func slowHandler(r *http.Request, log httplog.Entry) (interface{}, int, error) {
	start := time.Now()
	time.Sleep(1 * time.Second)
	return time.Since(start).String(), 200, nil
}

func slowerHandler(r *http.Request, log httplog.Entry) (interface{}, int, error) {
	start := time.Now()
	time.Sleep(5 * time.Second)
	return time.Since(start).String(), 200, nil
}

func catchAllHandler(r *http.Request, log httplog.Entry) (interface{}, int, error) {
	return "404", http.StatusNotFound, nil
}
```
