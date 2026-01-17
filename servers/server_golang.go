package main

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"sync"
)

const (
	HOST = "localhost"
	PORT = "8080"
)

var (
	tow_lines []byte = []byte{13, 10, 13, 10}
)

func recvSend(conn net.Conn) {
	defer conn.Close()

	for {
		buffer := make([]byte, 1024)
		data, _ := conn.Read(buffer)
		buffer = append(buffer, buffer[:data]...)

		if bytes.Contains(buffer, tow_lines) {
			parts := bytes.SplitN(buffer, tow_lines, 2)[1]

			response := []byte("HTTP/1.1 200 OK\r\n" +
				"Content-Type: text/html\r\n" +
				"Content-Length: " + strconv.Itoa(len(string(parts))) + "\r\n" +
				"\r\n")

			conn.Write(append(response, parts...))

		} else {
			response := []byte("HTTP/1.1 200 OK\r\n" +
				"Content-Type: text/html\r\n" +
				"Content-Length: 0" + "\r\n" +
				"\r\n")

			conn.Write(response)
		}
	}
}

func runcode() {
	listener, err := net.Listen("tcp4", HOST+":"+PORT)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
	defer listener.Close()

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		go recvSend(clientConn)
	}
}

func main() {

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		runcode()
	}()

	fmt.Println("Server listening on http://" + HOST + ":" + PORT)
	wg.Wait()

}
