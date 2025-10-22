package regn

import (
	"bufio"
	"io"
	"regexp"
	"sync"

	"github.com/valyala/bytebufferpool"
)

type RegnError struct {
	Message string
}

var (
	bufferPool  bytebufferpool.Pool
	peekerPool  *sync.Pool = &sync.Pool{}
	flusherPool *sync.Pool = &sync.Pool{}

	statusRegex *regexp.Regexp = regexp.MustCompile(`HTTP/1.1 (\d{3})`)
	reasonRegex *regexp.Regexp = regexp.MustCompile(`HTTP/1.1 (\d{3} .*)`)

	contentLengthKey []byte = []byte{67, 111, 110, 116, 101, 110, 116, 45, 76, 101, 110, 103, 116, 104, 58, 32}

	lines     []byte = []byte{13, 10, 48, 13, 10, 13, 10}
	SpaceByte []byte = []byte(" ")
)

const (
	MethodPost    string = "POST"
	MethodGet     string = "GET"
	MethodPut     string = "PUT"
	MethodConnect string = "CONNECT"
	MethodOptions string = "OPTIONS"
	MethodTrace   string = "TRACE"
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

func formatInt(n int) string {
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

func intToBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

func BToInt(b []byte) int {
	n := 0
	for i := 0; i < len(b); i++ {
		n = n*10 + int(b[i]-'0')
	}
	return n
}
