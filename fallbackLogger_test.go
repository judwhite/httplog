package httplog

import (
	"fmt"
	"io"
	"testing"
)

func TestAddError(t *testing.T) {
	var got string
	old := logPrint
	logPrint = func(v ...interface{}) { got = fmt.Sprint(v...) }
	defer func() { logPrint = old }()

	const want = `[error] whoops err="EOF" stacktrace="github.com/judwhite/httplog/fallbackLogger_test.go:TestAddError:18"`

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
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:c:9, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:b:8, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:a:7, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddErrorWrapped:39"`

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

			entry := fallbackLogger{}
			entry.AddError(withStack(perror))
			entry.Error("panic recover")
		}
		logPrint = old
	}()

	f()

	return
}

func TestPanic(t *testing.T) {
	const want = `[error] panic recover err="EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:recoverPanic.func2:58, ` +
		`runtime/panic.go:gopanic:489, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cPanic:13, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bPanic:12, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aPanic:11, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:recoverPanic:64, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestPanic:79"`

	got := recoverPanic(aPanic)

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}

func TestWrappedPanic(t *testing.T) {
	const want = `[error] panic recover err="unexpected eof: EOF" stacktrace="` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:recoverPanic.func2:58, ` +
		`runtime/panic.go:gopanic:489, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cPanicWrapped:17, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bPanicWrapped:16, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aPanicWrapped:15, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:recoverPanic:64, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestWrappedPanic:96"`

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
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWrapped:21, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWrapped:20, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWrapped:19, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddErrorAllWrapped:116"`

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
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWrapped2:25, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWrapped2:24, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWrapped2:23, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddErrorSomeWrapped:137"`

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
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWithStack:29, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWithStack:28, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWithStack:27, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddErrorWithStack:158"`

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
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWithStack2:33, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWithStack2:32, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWithStack2:31, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddAllWithStack:179"`

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
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:cWithStack3:37, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:bWithStack3:36, ` +
		`github.com/judwhite/httplog/fallbackLogger_helpers_test.go:aWithStack3:35, ` +
		`github.com/judwhite/httplog/fallbackLogger_test.go:TestAddSomeWithStack:200"`

	entry := fallbackLogger{}
	entry.AddError(aWithStack3())
	entry.Error("whoops")

	if want != got {
		t.Errorf("\nwant:\n\t%s\ngot:\n\t%s", want, got)
	}
}
