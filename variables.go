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

	lenRegex *regexp.Regexp = regexp.MustCompile(`Content-Length: (\d+)`)

	lines     []byte = []byte{48, 13, 10, 13, 10}
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

func genPeeker(reader io.Reader) *bufio.Reader {
	nr := peekerPool.Get()

	if nr == nil {
		return bufio.NewReader(reader)
	}

	nrr, _ := nr.(*bufio.Reader)
	nrr.Reset(reader)
	return nrr
}

func genFlusher(writer io.Writer) *bufio.Writer {
	nw := flusherPool.Get()

	if nw == nil {
		return bufio.NewWriter(writer)
	}

	nww, _ := nw.(*bufio.Writer)
	nww.Reset(writer)
	return nww
}
