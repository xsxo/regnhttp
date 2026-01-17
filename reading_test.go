package regn

import (
	"testing"
	"time"
)

func TestEmptyResponse(T *testing.T) {
	res := Response()
	if res.Header.Get("User-Agent") != "" {
		T.Error("Error: header parsing -> REQ.Header.Get")
	} else if res.Body() != nil {
		T.Error("Error: RES.Body function")
	} else if res.StatusCodeInt() != 0 {
		T.Error("Error: RES.StatusCode function")
	} else if res.ReasonString() != "" {
		T.Error("Error : " + res.StatusCodeString())
	} else if len(res.Header.GetAll()) != 0 {
		T.Error("Error: header parsing -> REQ.Header.GetAll")
	}
}

func TestReadingResponse(T *testing.T) {
	req := Request()
	res := Response()

	defer req.Close()
	defer res.Close()

	req.SetMethod("POST")
	req.SetURL("https://httpbin.org/post?name=ndoshy")

	req.SetBodyString(`{"name":"NdOShy"}`)

	clt := Client{
		Timeout:     time.Duration(20 * time.Second),
		TimeoutRead: time.Duration(20 * time.Second),
	}

	if err := clt.Connect(req); err != nil {
		T.Error("Error: Client.Connect function with 'httpbin.org' host")
	}

	if err := clt.Do(req, res); err != nil {
		T.Error("Error Client.Do: " + err.Error())
	}

	if res.StatusCodeInt() != 200 {
		T.Error("Error StatusCode: " + res.StatusCodeString())
	}

	if res.Body() == nil {
		T.Error("Error: RES.Body function")
	}

	if res.BodyString() == "" {
		T.Error("Error: RES.BodyString function")
	}
}
