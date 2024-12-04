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

func Request() *RequestType {
	to_return := &RequestType{Header: &ConnectionInformation{raw: *bytes_pool.Get()}}
	to_return.Header.raw.WriteString("S S HTTP/1.1\r\n")
	to_return.Header.raw.WriteString("User-Agent: " + Name + "/" + Version + Author + "\r\n")
	to_return.Header.raw.WriteString("Connection: Keep-Alive\r\n")
	to_return.Header.raw.WriteString("\r\n")

	return to_return
}

// func (REQ *RequestType) ReturnBytes() []byte {
// 	return REQ.Header.raw.B
// }

// func Http2Request() *RequestType {
// 	therequest := &RequestType{Header: &ConnectionInformation{raw: *bytes_pool.Get()}}
// 	therequest.upgradeH2c()

// 	therequest.Header.Set("User-Agent", Name+"/"+Version+Author)
// 	return therequest
// }

func (REQ *RequestType) Close() {
	REQ.Header.raw.Reset()
	bytes_pool.Put(&REQ.Header.raw)

	if REQ.Header.hpackHeaders != nil {
		REQ.Header.rawBody.Reset()
		bytes_pool.Put(&REQ.Header.rawBody)

		REQ.Header.hpackHeaders = nil
		REQ.Header.hpackEncoder = nil
	}
}

// func (REQ *RequestType) upgradeH2c() {
// 	REQ.Header.raw.Reset()
// 	REQ.Header.rawBody = *bytes_pool.Get()
// 	REQ.Header.hpackEncoder = hpack.NewEncoder(&REQ.Header.raw)

// 	if len(REQ.Header.raw.B) != 0 {
// 		REQ.Header.hpackHeaders = []hpack.HeaderField{}
// 		REQ.Header.Remove("Host")
// 		REQ.Header.Remove("Content-Length")
// 		MethodLine := bytes.Split(REQ.Header.raw.B, space_line)

// 		if len(MethodLine) >= 1 {
// 			REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: ":method", Value: string(MethodLine[0])})
// 		}

// 		if len(MethodLine) >= 2 {
// 			REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: ":path", Value: string(MethodLine[1])})
// 		}

// 		MethodLine = nil

// 		REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: ":scheme", Value: "https"})
// 		REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: ":authority", Value: REQ.Header.myhost})

// 		TheHeaders_lines := bytes.Split(REQ.Header.raw.B, one_line)
// 		TheHeaders_lines = TheHeaders_lines[1:]

// 		for _, xo := range TheHeaders_lines {
// 			Head := bytes.Split(xo, []byte(": "))
// 			if len(Head) >= 2 {
// 				REQ.Header.hpackHeaders = append(REQ.Header.hpackHeaders, hpack.HeaderField{Name: string(Head[0]), Value: string(bytes.Join(Head[1:], []byte(": ")))})
// 			}
// 			Head = nil
// 			xo = nil
// 		}

// 		TheHeaders_lines = nil

// 		FullRequest := bytes.Split(REQ.Header.raw.B, tow_lines)
// 		REQ.Header.raw.Reset()

// 		for _, Head := range REQ.Header.hpackHeaders {
// 			REQ.Header.hpackEncoder.WriteField(Head)
// 		}

// 		if len(FullRequest) == 2 {
// 			REQ.Header.rawBody.Write(FullRequest[1])
// 		}

// 		REQ.Header.raw.Reset()
// 		FullRequest = nil
// 	} else {
// 		REQ.Header.hpackHeaders = []hpack.HeaderField{
// 			{Name: ":method", Value: ""},
// 			{Name: ":path", Value: ""},
// 			{Name: ":scheme", Value: "https"},
// 			{Name: ":authority", Value: ""},
// 		}
// 	}
// 	for _, xo := range REQ.Header.hpackHeaders {
// 		REQ.Header.hpackEncoder.WriteField(xo)
// 	}
// }

