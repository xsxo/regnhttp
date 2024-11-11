package main

// import library
import (
	"fmt"
	"log"
	"time"

	fiberhttp "github.com/xsxo/fiberhttp-go"
)

func Example() {
	// create client object for each goroutine
	clt := fiberhttp.Client{}

	// set timeout connection
	clt.Timeout = 20 // not required

	// to set proxy connection
	// clt.Porxy("http://username:password@host:port")
	// clt.Porxy("http://host:port")

	// create request object
	req := fiberhttp.Request()

	// set meothod
	req.SetMethod("GET")

	// set url
	req.SetURL("https://httpbin.org/get?name=ndoshy")

	// create connection with server before send request
	err := clt.Connect(req) // not required

	if err != nil {
		log.SetFlags(0)
		log.Fatal("Err:", err.Error())
	} else {
		fmt.Println("connected with 'httpbin.org' host")
	}

	// create automaticly response object
	res, err := clt.Do(req)

	if err != nil {
		log.SetFlags(0)
		log.Fatal("Err:", err.Error())
	}

	// print status code of response
	fmt.Println(res.StatusCode())

	// read response with json format
	Json, err := res.Json()

	if err != nil {
		log.SetFlags(0)
		log.Fatal("Err:", err.Error())
	}

	args := Json["args"].(map[string]interface{})
	if args["name"] == "ndoshy" {
		fmt.Println(true)
	} else {
		fmt.Println(false)
	}

}

func main() {

	for xo := 0; xo != 2; xo++ {
		go Example()
	}

	time.Sleep(200 * time.Second)

}
