package regn

import (
	"strings"
	"testing"
)

func TestReading(t *testing.T) {
	c := Client{}
	r := Request(4 * 1024)
	s := Response(4 * 1024)

	r.SetMethod("POST")
	r.SetURL("http://localhost:8080/python3")
	r.Header.Add("Lang", "Python3")
	r.SetBodyString("Hi From FirstBody")

	if !strings.Contains(r.RawString(), "Lang: Python3\r\n") {
		t.Error("Request.Header.Add function 0")
	} else if !strings.Contains(r.RawString(), "\r\n\r\nHi From FirstBody") {
		t.Error("Request.SetBodyString function 0")
	} else if !strings.Contains(r.RawString(), "POST") {
		t.Error("Request.SetMethod function 0")
	} else if !strings.Contains(r.RawString(), "/python3") {
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

	if !strings.Contains(r.RawString(), "Lang: Golang\r\n") || strings.Contains(r.RawString(), "Lang: Python3\r\n") {
		t.Error("Request.Header.Add function 1")
	} else if !strings.Contains(r.RawString(), "\r\n\r\nHi From LastBody") || strings.Contains(r.RawString(), "\r\n\r\nHi From FirstBody") {
		t.Error("Request.SetBodyString function 1")
	} else if !strings.Contains(r.RawString(), "PUT") || strings.Contains(r.RawString(), "POST") {
		t.Error("Request.SetMethod function 1")
	} else if !strings.Contains(r.RawString(), "/golang") || strings.Contains(r.RawString(), "/python3") {
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
