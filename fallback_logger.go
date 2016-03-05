package httplog

import (
	"fmt"
	"log"
	"runtime"
	"strings"
)

// fallbackLogger is used if Server.NewLogEntry is not set. It's not meant to
// be particularly good. README.md contains an example of settings this up.
type fallbackLogger struct {
	msg string
}

func (e *fallbackLogger) AddField(key string, value interface{}) {
	if e.msg != "" {
		e.msg += " "
	}
	e.msg += fmt.Sprintf("%s=\"%v\"", key, value)
}

func (e *fallbackLogger) AddFields(fields map[string]interface{}) {
	for k, v := range fields {
		e.AddField(k, v)
	}
}

func (e *fallbackLogger) AddCallstack() {
	var cs []string
	for i := 0; ; i++ {
		file, line := getCaller(2 + i)
		if file == "???" {
			break
		}
		if strings.HasSuffix(file, ".s") ||
			strings.HasPrefix(file, "http/server.go") ||
			strings.HasPrefix(file, "runtime/proc.go") {
			continue
		}
		cs = append(cs, fmt.Sprintf("%s:%d", file, line))
	}
	e.AddField("callstack", strings.Join(cs, ", "))
}

func (e *fallbackLogger) Info(args ...interface{}) {
	e.Write("info", "", args...)
}

func (e *fallbackLogger) Infof(format string, args ...interface{}) {
	e.Write("info", format, args...)
}

func (e *fallbackLogger) Warn(args ...interface{}) {
	e.Write("warn", "", args...)
}

func (e *fallbackLogger) Warnf(format string, args ...interface{}) {
	e.Write("warn", format, args...)
}

func (e *fallbackLogger) Error(args ...interface{}) {
	e.Write("error", "", args...)
}

func (e *fallbackLogger) Errorf(format string, args ...interface{}) {
	e.Write("error", format, args...)
}

func (e *fallbackLogger) Write(level, format string, args ...interface{}) {
	msg := fmt.Sprintf("[%s] ", level)
	if format != "" {
		msg += fmt.Sprintf(format, args...)
	} else {
		msg += fmt.Sprint(args...)
	}
	if msg != "" {
		msg += " "
	}
	msg += e.msg
	log.Print(msg)
}

func getCaller(skip int) (file string, line int) {
	var ok bool
	_, file, line, ok = runtime.Caller(skip)
	if !ok {
		file = "???"
		line = 0
	} else {
		short := file
		count := 0
		for i := len(file) - 1; i > 0; i-- {
			if file[i] == '/' {
				count++
				if count == 2 {
					short = file[i+1:]
					break
				}
			}
		}
		file = short
	}
	return
}
