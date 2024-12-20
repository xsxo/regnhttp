package benchmark

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"testing"

	regn "github.com/xsxo/regnhttp"
)

func BenchmarkRegnhttp(b *testing.B) {
	b.StopTimer()
	request := regn.Http2Request()
	defer request.Close()
	response := regn.Http2Response()
	defer response.Close()

	c := &regn.Client{TlsConfig: &tls.Config{InsecureSkipVerify: true}}
	defer c.Close()

	request.SetURL("https://nghttp2.org/httpbin/post")
	request.SetMethod(regn.MethodPost)

	c.Http2Upgrade()
	c.Connect(request)

	b.StartTimer()

	for xo := 1; xo != int(c.Http2MaxIds); xo++ {
		fmt.Println("Done:", Corrects, "Error:", Errors)
		request.SetBodyString("number=" + strconv.Itoa(xo))
		if err := c.Http2SendRequest(request, uint32(xo)); err != nil {
			panic("error:" + err.Error())
		}
	}

	for xo := 1; xo != int(c.Http2MaxIds); xo++ {
		fmt.Println("Done:", Corrects, "Error:", Errors)
		if err := c.Http2ReadRespone(response, uint32(xo)); err != nil {
			Errors++
		} else if strings.Contains(response.BodyString(), "number="+strconv.Itoa(xo)) {
			Corrects++
		} else {
			Errors++
		}
	}

	fmt.Println("Done:", Corrects, "Error:", Errors)
	request.Close()
	response.Close()
	c.Close()
}
