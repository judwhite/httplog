package httplog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestHandler(t *testing.T) {
	// arrange
	jsonStruct := struct {
		Field1 string `json:"field1"`
		Field2 bool   `json:"field2"`
	}{
		Field1: strings.Repeat("a", 1000),
		Field2: true,
	}

	uncompressedJSONBytes, err := json.Marshal(jsonStruct)
	if err != nil {
		t.Fatal(err)
	}

	jsonString := string(uncompressedJSONBytes)

	compressedJSONBytes := []byte{0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0xaa, 0x56, 0x4a, 0xcb, 0x4c, 0xcd,
		0x49, 0x31, 0x54, 0xb2, 0x52, 0x4a, 0x1c, 0x5, 0xa3, 0x60, 0x14, 0xc, 0x7b, 0xa0, 0xa4, 0x3, 0xc9, 0xf3, 0x46,
		0x4a, 0x56, 0x25, 0x45, 0xa5, 0xa9, 0xb5, 0x80, 0x0, 0x0, 0x0, 0xff, 0xff, 0x60, 0x63, 0x6d, 0xf6, 0x3, 0x4,
		0x0, 0x0}

	type clientCase struct {
		AcceptEncoding          string
		ExpectedBody            []byte
		ExpectedContentEncoding string
	}

	serverCases := []struct {
		Body        interface{}
		MIMEType    string
		ClientCases []clientCase
	}{
		{jsonStruct, "", []clientCase{
			{"gzip", compressedJSONBytes, "gzip"},
			{"", uncompressedJSONBytes, ""},
		}},
		{jsonString, "", []clientCase{
			{"gzip", compressedJSONBytes, "gzip"},
			{"", uncompressedJSONBytes, ""},
		}},
		{uncompressedJSONBytes, "application/json", []clientCase{
			{"gzip", compressedJSONBytes, "gzip"},
			{"", uncompressedJSONBytes, ""},
		}},
		{uncompressedJSONBytes, "", []clientCase{
			{"gzip", uncompressedJSONBytes, ""},
			{"", uncompressedJSONBytes, ""},
		}},
		{compressedJSONBytes, "application/json", []clientCase{
			{"gzip", compressedJSONBytes, "gzip"},
			{"", uncompressedJSONBytes, ""},
		}},
		{compressedJSONBytes, "", []clientCase{
			{"gzip", compressedJSONBytes, "gzip"},
			{"", uncompressedJSONBytes, ""},
		}},
	}

	// arrange http test server
	var returnBody interface{}
	var returnMIMEType string

	var s Server
	s.NewLogEntry = func() Entry { return &nullLogger{} }
	defer s.Shutdown()

	handler := Handler{Name: "test", Func: func(_ *http.Request, _ Entry) (Response, error) {
		return Response{Body: returnBody, Headers: []Header{{"Content-Type", returnMIMEType}}}, nil
	}}
	handlerFunc := s.Handle(handler)

	ts := httptest.NewServer(http.HandlerFunc(handlerFunc))
	defer ts.Close()

	for i, sc := range serverCases {
		returnBody = sc.Body
		returnMIMEType = sc.MIMEType

		var returnDescription string
		if reflect.DeepEqual(sc.Body, jsonStruct) {
			returnDescription = "json-struct"
		} else if reflect.DeepEqual(sc.Body, jsonString) {
			returnDescription = "json-string"
		} else if reflect.DeepEqual(sc.Body, uncompressedJSONBytes) {
			returnDescription = "uncompressed-bytes"
		} else if reflect.DeepEqual(sc.Body, compressedJSONBytes) {
			returnDescription = "compressed-bytes"
		}

		serverTestName := fmt.Sprintf("return:%q/mime-type:%q/i:%d", returnDescription, sc.MIMEType, i)

		for j, cc := range sc.ClientCases {
			clientTestName := fmt.Sprintf("client-accept-encoding:%q/j:%d", cc.AcceptEncoding, j)
			req, err := http.NewRequest("GET", ts.URL, nil)
			if err != nil {
				t.Error(err)
				return
			}

			req.Header.Set("Accept-Encoding", cc.AcceptEncoding)

			// act
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Error(err)
				return
			}
			defer resp.Body.Close()

			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Error(err)
				return
			}

			// assert
			if !bytes.Equal(b, cc.ExpectedBody) {
				dataToString := func(d []byte) interface{} {
					if bytes.Equal(d, compressedJSONBytes) {
						return "compressed"
					} else if bytes.Equal(d, uncompressedJSONBytes) {
						return "uncompressed"
					}
					return d
				}

				got := dataToString(b)
				want := dataToString(cc.ExpectedBody)

				t.Errorf("\n\t%s %s:\n\tBody want: %v\n\t     got:  %v", serverTestName, clientTestName, want, got)
			}

			if resp.Header.Get("Content-Encoding") != cc.ExpectedContentEncoding {
				t.Errorf("\n\t%s %s:\n\tContent-Encoding want: %v\n\t                 got:  %v",
					serverTestName,
					clientTestName,
					cc.ExpectedContentEncoding,
					resp.Header.Get("Content-Encoding"))
			}
		}
	}
}

type nullLogger struct{}

func (*nullLogger) AddField(key string, value interface{})          {}
func (*nullLogger) AddFields(fields map[string]interface{})         {}
func (*nullLogger) AddError(err error)                              {}
func (*nullLogger) Info(args ...interface{})                        {}
func (*nullLogger) Infof(format string, args ...interface{})        {}
func (*nullLogger) Warn(args ...interface{})                        {}
func (*nullLogger) Warnf(format string, args ...interface{})        {}
func (*nullLogger) Error(args ...interface{})                       {}
func (*nullLogger) Errorf(format string, args ...interface{})       {}
func (*nullLogger) Write(level, format string, args ...interface{}) {}
