package logrjack

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Sirupsen/logrus"
)

type entry struct {
	entry *logrus.Entry
}

// Entry is the interface returned by NewEntry.
//
// Use AddField or AddFields (opposed to Logrus' WithField)
// to add new fields to an entry.
//
// To add source file and line information call AddErrFields.
// Typically this value will be 2 for normal calls or 5 if you're in
// a panic recover.
//
// Call the Info, Warn, or Error methods to write the log entry.
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
	String() string
}

var std *logrus.Logger

func init() {
	std = logrus.StandardLogger()
}

// NewEntry creates a new log entry
func NewEntry() Entry {
	e := &entry{}
	e.entry = logrus.NewEntry(std)
	return e
}

// String returns the string representation from the reader and
// ultimately the formatter.
func (e entry) String() string {
	s, err := e.entry.String()
	if err != nil {
		return fmt.Sprintf("%s - <%s>", s, err)
	}
	return s
}

// AddField adds a single field to the Entry.
func (e *entry) AddField(key string, value interface{}) {
	e.entry = e.entry.WithField(key, value)
}

// AddFields adds a map of fields to the Entry.
func (e *entry) AddFields(fields map[string]interface{}) {
	logrusFields := logrus.Fields{}
	for k, v := range fields {
		logrusFields[k] = v
	}
	e.entry = e.entry.WithFields(logrusFields)
}

// AddCallstack adds the current callstack to the Entry using the key "callstack".
//
// Excludes runtime/proc.go, http/server.go, and files ending in .s from the callstack.
func (e *entry) AddCallstack() {
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

// Info logs the Entry with a status of "info"
func (e *entry) Info(args ...interface{}) {
	e.entry.Info(args...)
}

// Info logs the Entry with a status of "info"
func (e *entry) Infof(format string, args ...interface{}) {
	e.entry.Infof(format, args...)
}

// Warn logs the Entry with a status of "warn"
func (e *entry) Warn(args ...interface{}) {
	e.entry.Warn(args...)
}

// Warnf logs the Entry with a status of "warn"
func (e *entry) Warnf(format string, args ...interface{}) {
	e.entry.Warnf(format, args...)
}

// Error logs the Entry with a status of "error"
func (e *entry) Error(args ...interface{}) {
	e.entry.Error(args...)
}

// Errorf logs the Entry with a status of "error"
func (e *entry) Errorf(format string, args ...interface{}) {
	e.entry.Errorf(format, args...)
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
