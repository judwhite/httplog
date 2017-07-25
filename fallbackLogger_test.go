package httplog

import (
	"fmt"
	"io"
	"testing"

	"github.com/pkg/errors"
)

func TestAddError(t *testing.T) {
	var got string
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() { logPrint = old }()

	const want = `[error] whoops err="EOF"`

	entry := fallbackLogger{}
	entry.AddError(io.EOF)
	entry.Error("whoops")

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func TestAddErrorWrapped(t *testing.T) {
	var got string
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() { logPrint = old }()

	const want = `[error] whoops err="unexpected eof: EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:c:11, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:b:10, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:a:9, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddErrorWrapped:41"`

	entry := fallbackLogger{}
	entry.AddError(a())
	entry.Error("whoops")

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func recoverPanic(f func()) (got string) {
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() {
		if perr := recover(); perr != nil {
			perror, ok := perr.(error)
			if !ok {
				perror = fmt.Errorf("%v", perr)
			}

			perror = errors.WithStack(perror)

			entry := fallbackLogger{}
			entry.AddError(perror)
			entry.Error("panic recover")
		}
		logPrint = old
	}()

	f()

	return
}

func TestPanic(t *testing.T) {
	const want = `[error] panic recover err="EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:recoverPanic.func2:59, ` +
		`runtime/panic.go:gopanic:489, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cPanic:15, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bPanic:14, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aPanic:13, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:recoverPanic:68, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestPanic:83"`

	got := recoverPanic(aPanic)

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func TestWrappedPanic(t *testing.T) {
	const want = `[error] panic recover err="unexpected eof: EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:recoverPanic.func2:59, ` +
		`runtime/panic.go:gopanic:489, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cPanicWrapped:19, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bPanicWrapped:18, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aPanicWrapped:17, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:recoverPanic:68, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestWrappedPanic:100"`

	got := recoverPanic(aPanicWrapped)

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func TestAddErrorAllWrapped(t *testing.T) {
	var got string
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() { logPrint = old }()

	const want = `[error] whoops err="aWrapped: bWrapped: unexpected eof: EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWrapped:23, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWrapped:22, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWrapped:21, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddErrorAllWrapped:120"`

	entry := fallbackLogger{}
	entry.AddError(aWrapped())
	entry.Error("whoops")

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func TestAddErrorSomeWrapped(t *testing.T) {
	var got string
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() { logPrint = old }()

	const want = `[error] whoops err="aWrapped: unexpected eof: EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWrapped2:27, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWrapped2:26, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWrapped2:25, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddErrorSomeWrapped:141"`

	entry := fallbackLogger{}
	entry.AddError(aWrapped2())
	entry.Error("whoops")

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func TestAddErrorWithStack(t *testing.T) {
	var got string
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() { logPrint = old }()

	const want = `[error] whoops err="EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWithStack:31, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWithStack:30, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWithStack:29, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddErrorWithStack:162"`

	entry := fallbackLogger{}
	entry.AddError(aWithStack())
	entry.Error("whoops")

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func TestAddAllWithStack(t *testing.T) {
	var got string
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() { logPrint = old }()

	const want = `[error] whoops err="EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWithStack2:35, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWithStack2:34, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWithStack2:33, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddAllWithStack:183"`

	entry := fallbackLogger{}
	entry.AddError(aWithStack2())
	entry.Error("whoops")

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func TestAddSomeWithStack(t *testing.T) {
	var got string
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() { logPrint = old }()

	const want = `[error] whoops err="EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWithStack3:39, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWithStack3:38, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWithStack3:37, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddSomeWithStack:204"`

	entry := fallbackLogger{}
	entry.AddError(aWithStack3())
	entry.Error("whoops")

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}
