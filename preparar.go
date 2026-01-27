package regn

import (
	"bytes"
	"net/url"
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
	copy(REQ.Header.raw[REQ.Header.position:], []byte("GET /golang HTTP/1.1\r\n"+"User-Agent: "+Name+"/"+Version+Author+"\r\n"+"Connection: Keep-Alive\r\n"+"\r\n"))
	REQ.Header.position += 89
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
	REQ.Header.position += len(METHOD) - indexS
	if len(METHOD) > indexS {
		REQ.Header.raw = REQ.Header.raw[:REQ.Header.position]
	}
	copy(REQ.Header.raw[len(METHOD):], REQ.Header.raw[indexS:])
	copy(REQ.Header.raw, bytes.ToUpper([]byte(METHOD)))
	if REQ.Header.position > REQ.Header.bufferSize {
		panic("regn.Request " + IntToString(REQ.Header.bufferSize) + " buffer is small\nupper it to " + IntToString(REQ.Header.position))
	}
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
	if Parse.Path == "" && Parse.RawQuery == "" {
		api = []byte("/")
	} else if Parse.RawQuery != "" {
		api = []byte(Parse.Path + "?" + Parse.RawQuery)
	} else {
		api = []byte(Parse.Path)
	}

	indexSpaceOne := bytes.Index(REQ.Header.raw, SpaceByte) + 1
	indexSpaceTow := bytes.Index(REQ.Header.raw[indexSpaceOne:], SpaceByte) + indexSpaceOne
	REQ.Header.position += len(api) - len(REQ.Header.raw[indexSpaceOne:indexSpaceTow])
	if len(api) > len(REQ.Header.raw[indexSpaceOne:indexSpaceTow]) {
		REQ.Header.raw = REQ.Header.raw[:REQ.Header.position]
	}

	copy(REQ.Header.raw[indexSpaceOne+len(api):], REQ.Header.raw[indexSpaceOne+len(REQ.Header.raw[indexSpaceOne:indexSpaceTow]):])
	copy(REQ.Header.raw[indexSpaceOne:], api)

	if REQ.Header.position > REQ.Header.bufferSize {
		panic("regn.Request " + IntToString(REQ.Header.bufferSize) + " buffer is small\nupper it to " + IntToString(REQ.Header.position))
	}
	REQ.Header.Set("Host", REQ.Header.myhost)
}

func (REQ *ConnectionInformation) Set(key string, value string) {
	key += ": "
	indexKey := bytes.Index(REQ.raw, []byte("\r\n"+key)) + 2
	if indexKey != 1 {
		indexN := bytes.Index(REQ.raw[indexKey:], lines[5:]) + indexKey
		REQ.position += len(value) - len(REQ.raw[indexKey+len(key):indexN])
		if REQ.position > REQ.bufferSize {
			panic("regn.Request " + IntToString(REQ.bufferSize) + " buffer is small\nupper it to " + IntToString(REQ.position))
		}
		if len(value) > len(REQ.raw[indexKey+len(key):indexN]) {
			REQ.raw = REQ.raw[:REQ.position]
		}
		copy(REQ.raw[indexKey+len(key)+len(value):], REQ.raw[indexKey+len(key)+len(REQ.raw[indexKey+len(key):indexN]):])
		copy(REQ.raw[indexKey+len(key):], []byte(value))
	} else {
		indexRN := bytes.Index(REQ.raw, lines[5:])
		REQ.position += len(key) + len(value) + 2
		if REQ.position > REQ.bufferSize {
			panic("regn.Request " + IntToString(REQ.bufferSize) + " buffer is small\nupper it to " + IntToString(REQ.position))
		}
		REQ.raw = REQ.raw[:REQ.position]
		copy(REQ.raw[indexRN+2+len(key)+len(value)+2:], REQ.raw[indexRN+2:])
		copy(REQ.raw[indexRN+2:], []byte(key+value+"\r\n"))
	}
}

func (REQ *ConnectionInformation) Del(key string) {
	indexKey := bytes.Index(REQ.raw, []byte(key))
	if indexKey != -1 {
		indexRN := bytes.Index(REQ.raw[indexKey:], lines[5:]) + indexKey + 2
		REQ.position -= len(REQ.raw[indexKey:indexRN])
		copy(REQ.raw[indexKey:], REQ.raw[indexRN:])
	}
}

func (REQ *RequestType) SetBody(RawBody []byte) {
	indexL := bytes.Index(REQ.Header.raw, contentLengthKey)
	contentLength := IntToBytes(len(RawBody))
	if indexL != -1 {
		indexN := bytes.Index(REQ.Header.raw[indexL:], lines[5:]) + indexL
		REQ.Header.position += len(contentLength) - len(REQ.Header.raw[indexL+16:indexN])
		if len(contentLength) > len(REQ.Header.raw[indexL+16:indexN]) {
			REQ.Header.raw = REQ.Header.raw[:REQ.Header.position]
		}
		copy(REQ.Header.raw[indexL+16+len(contentLength):], REQ.Header.raw[indexL+16+len(REQ.Header.raw[indexL+16:indexN]):])
		copy(REQ.Header.raw[indexL+16:], contentLength)
		indexB := bytes.Index(REQ.Header.raw, lines[3:]) + 4
		REQ.Header.position += len(RawBody) - len(REQ.Header.raw[indexB:REQ.Header.position])
		if REQ.Header.position > REQ.Header.bufferSize {
			panic("regn.Request " + IntToString(REQ.Header.bufferSize) + " buffer is small\nupper it to " + IntToString(REQ.Header.position))
		} else if len(RawBody) > len(REQ.Header.raw[indexB:REQ.Header.position]) {
			REQ.Header.raw = REQ.Header.raw[:REQ.Header.position]
		}
		copy(REQ.Header.raw[indexB:REQ.Header.position], RawBody)
	} else {
		indexH := bytes.Index(REQ.Header.raw, lines[5:]) + 2
		REQ.Header.position += len(contentLengthKey) + len(contentLength) + 2
		REQ.Header.raw = REQ.Header.raw[:REQ.Header.position]
		copy(REQ.Header.raw[indexH+len(contentLengthKey)+len(contentLength)+2:], REQ.Header.raw[indexH:])
		copy(REQ.Header.raw[indexH:], contentLengthKey)
		copy(REQ.Header.raw[indexH+16:], contentLength)
		copy(REQ.Header.raw[indexH+16+len(contentLength):], lines[5:])
		indexR := bytes.Index(REQ.Header.raw, lines[3:]) + 4
		REQ.Header.position += len(RawBody)
		if REQ.Header.position > REQ.Header.bufferSize {
			panic("regn.Request " + IntToString(REQ.Header.bufferSize) + " buffer is small\nupper it to " + IntToString(REQ.Header.position))
		}
		REQ.Header.raw = REQ.Header.raw[:REQ.Header.position]
		copy(REQ.Header.raw[indexR:], RawBody)
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
