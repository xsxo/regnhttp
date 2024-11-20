package regn

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"net"
	"net/url"
	"strconv"
	"time"
)

type __inforamtion__ struct {
	use_proxy      bool
	host_connected string
	connection     net.Conn
	run            bool

	peeker  *bufio.Reader
	flusher *bufio.Writer

	authorization string
	host_proxy    string
	port_proxy    string
}

type Client struct {
	confgiuration *__inforamtion__
	Timeout       int
	Deadline      int
}

func (e *RegnError) Error() string {
	return "RegnHTTP Error: " + e.Message
}

func ___connect_net___(host string, port string, thetimeout int) (net.Conn, error) {
	if thetimeout == 0 {
		thetimeout = 10
	}

	if port != "443" {
		TheConn, err := net.DialTimeout("tcp", host+":"+port, time.Duration(thetimeout)*time.Second)

		if err != nil {
			er := &RegnError{}
			er.Message = "Field connection with '" + host + "' host\nnethttp Error: " + err.Error()
			return nil, er
		}

		// TheConn = net.Conn(TheConn)
		return TheConn, nil

	} else {
		TheConn, err := tls.Dial("tcp4", host+":"+port, &tls.Config{ServerName: host})

		if err != nil {
			er := &RegnError{}
			er.Message = "Field connection with '" + host + "' host"
			return nil, er
		}

		// TheConn := net.Conn(TlsConn)
		// return &TheConn
		return TheConn, nil
	}
}

func ___connect_to_host___(cn *Client, host_port string, authorization string) error {
	// therequest := []byte("CONNECT " + host_port + " HTTP/1.1\r\n" + authorization + "Host: " + host_port + "\r\n\r\n")

	cn.create_line()

	therequest := Request()
	therequest.SetMethod("CONNECT")
	therequest.theybytesapi = host_port
	therequest.Header.Set("Host", host_port)
	therequest.Header.Set("Authorization", authorization)
	therequest.release()

	cn.confgiuration.flusher.Write(therequest.Header.raw.Bytes())
	if err := cn.confgiuration.flusher.Flush(); err != nil {
		// cn.confgiuration.connection.Close()
		cn.confgiuration.connection.Close()
		cn.close_line()
		return &RegnError{Message: "Field connection with '" + host_port + "' addr"}
	}
	therequest.Close()

	cn.confgiuration.peeker.Peek(1)

	res, err := cn.confgiuration.peeker.Peek(cn.confgiuration.peeker.Buffered())

	if err != nil {
		cn.confgiuration.connection.Close()
		cn.close_line()
		cn.confgiuration.connection = nil
		return &RegnError{Message: "Field connection with '" + host_port + "' addr"}
	}

	if len(code_regexp.FindSubmatch(res)) == 0 {
		cn.confgiuration.connection.Close()
		cn.close_line()
		cn.confgiuration.connection = nil
		return &RegnError{Message: "Field connection with '" + host_port + "' addr"}
	}

	return nil
}

func (cn *Client) Porxy(Url string) {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	}
	if cn.confgiuration.connection != nil {
		cn.confgiuration.connection.Close()
		cn.close_line()
		cn.confgiuration.connection = nil
	}

	cn.confgiuration.host_connected = ""
	cn.confgiuration.use_proxy = true

	Parse, err := url.Parse(Url)
	if err != nil {
		return
	}

	if Parse.Host == "" {
		panic("REGN HTTP error: No host proxy url supplied.")
	} else if Parse.Port() == "" {
		panic("REGN HTTP error: No host proxy url supplied.")
	}

	cn.confgiuration.host_proxy = Parse.Hostname()
	cn.confgiuration.port_proxy = Parse.Port()

	if Parse.User.Username() != "" {

		password, _ := Parse.User.Password()
		credentials := Parse.User.Username() + ":" + password
		cn.confgiuration.authorization = ""
		cn.confgiuration.authorization = base64.StdEncoding.EncodeToString([]byte(credentials))
	}
}

func (cn *Client) Close() {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	}

	if cn.confgiuration.connection != nil {
		cn.confgiuration.connection.Close()
		cn.close_line()
		cn.confgiuration.connection = nil
	}

	cn.confgiuration.host_connected = ""
}

func (cn *Client) close_line() {
	if cn.confgiuration.peeker != nil {
		nrpool.Put(cn.confgiuration.peeker)
	}

	if cn.confgiuration.flusher != nil {
		nwpool.Put(cn.confgiuration.flusher)
	}
}

func (cn *Client) create_line() {
	cn.close_line()
	cn.confgiuration.peeker = get_reader(cn.confgiuration.connection)
	cn.confgiuration.flusher = get_writer(cn.confgiuration.connection)
}

