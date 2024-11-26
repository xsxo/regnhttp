package regn

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strconv"
	"strings"

	"github.com/valyala/bytebufferpool"
)

type ConnectionInformation struct {
	myport string
	myhost string
	raw    bytebufferpool.ByteBuffer
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

func (REQ *RequestType) Close() {
	REQ.Header.raw.Reset()
	bytes_pool.Put(&REQ.Header.raw)
}

func (REQ *RequestType) SetMethod(METHOD string) {
	TheMethod := []byte(strings.ToUpper(METHOD))
	METHOD = ""

	lines := bytes.Split(REQ.Header.raw.B, space_line)

	lines[0] = TheMethod
	TheMethod = nil

	new_requests := bytes.Join(lines, space_line)

	REQ.Header.raw.Reset()
	REQ.Header.raw.Write(new_requests)
	new_requests = nil
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

func (REQ *ConnectionInformation) Set(key string, value string) {
	data := REQ.raw.B
	lowerSearch := []byte(strings.ToLower(key + ": "))
	lowerData := bytes.ToLower(data)

	start := bytes.Index(lowerData, lowerSearch)
	lowerSearch = nil

	if start != -1 {
		end := bytes.Index(lowerData[start:], one_line)
		end += start + 2
		data = append(data[:start], data[end:]...)
		// fmt.Println("end: ", end)
		// fmt.Println("start: ", start)
	} else {
		start = bytes.Index(lowerData, one_line) + 2
	}

	newHeader := []byte(key + ": " + value + "\r\n")
	// insertPosition := bytes.Index(data, one_line) + 2
	// fmt.Println("insertPosition: ", insertPosition)
	// data = append(data[:insertPosition], append(newHeader, data[insertPosition:]...)...)

	data = append(data[:start], append(newHeader, data[start:]...)...)
	newHeader = nil

	REQ.raw.Reset()
	REQ.raw.Write(data)

	lowerData = nil
	data = nil
}

func (REQ *ConnectionInformation) Add(key string, value string) {
	data := REQ.raw.B
	lowerSearch := []byte(strings.ToLower(key + ": "))
	lowerData := bytes.ToLower(data)

	start := bytes.Index(lowerData, lowerSearch)
	lowerSearch = nil

	if start != -1 {
		end := bytes.Index(lowerData[start:], one_line)
		end += start + 2
		data = append(data[:start], data[end:]...)
		// fmt.Println("end: ", end)
		// fmt.Println("start: ", start)
	} else {
		start = bytes.Index(lowerData, one_line)
	}

	newHeader := []byte(key + ": " + value + "\r\n")
	// insertPosition := bytes.Index(data, one_line) + 2
	// fmt.Println("insertPosition: ", insertPosition)
	// data = append(data[:insertPosition], append(newHeader, data[insertPosition:]...)...)

	data = append(data[:start], append(newHeader, data[start:]...)...)
	newHeader = nil

	REQ.raw.Reset()
	REQ.raw.Write(data)

	lowerData = nil
	data = nil
}

func (REQ *ConnectionInformation) Del(key string) {
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

func (REQ *ConnectionInformation) Remove(key string) {
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

func (REQ *RequestType) SetBody(RawBody []byte) {
	lines := bytes.Split(REQ.Header.raw.B, tow_lines)

	lines[1] = RawBody
	LenBody := len(RawBody)
	RawBody = nil

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

func (REQ *RequestType) SetBodyString(RawBody string) {
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

func (REQ *RequestType) SetJson(RawJson map[string]string) error {
	TheBody, err := json.Marshal(RawJson)

	if err != nil {
		return &RegnError{Message: "Field encoding map to json format; use map[string]string"}
	}

	REQ.SetBody(TheBody)
	REQ.Header.Set("Content-Type", "application/json")

	return nil
}
