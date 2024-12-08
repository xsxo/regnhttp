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
	thebuffer     bytebufferpool.ByteBuffer
	decoder       *hpack.Decoder
	headers       []hpack.HeaderField
	contectLegnth int
}

type ResponseType struct {
	Header *headStruct
}

func (RES *ResponseType) Close() {
	RES.Header.thebuffer.Reset()
	bufferPool.Put(&RES.Header.thebuffer)
}

func Response() *ResponseType {
	return &ResponseType{Header: &headStruct{thebuffer: *bufferPool.Get()}}
}

func Http2Response() *ResponseType {
	toReturn := &ResponseType{Header: &headStruct{thebuffer: *bufferPool.Get(), headers: []hpack.HeaderField{}}}

	toReturn.Header.decoder = hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		toReturn.Header.headers = append(toReturn.Header.headers, f)
		if strings.EqualFold(f.Name, "content-length") {
			toReturn.Header.contectLegnth, _ = strconv.Atoi(f.Value)
		}
	})

	return toReturn
}

func (RES *ResponseType) Http2Upgrade() {
	RES.Header.headers = []hpack.HeaderField{}
	RES.Header.decoder = hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		RES.Header.headers = append(RES.Header.headers, f)
		if strings.EqualFold(f.Name, "content-length") {
			RES.Header.contectLegnth, _ = strconv.Atoi(f.Value)
		}
	})
}

func (RES *ResponseType) HttpDowngrade() {
	if RES.Header.decoder != nil {
		RES.Header.headers = nil
		RES.Header.decoder.Close()
		RES.Header.decoder = nil
	}
}

func (RES *ResponseType) StatusCode() int {
	if RES.Header.decoder != nil {
		if len(RES.Header.headers) == 0 {
			return 0
		} else if statusCode, err := strconv.Atoi(RES.Header.headers[0].Value); err == nil {
			return statusCode
		} else {
			return 0
		}
	} else {
		matches := statusRegex.FindSubmatch(RES.Header.thebuffer.B)
		if len(matches) < 2 {
			return 0
		} else if statusCode, err := strconv.Atoi(string(matches[1])); err != nil {
			matches[0] = nil
			matches[1] = nil
			return 0
		} else {
			matches[0] = nil
			matches[1] = nil
			return statusCode
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
		matches := reasonRegex.FindSubmatch(RES.Header.thebuffer.B)

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
		splied := bytes.SplitN(RES.Header.thebuffer.B, lines[1:], 2)

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

func (HEAD *headStruct) Get(name string) string {
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
