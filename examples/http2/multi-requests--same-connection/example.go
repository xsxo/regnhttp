package main

import (
	"fmt"

	regn "github.com/xsxo/regnhttp"
)

func main() {
	c := regn.Client{}
	c.Http2Upgrade()

	// can use same request object with more than one response objects
	// but can't use same response object with more than one requests objects
	r := regn.Http2Request()
	r.SetMethod("GET")
	r.SetURL("https://ipinfo.io/json")

	// need to connect to can read Http2MaxStreams() function
	// cuz if client not connect with the server Http2MaxStreams() function will return 0
	if err := c.Connect(r); err != nil {
		panic(err)
	}

	responses := []*regn.ResponseType{}

	// max requests can send to server at same connection
	// most server will set this numberis 100 (default value of nginx server)
	max_requests := c.Http2MaxStreams()
	fmt.Println("max requests can send at same connection", max_requests)

	// send requests
	for xo := 0; xo != int(max_requests); xo++ {
		s := regn.Http2Response()
		if err := c.Http2WriteRequest(r, s); err != nil {
			panic(err)
		}
		responses = append(responses, s) // need to save response object to read it when the client reseve all packets
	}

	// read responses
	for loop, s := range responses {
		if err := c.Http2ReadRespone(s); err != nil {
			panic(err)
		}
		json, err := s.BodyJson()
		if err != nil {
			panic(err)
		}
		fmt.Println("loop:", loop, "request id:", s.Header.StreamId, "ip:", json["ip"])
	}
}
