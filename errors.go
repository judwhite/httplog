package httplog

import (
	"strings"

	"fmt"
	"strconv"

	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

type causer interface {
	Cause() error
}

// FilterStackTrace is called by the StackTrace function to filter frames.
// This variable can be set to a custom function.
var FilterStackTrace = func(filename string) bool {
	return strings.HasSuffix(filename, ".s") ||
		strings.HasPrefix(filename, "http/server.go") ||
		strings.HasPrefix(filename, "runtime/proc.go") ||
		filename == "testing/testing.go"
}

// Frame contains information about a stack trace frame.
type Frame interface {
	File() string
	Func() string
	Line() int
}

type frame struct {
	file     string
	funcname string
	line     int
}

func (f frame) File() string {
	return f.file
}

func (f frame) Func() string {
	return f.funcname
}

func (f frame) Line() int {
	return f.line
}

func getFrame(errorsFrame errors.Frame) Frame {
	if f, ok := interface{}(errorsFrame).(Frame); ok {
		return f
	}

	// support versions of pkg/errors which don't have File/Func/Line methods
	// basically parse it out of the text :/ waiting for https://github.com/pkg/errors/pull/100 to land

	// line number
	lineStr := fmt.Sprintf("%d", errorsFrame)
	line, err := strconv.Atoi(lineStr)
	if err != nil {
		line = 0
	}

	output := fmt.Sprintf("%+v", errorsFrame)

	parts := strings.Split(output, "\n")
	if len(parts) != 2 {
		return nil
	}

	// func name
	slashIndex := strings.LastIndex(parts[0], "/")
	funcname := parts[0][slashIndex+1:]
	dotIndex := strings.Index(funcname, ".")
	if dotIndex == -1 {
		return nil
	}
	funcname = funcname[dotIndex+1:]

	// file name
	file := strings.Trim(parts[1], "\n\r\t")
	srcIndex := strings.Index(file, "/src/")
	if srcIndex == -1 {
		return nil
	}
	file = file[srcIndex+5:]
	colonIndex := strings.LastIndex(file, ":")
	if colonIndex != -1 {
		file = file[:colonIndex]
	}

	return frame{file, funcname, line}
}

// StackTrace returns the stack frames of an error created by github.com/pkg/errors.
// This function calls httplog.FilterStackTrace to filter frames.
func StackTrace(err error) []Frame {
	if err == nil {
		return nil
	}

	var stackTrace []Frame
	if st, ok := err.(stackTracer); ok {
		for _, frame := range st.StackTrace() {
			f := getFrame(frame)
			if f != nil {
				filename := f.File()
				if FilterStackTrace(filename) {
					continue
				}
				stackTrace = append(stackTrace, f)
			}
		}
	}

	if cause, ok := err.(causer); ok {
		st := StackTrace(cause.Cause())

		if len(st) >= len(stackTrace) {
			// remove duplicate stack traces caused by multiple calls to Wrap/WithStack
			diff := false
			for i, j := len(st)-1, len(stackTrace)-1; j >= 0; {
				if st[i].File() != stackTrace[j].File() || st[i].Line() != stackTrace[j].Line() {
					diff = true
					break
				}
				i--
				j--
			}
			if diff {
				stackTrace = append(stackTrace, st...)
			} else {
				stackTrace = st
			}
		}
	}

	return stackTrace
}
