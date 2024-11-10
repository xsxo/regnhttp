package fiberhttp

import (
	"encoding/json"
	"log"
	"net/url"
	"strconv"
	"strings"
)

type ConnectionInformation struct {
	myport          string
	myhost          string
	thebytesheaders map[string]string
	// run             bool
	raw []byte
}

type request struct {
	theybytesmethod []byte
	theybytesapi    []byte
	theybytesbody   []byte
	Header          *ConnectionInformation

	userjson bool
}

func Request() request {
	return request{Header: &ConnectionInformation{}}
}

func (REQ *request) SetMethod(METHOD string) {
	REQ.Header.raw = nil

	REQ.theybytesmethod = []byte(strings.ToUpper(METHOD))
}

func (REQ *request) SetURL(Url string) {
	REQ.Header.raw = nil
	Parse, err := url.Parse(Url)

	if err != nil {
		log.SetFlags(0)
		log.Fatalln("FiberHTTP Error: Invalid URL  '" + err.Error() + "'")
	} else if Parse.Scheme == "" {
		log.SetFlags(0)
		log.Fatalln("FiberHTTP Error: Invalid URL '" + Url + "': No scheme supplied, Perhaps you meant 'https://" + Url + "' ?")
	}

	if Parse.Port() != "" {
		REQ.Header.myport = Parse.Port()
	} else if Parse.Scheme == "https" {
		REQ.Header.myport = "443"
	} else {
		REQ.Header.myport = "80"
	}

	if Parse.Hostname() == "" {
		log.SetFlags(0)
		log.Fatalln("FiberHTTP Error: Invalid URL '" + Url + "': No host supplied")
	} else {
		REQ.Header.myhost = Parse.Hostname()
	}

	if Parse.Path == "" {
		REQ.theybytesapi = []byte("/")
	} else {
		REQ.theybytesapi = []byte(Parse.Path)
	}

	if Parse.RawQuery != "" {
		REQ.theybytesapi = append(REQ.theybytesapi, []byte("?"+Parse.RawQuery)...)
	}
}

func (REQ *ConnectionInformation) Set(key string, value string) {
	REQ.raw = nil

	if REQ.thebytesheaders == nil {
		REQ.thebytesheaders = make(map[string]string)
	}

	REQ.thebytesheaders[key] = value
}

func (REQ *ConnectionInformation) Add(key string, value string) {
	REQ.raw = nil

	if REQ.thebytesheaders == nil {
		REQ.thebytesheaders = make(map[string]string)
	}

	REQ.thebytesheaders[key] = value
}

func (REQ *ConnectionInformation) Del(key string) {
	REQ.raw = nil

	if REQ.thebytesheaders == nil {
		return
	}

	delete(REQ.thebytesheaders, key)
}

func (REQ *ConnectionInformation) Remove(key string) {

	REQ.raw = nil

	if REQ.thebytesheaders == nil {
		return
	}

	delete(REQ.thebytesheaders, key)
}

func (REQ *request) SetBody(RawBody string) {
	REQ.Header.raw = nil

	REQ.userjson = false
	REQ.theybytesbody = []byte(RawBody)
}

func (REQ *request) SetJson(RawJson map[string]string) error {
	NewErr := &FiberhttpError{}
	var err error

	REQ.Header.raw = nil

	REQ.userjson = true
	REQ.theybytesbody, err = json.Marshal(RawJson)

	if err != nil {
		NewErr.Message = "Field encoding map to json format; use map[string]string"
		return NewErr
	}

	return nil
}

func (REQ *request) release() error {
	err := &FiberhttpError{}

	if REQ.theybytesmethod == nil {
		err.Message = "No URL supplied"
		return err
	}

	REQ.Header.raw = nil
	REQ.Header.raw = REQ.theybytesmethod
	REQ.Header.raw = append(REQ.Header.raw, 32)
	REQ.Header.raw = append(REQ.Header.raw, REQ.theybytesapi...)
	REQ.Header.raw = append(REQ.Header.raw, 32, 72, 84, 84, 80, 47, 49, 46, 49, 13, 10)

	if REQ.Header.thebytesheaders == nil {
		REQ.Header.thebytesheaders = make(map[string]string)
	}

	var keys []string
	for key := range REQ.Header.thebytesheaders {
		keys = append(keys, strings.ToLower(key))
	}
	StringHeaders := strings.Join(keys, ", ")

	if !strings.Contains(StringHeaders, "user-agent") {
		REQ.Header.thebytesheaders["User-Agent"] = "Mozilla/5.0 Firefox/132.0"
	}

	if !strings.Contains(StringHeaders, "host") {
		REQ.Header.thebytesheaders["Host"] = REQ.Header.myhost
	}

	if !strings.Contains(StringHeaders, "connection") {
		REQ.Header.thebytesheaders["Connection"] = "Keep-Alive"
	}

	if REQ.theybytesbody != nil && !strings.Contains(StringHeaders, "content-length") {
		REQ.Header.thebytesheaders["Content-Length"] = strconv.Itoa(len(REQ.theybytesbody))
	}

	if REQ.userjson && !strings.Contains(StringHeaders, "content-type") {
		REQ.Header.thebytesheaders["Content-Type"] = "application/json"
	}

	for key, value := range REQ.Header.thebytesheaders {
		REQ.Header.raw = append(REQ.Header.raw, []byte(key+": "+value+"\r\n")...)
	}
	REQ.Header.raw = append(REQ.Header.raw, 13, 10)
	REQ.Header.raw = append(REQ.Header.raw, []byte(REQ.theybytesbody)...)

	return nil
}
