package fiberhttp

import (
	"regexp"

	"github.com/valyala/bytebufferpool"
)

var (
	bytes_pool  bytebufferpool.Pool
	contetre    *regexp.Regexp = regexp.MustCompile(`Content-Length: (\d+)`)
	code_regexp *regexp.Regexp = regexp.MustCompile(`HTTP/1.1 (\d{3})`)
	tow_lines   [4]byte        = [4]byte{13, 10, 13, 10}
	zero_lines  [5]byte        = [5]byte{48, 13, 10, 13, 10}
	proxybasic  [27]byte       = [27]byte{80, 114, 111, 120, 121, 45, 65, 117, 116, 104, 111, 114, 105, 122, 97, 116, 105, 111, 110, 58, 32, 66, 97, 115, 105, 99, 32}
)

var (
	status_code_regexp *regexp.Regexp = regexp.MustCompile(`HTTP/1.1 (\d{3})`)
	reason_regexp      *regexp.Regexp = regexp.MustCompile(`HTTP/1.1 (\d{3} .*)`)
)
