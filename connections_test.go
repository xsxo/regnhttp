package regn

import (
	"crypto/tls"
	"testing"
	"time"
)

func TestConnectFunction(T *testing.T) {
	req := Request()
	req.SetMethod("GET")
	req.SetURL("https://localhost:9911/")

	clt := Client{
		Timeout:     time.Duration(20 * time.Second),
		TimeoutRead: time.Duration(20 * time.Second),
		TLSConfig:   &tls.Config{InsecureSkipVerify: true},
	}

	if err := clt.Connect(req); err != nil {
		T.Error("Error: Client.Connect function with 'localhost' host")
	}

	clt.Close()
	req.Close()
}
