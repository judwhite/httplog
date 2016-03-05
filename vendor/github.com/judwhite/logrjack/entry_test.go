package logrjack

import (
	"strings"
	"testing"
)

func TestAddCallstack(t *testing.T) {
	// arrange
	e := NewEntry()

	// act
	e.AddCallstack()

	// assert
	actual := e.String()
	if !strings.Contains(actual, "callstack=\"logrjack/entry_test.go:13, ") {
		t.Fatalf("could not find expected callstack string: %s", actual)
	}
}
