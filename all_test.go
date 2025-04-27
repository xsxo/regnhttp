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

	r.Header.Add("head-one", "changed")
	r.SetBodyString("changed body")
	if err := c.Do(r, s); err != nil {
		t.Error("Do function 1")
	} else if !strings.Contains(s.BodyString(), `"Head-One": "changed"`) {
		t.Error("Header.Add function -> Change")
	} else if !strings.Contains(s.BodyString(), `changed body`) {
		t.Error("r.SetBodyString function -> Change")
	}

	r.Header.Del("head-one")
	r.SetBody(nil)
	if err := c.Do(r, s); err != nil {
		t.Error("Do function 2")
	} else if strings.Contains(s.BodyString(), `"Head-One": "changed"`) {
		t.Error("Del Head function")
	} else if strings.Contains(s.BodyString(), `changed body`) {
		t.Error("r.SetBodyString function -> nil")
	}

	r.SetMethod("GET")
	r.SetURL("https://www.instagram.com/0_11/")
	r.Header.Add("Accept", "*/*")
	r.Header.Add("Accept-Language", "en-US,en;q=0.5")
	r.Header.Add("X-IG-App-ID", "936619743392459")
	r.Header.Add("X-ASBD-ID", "359341")
	r.Header.Add("X-IG-WWW-Claim", "0")
	r.Header.Add("X-Requested-With", "XMLHttpRequest")
	r.Header.Add("DNT", "1")
	r.Header.Add("Sec-GPC", "1")
	r.Header.Add("Sec-Fetch-Dest", "empty")
	r.Header.Add("Sec-Fetch-Mode", "cors")
	r.Header.Add("Sec-Fetch-Site", "same-origin")
	r.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:138.0) Gecko/tsd-34sj Firefox/138.0")

	if err := c.Do(r, s); err != nil {
		t.Error("Do function 3")
	} else if !strings.Contains(s.BodyString(), `</html>`) {
		t.Error("Reading Html Page")
	}

	c.Close()
	r.Close()
	s.Close()
}
