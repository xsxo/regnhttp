package fiberhttp

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"log"
	"net"
	"net/url"
	"strconv"
	"time"
)

type __inforamtion__ struct {
	use_proxy      bool
	host_connected string
	connection     net.Conn
	thereader      *bufio.Reader
	thewriter      *bufio.Writer
	run            bool

	host_proxy    string
	port_proxy    string
	authorization []byte
}

type FiberhttpError struct {
	Message string
}

type Client struct {
	confgiuration *__inforamtion__
	Timeout       int
}

func (e *FiberhttpError) Error() string {
	return "FiberHTTP Error :" + e.Message
}

func ___connect_net___(host string, port string, thetimeout int) (net.Conn, error) {
	if port != "443" {
		TheConn, err := net.DialTimeout("tcp4", host+":"+port, time.Duration(thetimeout)*time.Second)
		// TheConn, err := net.Dial("tcp", host+":"+port)
		if err != nil {
			er := &FiberhttpError{}
			er.Message = "Field connection with '" + host + "' host"
			return nil, er
		}

		return TheConn, nil

	} else {
		TheConn, err := tls.Dial("tcp4", host+":"+port, &tls.Config{ServerName: host})

		if err != nil {
			er := &FiberhttpError{}
			er.Message = "Field connection with '" + host + "' host"
			return nil, er
		}

		return TheConn, nil
	}
}

func ___connect_to_host___(__my *__inforamtion__, host_port string, authorization string) error {
	er := &FiberhttpError{}
	therequest := []byte("CONNECT " + host_port + " HTTP/1.1\r\n" + authorization + "Host: " + host_port + "\r\n\r\n")

	buffer := make([]byte, 4096)

	__my.connection.Write(therequest)
	if _, err := __my.connection.Write(therequest); err != nil {
		er.Message = "Field connection with '" + host_port + "' addr"
		__my.connection.Close()
		__my.connection = nil
		return er
	}

	data, err := __my.connection.Read(buffer)

	if err != nil {
		er.Message = "Field connection with '" + host_port + "' addr"
		__my.connection.Close()
		__my.connection = nil
		return er
	}

	if len(code_regexp.FindSubmatch(buffer[:data])) == 0 {
		er.Message = "Field connection with '" + host_port + "' addr"
		__my.connection.Close()
		__my.connection = nil
		return er
	}

	return nil
}

func (cn *Client) Porxy(Url string) {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	}
	if cn.confgiuration.connection != nil {
		cn.confgiuration.connection.Close()
		cn.confgiuration.connection = nil
	}

	cn.confgiuration.host_connected = ""
	cn.confgiuration.use_proxy = true

	Parse, err := url.Parse(Url)
	if err != nil {
		return
	}

	if Parse.Host == "" {
		log.SetFlags(0)
		log.Fatalln("Fiberhttp error: No host proxy url supplied.")
		return
	} else if Parse.Port() == "" {
		log.SetFlags(0)
		log.Fatalln("Fiberhttp error: No host proxy url supplied.")
		return
	}

	cn.confgiuration.host_proxy = Parse.Hostname()
	cn.confgiuration.port_proxy = Parse.Port()

	if Parse.User.Username() != "" {

		password, _ := Parse.User.Password()
		credentials := Parse.User.Username() + ":" + password
		cn.confgiuration.authorization = nil

		cn.confgiuration.authorization = append(cn.confgiuration.authorization, proxybasic[:]...)
		cn.confgiuration.authorization = append(cn.confgiuration.authorization, []byte(base64.StdEncoding.EncodeToString([]byte(credentials)))...)
		cn.confgiuration.authorization = append(cn.confgiuration.authorization, []byte{13, 10}...)
	}
}

func (cn *Client) Close() {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	}

	if cn.confgiuration.connection != nil {
		cn.confgiuration.connection.Close()
		cn.confgiuration.connection = nil
	}
}

