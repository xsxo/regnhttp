package regn

import (
	"crypto/tls"
	"testing"
)

func TestConnectFunction(T *testing.T) {
	req := Request()
	req.SetMethod("GET")
	req.SetURL("https://localhost:9911/")

	clt := Client{
		Timeout:     10,
		TimeoutRead: 10,
		TLSConfig:   &tls.Config{InsecureSkipVerify: true},
	}

	if err := clt.Connect(req); err != nil {
		T.Error("Error: Client.Connect function with 'localhost' host")
	}

	clt.Close()
	req.Close()
}
