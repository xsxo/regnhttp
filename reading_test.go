package regn

import (
	"strings"
	"testing"
)

func TestReading(t *testing.T) {
	c := Client{}
	r := Request()
	s := Response()

	r.SetMethod("POST")
	r.SetURL("http://localhost:8080/python3")
	r.Header.Add("Lang", "Python3")
	r.SetBodyString("Hi From FirstBody")

	if !strings.Contains(r.Header.raw.String(), "Lang: Python3\r\n") {
		t.Error("Request.Header.Add function 0")
	} else if !strings.Contains(r.Header.raw.String(), "\r\n\r\nHi From FirstBody") {
		t.Error("Request.SetBodyString function 0")
	} else if !strings.Contains(r.Header.raw.String(), "POST") {
		t.Error("Request.SetMethod function 0")
	} else if !strings.Contains(r.Header.raw.String(), "/python3") {
		t.Error("Request.SetURL function 0")
	}

	if err := c.Do(r, s); err != nil {
		t.Error("Do function 0")
	} else if s.Header.Get("Lang") != "Python3" {
		t.Error("Response.Header.GET function 0")
	} else if s.BodyString() != "Hi From FirstBody" {
		t.Error("Response.BodyString function 0")
	} else if s.StatusCodeInt() != 200 {
		t.Error("Response.StatusCodeInt function 0")
	} else if s.StatusCodeString() != "200" {
		t.Error("Response.StatusCodeString function 0")
	} else if s.StatusCodeInt() != 200 {
		t.Error("Response.StatusCodeString function 0")
	} else if s.ReasonString() != "200 OK" {
		t.Error("Response.StatusCodeString function 0")
	}

	r.SetMethod("PUT")
	r.SetURL("http://localhost:11/golang")
	r.Header.Add("Lang", "Golang")
	r.SetBodyString("Hi From LastBody")

	if strings.Contains(r.RawString(), "POST") || !strings.Contains(r.RawString(), "PUT") {
		t.Error("Response.Header.GET function 1")
	}

	if !strings.Contains(r.Header.raw.String(), "Lang: Golang\r\n") || strings.Contains(r.Header.raw.String(), "Lang: Python3\r\n") {
		t.Error("Request.Header.Add function 1")
	} else if !strings.Contains(r.Header.raw.String(), "\r\n\r\nHi From LastBody") || strings.Contains(r.Header.raw.String(), "\r\n\r\nHi From FirstBody") {
		t.Error("Request.SetBodyString function 1")
	} else if !strings.Contains(r.Header.raw.String(), "PUT") || strings.Contains(r.Header.raw.String(), "POST") {
		t.Error("Request.SetMethod function 1")
	} else if !strings.Contains(r.Header.raw.String(), "/golang") || strings.Contains(r.Header.raw.String(), "/python3") {
		t.Error("Request.SetURL function 1")
	}

	if err := c.Do(r, s); err != nil {
		t.Error("Do function 1")
	} else if s.Header.Get("Lang") != "Golang" || strings.Contains(s.Header.Get("Lang"), "Python3") {
		t.Error("Response.Header.GET function 1")
	} else if s.BodyString() != "Hi From LastBody" || strings.Contains(s.BodyString(), "Hi From FirstBody") {
		t.Error("Response.BodyString function 1")
	} else if s.StatusCodeInt() != 200 {
		t.Error("Response.StatusCodeInt function 1")
	} else if s.StatusCodeString() != "200" {
		t.Error("Response.StatusCodeString function 1")
	} else if s.StatusCodeInt() != 200 {
		t.Error("Response.StatusCodeString function 0")
	} else if s.ReasonString() != "200 OK" {
		t.Error("Response.StatusCodeString function 0")
	}

	c.Close()
	r.SetMethod("GET")
	r.SetURL("https://github.com/")
	r.SetBody(nil)
	if err := c.Do(r, s); err != nil {
		t.Error("Do function 3")
	} else if !strings.Contains(s.BodyString(), `</html>`) {
		t.Error("Reading Html Page")
	}

	c.Close()
	r.Close()
	s.Close()
}
