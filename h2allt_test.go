package regn

import (
	"fmt"
	"strings"
	"testing"
)

func Test_HTTP_2(t *testing.T) {
	c := Client{}
	c.Http2Upgrade()

	r := Http2Request()
	r.SetMethod("POST")
	r.SetURL("https://nghttp2.org/httpbin/post")
	r.Header.Add("head-one", "added")
	s := Http2Response()

	r.Header.Add("head-one", "added")
	r.SetBodyString("added body")
	if err := c.Do(r, s); err != nil {
		panic("Do function 0; err: " + err.Error())
	} else if !strings.Contains(s.BodyString(), `"Head-One":"added"`) {
		fmt.Println(s.BodyString())
		panic("Header.Add function")
	} else if !strings.Contains(s.BodyString(), `added body`) {
		fmt.Println(s.BodyString())
		panic("r.SetBodyString function")
	}
	c.Close()

	r.Header.Add("head-tow", "changed")
	r.SetBodyString("changed body")
	if err := c.Do(r, s); err != nil {
		panic("Do function 1; err: " + err.Error())
	} else if !strings.Contains(s.BodyString(), `"Head-Tow":"changed"`) {
		fmt.Println(s.BodyString())
		panic("Header.Add function 1")
	} else if !strings.Contains(s.BodyString(), `changed body`) {
		fmt.Println(s.BodyString())
		panic("r.SetBodyString function 1")
	}
	c.Close()

	r.Header.Del("head-tow")
	r.SetBody(nil)
	if err := c.Do(r, s); err != nil {
		panic("Do function 2; err: " + err.Error())
	} else if strings.Contains(s.BodyString(), `"Head-Tow"`) {
		fmt.Println(s.BodyString())
		panic("Header.Del function")
	} else if strings.Contains(s.BodyString(), `changed body`) {
		fmt.Println(s.BodyString())
		panic("r.SetBody function")
	}

	r.SetMethod("GET")
	r.SetURL("https://nasa.com/")

	if err := c.Do(r, s); err != nil {
		panic("Do function 3; err" + err.Error())
	} else if !strings.Contains(s.BodyString(), `</html>`) {
		fmt.Println(s.BodyString())
		panic("Reading Html Page")
	}

	c.Close()
	r.Close()
	s.Close()
}
