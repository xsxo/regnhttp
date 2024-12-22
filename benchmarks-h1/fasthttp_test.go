package benchmark

import (
	"strconv"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func BenchmarkFasthttp(b *testing.B) {
	b.StopTimer()

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()

	request.SetRequestURI("http://localhost")
	request.Header.SetMethod(fasthttp.MethodPost)

	c := &fasthttp.Client{}

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
