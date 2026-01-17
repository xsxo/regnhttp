package regn

import (
	"bytes"
	"strings"

	"github.com/valyala/bytebufferpool"
)

type headStruct struct {
	theBuffer bytebufferpool.ByteBuffer
}

type ResponseType struct {
	Header *headStruct
}

func (RES *ResponseType) Close() {
	RES.Header.theBuffer.Reset()
	bufferPool.Put(&RES.Header.theBuffer)
}

func (RES *ResponseType) Reset() {
	RES.Header.theBuffer.Reset()
}

func Response() *ResponseType {
	return &ResponseType{Header: &headStruct{theBuffer: *bufferPool.Get()}}
}

func (RES *ResponseType) StatusCode() []byte {
	index1 := bytes.IndexByte(RES.Header.theBuffer.B, ' ')
	if index1 == -1 {
		return nil
	}

	index2 := index1 + 1 + bytes.IndexByte(RES.Header.theBuffer.B[index1+1:], ' ')
	if index2 == -1 {
		return nil
	}

	return RES.Header.theBuffer.B[index1+1 : index2]
}

func (RES *ResponseType) StatusCodeString() string {
	return string(RES.StatusCode())
}

func (RES *ResponseType) StatusCodeInt() int {
	return bToInt(RES.StatusCode())
}

func (RES *ResponseType) Reason() []byte {
	index1 := bytes.IndexByte(RES.Header.theBuffer.B, ' ')
	if index1 == -1 {
		return nil
	}
	return RES.Header.theBuffer.B[index1+1 : bytes.Index(RES.Header.theBuffer.B, lines[5:])]
}

func (RES *ResponseType) ReasonString() string {
	return string(RES.Reason())
}

func (RES *ResponseType) BodyString() string {
	return string(RES.Body())
}

func (RES *ResponseType) Body() []byte {
	idx := bytes.Index(RES.Header.theBuffer.B, lines[3:])
	if idx == -1 {
		return nil
	}
	return RES.Header.theBuffer.B[idx+4:]
}

func (HEAD *headStruct) GetAll() map[string]string {
	forReturn := make(map[string]string)

	forNothing := strings.Split(HEAD.theBuffer.String(), "\r\n")[1:]

	for _, res := range forNothing {
		index1 := strings.Index(res, ": ")

		if index1 == -1 {
			break
		}

		forReturn[res[:index1]] = res[index1+2:]
	}

	return forReturn
}

func (HEAD *headStruct) Get(name string) string {
	forNothing := strings.Split(HEAD.theBuffer.String(), "\r\n")[1:]

	for _, res := range forNothing {
		index1 := strings.Index(res, ": ")

		if index1 == -1 {
			break
		} else if res[:index1] == name {
			return res[index1+2:]
		}
	}
	return ""
}

func (RES *ResponseType) Raw() []byte {
	return RES.Header.theBuffer.B
}

func (RES *ResponseType) RawString() string {
	return RES.Header.theBuffer.String()
}
