package regn_test

import (
	"strings"
	"testing"

	regn "github.com/xsxo/regnhttp"
)

var r *regn.RequestType = regn.Request(32 * 6 * 1024)

func prepare_request_new() {
	r.SetMethod(regn.MethodPost)
	r.SetURL("https://localhost:8080/api")
	r.Header.Set("Key1", "REGN HTTP v0.0.0-rc @xsxo - GitHub.com")
	r.Header.Set("Other1", "REGN HTTP v0.0.0-rc @xsxo - GitHub.com")
	r.SetBody([]byte("REGN HTTP TEST BODY"))
}

func Test_prepareNew(t *testing.T) {
	prepare_request_new()

	methods := []string{regn.MethodConnect, regn.MethodDelete, regn.MethodGet, regn.MethodHead, regn.MethodOptions, regn.MethodPatch}

	for xo := 0; xo != len(methods); xo++ {
		method := methods[xo]
		r.SetMethod(method)
		if r.String()[:strings.Index(r.String(), " ")] != method {
			t.Error(`r.String()[:strings.Index(r.String(), " ")] != method`)
		} else if strings.Count(r.String(), method) != 1 {
			t.Error(`strings.Count(r.String(), method) != 1`)
		}
	}

	for xo := len(methods) - 1; xo != 0; xo-- {
		method := methods[xo]
		r.SetMethod(method)
		if r.String()[:strings.Index(r.String(), " ")] != method {
			t.Error(`r.String()[:strings.Index(r.String(), " ")] != method`)
		} else if strings.Count(r.String(), method) != 1 {
			t.Error(`strings.Count(r.String(), method) != 1`)
		}
	}

	for xo := 0; xo != 4096; xo++ {
		str := strings.Repeat("hello_world", xo)

		r.SetURL("https://localhost/" + str + "/")
		r.Header.Set("Key1", str)
		r.Header.Add("Other1", str)
		r.SetBodyString(str)

		indexBody := strings.Index(r.String(), "\r\n\r\n") + 4
		indexURL1 := strings.Index(r.String(), " ") + 1
		indexURL2 := strings.Index(r.String()[indexURL1:], " ") + indexURL1

		if r.String()[indexURL1:indexURL2] != "/"+str+"/" {
			t.Error(`r.String()[indexURL1:indexURL2] != "/"+str+"/"`)
		} else if strings.Count(r.String(), "Key1") != 1 {
			t.Error(`strings.Count(r.String(), "Key1")`)
		} else if !strings.Contains(r.String(), "Key1: "+str+"\r\n") {
			t.Error(`!strings.Contains(r.String(), "Key1: "+str+"\r\n")`)
		} else if strings.Count(r.String(), "Other1") != 1 {
			t.Error(`strings.Count(r.String(), "Other1")`)
		} else if !strings.Contains(r.String(), "Other1: "+str+"\r\n") {
			t.Error(`!strings.Contains(r.String(), "Other1: "+str+"\r\n")`)
		} else if r.String()[indexBody:] != str || len(r.String()[indexBody:]) != len(str) {
			t.Error(`r.String()[indexBody:] != str`)
		}

		r.SetURL("https://localhost/")
		r.Header.Del("Key1")
		r.Header.Remove("Other1")
		r.SetBody(nil)

		indexBody = strings.Index(r.String(), "\r\n\r\n") + 4
		indexURL1 = strings.Index(r.String(), " ") + 1
		indexURL2 = strings.Index(r.String()[indexURL1:], " ") + indexURL1
		if r.String()[indexURL1:indexURL2] != "/" {
		} else if strings.Count(r.String(), "Key1") != 0 {
			t.Error(`strings.Count(r.String(), "Key1")`)
		} else if strings.Contains(r.String(), "Key1: "+str+"\r\n") {
			t.Error(`strings.Contains(r.String(), "Key1: "+str+"\r\n")`)
		} else if strings.Count(r.String(), "Other1") != 0 {
			t.Error(`strings.Count(r.String(), "Other1")`)
		} else if strings.Contains(r.String(), "Other1: "+str+"\r\n") {
			t.Error(`strings.Contains(r.String(), "Other1: "+str+"\r\n")`)
		} else if r.String()[indexBody:] != "" {
			t.Error(`r.String()[indexBody:] != ""`)
		} else if strings.Contains(r.String(), "hello_world") {
			t.Error(`strings.Contains(r.String(), "hello_world")`)
		}
	}

	for xo := 4096; xo != 0; xo-- {
		str := strings.Repeat("hello_world", xo)

		r.SetURL("https://localhost/" + str + "/")
		r.Header.Set("Key1", str)
		r.Header.Add("Other1", str)
		r.SetBodyString(str)

		indexBody := strings.Index(r.String(), "\r\n\r\n") + 4

		indexURL1 := strings.Index(r.String(), " ") + 1
		indexURL2 := strings.Index(r.String()[indexURL1:], " ") + indexURL1

		if r.String()[indexURL1:indexURL2] != "/"+str+"/" {
			t.Error(`r.String()[indexURL1:indexURL2] != "/"+str+"/"`)
		} else if strings.Count(r.String(), "Key1") != 1 {
			t.Error(`strings.Count(r.String(), "Key1")`)
		} else if !strings.Contains(r.String(), "Key1: "+str+"\r\n") {
			t.Error(`!strings.Contains(r.String(), "Key1: "+str+"\r\n")`)
		} else if strings.Count(r.String(), "Other1") != 1 {
			t.Error(`strings.Count(r.String(), "Other1")`)
		} else if !strings.Contains(r.String(), "Other1: "+str+"\r\n") {
			t.Error(`!strings.Contains(r.String(), "Other1: "+str+"\r\n")`)
		} else if r.String()[indexBody:] != str || len(r.String()[indexBody:]) != len(str) {
			t.Error(`r.String()[indexBody:] != str`)
		}

		r.SetURL("https://localhost/")
		r.Header.Del("Key1")
		r.Header.Remove("Other1")
		r.SetBody(nil)

		indexBody = strings.Index(r.String(), "\r\n\r\n") + 4
		indexURL1 = strings.Index(r.String(), " ") + 1
		indexURL2 = strings.Index(r.String()[indexURL1:], " ") + indexURL1
		if r.String()[indexURL1:indexURL2] != "/" {
		} else if strings.Count(r.String(), "Key1") != 0 {
			t.Error(`strings.Count(r.String(), "Key1")`)
		} else if strings.Contains(r.String(), "Key1: "+str+"\r\n") {
			t.Error(`strings.Contains(r.String(), "Key1: "+str+"\r\n")`)
		} else if strings.Count(r.String(), "Other1") != 0 {
			t.Error(`strings.Count(r.String(), "Other1")`)
		} else if strings.Contains(r.String(), "Other1: "+str+"\r\n") {
			t.Error(`strings.Contains(r.String(), "Other1: "+str+"\r\n")`)
		} else if r.String()[indexBody:] != "" {
			t.Error(`r.String()[indexBody:] != ""`)
		} else if strings.Contains(r.String(), "hello_world") {
			t.Error(`strings.Contains(r.String(), "hello_world")`)
		}
	}
}
