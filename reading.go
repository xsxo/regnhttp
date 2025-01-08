package regn

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/valyala/bytebufferpool"
	"golang.org/x/net/http2/hpack"
)

type headStruct struct {
	theBuffer bytebufferpool.ByteBuffer
	theHeader bytebufferpool.ByteBuffer
	upgraded  bool
}

type ResponseType struct {
	Header *headStruct
}

func (RES *ResponseType) Close() {
	RES.Header.theBuffer.Reset()
	bufferPool.Put(&RES.Header.theBuffer)
}

func Response() *ResponseType {
	return &ResponseType{Header: &headStruct{theBuffer: *bufferPool.Get()}}
}

func Http2Response() *ResponseType {
	toReturn := &ResponseType{Header: &headStruct{theBuffer: *bufferPool.Get(), theHeader: *bufferPool.Get(), upgraded: true}}

	return toReturn
}

func (RES *ResponseType) Http2Upgrade() {
	if !RES.Header.upgraded {
		RES.Header.upgraded = true
		RES.Header.theHeader = *bufferPool.Get()
	}
}

func (RES *ResponseType) HttpDowngrade() {
	if RES.Header.upgraded {
		RES.Header.upgraded = false
		bufferPool.Put(&RES.Header.theHeader)
	}
}

func (RES *ResponseType) StatusCode() int {
	if RES.Header.upgraded {
		code := RES.Header.Get(":status")
		if code == "" {
			return 0
		} else if statusCode, err := strconv.Atoi(code); err == nil {
			return statusCode
		}
	} else {
		matches := statusRegex.FindSubmatch(RES.Header.theBuffer.B)
		if len(matches) < 2 {
			return 0
		} else if statusCode, err := strconv.Atoi(string(matches[1])); err == nil {
			matches[0] = nil
			matches[1] = nil
			return statusCode
		}
	}

	return 0
}

func (RES *ResponseType) Reason() string {
	if RES.Header.upgraded {
		return RES.Header.Get(":status")
	} else {
		matches := reasonRegex.FindSubmatch(RES.Header.theBuffer.B)

		if len(matches) < 2 {
			return ""
		}

		toReturn := string(matches[1])

		matches[0] = nil
		matches[1] = nil
		return toReturn
	}
}

func (RES *ResponseType) BodyString() string {
	if RES.Header.upgraded {
		return RES.Header.theBuffer.String()
	} else {
		out := strings.SplitN(RES.Header.theBuffer.String(), "\r\n\r\n", 2)
		if len(out) < 2 {
			return ""
		}

		out[0] = ""
		return out[1]
	}
}

func (RES *ResponseType) Body() []byte {
	if RES.Header.upgraded {
		return RES.Header.theBuffer.B
	} else {
		splied := bytes.SplitN(RES.Header.theBuffer.B, lines[1:], 2)

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
		NewErr.Message = "field decode body to json format"
		return result, NewErr
	}

	return result, nil
}

func (HEAD *headStruct) GetAll() map[string]string {
	forReturn := make(map[string]string)

	if HEAD.upgraded {
		headers := []hpack.HeaderField{}
		decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
			headers = append(headers, f)
		})
		decoder.Write(HEAD.theHeader.B)

		for _, h := range headers {
			forReturn[h.Name] = h.Value
		}
	} else {
		forNothing := strings.Split(HEAD.theBuffer.String(), "\n")[1:]

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

func (HEAD *headStruct) Get(name string) string {
	if HEAD.upgraded {
		headers := []hpack.HeaderField{}
		decoder := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
			headers = append(headers, f)
		})
		decoder.Write(HEAD.theHeader.B)

		for _, h := range headers {
			if strings.EqualFold(name, h.Name) {
				return h.Value
			}
		}
	} else {
		forNothing := strings.Split(HEAD.theBuffer.String(), "\n")[1:]
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
