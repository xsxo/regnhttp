package benchmark

import (
	"crypto/tls"
	"strconv"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func BenchmarkFasthttp(b *testing.B) {
	b.StopTimer()

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()

	request.SetRequestURI("https://localhost")
	request.Header.SetMethod(fasthttp.MethodPost)

	c := &fasthttp.Client{TLSConfig: &tls.Config{InsecureSkipVerify: true}}

	b.StartTimer()
	for Corrects != RequestsNumber && Errors != RequestsNumber {
		stringBody := "number=" + strconv.Itoa(Corrects)
		request.SetBodyString(stringBody)
		if err := c.Do(request, response); err != nil {
			Errors++
		} else if strings.Contains(string(response.Body()), stringBody) {
			Corrects++
		} else {
			Errors++
		}
	}

	fasthttp.ReleaseRequest(request)
	fasthttp.ReleaseResponse(response)
	c.CloseIdleConnections()
}
