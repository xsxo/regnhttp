package main

import (
	"fmt"
	"strings"
	"time"

	regn "github.com/xsxo/regnhttp"
)

func main() {
	// create request object
	req := regn.Request()

	// create response objects
	res := regn.Response()

	// set post method of request
	req.SetMethod(regn.MethodPost)

	// set api url of request
	req.SetURL("http://i.instagram.com/api/v1/bloks/apps/com.bloks.www.fxim.settings.username.change.async/")

	// set headers of request
	req.Header.Set("User-Agent", "Instagram 309.1.0.41.113 Android (28/9; 240dpi; 720x1280; OnePlus; A5010; A5010; intel; en_US; 541635897)")
	req.Header.Set("X-Bloks-Version-Id", "9fc6a7a4a577456e492c189810755fe22a6300efc23e4532268bca150fe3e27a")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	// create client object
	cn := regn.Client{}

	// set timeout (not required)
	cn.Timeout = time.Duration(20 * time.Second)

	// set proxy (not required)
	proxy := "127.0.0.1:9911"
	cn.Proxy("http://" + proxy)

	// create connection with the server before send the request's (not required)
	// note: this function used for one time before send the request's don't set the function if for loop
	cn.Connect(req)

	stop := false
	for !stop {
		// change authorization without need to create new request
		req.Header.Set("Authorization", "Base64-Code")

		// set body of request
		req.SetBodyString("username=" + "TargetUsernameHere" + "&identity_ids_DEPRECATED=" + "FbidHere" + "&family_device_id=1873897-72ea-45a1-baa3-b112600520303102cf&operation_type=MUTATE")

		// send the request + check response
		if err := cn.Do(req, res); err != nil {
			fmt.Println("Error Send:", err.Error())
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
