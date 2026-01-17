package benchmark

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"testing"

	"golang.org/x/net/http2"
)

func BenchmarkNethttp(b *testing.B) {
	b.StopTimer()

	request, _ := http.NewRequest("POST", "https://localhost:8080", nil)

	tr := &http2.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	c := &http.Client{
		Transport: tr,
	}

	var bodyBuffer bytes.Buffer

	b.StartTimer()
	for Corrects != RequestsNumber && Errors != RequestsNumber {
		Body := []byte("number=" + strconv.Itoa(Corrects))

		bodyBuffer.Reset()
		bodyBuffer.Write(Body)
		request.Body = io.NopCloser(&bodyBuffer)
		request.ContentLength = int64(bodyBuffer.Len())

		resp, err := c.Do(request)
		if err != nil {
			Errors++
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		if bytes.Contains(Body, respBody) {
			Corrects++
		} else {
			Errors++
		}
	}

	fmt.Println("Corrects:", Corrects, "; Errors:", Errors)
	c.CloseIdleConnections()
}
