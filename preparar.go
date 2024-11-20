package regn

import (
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/valyala/bytebufferpool"
)

type ConnectionInformation struct {
	myport          string
	myhost          string
	thebytesheaders map[string]string
	raw             bytebufferpool.ByteBuffer
}

type RequestType struct {
	theybytesmethod string
	theybytesapi    string
	theybytesbody   []byte
	Header          *ConnectionInformation

	userjson bool
}

func Request() *RequestType {
	return &RequestType{Header: &ConnectionInformation{raw: *bytes_pool.Get()}}
}

func (REQ *RequestType) Close() {
	REQ.Header.raw.Reset()
	bytes_pool.Put(&REQ.Header.raw)
}

func (REQ *RequestType) SetMethod(METHOD string) {
	REQ.Header.raw.Reset()

	REQ.theybytesmethod = strings.ToUpper(METHOD)
}

func (REQ *RequestType) SetURL(Url string) {
	REQ.Header.raw.Reset()
	Parse, err := url.Parse(Url)

	if err != nil {
		panic("RegnHTTP Error: Invalid URL  '" + err.Error() + "'")
	} else if Parse.Scheme == "" {
		panic("RegnHTTP Error: Invalid URL '" + Url + "': No scheme supplied, Perhaps you meant 'https://" + Url + "' ?")
	}

	if Parse.Port() != "" {
		REQ.Header.myport = Parse.Port()
	} else if Parse.Scheme == "https" {
		REQ.Header.myport = "443"
	} else {
		REQ.Header.myport = "80"
	}

	if Parse.Hostname() == "" {
		panic("RegnHTTP Error: Invalid URL '" + Url + "': No host supplied")
	} else {
		REQ.Header.myhost = Parse.Hostname()
	}

	if Parse.Path == "" {
		REQ.theybytesapi = "/"
	} else {
		REQ.theybytesapi = Parse.Path
	}

	if Parse.RawQuery != "" {
		REQ.theybytesapi = REQ.theybytesapi + "?" + Parse.RawQuery
	}
}

func (REQ *ConnectionInformation) Set(key string, value string) {
	REQ.raw.Reset()

	if REQ.thebytesheaders == nil {
		REQ.thebytesheaders = make(map[string]string)
	}

	REQ.thebytesheaders[key] = value
}

func (REQ *ConnectionInformation) Add(key string, value string) {
	REQ.raw.Reset()

	if REQ.thebytesheaders == nil {
		REQ.thebytesheaders = make(map[string]string)
	}

	REQ.thebytesheaders[key] = value
}

func (REQ *ConnectionInformation) Del(key string) {
	REQ.raw.Reset()

	if REQ.thebytesheaders == nil {
		return
	}

	delete(REQ.thebytesheaders, key)
}

func (REQ *ConnectionInformation) Remove(key string) {

	REQ.raw.Reset()

	if REQ.thebytesheaders == nil {
		return
	}

	delete(REQ.thebytesheaders, key)
}

func (REQ *RequestType) SetBody(RawBody []byte) {
	REQ.Header.raw.Reset()
	REQ.userjson = false
	REQ.theybytesbody = RawBody
}

func (REQ *RequestType) SetBodyString(RawBody string) {
	REQ.Header.raw.Reset()
	REQ.userjson = false
	REQ.theybytesbody = []byte(RawBody)
}

func (REQ *RequestType) SetJson(RawJson map[string]string) error {
	NewErr := &RegnError{}
	var err error

	REQ.Header.raw.Reset()

	REQ.userjson = true
	REQ.theybytesbody, err = json.Marshal(RawJson)

	if err != nil {
		NewErr.Message = "Field encoding map to json format; use map[string]string"
		return NewErr
	}

	return nil
}

func (REQ *RequestType) release() error {
	err := &RegnError{}

	if len(REQ.theybytesmethod) == 0 {
		err.Message = "No URL supplied"
		return err
	}

	REQ.Header.raw.Reset()
	REQ.Header.raw.WriteString(REQ.theybytesmethod)
	// REQ.Header.raw.WriteString("\r")
	REQ.Header.raw.WriteString(" " + REQ.theybytesapi)
	REQ.Header.raw.WriteString(" HTTP/1.1\r\n")

	for key, value := range REQ.Header.thebytesheaders {
		REQ.Header.raw.WriteString(key + ": " + value + "\r\n")
	}

	lower := strings.ToLower(REQ.Header.raw.String())

	if !strings.Contains(lower, "user-agent: ") {
		REQ.Header.raw.WriteString("User-Agent: " + Name + "/" + Version + Author + "\r\n")
	}

	if !strings.Contains(lower, "host: ") {
		REQ.Header.raw.WriteString("Host: " + REQ.Header.myhost + "\r\n")
	}

	if !strings.Contains(lower, "connection: ") {
		REQ.Header.raw.WriteString("Connection: Keep-Alive\r\n")
	}

	if !strings.Contains(lower, "content-length: ") {
		REQ.Header.raw.WriteString("Content-Length: " + strconv.Itoa(len(REQ.theybytesbody)) + "\r\n")
	}

	REQ.Header.raw.WriteString("\r\n")

	REQ.Header.raw.Write(REQ.theybytesbody)

	return nil
}
