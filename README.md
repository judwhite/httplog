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
