package regn

import (
	"bytes"
	"net/url"
	"strings"
)

type ConnectionInformation struct {
	myport string
	myhost string
	mytls  bool

	raw        []byte
	bufferSize int
	position   int
}

type RequestType struct {
	Header *ConnectionInformation
}

func (REQ *RequestType) Close() {
	REQ.Header.raw = nil
	REQ.Header.bufferSize = 0
}

func (REQ *RequestType) Reset() {
	REQ.Header.raw = REQ.Header.raw[:0]
	REQ.Header.raw = REQ.Header.raw[:REQ.Header.bufferSize]
}

func Request(bufferSize int) *RequestType {
	if bufferSize < 128 {
		panic("can not using bufferSize < 128 in `regn.Request` function")
	}

	toReturn := &RequestType{Header: &ConnectionInformation{raw: make([]byte, 0, bufferSize), bufferSize: bufferSize}}
	toReturn.Header.raw = toReturn.Header.raw[:bufferSize]
	copy(toReturn.Header.raw[toReturn.Header.position:], []byte("GET /golang HTTP/1.1\r\n"+"User-Agent: "+Name+"/"+Version+Author+"\r\n"+"Connection: Keep-Alive\r\n"+"\r\n"))
	toReturn.Header.position += 89
	return toReturn
}

func (REQ *RequestType) SetMethod(METHOD string) {
	indexS := bytes.Index(REQ.Header.raw, SpaceByte)
	REQ.Header.raw = append([]byte(strings.ToUpper(METHOD)), REQ.Header.raw[indexS:]...)
	REQ.Header.position -= indexS
	REQ.Header.position += len(METHOD)
}

func (REQ *RequestType) SetURL(Url string) {
	Parse, err := url.Parse(Url)

	if err != nil {
		panic("invalid url request \n" + err.Error())
	} else if Parse.Scheme == "" {
		panic("no supplied url scheme")
	}

	if Parse.Port() != "" {
		REQ.Header.myport = Parse.Port()
		if Parse.Scheme == "https" {
			REQ.Header.mytls = true
		} else {
			REQ.Header.mytls = false
		}
	} else if Parse.Scheme == "https" {
		REQ.Header.myport = "443"
		REQ.Header.mytls = false
	} else {
		REQ.Header.myport = "80"
		REQ.Header.mytls = false
	}

	if Parse.Hostname() == "" {
		panic("no supplied hostname")
	} else {
		REQ.Header.myhost = Parse.Hostname()
	}

	var api []byte
	if Parse.Path == "" {
		api = []byte("/")
	} else {
		api = []byte(Parse.Path)
	}

	if Parse.RawQuery != "" {
		query := []byte("?" + Parse.RawQuery)
		api = append(api, query...)
		query = nil
	}

	indexSpaceOne := bytes.Index(REQ.Header.raw, SpaceByte) + 1
	indexSpaceTow := bytes.Index(REQ.Header.raw[indexSpaceOne:], SpaceByte) + indexSpaceOne
	REQ.Header.position -= len(REQ.Header.raw[indexSpaceOne:indexSpaceTow])
	REQ.Header.position += len(api)

	REQ.Header.raw = append(REQ.Header.raw[:indexSpaceOne], append(api, REQ.Header.raw[indexSpaceTow:]...)...)
	REQ.Header.Set("Host", REQ.Header.myhost)
}

func (REQ *ConnectionInformation) Set(key string, value string) {
	indexKey := bytes.Index(REQ.raw, append(lines[5:], []byte(key)...)) + 2
	if indexKey != 1 {
		indexRN := bytes.Index(REQ.raw[indexKey:], lines[5:]) + indexKey
		REQ.position -= len(REQ.raw[indexKey:indexRN])
		REQ.position += 2 + len(key) + len(value)
		REQ.raw = append(REQ.raw[:indexKey], append([]byte(key+": "+value), REQ.raw[indexRN:]...)...)
	} else {
		indexRN := bytes.Index(REQ.raw, lines[5:])
		REQ.position += 4 + len(key) + len(value)
		REQ.raw = append(REQ.raw[:indexRN+2], append([]byte(key+": "+value), REQ.raw[indexRN:]...)...)
	}
}

func (REQ *ConnectionInformation) Del(key string) {
	indexKey := bytes.Index(REQ.raw, []byte(key))
	if indexKey != -1 {
		indexRN := bytes.Index(REQ.raw[indexKey:], lines[5:]) + indexKey + 2
		REQ.position -= len(REQ.raw[indexKey:indexRN])
		REQ.raw = append(REQ.raw[:indexKey], REQ.raw[indexRN:]...)
	}
}

func (REQ *RequestType) SetBody(RawBody []byte) {
	indexL := bytes.Index(REQ.Header.raw, contentLengthKey)
	contentLength := intToB(len(RawBody))
	if indexL != -1 {
		indexN := bytes.Index(REQ.Header.raw[indexL:], lines[5:]) + indexL
		copy(REQ.Header.raw[indexL+16+len(contentLength):], REQ.Header.raw[indexL+16+len(REQ.Header.raw[indexL+16:indexN]):])
		copy(REQ.Header.raw[indexL+16:], contentLength)
		indexB := bytes.Index(REQ.Header.raw, lines[3:]) + 4
		copy(REQ.Header.raw[indexB:], RawBody)
		REQ.Header.position += len(contentLength) - len(REQ.Header.raw[indexL+16:indexN])
		REQ.Header.position -= len(REQ.Header.raw[indexB:REQ.Header.position]) - len(RawBody)
	} else {
		indexR := bytes.Index(REQ.Header.raw, lines[3:])
		copy(REQ.Header.raw[indexR+2:], contentLengthKey)
		REQ.Header.position += len(contentLengthKey) - 2
		copy(REQ.Header.raw[REQ.Header.position:], contentLength)
		REQ.Header.position += len(contentLength)
		copy(REQ.Header.raw[REQ.Header.position:], lines[3:])
		REQ.Header.position += 4
		copy(REQ.Header.raw[REQ.Header.position:], RawBody)
		REQ.Header.position += len(RawBody)
	}
}

func (REQ *RequestType) SetBodyString(RawBody string) {
	REQ.SetBody([]byte(RawBody))
}

func (REQ *ConnectionInformation) Add(key string, value string) {
	REQ.Set(key, value)
}

func (REQ *ConnectionInformation) Remove(key string) {
	REQ.Del(key)
}

func (REQ *RequestType) RawString() string {
	return string(REQ.Header.raw[:REQ.Header.position])
}

func (REQ *RequestType) Raw() []byte {
	return REQ.Header.raw[:REQ.Header.position]
}
