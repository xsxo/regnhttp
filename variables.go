package regn

import (
	"bufio"
	"io"
	"sync"
)

type RegnError struct {
	Message string
}

var (
	peekerPool  *sync.Pool = &sync.Pool{}
	flusherPool *sync.Pool = &sync.Pool{}

	contentLengthKey []byte = []byte("Content-Length: ")
	chunkedValue            = []byte(": chunked")
	lines            []byte = []byte("\r\n\r\n")
	line             []byte = []byte("\r\n")
	spaceByte        []byte = []byte(" ")
)

const (
	MethodGet     = "GET"
	MethodHead    = "HEAD"
	MethodPost    = "POST"
	MethodPut     = "PUT"
	MethodPatch   = "PATCH"
	MethodDelete  = "DELETE"
	MethodConnect = "CONNECT"
	MethodOptions = "OPTIONS"
	MethodTrace   = "TRACE"
)

func (e *RegnError) Error() string {
	return "regnhttp error: " + e.Message
}

func genPeeker(reader io.Reader, size int) *bufio.Reader {
	nr := peekerPool.Get()

	if nr == nil {
		return bufio.NewReaderSize(reader, size)
	}

	nrr, _ := nr.(*bufio.Reader)
	nrr.Reset(reader)
	return nrr
}

func genFlusher(writer io.Writer, size int) *bufio.Writer {
	nw := flusherPool.Get()

	if nw == nil {
		return bufio.NewWriterSize(writer, size)
	}

	nww, _ := nw.(*bufio.Writer)
	nww.Reset(writer)
	return nww
}

func BytesToInt(b []byte) int {
	n := 0
	for i := 0; i < len(b); i++ {
		n = n*10 + int(b[i]-'0')
	}
	return n
}

func IntToBytes(n int) []byte {
	if n == 0 {
		return []byte{'0'}
	}

	var buf [20]byte
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}

	return buf[i:]
}

func IntToString(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := [20]byte{}
	i := len(buf)

	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func StringToInt(s string) int {
	if len(s) == 0 {
		return 0
	}

	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}

func hexBytesToInt(b []byte) (int, bool) {
	n := 0
	for _, c := range b {
		n <<= 4
		switch {
		case c >= '0' && c <= '9':
			n |= int(c - '0')
		case c >= 'a' && c <= 'f':
			n |= int(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			n |= int(c - 'A' + 10)
		default:
			return 0, false
		}
	}
	return n, true
}
