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
	} else if res.StatusCode() != 0 {
		T.Error("Error: RES.StatusCode function")
	} else if res.Reason() != "" {
		T.Error("Error : " + formatInt(res.StatusCode()))
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

	JsonREQ := map[string]string{"name": "NdOShy"}

	if NewErr := req.SetJson(JsonREQ); NewErr != nil {
		T.Error("Error: parsing map to json -> REQ.SetJson function")
	}

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

	if res.StatusCode() != 200 {
		T.Error("Error StatusCode: " + formatInt(res.StatusCode()))
	}

	if res.Body() == nil {
		T.Error("Error: RES.Body function")
	}

	if res.BodyString() == "" {
		T.Error("Error: RES.BodyString function")
	}

	JsonRES, Err := res.BodyJson()

	if Err != nil {
		T.Error("Error: parsing map to json -> RES.Json function")
	}

	JsonJson := JsonRES["json"].(map[string]interface{})

	if JsonJson["name"] != JsonREQ["name"] {
		T.Error("Error: json parsing -> RES.Json + REQ.SetJson functions")
	}

	if res.Header.Get("Connection") != "keep-alive" {
		T.Error("Error: header parsing -> res.Header.Get(Connection) function")
	}

}
