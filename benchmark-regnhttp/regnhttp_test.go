package regn

import (
	"crypto/tls"
	"fmt"

	// "fmt"
	"strconv"
	"strings"
	"testing"

	regn "github.com/xsxo/regnhttp"
)

var RequestsNumber int = 59599
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

	for xo := 1; xo != RequestsNumber; xo += 2 {
		// fmt.Println(xo)
		request.SetBodyString("id=" + strconv.Itoa(xo))
		if err := c.Http2SendRequest(request, uint32(xo)); err != nil {
			panic(err.Error())
		}
	}

	for xo := 1; xo != RequestsNumber; xo += 2 {
		if xo == 1025 || xo == 4835 || xo == 8351 {
			continue
		}

		if err := c.Http2ReadRespone(response, uint32(xo)); err != nil {
			Errors++
		} else if strings.Contains(response.BodyString(), "id="+strconv.Itoa(xo)) {
			Corrects++
		} else {
			Errors++
		}
		// fmt.Println(response.BodyString())
	}

	fmt.Println("Corrects:", Corrects, "; Errors:", Errors)
	request.Close()
	response.Close()
	c.Close()
}
