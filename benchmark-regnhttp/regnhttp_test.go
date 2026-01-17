package regn

import (
	"fmt"
	"sync"

	// "fmt"

	"testing"
)

var RequestsNumber int = 9000
var Errors int
var Corrects int
var Groups sync.WaitGroup

func BenchmarkRegnhttp(b *testing.B) {
	b.StopTimer()

	// c := regn.Client{TLSConfig: &tls.Config{InsecureSkipVerify: true}}
	// defer c.Close()

	// c.Http2Upgrade()

	// r := regn.Http2Request()
	// r.SetMethod(regn.MethodPost)
	// r.SetURL("https://localhost:9911")
	// if err := c.Connect(r); err != nil {
	// 	panic(err)
	// }
	// r.Close()

	// b.StartTimer()

	// request := regn.Http2Request()
	// request.SetMethod(regn.MethodPost)
	// request.SetURL("https://localhost:9911")

	// responses := []*regn.ResponseType{}
	// for xo := 0; xo != RequestsNumber; xo++ {
	// 	s := regn.Http2Response()
	// 	body := "id=" + strconv.Itoa(xo)
	// 	request.SetBodyString(body)
	// 	if err := c.Http2WriteRequest(request, s); err != nil {
	// 		Errors++
	// 		continue
	// 	}
	// 	responses = append(responses, s)
	// }

	// for xo, s := range responses {
	// 	if err := c.Http2ReadRespone(s); err != nil {
	// 		Errors++
	// 		continue
	// 	} else if strings.Contains(s.BodyString(), "id="+strconv.Itoa(xo)) {
	// 		Corrects++
	// 	} else {
	// 		Errors++
	// 	}
	// }

	fmt.Println("Corrects:", Corrects, "; Errors:", Errors)
	// r.Close()
	// c.Close()
}
