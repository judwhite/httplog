package httplog

import (
	"fmt"
	"log"
	"strings"
)

var logPrint = log.Print

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

func (e *fallbackLogger) AddError(err error) {
	e.AddField("err", err)

	var st []frame

	if errStack, ok := err.(*errorStack); ok {
		st = errStack.StackTrace()
	} else {
		st = stackTrace()
		if len(st) < 2 {
			return
		}
		st = st[1:]
	}

	var cs []string
	for _, frame := range st {
		cs = append(cs, fmt.Sprintf("%s:%s:%d", frame.Path(), frame.Func(), frame.Line()))
	}

	if len(cs) > 0 {
		e.AddField("stacktrace", strings.Join(cs, ", "))
	}
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
	logPrint(msg)
}