func (cn *Client) Connect(REQ *RequestType) error {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	}

	if cn.confgiuration.run {
		cn.confgiuration.run = false
		panic("REGN HTTP Error: The client struct isn't support pool connections\ncreate a client for each connection || use sync.Pool for pool connections")
	}

	if cn.confgiuration.use_proxy {
		if cn.confgiuration.connection == nil {
			var err error
			cn.confgiuration.connection, err = ___connect_net___(cn.confgiuration.host_proxy, cn.confgiuration.port_proxy, cn.Timeout)
			if err != nil {
				return err
			}

			if err := ___connect_to_host___(cn, REQ.Header.myhost+":"+REQ.Header.myport, cn.confgiuration.authorization); err != nil {
				return err
			}

			cn.confgiuration.host_connected = REQ.Header.myhost

		} else if cn.confgiuration.host_connected != REQ.Header.myhost {
			if err := ___connect_to_host___(cn, REQ.Header.myhost+":"+REQ.Header.myport, cn.confgiuration.authorization); err != nil {
				return err
			}

			cn.confgiuration.host_connected = REQ.Header.myhost

		}

		if REQ.Header.myport == "443" {
			config := &tls.Config{
				ServerName: REQ.Header.myhost,
			}
			tlsConn := tls.Client(cn.confgiuration.connection, config)
			// tcc := net.Conn(tlsConn)
			cn.confgiuration.connection = tlsConn
		}

	} else {
		if cn.confgiuration.connection == nil {
			var err error
			cn.confgiuration.connection, err = ___connect_net___(REQ.Header.myhost, REQ.Header.myport, cn.Timeout)

			if err != nil {
				return err
			}
			cn.confgiuration.host_connected = REQ.Header.myhost
			cn.create_line()

		} else if cn.confgiuration.host_connected != REQ.Header.myhost {
			var err error
			if cn.confgiuration.connection != nil {
				cn.close_line()
				cn.confgiuration.connection.Close()
				cn.confgiuration.connection = nil
			}

			cn.confgiuration.connection, err = ___connect_net___(REQ.Header.myhost, REQ.Header.myport, cn.Timeout)
			if err != nil {
				return err
			}

			cn.confgiuration.host_connected = REQ.Header.myhost
			cn.create_line()
		}
	}

	return nil
}

func (cn *Client) Send(REQ *RequestType, RES *ResponseType) error {
	if NewErr := cn.Connect(REQ); NewErr != nil {
		cn.confgiuration.run = false
		return NewErr
	}

	cn.confgiuration.run = true

	if REQ.Header.raw.Len() == 0 {
		if NewErr := REQ.release(); NewErr != nil {
			cn.confgiuration.run = false
			return NewErr
		}
	}

	if cn.Deadline != 0 {
		cn.confgiuration.connection.SetDeadline(time.Now().Add(time.Duration(cn.Deadline) * time.Second))
		cn.Deadline = 0
	}

	cn.confgiuration.flusher.Write(REQ.Header.raw.Bytes())
	if NewErr := cn.confgiuration.flusher.Flush(); NewErr != nil {
		cn.confgiuration.connection.Close()
		cn.close_line()
		cn.confgiuration.connection = nil
		cn.confgiuration.host_connected = ""
		cn.confgiuration.run = false
		return &RegnError{Message: " Error writing: " + NewErr.Error()}
	}

	RES.Header.thebuffer.Reset()
	for {
		cn.confgiuration.peeker.Peek(1)
		le := cn.confgiuration.peeker.Buffered()
		test, _ := cn.confgiuration.peeker.Peek(le)
		RES.Header.thebuffer.Write(test)
		cn.confgiuration.peeker.Discard(le)

		if bytes.Contains(RES.Header.thebuffer.B, tow_lines) {
			contentLengthMatch := contetre.FindSubmatch(RES.Header.thebuffer.B) // changed form (*RES.Header.thebuffer).B
			if len(contentLengthMatch) > 1 {
				contentLength, _ := strconv.Atoi(string(contentLengthMatch[1]))
				if len(bytes.SplitN(RES.Header.thebuffer.B, tow_lines, 2)[1]) >= contentLength {
					break
				}
			} else if bytes.Contains(RES.Header.thebuffer.B, zero_lines) {
				break
			}

		} else if le == 0 {
			cn.confgiuration.connection.Close()
			cn.close_line()
			cn.confgiuration.connection = nil
			cn.confgiuration.host_connected = ""

			cn.confgiuration.run = false

			return nil
		}
	}

	cn.confgiuration.run = false
	return nil

}

func (cn *Client) Do(REQ *RequestType, RES *ResponseType) error {
	return cn.Send(REQ, RES)
}

func (cn *Client) Action(REQ *RequestType, RES *ResponseType) error {
	return cn.Send(REQ, RES)
}
