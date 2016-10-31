// +build gofuzz

package httplog

func Fuzz(data []byte) int {
	str := string(data)
	acceptsGzip(str)
	return 0
}
