package regn

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/valyala/bytebufferpool"
	"golang.org/x/net/http2/hpack"
)

type ConnectionInformation struct {
	myport       string
	myhost       string
	hpackHeaders []hpack.HeaderField
	hpackEncoder *hpack.Encoder
	rawBody      bytebufferpool.ByteBuffer
	raw          bytebufferpool.ByteBuffer
}

type RequestType struct {
	Header *ConnectionInformation
}

func (REQ *RequestType) Close() {
	REQ.Header.raw.Reset()
	bufferPool.Put(&REQ.Header.raw)

	if REQ.Header.hpackHeaders != nil {
		REQ.Header.rawBody.Reset()
		bufferPool.Put(&REQ.Header.rawBody)

		REQ.Header.hpackHeaders = nil
		REQ.Header.hpackEncoder = nil
	}
}

func Request() *RequestType {
	toReturn := &RequestType{Header: &ConnectionInformation{raw: *bufferPool.Get()}}
	toReturn.Header.raw.WriteString("S S HTTP/1.1\r\n")
	toReturn.Header.raw.WriteString("User-Agent: " + Name + "/" + Version + Author + "\r\n")
	toReturn.Header.raw.WriteString("Connection: Keep-Alive\r\n")
	toReturn.Header.raw.WriteString("\r\n")

	return toReturn
}

func (REQ *RequestType) ReturnBytes() []byte {
	return REQ.Header.raw.B
}

func Http2Request() *RequestType {
	therequest := &RequestType{Header: &ConnectionInformation{raw: *bufferPool.Get()}}
	therequest.Http2Upgrade()

	therequest.Header.Set("User-Agent", Name+"/"+Version+Author)
	return therequest
}

func (REQ *RequestType) HttpDowngrade() {
	if REQ.Header.hpackEncoder != nil {
		REQ.Header.raw.Reset()

		REQ.Header.raw.WriteString(REQ.Header.hpackHeaders[0].Value + " ")

		REQ.Header.raw.WriteString(REQ.Header.hpackHeaders[1].Value + " HTTP/1.1\r\n")

		REQ.Header.raw.WriteString("Host: " + REQ.Header.hpackHeaders[3].Value + "\r\n")
		REQ.Header.hpackHeaders = REQ.Header.hpackHeaders[3:]

		for _, head := range REQ.Header.hpackHeaders {
			REQ.Header.raw.WriteString(head.Name + ": " + head.Value + "\r\n")
		}

		if REQ.Header.rawBody.Len() != 0 {
			length := strconv.Itoa(REQ.Header.rawBody.Len())
			REQ.Header.raw.WriteString("Content-Length: " + length + "\r\n")
		}

		REQ.Header.raw.WriteString("\r\n")
		REQ.Header.raw.Write(REQ.Header.rawBody.B)

		REQ.Header.hpackHeaders = nil
		REQ.Header.hpackEncoder = nil
		REQ.Header.rawBody.Reset()
		bufferPool.Put(&REQ.Header.rawBody)
	}
}

func (REQ *RequestType) Http2Upgrade() {
	REQ.Header.raw.Reset()
	REQ.Header.rawBody = *bufferPool.Get()
	REQ.Header.hpackEncoder = hpack.NewEncoder(&REQ.Header.raw)

	if len(REQ.Header.raw.B) != 0 {
		REQ.Header.hpackHeaders = []hpack.HeaderField{}
		REQ.Header.Remove("Host")
		REQ.Header.Remove("Content-Length")
		MethodLine := bytes.Split(REQ.Header.raw.B, SpaceByte)

		if len(MethodLine) >= 1 {
			REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: ":method", Value: string(MethodLine[0])})
		}

		if len(MethodLine) >= 2 {
			REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: ":path", Value: string(MethodLine[1])})
		}

		MethodLine = nil

		REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: ":scheme", Value: "https"})
		REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: ":authority", Value: REQ.Header.myhost})

		HeadersLines := bytes.Split(REQ.Header.raw.B, lines[3:])
		HeadersLines = HeadersLines[1:]

		for _, xo := range HeadersLines {
			Head := bytes.Split(xo, []byte(": "))
			if len(Head) >= 2 {
				REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: string(Head[0]), Value: string(bytes.Join(Head[1:], []byte(": ")))})
			}
			Head = nil
			xo = nil
		}

		HeadersLines = nil

		FullRequest := bytes.Split(REQ.Header.raw.B, lines[1:])
		REQ.Header.raw.Reset()

		for _, Head := range REQ.Header.hpackHeaders {
			REQ.Header.hpackEncoder.WriteField(Head)
		}

		if len(FullRequest) == 2 {
			REQ.Header.rawBody.Write(FullRequest[1])
		}

		REQ.Header.raw.Reset()
		FullRequest = nil
	} else {
		REQ.Header.hpackHeaders = []hpack.HeaderField{
			{Name: ":method", Value: ""},
			{Name: ":path", Value: ""},
			{Name: ":scheme", Value: "https"},
			{Name: ":authority", Value: ""},
		}
	}
	for _, xo := range REQ.Header.hpackHeaders {
		REQ.Header.hpackEncoder.WriteField(xo)
	}
}

