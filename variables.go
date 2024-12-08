package regn

import (
	"bufio"
	"net"
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

func genPeeker(Conn net.Conn) *bufio.Reader {
	nr := peekerPool.Get()

	if nr == nil {
		return bufio.NewReader(Conn)
	}

	nrr, _ := nr.(*bufio.Reader)
	nrr.Reset(Conn)
	return nrr
}

func genFlusher(Conn net.Conn) *bufio.Writer {
	nw := flusherPool.Get()

	if nw == nil {
		return bufio.NewWriter(Conn)
	}

	nww, _ := nw.(*bufio.Writer)
	nww.Reset(Conn)
	return nww
}
