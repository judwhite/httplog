package httplog

import (
	"io"

	"github.com/pkg/errors"
)

func a() error { return b() }
func b() error { return c() }
func c() error { return errors.Wrap(io.EOF, "unexpected eof") }

func aPanic() { bPanic() }
func bPanic() { cPanic() }
func cPanic() { panic(io.EOF) }

func aPanicWrapped() { bPanicWrapped() }
func bPanicWrapped() { cPanicWrapped() }
func cPanicWrapped() { panic(errors.Wrap(io.EOF, "unexpected eof")) }

func aWrapped() error { return errors.Wrap(bWrapped(), "aWrapped") }
func bWrapped() error { return errors.Wrap(cWrapped(), "bWrapped") }
func cWrapped() error { return errors.Wrap(io.EOF, "unexpected eof") }

func aWrapped2() error { return errors.Wrap(bWrapped2(), "aWrapped") }
func bWrapped2() error { return cWrapped2() }
func cWrapped2() error { return errors.Wrap(io.EOF, "unexpected eof") }

func aWithStack() error { return bWithStack() }
func bWithStack() error { return cWithStack() }
func cWithStack() error { return errors.WithStack(io.EOF) }

func aWithStack2() error { return errors.WithStack(bWithStack2()) }
func bWithStack2() error { return errors.WithStack(cWithStack2()) }
func cWithStack2() error { return errors.WithStack(io.EOF) }

func aWithStack3() error { return errors.WithStack(bWithStack3()) }
func bWithStack3() error { return cWithStack3() }
func cWithStack3() error { return errors.WithStack(io.EOF) }
