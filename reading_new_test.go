package regn

import (
	"strings"
	"testing"
)

func TestReadingNew(t *testing.T) {
	c := Client{}
	r := Request(4 * 1024)
	s := Response(4096 * 1024)

	r.SetMethod("GET")
	r.SetURL("https://github.com/")
	r.Header.Set("Accept", "*/*")
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
	r.SetBody(nil)
	if err := c.Do(r, s); err != nil {
		t.Error("Do function 3")
	} else if s.ReasonString() != "200 OK" {
		t.Error("Reading chuncked response 'ReasonString'")
	} else if !strings.Contains(s.BodyString(), "</html>") {
		t.Error("Reading chuncked response 'BodyString'")
	}

	r.SetMethod("GET")
	r.SetURL("https://error_server.error/")
	if err := c.Do(r, s); err == nil {
		t.Error("Do function 4")
	} else if s.RawString() != "" {
		t.Error("Clean response 's.RawString()'")
	} else if s.Body() != nil {
		t.Error("Clean response 's.Body()'")
	} else if s.StatusCodeInt() != 0 {
		t.Error("Clean response 's.StatusCodeInt()'")
	} else if s.Reason() != nil {
		t.Error("Clean response 's.StatusCodeInt()'")
	}

	c.Close()
	r.Close()
	s.Close()
}
