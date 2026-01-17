package regn

import (
	"bytes"
	"testing"
)

var req *RequestType

func prepare_request() {
	req = Request()
	req.SetMethod(MethodPost)
	req.SetURL("https://localhost:8080/api")
	req.Header.Set("Message1", "REGN HTTP v0.0.0-rc @xsxo - GitHub.com")
	req.Header.Set("Message2", "REGN HTTP v0.0.0-rc @xsxo - GitHub.com")
	req.SetBody([]byte("REGN HTTP TEST BODY"))
}

func clear_request() {
	req.Header.Del("Message1")
	req.Header.Remove("Message2")
	req.SetBody(nil)
}

func Test_SetHeaders(t *testing.T) {
	prepare_request()
	if !bytes.Contains(req.Header.raw.B, []byte(" /api ")) {
		t.Error("error when prepare RequestType.SetURL function")
	} else if !bytes.Contains(req.Header.raw.B, []byte("Message1: ")) {
		t.Error("error when prepare RequestType.Header.Set function")
	} else if !bytes.Contains(req.Header.raw.B, []byte("Message2: ")) {
		t.Error("error when prepare RequestType.Header.Add function")
	} else if !bytes.Contains(req.Header.raw.B, []byte("\r\n\r\nREGN HTTP TEST BODY")) {
		t.Error("error when prepare RequestType.SetBody function")
	} else if !bytes.Contains(req.Header.raw.B, []byte("Content-Length: ")) {
		t.Error("error when prepare RequestType.SetBody function")
	}

	clear_request()
	if bytes.Contains(req.Header.raw.B, []byte("Message1: ")) {
		t.Error("error when prepare RequestType.Header.Del function")
	} else if bytes.Contains(req.Header.raw.B, []byte("Message2: ")) {
		t.Error("error when prepare RequestType.Header.Remove function")
	} else if bytes.Contains(req.Header.raw.B, []byte("\r\n\r\nREGN HTTP TEST BODY")) {
		t.Error("error when prepare RequestType.SetBody function")
	}
}
