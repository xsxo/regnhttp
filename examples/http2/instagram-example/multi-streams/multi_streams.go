package main

import (
	"fmt"
	"strings"

	regn "github.com/xsxo/regnhttp"
)

func main() {
	// create http2 request object
	req := regn.Http2Request()

	// create http2 response objects
	res := regn.Http2Response()

	// set post method of request
	req.SetMethod(regn.MethodPost)

	// set api url of request
	req.SetURL("http://i.instagram.com/api/v1/bloks/apps/com.bloks.www.fxim.settings.username.change.async/")

	// set headers of request
	req.Header.Set("User-Agent", "Instagram 309.1.0.41.113 Android (28/9; 240dpi; 720x1280; OnePlus; A5010; A5010; intel; en_US; 541635897)")
	req.Header.Set("X-Bloks-Version-Id", "9fc6a7a4a577456e492c189810755fe22a6300efc23e4532268bca150fe3e27a")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	// create http2 client object
	cn := regn.Client{}
	cn.Http2Upgrade()

	// set timeout (not required)
	cn.Timeout = 40

	// set proxy (not required)
	proxy := "127.0.0.1:9911"
	cn.Proxy("http://" + proxy)

	// create connection with the server before send the request's (not required)
	// note: this function used for one time before send the request's don't set the function if for loop
	cn.Connect(req)

	max_streams := cn.Http2MaxStreams()

	loop := 1
	for loop < int(max_streams) {
		loop += 2
		// change authorization without need to create new request
		req.Header.Set("Authorization", "Base64-Code")

		// set body of request
		req.SetBodyString("username=" + "TargetUsernameHere" + "&identity_ids_DEPRECATED=" + "FbidHere" + "&family_device_id=1873897-72ea-45a1-baa3-b112600520303102cf&operation_type=MUTATE")

		// send http2 request with odd stream id (require odd stream id)
		cn.Http2SendRequest(req, uint32(loop))
	}

	for loop != 1 {
		loop -= 2

		// read http response with odd stream id you send (require odd stream id)
		if err := cn.Http2ReadRespone(res, uint32(loop)); err != nil {
			fmt.Println("Error Send:", err.Error())
			break
		} else {
			if strings.Contains(res.BodyString(), `\"`+"TargetUsernameHere"+`\"`) {
				fmt.Println("catch it")
				break
			} else if strings.Contains(res.BodyString(), `\"Username is not available\"`) {
				fmt.Println("attempt")
			} else {
				fmt.Println("ResponseError:", res.BodyString())
			}
		}
	}
}
