package FiberHTTP

import (
	"encoding/json"
	"strconv"
	"strings"
	"unicode/utf8"
)

type headers_struct struct {
	theybytesheaders []byte
}

type readresponse struct {
	content []byte
	Header  *headers_struct
}

func (RES *readresponse) StatusCode() int {
	matches := status_code_regexp.FindSubmatch(RES.Header.theybytesheaders)
	status_code, _ := strconv.Atoi(string(matches[1]))
	return status_code
}

func (RES *readresponse) Reason() string {
	matches := reason_regexp.FindSubmatch(RES.Header.theybytesheaders)
	return string(matches[1])
}

func (RES *readresponse) StringBody() (string, error) {
	err := &FiberhttpError{}

	if !utf8.Valid(RES.content) {
		err.Message = "Field decode body to UTF-8 string"
		return "", err
	}

	return string(RES.content), nil
}

func (RES *readresponse) Body() []byte {
	return RES.content
}

func (RES *readresponse) Json() (map[string]interface{}, error) {
	NewErr := &FiberhttpError{}

	var result map[string]interface{}
	err := json.Unmarshal([]byte(string(RES.content)), &result)

	if err != nil {
		NewErr.Message = "Field decode body to json format"
		return result, NewErr
	}

	return result, nil
}

func (HEAD *headers_struct) GetAll() map[string]string {
	forReturn := make(map[string]string)
	forNothing := strings.Split(string(HEAD.theybytesheaders), "\n")[1:]

	for _, res := range forNothing {
		if !strings.Contains(res, ": ") {
			break
		}
		parts := strings.SplitN(res, ": ", 2)
		if len(parts) == 2 {
			key := parts[0]
			value := strings.TrimSpace(parts[1])
			forReturn[key] = value
		}
	}

	return forReturn
}

func (HEAD *headers_struct) Get(key string) string {
	forNothing := strings.Split(string(HEAD.theybytesheaders), "\n")[1:]

	for _, res := range forNothing {
		if !strings.Contains(res, ": ") {
			break
		}
		parts := strings.SplitN(res, ": ", 2)
		if len(parts) == 2 {
			if parts[0] == key {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}
