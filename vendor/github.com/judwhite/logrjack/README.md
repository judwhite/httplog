# logrjack
[Logrus](https://github.com/Sirupsen/logrus) (structured, leveled logging) and [Lumberjack](https://github.com/natefinch/lumberjack) (rolling logs).

# Example

```go
package main

import (
	"errors"
	"fmt"
	"runtime"
	"time"

	log "github.com/judwhite/logrjack"
)

type fields map[string]interface{} // optional, makes the call the AddFields look nice

func main() {
	start := time.Now()

	// Setup the log file name, log rolling options
	log.Setup(log.Settings{
		Filename:    "example.log", // optional, defaults to ./logs/<procname>.log
		MaxSizeMB:   100,
		MaxAgeDays:  30,
		WriteStdout: true,
	})

	// Simple log message
	log.Info("Welcome!")

	// AddFields example
	entry := log.NewEntry()
	entry.AddFields(fields{
		"runtime_version": runtime.Version(),
		"arch":            runtime.GOARCH,
	})
	entry.Info("OK")

	// error/panic examples
	errorExamples()

	// AddField example
	entry = log.NewEntry()
	entry.AddField("uptime", time.Since(start))
	entry.Info("Shutting down.")
}

func errorExamples() {
	// Error and panic examples. You might use this in an HTTP handler for
	// errors, or as part of the middleware for panics.

	// numerator, denominator
	n, d := 10, 0

	// 'divide' panics, recovers, adds the callstack, and returns an error
	entry := log.NewEntry()
	if res, err := divide(n, d, entry); err != nil {
		entry.Error(err)
	} else {
		entry.Infof("%d/%d=%d", n, d, res)
	}

	// 'safeDivide' checks if d == 0. if so it adds the callstack and returns an error
	entry = log.NewEntry()
	if res, err := safeDivide(n, d, entry); err != nil {
		entry.Error(err)
	} else {
		entry.Infof("%d/%d=%d", n, d, res)
	}
}

func divide(n, d int, entry log.Entry) (res int, err error) {
	defer func() {
		perr := recover()
		if perr != nil {
			entry.AddCallstack()
			var ok bool
			err, ok = perr.(error)
			if ok {
				return
			}
			err = errors.New(fmt.Sprintf("%v", perr))
		}
	}()

	res = n / d
	return
}

func safeDivide(n, d int, entry log.Entry) (int, error) {
	if d == 0 {
		entry.AddCallstack()
		return 0, errors.New("d must not equal 0")
	}
	return n / d, nil
}
```

Outputs:

```
time="2016-03-04T02:29:30-06:00" level=info msg="Welcome!" 
time="2016-03-04T02:29:30-06:00" level=info msg=OK arch=amd64 runtime_version=go1.6 
time="2016-03-04T02:29:30-06:00" level=error msg="runtime error: integer divide by zero" callstack="logrjack_test/main.go:72, runtime/panic.go:426, runtime/panic.go:27, runtime/signal_windows.go:166, logrjack_test/main.go:82, logrjack_test/main.go:53, logrjack_test/main.go:36" 
time="2016-03-04T02:29:30-06:00" level=error msg="d must not equal 0" callstack="logrjack_test/main.go:88, logrjack_test/main.go:61, logrjack_test/main.go:36" 
time="2016-03-04T02:29:30-06:00" level=info msg="Shutting down." uptime=1.0001ms 
```

Output if `d` is changed to non-zero (the happy path):

```
time="2016-03-04T02:33:18-06:00" level=info msg="Welcome!" 
time="2016-03-04T02:33:18-06:00" level=info msg=OK arch=amd64 runtime_version=go1.6 
time="2016-03-04T02:33:18-06:00" level=info msg="10/5=2" 
time="2016-03-04T02:33:18-06:00" level=info msg="10/5=2" 
time="2016-03-04T02:33:18-06:00" level=info msg="Shutting down." uptime=1.0001ms 
```

# Notes

This package is meant to output [logfmt](https://github.com/kr/logfmt) formatted text to a file and optionally stdout. It doesn't expose the hooks available in Logrus or extra features in Lumberjack.

Notable differences from Logrus:
- `Entry` is an interface.
- Instead of calling `WithField` and receiving `*Entry`, call `AddField` like above. The Logrus `Entry` is wrapped in an unexported type. The downside is you can't setup a base `Entry` to be passed to multiple routines which write their separate log entries from the same base. Call `NewEntry` and copy the values if you want this behavior.
- A handy `AddCallstack` method, useful for logging errors and panics. It adds a key named `callstack` and is formatted `dir/file.go:##, dir/file2.go:##`. `runtime/proc.go`, `http/server.go`, and files which end in `.s`, such as `runtime/asm_amd64.s`, are omitted from the callstack. All other entries are included, including `runtime/panic.go` in a panic recovery.
- `AddField` takes a `map[string]interface{}` so the interface can be implemented in a nested-vendor setup. You can create your own type as above to shorten the code.

Notable differences from Lumberjack:
- If left unspecified, the default filename is `<processpath>/logs/<processname>.log`.

`./vendor` notes:
- Logrus is v0.9.0 https://github.com/Sirupsen/logrus/commit/be52937128b38f1d99787bb476c789e2af1147f1.
- Lumberjack is `gopkg.in/natefinch/lumberjack.v2`.
- Feel free to swap out for a newer version or delete the vendor directory.

# Why?

I got tired of copy-pasting this type of code all over the place. It's a convenience for myself, hopefully someone else will find it useful :)

# License

MIT.

At the time of this writing both Logrus and Lumberjack are also licensed under MIT.
