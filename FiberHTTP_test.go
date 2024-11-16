package regn

import (
	"bytes"
	"strconv"
	"sync"
	"testing"
)

const number int = 4000000
const threads int = 200

var counter struct {
	ok int
	er int
}

func testregn() {
	var CN = Client{}
	defer CN.Close()

	REQ := Request()
	RES := Response()
	defer RES.Close()

	REQ.SetMethod("POST")
	REQ.SetURL("http://localhost:9911/")

	for counter.ok < number {
		NewBody := "number=" + strconv.Itoa(number)
		REQ.SetBody(NewBody)
		REQ.release()

		err := CN.Send(REQ, RES)
		if bytes.Contains(RES.Header.thebuffer.B, []byte(NewBody)) {
			counter.ok++
		} else if err != nil {
			counter.er++
		} else {
			counter.er++
		}
	}
}

func Benchmarkregn(b *testing.B) {
	var wg sync.WaitGroup
	for i := 0; i < threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			testregn()
		}()
	}
	wg.Wait()
}
