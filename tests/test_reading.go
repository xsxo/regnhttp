package regn

import (
	"strconv"
	"testing"
)

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
		Timeout:  10,
		Deadline: 10,
	}

	if err := clt.Connect(req); err != nil {
		T.Error("Error: Client.Connect function with 'httpbin.org' host")
	}

	if err := clt.Do(req, res); err != nil {
		T.Error("Error Client.Do: " + err.Error())
	}

	if res.StatusCode() != 200 {
		T.Error("Error StatusCode: " + strconv.Itoa(res.StatusCode()))
	}

	JsonRES, Err := res.Json()

	if Err != nil {
		T.Error("Error: parsing map to json -> RES.Json function")
	}

	JsonJson := JsonRES["Json"].(map[string]interface{})

	if JsonJson["name"] != JsonREQ["name"] {
		T.Error("Error: json parsing -> RES.Json + REQ.SetJson functions")
	}

	if res.Header.Get("User-Agent") == "" {
		T.Error("Error: header parsing -> REQ.Header.Set(User-Agent) function")
	}

}