func (REQ *RequestType) SetMethod(METHOD string) {
	if REQ.Header.hpackHeaders != nil {
		REQ.Header.Set(":method", strings.ToUpper(METHOD))
	} else {
		TheMethod := []byte(strings.ToUpper(METHOD))
		lines := bytes.Split(REQ.Header.raw.B, space_line)
		lines[0] = TheMethod
		TheMethod = nil

		new_requests := bytes.Join(lines, space_line)

		REQ.Header.raw.Reset()
		REQ.Header.raw.Write(new_requests)
		new_requests = nil
	}
}

func (REQ *RequestType) SetURL(Url string) {
	Parse, err := url.Parse(Url)

	if err != nil {
		panic("REGNHTTP: Invalid URL  '" + err.Error() + "'")
	} else if Parse.Scheme == "" {
		panic("REGNHTTP: Invalid URL '" + Url + "': No scheme supplied, Perhaps you meant 'https://" + Url + "' ?")
	}

	if Parse.Port() != "" {
		REQ.Header.myport = Parse.Port()
	} else if Parse.Scheme == "https" {
		REQ.Header.myport = "443"
	} else {
		REQ.Header.myport = "80"
	}

	if Parse.Hostname() == "" {
		panic("REGNHTTP: Invalid URL '" + Url + "': No host supplied")
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

		lines := bytes.Split(REQ.Header.raw.B, space_line)

		lines[1] = api
		new_requests := bytes.Join(lines, space_line)
		api = nil

		REQ.Header.raw.Reset()
		REQ.Header.raw.Write(new_requests)

		new_requests = nil

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
			end := bytes.Index(lowerData[start:], one_line)
			end += start + 2
			data = append(data[:start], data[end:]...)
		} else {
			start = bytes.Index(lowerData, one_line) + 2
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
			end := bytes.Index(lowerData[start:], one_line)
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
		lines := bytes.Split(REQ.Header.raw.B, tow_lines)

		lines[1] = RawBody
		LenBody := len(RawBody)
		RawBody = nil

		new_requests := bytes.Join(lines, tow_lines)

		lowerSearch := []byte(strings.ToLower("Content-Length: "))
		lowerData := bytes.ToLower(new_requests)

		start := bytes.Index(lowerData, lowerSearch)
		lowerSearch = nil
		if start != -1 {
			end := bytes.Index(lowerData[start:], one_line)
			end += start + 2
			new_requests = append(new_requests[:start], new_requests[end:]...)
		} else {
			start = bytes.Index(lowerData, one_line) + 2
		}

		newHeader := []byte("Content-Length: " + strconv.Itoa(LenBody) + "\r\n")

		new_requests = append(new_requests[:start], append(newHeader, new_requests[start:]...)...)
		newHeader = nil

		REQ.Header.raw.Reset()
		REQ.Header.raw.Write(new_requests)

		lowerData = nil
		new_requests = nil
	}

}

func (REQ *RequestType) SetBodyString(RawBody string) {
	if REQ.Header.hpackHeaders != nil {
		REQ.Header.rawBody.Reset()
		RawByte := []byte(RawBody)
		REQ.Header.rawBody.Write(RawByte)
		RawByte = nil

	} else {
		lines := bytes.Split(REQ.Header.raw.B, tow_lines)

		lines[1] = []byte(RawBody)
		LenBody := len(RawBody)

		new_requests := bytes.Join(lines, tow_lines)

		lowerSearch := []byte(strings.ToLower("Content-Length" + ": "))
		lowerData := bytes.ToLower(new_requests)

		start := bytes.Index(lowerData, lowerSearch)
		lowerSearch = nil
		if start != -1 {
			end := bytes.Index(lowerData[start:], one_line)
			end += start + 2
			new_requests = append(new_requests[:start], new_requests[end:]...)
		} else {
			start = bytes.Index(lowerData, one_line) + 2
		}

		newHeader := []byte("Content-Length" + ": " + strconv.Itoa(LenBody) + "\r\n")

		new_requests = append(new_requests[:start], append(newHeader, new_requests[start:]...)...)
		newHeader = nil

		REQ.Header.raw.Reset()
		REQ.Header.raw.Write(new_requests)

		lowerData = nil
		new_requests = nil
	}
}

func (REQ *RequestType) SetJson(RawJson map[string]string) error {
	TheBody, err := json.Marshal(RawJson)

	if err != nil {
		return &RegnError{Message: "Field encoding map to json format; use map[string]string"}
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
