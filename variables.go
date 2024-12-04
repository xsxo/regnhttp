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
	bytes_pool bytebufferpool.Pool
	nwpool     *sync.Pool = &sync.Pool{}
	nrpool     *sync.Pool = &sync.Pool{}

	status_code_regexp *regexp.Regexp = regexp.MustCompile(`HTTP/1.1 (\d{3})`)
	reason_regexp      *regexp.Regexp = regexp.MustCompile(`HTTP/1.1 (\d{3} .*)`)

	contetre *regexp.Regexp = regexp.MustCompile(`Content-Length: (\d+)`)

	// delete all this var's and keep zero_lines -> the_lines
	tow_lines  []byte = []byte{13, 10, 13, 10}
	zero_lines []byte = []byte{48, 13, 10, 13, 10}
	one_line   []byte = []byte{13, 10}
	space_line []byte = []byte{32}
)

const (
	MethodPost    string = "POST"
	MethodGet     string = "GET"
	MethodPut     string = "PUT"
	MethodConnect string = "CONNECT"
	MethodOptions string = "OPTIONS"
	MethodTrace   string = "TRACE"
)

func get_reader(Conn net.Conn) *bufio.Reader {
	nr := nrpool.Get()

	if nr == nil {
		return bufio.NewReader(Conn)
	}

	nrr, _ := nr.(*bufio.Reader)
	nrr.Reset(Conn)
	return nrr
}

func get_writer(Conn net.Conn) *bufio.Writer {
	nw := nwpool.Get()

	if nw == nil {
		return bufio.NewWriter(Conn)
	}

	nww, _ := nw.(*bufio.Writer)
	nww.Reset(Conn)
	return nww
}
