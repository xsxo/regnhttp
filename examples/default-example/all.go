package main

// import library
import (
	"fmt"
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
	// ex: clt.Proxy("http://username:password@localhost:9911")

	// create request object
	req := regn.Http2Request()
	defer req.Close()

	// create response object
	res := regn.Http2Response()
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

	// read json response
	Json, err := res.BodyJson()

	if err != nil {
		panic("Err: " + err.Error())
	}

	args := Json["args"].(map[string]interface{})
	if args["name"] == "ndoshy" {
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
