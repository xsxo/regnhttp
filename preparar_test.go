package regn

import (
	"bytes"
	"strings"
	"testing"
)

var req *RequestType

func prepare_request() {
	req = Request(4 * 1024)
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

func testMethod(t *testing.T) {
	req = Request(4 * 1024)

	req.SetMethod(MethodPut)
	if raw := req.RawString(); raw[:len(MethodPut)] != MethodPut || raw[len(MethodPut)] != ' ' {
		t.Error("error when prepare RequestType.SetMethod function `set`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}

	req.SetMethod(MethodConnect)
	if raw := req.RawString(); raw[:len(MethodConnect)] != MethodConnect || raw[len(MethodConnect)] != ' ' {
		t.Error("error when prepare RequestType.SetMethod function `from lower to upper`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}

	req.SetMethod(MethodPost)
	if raw := req.RawString(); raw[:len(MethodPost)] != MethodPost || raw[len(MethodPost)] != ' ' {
		t.Error("error when prepare RequestType.SetMethod function `from upper to lower`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}

}

func testURL(t *testing.T) {
	req = Request(4 * 1024)

	req.SetURL("https://localhost:8080/api")
	if !strings.Contains(req.RawString(), " /api ") {
		t.Error("error when prepare RequestType.SetURL function `from lower to upper`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}

	req.SetURL("https://localhost:8080/")
	if !strings.Contains(req.RawString(), " / ") {
		t.Error("error when prepare RequestType.SetURL function `from upper to lower`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}
}

func testHeader(t *testing.T) {
	req = Request(4 * 1024)

	req.Header.Set("lanugage", "english")
	if !strings.Contains(req.RawString(), "lanugage: english\r\n") {
		t.Error("error when prepare RequestType.HeaderSet function `set`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}

	req.Header.Set("lanugage", "en")
	if !strings.Contains(req.RawString(), "lanugage: en\r\n") {
		t.Error("error when prepare RequestType.HeaderSet function `from upper to lower`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}

	req.Header.Set("lanugage", "_english_")
	if !strings.Contains(req.RawString(), "lanugage: _english_\r\n") {
		t.Error("error when prepare RequestType.HeaderSet function `from lower to upper`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}
}

func testBody(t *testing.T) {
	req = Request(4 * 1024)

	req.SetBodyString("hello")
	if !strings.Contains(req.RawString(), "\r\n\r\nhello") {
		t.Error("error when prepare RequestType.SetBodyString function `set`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}

	req.SetBodyString("h")
	if !strings.Contains(req.RawString(), "\r\n\r\nh") {
		t.Error("error when prepare RequestType.SetBodyString function `from upper to lower`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}

	req.SetBodyString("hello world")
	if !strings.Contains(req.RawString(), "\r\n\r\nhello world") {
		t.Error("error when prepare RequestType.SetBodyString function `from lower to upper`\n" + strings.ReplaceAll(req.RawString(), "\r\n", "\\r\\n"))
	}
}

func Test_SetHeaders(t *testing.T) {
	prepare_request()
	if !bytes.Contains(req.Header.raw, []byte(" /api ")) {
		t.Error("error when prepare RequestType.SetURL function")
	} else if !bytes.Contains(req.Header.raw, []byte("Message1: ")) {
		t.Error("error when prepare RequestType.Header.Set function")
	} else if !bytes.Contains(req.Header.raw, []byte("Message2: ")) {
		t.Error("error when prepare RequestType.Header.Add function")
	} else if !bytes.Contains(req.Header.raw, []byte("\r\n\r\nREGN HTTP TEST BODY")) {
		t.Error("error when prepare RequestType.SetBody function")
	} else if !bytes.Contains(req.Header.raw, []byte("Content-Length: ")) {
		t.Error("error when prepare RequestType.SetBody function")
	}

	clear_request()
	if bytes.Contains(req.Raw(), []byte("Message1: ")) {
		t.Error("error when prepare RequestType.Header.Del function")
	} else if bytes.Contains(req.Raw(), []byte("Message2: ")) {
		t.Error("error when prepare RequestType.Header.Remove function")
	} else if bytes.Contains(req.Raw(), []byte("\r\n\r\nREGN HTTP TEST BODY")) {
		t.Error("error when prepare RequestType.SetBody function")
	}

	testMethod(t)
	testURL(t)
	testHeader(t)
	testBody(t)
}
