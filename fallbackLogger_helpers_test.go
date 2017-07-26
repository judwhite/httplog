package httplog

import (
	"io"
)

func a() error { return b() }
func b() error { return c() }
func c() error { return wrap(io.EOF, "unexpected eof") }

func aPanic() { bPanic() }
func bPanic() { cPanic() }
func cPanic() { panic(io.EOF) }

func aPanicWrapped() { bPanicWrapped() }
func bPanicWrapped() { cPanicWrapped() }
func cPanicWrapped() { panic(wrap(io.EOF, "unexpected eof")) }

func aWrapped() error { return wrap(bWrapped(), "aWrapped") }
func bWrapped() error { return wrap(cWrapped(), "bWrapped") }
func cWrapped() error { return wrap(io.EOF, "unexpected eof") }

func aWrapped2() error { return wrap(bWrapped2(), "aWrapped") }
func bWrapped2() error { return cWrapped2() }
func cWrapped2() error { return wrap(io.EOF, "unexpected eof") }

func aWithStack() error { return bWithStack() }
func bWithStack() error { return cWithStack() }
func cWithStack() error { return withStack(io.EOF) }

func aWithStack2() error { return withStack(bWithStack2()) }
func bWithStack2() error { return withStack(cWithStack2()) }
func cWithStack2() error { return withStack(io.EOF) }

func aWithStack3() error { return withStack(bWithStack3()) }
func bWithStack3() error { return cWithStack3() }
func cWithStack3() error { return withStack(io.EOF) }
