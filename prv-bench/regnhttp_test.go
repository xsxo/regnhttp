package regn

import (
	"bytes"
	"crypto/tls"
	"fmt"

	// "fmt"
	"strconv"
	"testing"

	regn "github.com/xsxo/regnhttp"
)

var RequestsNumber int = 34796 * 6
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

	for xo := 1; xo != RequestsNumber; xo += 1 {
		raw_body := []byte("id=" + strconv.Itoa(xo))
		request.SetBody(raw_body)

		if err := c.Do(request, response); err != nil {
			Errors++
			fmt.Println(err.Error())
		} else if bytes.Contains(response.Body(), raw_body) {
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
