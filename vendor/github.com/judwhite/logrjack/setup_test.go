package logrjack

import (
	"path/filepath"
	"strings"
	"testing"
)

type testData struct {
	input    string
	expected string
	actual   string
}

func TestGetDefaultLogFilename(t *testing.T) {
	// arrange
	testData := []testData{
		{input: `c:\apps\server\run.exe`, expected: `c:\apps\server\logs\run.log`},
		{input: `c:\apps\server\run`, expected: `c:\apps\server\logs\run.log`},
		{input: `\\home\apps\server\run`, expected: `\\home\apps\server\logs\run.log`},
	}

	if filepath.Separator == '/' {
		testData = testData[0 : len(testData)-1] // remove UNC path test
		for i := 0; i < len(testData); i++ {
			// remove drive: and replace \ with /
			testData[i].input = strings.Replace(testData[i].input[2:], `\`, `/`, -1)
			testData[i].expected = strings.Replace(testData[i].expected[2:], `\`, `/`, -1)
		}
	}

	// act
	for i := 0; i < len(testData); i++ {
		testData[i].actual = getDefaultLogFilename(testData[i].input)
	}

	// assert
	for i := 0; i < len(testData); i++ {
		if testData[i].actual != testData[i].expected {
			t.Fatalf("idx: %d actual: %s expected: %s", i, testData[i].actual, testData[i].expected)
		}
	}
}
