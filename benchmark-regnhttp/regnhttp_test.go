package regn

import (
	"crypto/tls"
	"fmt"
	"strconv"
	"strings"
	"testing"

	regn "github.com/xsxo/regnhttp"
)

var RequestsNumber int = 100 // 17773
var Errors int
var Corrects int

func BenchmarkRegnhttp(b *testing.B) {
	b.StopTimer()
	request := regn.Http2Request()
	defer request.Close()
	response := regn.Http2Response()
	defer response.Close()

	c := regn.Client{TLSConfig: &tls.Config{InsecureSkipVerify: true}}
	defer c.Close()

	request.SetURL("https://localhost:9911")
	request.SetMethod(regn.MethodPost)

	c.Http2Upgrade()
	if err := c.Connect(request); err != nil {
		panic(err)
	}

	b.StartTimer()

	for xo := 1; xo < RequestsNumber*2-2; xo += 2 {
		request.SetBodyString("number=" + strconv.Itoa(xo))
		if err := c.Http2SendRequest(request, uint32(xo)); err != nil {
			panic(err.Error())
		}
	}

	for xo := 1; xo < RequestsNumber*2-2; xo += 2 {
		if err := c.Http2ReadRespone(response, uint32(xo)); err != nil {
			Errors++
		} else if strings.Contains(response.BodyString(), "number="+strconv.Itoa(xo)) {
			Corrects++
		} else {
			Errors++
		}
	}

	fmt.Println("Corrects:", Corrects, "; Errors:", Errors)
	request.Close()
	response.Close()
	c.Close()
}
