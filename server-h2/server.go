package main

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"

	"golang.org/x/net/http2"
)

func handler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	w.Header().Del("Date")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	w.Write(body)
}

func main() {
	http.HandleFunc("/", handler)

	http2Server := &http2.Server{
		MaxConcurrentStreams: 5000000,
	}

	server := &http.Server{
		Addr: ":443",
		TLSConfig: &tls.Config{
			InsecureSkipVerify: true,
			NextProtos:         []string{"h2"},
		},
	}

	http2.ConfigureServer(server, http2Server)

	fmt.Println("Starting HTTPS server on https://localhost:443")
	err := server.ListenAndServeTLS("server.crt", "server.key")
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		os.Exit(1)
	}
}
