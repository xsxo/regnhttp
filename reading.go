package regn

import (
	"bytes"
	"strings"
)

type headStruct struct {
	theBuffer  []byte
	bufferSize int
	position   int
}

type ResponseType struct {
	Header *headStruct
}

func (RES *ResponseType) Close() {
	RES.Header.theBuffer = nil
	RES.Header.bufferSize = 0
}

func (RES *ResponseType) Reset() {
	RES.Header.position = 0
	RES.Header.theBuffer = RES.Header.theBuffer[:0]
}

func Response(bufferSize int) *ResponseType {
	return &ResponseType{Header: &headStruct{theBuffer: make([]byte, 0, bufferSize), bufferSize: bufferSize}}
}

func (RES *ResponseType) StatusCode() []byte {
	index1 := bytes.IndexByte(RES.Header.theBuffer, ' ')
	if index1 == -1 {
		return nil
	}

	index2 := index1 + 1 + bytes.IndexByte(RES.Header.theBuffer[index1+1:RES.Header.position], ' ')
	if index2 == -1 {
		return nil
	}

	return RES.Header.theBuffer[index1+1 : index2]
}

func (RES *ResponseType) StatusCodeString() string {
	return string(RES.StatusCode())
}

func (RES *ResponseType) StatusCodeInt() int {
	return BytesToInt(RES.StatusCode())
}

func (RES *ResponseType) Reason() []byte {
	index1 := bytes.IndexByte(RES.Header.theBuffer, ' ')
	if index1 == -1 {
		return nil
	}
	return RES.Header.theBuffer[index1+1 : bytes.Index(RES.Header.theBuffer, line)]
}

func (RES *ResponseType) ReasonString() string {
	return string(RES.Reason())
}

func (RES *ResponseType) Body() []byte {
	idx := bytes.Index(RES.Header.theBuffer, lines)
	if idx == -1 {
		return nil
	}
	return RES.Header.theBuffer[idx+4 : RES.Header.position]
}

func (RES *ResponseType) BodyString() string {
	return string(RES.Body())
}

func (HEAD *headStruct) GetAll() map[string]string {
	forReturn := make(map[string]string)

	forNothing := strings.Split(string(HEAD.theBuffer), "\r\n")[1:]

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
	forNothing := strings.Split(string(HEAD.theBuffer), "\r\n")[1:]

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
	return RES.Header.theBuffer[:RES.Header.position]
}

func (RES *ResponseType) RawString() string {
	return string(RES.Header.theBuffer[:RES.Header.position])
}