func (cn *Client) Connect(REQ request) error {

	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	}

	if cn.Timeout == 0 {
		cn.Timeout = 10
	}

	if cn.confgiuration.use_proxy {
		if cn.confgiuration.connection == nil {
			var err error
			cn.confgiuration.connection, err = ___connect_net___(cn.confgiuration.host_proxy, cn.confgiuration.port_proxy, cn.Timeout)
			if err != nil {
				return err
			}

			if err := ___connect_to_host___(cn.confgiuration, REQ.Header.myhost+":"+REQ.Header.myport, string(cn.confgiuration.authorization)); err != nil {
				return err
			}

			cn.confgiuration.host_connected = REQ.Header.myhost

		} else if cn.confgiuration.host_connected != REQ.Header.myhost {
			if err := ___connect_to_host___(cn.confgiuration, REQ.Header.myhost+":"+REQ.Header.myport, string(cn.confgiuration.authorization)); err != nil {
				return err
			}

			cn.confgiuration.host_connected = REQ.Header.myhost

		}

		if REQ.Header.myport == "443" {
			config := &tls.Config{
				ServerName: REQ.Header.myhost,
			}
			cn.confgiuration.connection = tls.Client(cn.confgiuration.connection, config)
		}

		cn.confgiuration.thereader = bufio.NewReader(cn.confgiuration.connection)
		cn.confgiuration.thewriter = bufio.NewWriter(cn.confgiuration.connection)

	} else {
		if cn.confgiuration.connection == nil {
			var err error
			cn.confgiuration.connection, err = ___connect_net___(REQ.Header.myhost, REQ.Header.myport, cn.Timeout)

			if err != nil {
				return err
			}
			cn.confgiuration.host_connected = REQ.Header.myhost

		} else if cn.confgiuration.host_connected != REQ.Header.myhost {
			var err error
			if cn.confgiuration.connection != nil {
				cn.confgiuration.connection.Close()
				cn.confgiuration.connection = nil
			}

			cn.confgiuration.connection, err = ___connect_net___(REQ.Header.myhost, REQ.Header.myport, cn.Timeout)
			if err != nil {
				return err
			}

			cn.confgiuration.host_connected = REQ.Header.myhost
		}
		cn.confgiuration.thereader = bufio.NewReader(cn.confgiuration.connection)
		cn.confgiuration.thewriter = bufio.NewWriter(cn.confgiuration.connection)
	}

	return nil
}

func (cn *Client) Send(REQ request) (readresponse, error) {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	}

	if cn.confgiuration.run {
		log.SetFlags(0)
		log.Fatalln("fiberhttp error: the client isn't support pool connections\ncreate a client for each connection")
	}

	cn.confgiuration.run = true

	if NewErr := cn.Connect(REQ); NewErr != nil {
		cn.confgiuration.run = false
		return readresponse{}, NewErr
	}

	if REQ.Header.raw == nil {
		if NewErr := REQ.release(); NewErr != nil {
			cn.confgiuration.run = false
			return readresponse{}, NewErr
		}
	}

	cn.confgiuration.thewriter.Write(REQ.Header.raw)

	if NewErr := cn.confgiuration.thewriter.Flush(); NewErr != nil {
		cn.confgiuration.connection.Close()
		cn.confgiuration.connection = nil
		cn.confgiuration.host_connected = ""
		cn.confgiuration.run = false
		return readresponse{}, &FiberhttpError{Message: " Error writing '" + NewErr.Error() + "'"}
	}

	thend := bytes_pool.Get()
	thend.Reset()

	to_return := readresponse{Header: &headers_struct{}}
	var buffed int = 1
	for {
		test, err := cn.confgiuration.thereader.Peek(buffed)

		if err != nil {
			cn.confgiuration.connection.Close()
			cn.confgiuration.connection = nil
			cn.confgiuration.host_connected = ""
			cn.confgiuration.run = false
			return readresponse{content: nil, Header: &headers_struct{theybytesheaders: nil}}, &FiberhttpError{Message: "Error reading"}
		}

		cn.confgiuration.thereader.Discard(buffed)
		buffed = cn.confgiuration.thereader.Buffered()
		thend.Write(test)

		if bytes.Contains(thend.B, tow_lines[:]) {
			parts := bytes.SplitN(thend.B, tow_lines[:], 2)
			to_return.Header.theybytesheaders = append(to_return.Header.theybytesheaders, parts[0]...)
			to_return.content = append(to_return.content, parts[1]...)
			// headers.B = parts[0]
			// body.B = parts[1]

			contentLengthMatch := contetre.FindSubmatch(to_return.Header.theybytesheaders)
			if len(contentLengthMatch) > 1 {
				contentLength, _ := strconv.Atoi(string(contentLengthMatch[1]))
				if len(to_return.content) >= contentLength {
					break
				}
			} else if bytes.Contains(thend.B, zero_lines[:]) {
				break
			}
		}
	}

	bytes_pool.Put(thend)
	cn.confgiuration.run = false

	return to_return, nil
}

func (cn *Client) Do(REQ request) (readresponse, error) {
	return cn.Send(REQ)
}

func (cn *Client) Action(REQ request) (readresponse, error) {
	return cn.Send(REQ)
}