func (REQ *RequestType) SetMethod(METHOD string) {
	if REQ.Header.hpackHeaders != nil {
		REQ.Header.Set(":method", strings.ToUpper(METHOD))
	} else {
		TheMethod := []byte(strings.ToUpper(METHOD))
		SplitLines := bytes.Split(REQ.Header.raw.B, SpaceByte)
		SplitLines[0] = TheMethod
		TheMethod = nil

		newRequest := bytes.Join(SplitLines, SpaceByte)

		REQ.Header.raw.Reset()
		REQ.Header.raw.Write(newRequest)
		newRequest = nil
	}
}

func (REQ *RequestType) SetURL(Url string) {
	Parse, err := url.Parse(Url)

	if err != nil {
		panic("invalid url request \n" + err.Error())
	} else if Parse.Scheme == "" {
		panic("no supplied url scheme \n" + err.Error())
	}

	if Parse.Port() != "" {
		REQ.Header.myport = Parse.Port()
	} else if Parse.Scheme == "https" {
		REQ.Header.myport = "443"
	} else {
		REQ.Header.myport = "80"
	}

	if Parse.Hostname() == "" {
		panic("no supplied hostname")
	} else {
		REQ.Header.myhost = Parse.Hostname()
	}

	if REQ.Header.hpackHeaders != nil {
		var api string
		if Parse.Path == "" {
			api = "/"
		} else {
			api = Parse.Path
		}

		if Parse.RawQuery != "" {
			api = api + "?" + Parse.RawQuery
		}
		REQ.Header.Set(":authority", REQ.Header.myhost)
		REQ.Header.Set(":path", api)
	} else {
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

		SplitLines := bytes.Split(REQ.Header.raw.B, SpaceByte)

		SplitLines[1] = api
		newRequest := bytes.Join(SplitLines, SpaceByte)
		api = nil

		REQ.Header.raw.Reset()
		REQ.Header.raw.Write(newRequest)

		newRequest = nil

		REQ.Header.Set("Host", REQ.Header.myhost)
	}
}

func (REQ *ConnectionInformation) Set(key string, value string) {
	if REQ.hpackHeaders != nil {
		REQ.raw.Reset()
		REQ.hpackEncoder = hpack.NewEncoder(&REQ.raw)

		Head := hpack.HeaderField{Name: strings.ToLower(key), Value: value}
		for r, xo := range REQ.hpackHeaders {
			if strings.EqualFold(xo.Name, key) {
				REQ.hpackHeaders[r] = Head
				REQ.hpackEncoder.WriteField(Head)
				Head.Name = ""
			} else {
				REQ.hpackEncoder.WriteField(xo)
			}
		}

		if Head.Name != "" {
			REQ.hpackHeaders = append(REQ.hpackHeaders, Head)
			REQ.hpackEncoder.WriteField(Head)
		}

	} else {
		data := REQ.raw.B
		lowerSearch := []byte(strings.ToLower(key + ": "))
		lowerData := bytes.ToLower(data)

		start := bytes.Index(lowerData, lowerSearch)
		lowerSearch = nil

		if start != -1 {
			end := bytes.Index(lowerData[start:], lines[3:])
			end += start + 2
			data = append(data[:start], data[end:]...)
		} else {
			start = bytes.Index(lowerData, lines[3:]) + 2
		}

		newHeader := []byte(key + ": " + value + "\r\n")

		data = append(data[:start], append(newHeader, data[start:]...)...)
		newHeader = nil

		REQ.raw.Reset()
		REQ.raw.Write(data)

		lowerData = nil
		data = nil
	}
}

