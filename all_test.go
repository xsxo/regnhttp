package regn

import (
	"strings"
	"testing"
)

func Test_HTTP1_1(t *testing.T) {
	c := Client{}
	r := Request()
	s := Response()

	r.SetMethod("POST")
	r.SetURL("https://httpbin.org/post")

	r.Header.Add("head-one", "added")
	r.SetBodyString("added body")
	if err := c.Do(r, s); err != nil {
		t.Error("Do function 0")
	} else if !strings.Contains(s.BodyString(), `"Head-One": "added"`) {
		t.Error("Header.Add function")
	} else if !strings.Contains(s.BodyString(), `added body`) {
		t.Error("r.SetBodyString function")
	}
	c.Close()

	r.Header.Add("head-tow", "changed")
	r.SetBodyString("changed body")
	if err := c.Do(r, s); err != nil {
		t.Error("Do function 1")
	} else if !strings.Contains(s.BodyString(), `"Head-Tow": "changed"`) {
		t.Error("Header.Add function -> Change")
	} else if !strings.Contains(s.BodyString(), `changed body`) {
		t.Error("r.SetBodyString function -> Change")
	}
	c.Close()

	r.Header.Del("head-tow")
	r.SetBody(nil)
	if err := c.Do(r, s); err != nil {
		t.Error("Do function 2")
	} else if strings.Contains(s.BodyString(), `"Head-Tow": "changed"`) {
		t.Error("Del Head function")
	} else if strings.Contains(s.BodyString(), `changed body`) {
		t.Error("r.SetBodyString function -> nil")
	}
	c.Close()

	r.SetMethod("GET")
	r.SetURL("https://nasa.com/")

	if err := c.Do(r, s); err != nil {
		t.Error("Do function 3")
	} else if !strings.Contains(s.BodyString(), `</html>`) {
		t.Error("Reading Html Page")
	}

	c.Close()
	r.Close()
	s.Close()
}
