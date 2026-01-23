package main

// import library
import (
	"fmt"
	"strings"
	"sync"
	"time"

	regn "github.com/xsxo/regnhttp"
)

func Example() {
	// create client object for each goroutine
	clt := regn.Client{}

	// set timeout connection
	clt.Timeout = time.Duration(20 * time.Second) // not required
	clt.TimeoutRead = time.Duration(20 * time.Second)

	// to set proxy connection
	// clt.Proxy("http://username:password@host:port")
	// clt.Proxy("http://host:port")
	// ex: clt.Proxy("http://username:password@localhost:11")

	// create request object
	// 4 * 1024 is the buffer size of request (length of raw request)
	// use len(raw.Request()) to get buffer size of request
	req := regn.Request(4 * 1024)
	defer req.Close()

	// create response object with buffer size
	// 4 * 1024 is the buffer size of response (length of raw response)
	// use len(raw.Response()) to get buffer size of response
	res := regn.Response(4 * 1024)
	defer res.Close()

	// set meothod
	req.SetMethod(regn.MethodPost)

	// set url request + params
	req.SetURL("https://httpbin/post?name=ndoshy")

	// set header
	req.Header.Set("Authorization", "base64-code")

	// set body
	req.SetBodyString("Hello World!")

	// create connection with server before send request
	err := clt.Connect(req) // not required

	// check error
	if err != nil {
		panic(err.Error())
	} else {
		fmt.Println("connected with 'httpbin.org' host")
	}

	// create automaticly response object
	err = clt.Do(req, res)

	if err != nil {
		panic("Err: " + err.Error())
	}

	// read status code response
	fmt.Println(res.StatusCode())

	// read string body response
	fmt.Println(res.BodyString())

	// read string body response
	responseBody := res.BodyString()

	if strings.Contains(responseBody, `"name":"ndoshy"`) {
		fmt.Println(true)
	} else {
		fmt.Println(false)
	}

}

func main() {

	// create client each function
	// !! regn Client is'nt support pool connection

	var wg sync.WaitGroup
	for xo := 0; xo != 2; xo++ {
		wg.Add(1) // Increment the counter for each goroutine
		go func() {
			defer wg.Done()
			Example()
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()

}
