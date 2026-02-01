package regn

import (
	"strings"
	"testing"
)

var r *RequestType = Request(32 * 6 * 1024)

func prepare_request_new() {
	r.SetMethod(MethodPost)
	r.SetURL("https://localhost:8080/api")
	r.Header.Set("Key1", "REGN HTTP v0.0.0-rc @xsxo - GitHub.com")
	r.Header.Set("Other1", "REGN HTTP v0.0.0-rc @xsxo - GitHub.com")
	r.SetBody([]byte("REGN HTTP TEST BODY"))
}

func Test_prepareNew(t *testing.T) {
	prepare_request_new()

	methods := []string{MethodConnect, MethodDelete, MethodGet, MethodHead, MethodOptions, MethodPatch}

	for xo := 0; xo != len(methods); xo++ {
		method := methods[xo]
		r.SetMethod(method)
		if r.RawString()[:strings.Index(r.RawString(), " ")] != method {
			t.Error(`r.RawString()[:strings.Index(r.RawString(), " ")] != method`)
		} else if strings.Count(r.RawString(), method) != 1 {
			t.Error(`strings.Count(r.RawString(), method) != 1`)
		}
	}

	for xo := len(methods) - 1; xo != 0; xo-- {
		method := methods[xo]
		r.SetMethod(method)
		if r.RawString()[:strings.Index(r.RawString(), " ")] != method {
			t.Error(`r.RawString()[:strings.Index(r.RawString(), " ")] != method`)
		} else if strings.Count(r.RawString(), method) != 1 {
			t.Error(`strings.Count(r.RawString(), method) != 1`)
		}
	}

	for xo := 0; xo != 4096; xo++ {
		str := strings.Repeat("hello_world", xo)

		r.SetURL("https://localhost/" + str + "/")
		r.Header.Set("Key1", str)
		r.Header.Add("Other1", str)
		r.SetBodyString(str)

		indexBody := strings.Index(r.RawString(), "\r\n\r\n") + 4
		indexURL1 := strings.Index(r.RawString(), " ") + 1
		indexURL2 := strings.Index(r.RawString()[indexURL1:], " ") + indexURL1

		if r.RawString()[indexURL1:indexURL2] != "/"+str+"/" {
			t.Error(`r.RawString()[indexURL1:indexURL2] != "/"+str+"/"`)
		} else if strings.Count(r.RawString(), "Key1") != 1 {
			t.Error(`strings.Count(r.RawString(), "Key1")`)
		} else if !strings.Contains(r.RawString(), "Key1: "+str+"\r\n") {
			t.Error(`!strings.Contains(r.RawString(), "Key1: "+str+"\r\n")`)
		} else if strings.Count(r.RawString(), "Other1") != 1 {
			t.Error(`strings.Count(r.RawString(), "Other1")`)
		} else if !strings.Contains(r.RawString(), "Other1: "+str+"\r\n") {
			t.Error(`!strings.Contains(r.RawString(), "Other1: "+str+"\r\n")`)
		} else if r.RawString()[indexBody:] != str || len(r.RawString()[indexBody:]) != len(str) {
			t.Error(`r.RawString()[indexBody:] != str`)
		}

		r.SetURL("https://localhost/")
		r.Header.Del("Key1")
		r.Header.Remove("Other1")
		r.SetBody(nil)

		indexBody = strings.Index(r.RawString(), "\r\n\r\n") + 4
		indexURL1 = strings.Index(r.RawString(), " ") + 1
		indexURL2 = strings.Index(r.RawString()[indexURL1:], " ") + indexURL1
		if r.RawString()[indexURL1:indexURL2] != "/" {
		} else if strings.Count(r.RawString(), "Key1") != 0 {
			t.Error(`strings.Count(r.RawString(), "Key1")`)
		} else if strings.Contains(r.RawString(), "Key1: "+str+"\r\n") {
			t.Error(`strings.Contains(r.RawString(), "Key1: "+str+"\r\n")`)
		} else if strings.Count(r.RawString(), "Other1") != 0 {
			t.Error(`strings.Count(r.RawString(), "Other1")`)
		} else if strings.Contains(r.RawString(), "Other1: "+str+"\r\n") {
			t.Error(`strings.Contains(r.RawString(), "Other1: "+str+"\r\n")`)
		} else if r.RawString()[indexBody:] != "" {
			t.Error(`r.RawString()[indexBody:] != ""`)
		} else if strings.Contains(r.RawString(), "hello_world") {
			t.Error(`strings.Contains(r.RawString(), "hello_world")`)
		}
	}

	for xo := 4096; xo != 0; xo-- {
		str := strings.Repeat("hello_world", xo)

		r.SetURL("https://localhost/" + str + "/")
		r.Header.Set("Key1", str)
		r.Header.Add("Other1", str)
		r.SetBodyString(str)

		indexBody := strings.Index(r.RawString(), "\r\n\r\n") + 4

		indexURL1 := strings.Index(r.RawString(), " ") + 1
		indexURL2 := strings.Index(r.RawString()[indexURL1:], " ") + indexURL1

		if r.RawString()[indexURL1:indexURL2] != "/"+str+"/" {
			t.Error(`r.RawString()[indexURL1:indexURL2] != "/"+str+"/"`)
		} else if strings.Count(r.RawString(), "Key1") != 1 {
			t.Error(`strings.Count(r.RawString(), "Key1")`)
		} else if !strings.Contains(r.RawString(), "Key1: "+str+"\r\n") {
			t.Error(`!strings.Contains(r.RawString(), "Key1: "+str+"\r\n")`)
		} else if strings.Count(r.RawString(), "Other1") != 1 {
			t.Error(`strings.Count(r.RawString(), "Other1")`)
		} else if !strings.Contains(r.RawString(), "Other1: "+str+"\r\n") {
			t.Error(`!strings.Contains(r.RawString(), "Other1: "+str+"\r\n")`)
		} else if r.RawString()[indexBody:] != str || len(r.RawString()[indexBody:]) != len(str) {
			t.Error(`r.RawString()[indexBody:] != str`)
		}

		r.SetURL("https://localhost/")
		r.Header.Del("Key1")
		r.Header.Remove("Other1")
		r.SetBody(nil)

		indexBody = strings.Index(r.RawString(), "\r\n\r\n") + 4
		indexURL1 = strings.Index(r.RawString(), " ") + 1
		indexURL2 = strings.Index(r.RawString()[indexURL1:], " ") + indexURL1
		if r.RawString()[indexURL1:indexURL2] != "/" {
		} else if strings.Count(r.RawString(), "Key1") != 0 {
			t.Error(`strings.Count(r.RawString(), "Key1")`)
		} else if strings.Contains(r.RawString(), "Key1: "+str+"\r\n") {
			t.Error(`strings.Contains(r.RawString(), "Key1: "+str+"\r\n")`)
		} else if strings.Count(r.RawString(), "Other1") != 0 {
			t.Error(`strings.Count(r.RawString(), "Other1")`)
		} else if strings.Contains(r.RawString(), "Other1: "+str+"\r\n") {
			t.Error(`strings.Contains(r.RawString(), "Other1: "+str+"\r\n")`)
		} else if r.RawString()[indexBody:] != "" {
			t.Error(`r.RawString()[indexBody:] != ""`)
		} else if strings.Contains(r.RawString(), "hello_world") {
			t.Error(`strings.Contains(r.RawString(), "hello_world")`)
		}
	}
}
