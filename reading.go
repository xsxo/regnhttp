package regn

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/valyala/bytebufferpool"
	"golang.org/x/net/http2/hpack"
)

type headers_struct struct {
	thebuffer bytebufferpool.ByteBuffer
	decoder   *hpack.Decoder
	headers   []hpack.HeaderField
}

type ResponseType struct {
	Header *headers_struct
}

func Response() *ResponseType {
	return &ResponseType{Header: &headers_struct{thebuffer: *bytes_pool.Get()}}
}

// func Http2Response() *ResponseType {
// 	to_return := &ResponseType{Header: &headers_struct{thebuffer: *bytes_pool.Get(), headers: []hpack.HeaderField{}}}

// 	to_return.Header.decoder = hpack.NewDecoder(4096, func(f hpack.HeaderField) {
// 		to_return.Header.headers = append(to_return.Header.headers, f)
// 	})

// 	return to_return
// }

func (RES *ResponseType) Close() {
	RES.Header.thebuffer.Reset()
	bytes_pool.Put(&RES.Header.thebuffer)
}

// func (RES *ResponseType) upgradeH2c() {
// 	RES.Header.headers = []hpack.HeaderField{}
// 	RES.Header.decoder = hpack.NewDecoder(4096, func(f hpack.HeaderField) {
// 		RES.Header.headers = append(RES.Header.headers, f)
// 	})
// }

func (RES *ResponseType) StatusCode() int {
	if RES.Header.decoder != nil {
		if len(RES.Header.headers) == 0 {
			return 0
		} else if stauts_code, err := strconv.Atoi(RES.Header.headers[0].Value); err == nil {
			return stauts_code
		} else {
			return 0
		}
	} else {
		matches := status_code_regexp.FindSubmatch(RES.Header.thebuffer.B)
		if len(matches) < 2 {
			return 0
		} else if status_code, err := strconv.Atoi(string(matches[1])); err != nil {
			matches[0] = nil
			matches[1] = nil
			return 0
		} else {
			matches[0] = nil
			matches[1] = nil
			return status_code
		}
	}
}

func (RES *ResponseType) Reason() string {
	if RES.Header.decoder != nil {
		if len(RES.Header.headers) == 0 {
			return ""
		} else {
			return RES.Header.headers[0].Value
		}
	} else {
		matches := reason_regexp.FindSubmatch(RES.Header.thebuffer.B)

		if len(matches) < 2 {
			return ""
		}

		to_return := string(matches[1])

		matches[0] = nil
		matches[1] = nil
		return to_return
	}
}

func (RES *ResponseType) BodyString() string {
	if RES.Header.decoder != nil {
		return RES.Header.thebuffer.String()
	} else {
		out := strings.SplitN(RES.Header.thebuffer.String(), "\r\n\r\n", 2)
		if len(out) < 2 {
			return ""
		}

		out[0] = ""
		return out[1]
	}
}

func (RES *ResponseType) Body() []byte {
	if RES.Header.decoder != nil {
		return RES.Header.thebuffer.B
	} else {
		splied := bytes.SplitN(RES.Header.thebuffer.B, tow_lines, 2)

		if len(splied) < 2 {
			return nil
		}

		splied[0] = nil
		return splied[1]
	}
}

func (RES *ResponseType) Json() (map[string]interface{}, error) {
	NewErr := &RegnError{}

	var result map[string]interface{}
	err := json.Unmarshal(RES.Body(), &result)

	if err != nil {
		NewErr.Message = "Field decode body to json format"
		return result, NewErr
	}

	return result, nil
}

func (HEAD *headers_struct) GetAll() map[string]string {
	forReturn := make(map[string]string)

	if HEAD.decoder != nil {
		for _, h := range HEAD.headers {
			forReturn[h.Name] = h.Value
		}
	} else {
		forNothing := strings.Split(HEAD.thebuffer.String(), "\n")[1:]

		for _, res := range forNothing {
			if !strings.Contains(res, ": ") {
				break
			}
			parts := strings.SplitN(res, ": ", 2)
			if len(parts) >= 2 {
				name := parts[0]
				value := strings.TrimSpace(parts[1])
				forReturn[name] = value
			}
		}
	}

	return forReturn
}

func (HEAD *headers_struct) Get(name string) string {
	forNothing := strings.Split(HEAD.thebuffer.String(), "\n")[1:]

	if HEAD.decoder != nil {
		for _, h := range HEAD.headers {
			if strings.EqualFold(name, h.Name) {
				return h.Value
			}
		}
	} else {
		for _, res := range forNothing {
			if !strings.Contains(res, ": ") {
				break
			}
			parts := strings.SplitN(res, ": ", 2)
			if len(parts) >= 2 {
				if strings.EqualFold(parts[0], name) {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	return ""
}
