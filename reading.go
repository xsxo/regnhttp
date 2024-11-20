package regn

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/valyala/bytebufferpool"
)

type headers_struct struct {
	thebuffer bytebufferpool.ByteBuffer
	// *thebuffer bytebufferpool.ByteBuffer
}

type ResponseType struct {
	Header *headers_struct
}

func Response() *ResponseType {
	return &ResponseType{Header: &headers_struct{*bytes_pool.Get()}}
}

func (RES *ResponseType) Close() {
	RES.Header.thebuffer.Reset()
	bytes_pool.Put(&RES.Header.thebuffer)
}

func (RES *ResponseType) StatusCode() int {
	matches := status_code_regexp.FindSubmatch(RES.Header.thebuffer.B)
	status_code, _ := strconv.Atoi(string(matches[1]))

	matches[0] = nil
	matches[1] = nil
	return status_code
}

func (RES *ResponseType) Reason() string {
	matches := reason_regexp.FindSubmatch(RES.Header.thebuffer.B)
	to_return := string(matches[1])

	matches[0] = nil
	matches[1] = nil

	return to_return
}

func (RES *ResponseType) BodyString() string {
	out := strings.SplitN(RES.Header.thebuffer.String(), "\r\n\r\n", 2)
	if len(out) < 1 {
		return ""
	}

	out[0] = ""
	return out[1]
}

func (RES *ResponseType) Body() []byte {
	splied := bytes.SplitN(RES.Header.thebuffer.B, tow_lines, 2)

	if len(splied) > 1 {
		return nil
	}

	splied[0] = nil
	return splied[1]
}

func (RES *ResponseType) Json() (map[string]interface{}, error) {
	NewErr := &RegnError{}

	var result map[string]interface{}
	err := json.Unmarshal([]byte(string(bytes.SplitN(RES.Header.thebuffer.B, tow_lines, 2)[1])), &result)

	if err != nil {
		NewErr.Message = "Field decode body to json format"
		return result, NewErr
	}

	return result, nil
}

func (HEAD *headers_struct) GetAll() map[string]string {
	forReturn := make(map[string]string)
	forNothing := strings.Split(HEAD.thebuffer.String(), "\n")[1:]

	for _, res := range forNothing {
		if !strings.Contains(res, ": ") {
			break
		}
		parts := strings.SplitN(res, ": ", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := strings.TrimSpace(parts[1])
			forReturn[key] = value
		}
	}

	return forReturn
}

func (HEAD *headers_struct) Get(key string) string {
	forNothing := strings.Split(HEAD.thebuffer.String(), "\n")[1:]

	for _, res := range forNothing {
		if !strings.Contains(res, ": ") {
			break
		}
		parts := strings.SplitN(res, ": ", 2)
		if len(parts) == 2 {
			if parts[0] == key {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
