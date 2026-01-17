package regn

import (
	"bytes"
	"net/url"
	"strings"

	"github.com/valyala/bytebufferpool"
)

type ConnectionInformation struct {
	myport string
	myhost string
	mytls  bool

	raw bytebufferpool.ByteBuffer
}

type RequestType struct {
	Header *ConnectionInformation
}

func (REQ *RequestType) Close() {
	REQ.Header.raw.Reset()
	bufferPool.Put(&REQ.Header.raw)
}

func (REQ *RequestType) Reset() {
	REQ.Header.raw.Reset()
}

func Request() *RequestType {
	toReturn := &RequestType{Header: &ConnectionInformation{raw: *bufferPool.Get()}}
	toReturn.Header.raw.WriteString("GET /golang HTTP/1.1\r\n")
	toReturn.Header.raw.WriteString("User-Agent: " + Name + "/" + Version + Author + "\r\n")
	toReturn.Header.raw.WriteString("Connection: Keep-Alive\r\n")
	toReturn.Header.raw.WriteString("\r\n")

	return toReturn
}

func (REQ *RequestType) ReturnBytes() []byte {
	return REQ.Header.raw.B
}

func (REQ *RequestType) SetMethod(METHOD string) {
	REQ.Header.raw.B = append([]byte(strings.ToUpper(METHOD)), REQ.Header.raw.B[bytes.Index(REQ.Header.raw.B, SpaceByte):]...)
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

	REQ.Header.raw.B = bytes.Replace(REQ.Header.raw.B, REQ.Header.raw.B[bytes.Index(REQ.Header.raw.B, SpaceByte)+1:bytes.Index(REQ.Header.raw.B, httpVersion)-1], api, 1)
	REQ.Header.Set("Host", REQ.Header.myhost)
}

func (REQ *ConnectionInformation) Set(key string, value string) {
	REQ.Del(key)
	reqLineEnd := bytes.Index(REQ.raw.B, lines[5:])
	if reqLineEnd != -1 {
		insertPos := reqLineEnd + 2
		REQ.raw.B = append(REQ.raw.B[:insertPos], append([]byte(key+": "+value+"\r\n"), REQ.raw.B[insertPos:]...)...)
	}
}

func (REQ *ConnectionInformation) Del(key string) {
	start := bytes.Index(REQ.raw.B, []byte(key))
	if start != -1 {
		end := bytes.Index(REQ.raw.B[start:], []byte(lines[5:]))
		end += start + 2
		REQ.raw.B = append(REQ.raw.B[:start], REQ.raw.B[end:]...)
	}
}

func (REQ *RequestType) SetBody(RawBody []byte) {
	REQ.Header.Del("Content-Length")
	sepIndex := bytes.Index(REQ.Header.raw.B, lines[3:])
	REQ.Header.raw.B = append(REQ.Header.raw.B[:sepIndex+2], contentLengthKey...)
	REQ.Header.raw.B = append(REQ.Header.raw.B, intToB(len(RawBody))...)
	REQ.Header.raw.B = append(REQ.Header.raw.B, lines[3:]...)
	REQ.Header.raw.B = append(REQ.Header.raw.B, RawBody...)
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
	return REQ.Header.raw.String()
}

func (REQ *RequestType) Raw() []byte {
	return REQ.Header.raw.B
}
