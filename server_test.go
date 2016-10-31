package httplog

import (
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestAcceptsGzip(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		// some common Accept-Encoding values from https://gist.github.com/drwilco/aa44665c561453442fed
		// source article: https://www.fastly.com/blog/best-practices-for-using-the-vary-header
		{input: "             none", expected: false},
		{input: "             sdch", expected: false},
		{input: " gzip ", expected: true},
		{input: " gzip, deflate, sdch, br ", expected: true},
		{input: "", expected: false},
		{input: "FFFF, FFFFFFF", expected: false},
		{input: "FFFF,FFFFFFF,sdch", expected: false},
		{input: "deflate", expected: false},
		{input: "deflate, gzip", expected: true},
		{input: "deflate, gzip, x-gzip, identity, *;q=0", expected: true},
		{input: "deflate,gzip", expected: true},
		{input: "deflate,gzip,x-gzip,identity,*;q=0.001", expected: true},
		{input: "deflate;q=1.0, compress;q=0.5", expected: false},
		{input: "deflate;q=1.0, compress;q=0.5, gzip;q=0.5", expected: true},
		{input: "gzip", expected: true},
		{input: "gzip, deflate", expected: true},
		{input: "gzip, deflate, base64, quoted-printable", expected: true},
		{input: "gzip, deflate, br", expected: true},
		{input: "gzip, deflate, compress", expected: true},
		{input: "gzip, deflate, identity", expected: true},
		{input: "gzip, deflate, peerdist", expected: true},
		{input: "gzip, deflate, sdch, br", expected: true},
		{input: "gzip, deflate, sdch, identity", expected: true},
		{input: "gzip, deflate, x-gzip, identity; q=0.9", expected: true},
		{input: "gzip, deflate, x-gzip, x-deflate", expected: true},
		{input: "gzip, identity", expected: true},
		{input: "gzip, x-gzip, deflate, x-deflate, identity", expected: true},
		{input: "gzip,defalte", expected: true},
		{input: "gzip,deflate", expected: true},
		{input: "gzip,deflate,", expected: true},
		{input: "gzip,deflate,diff", expected: true},
		{input: "gzip,deflate,gzip", expected: true},
		{input: "gzip,deflate,identity", expected: true},
		{input: "gzip,deflate,identity,*;q=0.001", expected: true},
		{input: "gzip,deflate,lzma", expected: true},
		{input: "gzip,deflate,lzma,sdch", expected: true},
		{input: "gzip,deflate,sdch", expected: true},
		{input: "gzip; q=0", expected: false},
		{input: "gzip; q=0, deflate, sdch, br ", expected: false},
		{input: "gzip; q=0.0", expected: false},
		{input: "gzip; q=0.0, deflate, sdch, br ", expected: false},
		{input: "gzip; q=0.000, deflate, sdch, br ", expected: false},
		{input: "gzip; q=0.001, deflate, sdch, br ", expected: true},
		{input: "gzip; q=0.8", expected: true},
		{input: "gzip; q=0.8, deflate, sdch, br ", expected: true},
		{input: "gzip; q=1", expected: true},
		{input: "gzip; q=1, deflate, sdch, br ", expected: true},
		{input: "gzip;q=0", expected: false},
		{input: "gzip;q=0, deflate, sdch, br ", expected: false},
		{input: "gzip;q=0.0", expected: false},
		{input: "gzip;q=0.0, deflate, sdch, br ", expected: false},
		{input: "gzip;q=0.8", expected: true},
		{input: "gzip;q=0.8, deflate, sdch, br ", expected: true},
		{input: "gzip;q=1", expected: true},
		{input: "gzip;q=1, deflate, sdch, br ", expected: true},
		{input: "gzip;q=1.0, *;q=0.5", expected: true},
		{input: "gzip;q=1.0, deflate;q=0.8, chunked;q=0.6", expected: true},
		{input: "gzip;q=1.0,deflate;q=0.6,identity;q=0.3", expected: true},
		{input: "gzippy", expected: false},
		{input: "gzippy, deflate", expected: false},
		{input: "identity", expected: false},
		{input: "identity, deflate, gzip", expected: true},
		{input: "identity, gzip, deflate", expected: true},
		{input: "identity, gzip;, deflate", expected: false},
		{input: "identity, gzipp, deflate", expected: false},
		{input: "identity, gzipp;, deflate", expected: false},
		{input: "identity, gzip;q=0", expected: false},
		{input: "identity, gzip;q=0, deflate", expected: false},
		{input: "identity, gzip;q=0.0", expected: false},
		{input: "identity, gzip;q=0.0, deflate", expected: false},
		{input: "identity, gzip;q=0.8", expected: true},
		{input: "identity, gzip;q=0.8, deflate", expected: true},
		{input: "identity, gzip;q=1", expected: true},
		{input: "identity, gzip;q=1, deflate", expected: true},
		{input: "identity,gzip", expected: true},
		{input: "identity,gzip;", expected: false},
		{input: "identity,gzip;q", expected: false},
		{input: "identity,gzip;q=", expected: false},
		{input: "identity,gzip;q=0.001", expected: true},
		{input: "identity,gzip;q=0.000", expected: false},
		{input: "identity,gzip,deflate", expected: true},
		{input: "identity,gzip; q=0,deflate", expected: false},
		{input: "identity,gzip; q=1,deflate", expected: true},
		{input: "identity,gzip;q=0,deflate", expected: false},
		{input: "identity,gzip;q=1,deflate", expected: true},
		{input: "identity;q=1, *;q=0", expected: false},
		{input: "sdch", expected: false},
		{input: "x-gzip, gzip, deflate", expected: true},
		{input: "gzips   ,", expected: false},
		{input: "gzip   ,", expected: true},
		{input: "gzip;Q=1", expected: false},
		{input: "gzip;q:1", expected: false},
		{input: "gzi", expected: false},
		{input: "gzi,", expected: false},
		{input: "gzi,br", expected: false},
		{input: "gzi,br,", expected: false},
		{input: "gzi,br;q=1,", expected: false},
		{input: "br;q=1,gzip", expected: true},
		{input: "br;q=1,gzip;q=0", expected: false},
		{input: "gzip          s", expected: false},
	}

	dir := "fuzz/corpus"

	if err := os.MkdirAll(dir, 0666); err != nil {
		t.Fatal(err)
	}

	illegalChars := []byte{'?', '%', '*', ':', '|', '\\', '"', ' '}

	for _, c := range cases {
		h := crc32.NewIEEE()
		if _, err := h.Write([]byte(c.input)); err != nil {
			t.Fatal(err)
		}
		filename := []byte(c.input)
		if len(filename) != 0 {
			for i := 0; i < len(filename); i++ {
				for j := 0; j < len(illegalChars); j++ {
					if filename[i] == illegalChars[j] || filename[i] < 32 || filename[i] > 127 {
						filename[i] = '_'
					}
				}
			}
			fullPath := path.Join(dir, string(filename)) + fmt.Sprintf("-%x.txt", h.Sum32())

			if err := ioutil.WriteFile(fullPath, []byte(c.input), 0666); err != nil {
				t.Log(err)
			}
		}

		actual := acceptsGzip(c.input)
		if actual != c.expected {
			t.Errorf("input: %q want: %v got: %v", c.input, c.expected, actual)
		}
	}
}

func BenchmarkAcceptsGzip_Empty(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		acceptsGzip("")
	}
}

func BenchmarkAcceptsGzip_JustGzip(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		acceptsGzip("gzip")
	}
}

func BenchmarkAcceptsGzip_GzipPrefix(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		acceptsGzip("gzip, deflate")
	}
}

func BenchmarkAcceptsGzip_GzipSuffix(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		acceptsGzip("deflate, gzip")
	}
}

func BenchmarkAcceptsGzip_GzipMiddle(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		acceptsGzip("deflate, gzip, br")
	}
}

func BenchmarkAcceptsGzip_GzipMiddleWithQ(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		acceptsGzip("identity,gzip;q=1,deflate")
	}
}
