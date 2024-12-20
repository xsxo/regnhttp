package benchmark

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/valyala/fasthttp"
)

func BenchmarkFasthttp(b *testing.B) {
	b.StopTimer()

	request := fasthttp.AcquireRequest()
	response := fasthttp.AcquireResponse()

	request.SetRequestURI("http://localhost:9911")
	request.Header.SetMethod(fasthttp.MethodPost)

	c := &fasthttp.Client{}

	b.StartTimer()
	fmt.Println("start doing")
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

	fmt.Println("Done:", Corrects, "Error:", Errors)

	fasthttp.ReleaseRequest(request)
	fasthttp.ReleaseResponse(response)
	c.CloseIdleConnections()
}