func (REQ *ConnectionInformation) Del(key string) {
	if REQ.hpackHeaders != nil {
		REQ.raw.Reset()
		REQ.hpackEncoder = hpack.NewEncoder(&REQ.raw)
		for r, xo := range REQ.hpackHeaders {
			if strings.EqualFold(xo.Name, key) {
				REQ.hpackHeaders = remove(REQ.hpackHeaders, r)
			} else {
				REQ.hpackEncoder.WriteField(xo)
			}
		}
	} else {
		data := REQ.raw.B
		lowerSearch := []byte(strings.ToLower(key + ": "))
		lowerData := bytes.ToLower(data)

		start := bytes.Index(lowerData, lowerSearch)
		lowerSearch = nil

		if start != -1 {
			end := bytes.Index(lowerData[start:], lines[3:])
			lowerData = nil

			end += start + 2
			data = append(data[:start], data[end:]...)

			REQ.raw.Reset()
			REQ.raw.Write(data)
		}
		data = nil
	}
}

func (REQ *RequestType) SetBody(RawBody []byte) {
	if REQ.Header.hpackHeaders != nil {
		REQ.Header.rawBody.Reset()
		REQ.Header.rawBody.Write(RawBody)
	} else {
		splitLines := bytes.Split(REQ.Header.raw.B, lines[1:])

		splitLines[1] = RawBody
		LenBody := len(RawBody)
		RawBody = nil

		newRequest := bytes.Join(splitLines, lines[1:])

		lowerSearch := []byte(strings.ToLower("Content-Length: "))
		lowerData := bytes.ToLower(newRequest)

		start := bytes.Index(lowerData, lowerSearch)
		lowerSearch = nil
		if start != -1 {
			end := bytes.Index(lowerData[start:], lines[3:])
			end += start + 2
			newRequest = append(newRequest[:start], newRequest[end:]...)
		} else {
			start = bytes.Index(lowerData, lines[3:]) + 2
		}

		newHeader := []byte("Content-Length: " + strconv.Itoa(LenBody) + "\r\n")

		newRequest = append(newRequest[:start], append(newHeader, newRequest[start:]...)...)
		newHeader = nil

		REQ.Header.raw.Reset()
		REQ.Header.raw.Write(newRequest)

		lowerData = nil
		newRequest = nil
	}

}

func (REQ *RequestType) SetBodyString(RawBody string) {
	if REQ.Header.hpackHeaders != nil {
		REQ.Header.rawBody.Reset()
		RawByte := []byte(RawBody)
		REQ.Header.rawBody.Write(RawByte)
		RawByte = nil

	} else {
		splitLines := bytes.Split(REQ.Header.raw.B, lines[1:])

		splitLines[1] = []byte(RawBody)
		LenBody := len(RawBody)

		newRequest := bytes.Join(splitLines, lines[1:])

		lowerSearch := []byte(strings.ToLower("Content-Length" + ": "))
		lowerData := bytes.ToLower(newRequest)

		start := bytes.Index(lowerData, lowerSearch)
		lowerSearch = nil
		if start != -1 {
			end := bytes.Index(lowerData[start:], lines[1:])
			end += start + 2
			newRequest = append(newRequest[:start], newRequest[end:]...)
		} else {
			start = bytes.Index(lowerData, lines[1:]) + 2
		}

		newHeader := []byte("Content-Length" + ": " + strconv.Itoa(LenBody) + "\r\n")

		newRequest = append(newRequest[:start], append(newHeader, newRequest[start:]...)...)
		newHeader = nil

		REQ.Header.raw.Reset()
		REQ.Header.raw.Write(newRequest)

		lowerData = nil
		newRequest = nil
	}
}

func (REQ *RequestType) SetJson(RawJson map[string]string) error {
	TheBody, err := json.Marshal(RawJson)

	if err != nil {
		return &RegnError{Message: "field encode body to json format"}
	}

	REQ.SetBody(TheBody)
	REQ.Header.Set("Content-Type", "application/json")

	return nil
}

func (REQ *ConnectionInformation) Add(key string, value string) {
	REQ.Set(key, value)
}

func (REQ *ConnectionInformation) Remove(key string) {
	REQ.Del(key)
}

func remove(slice []hpack.HeaderField, index int) []hpack.HeaderField {
	return append(slice[:index], slice[index+1:]...)
}
