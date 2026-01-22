package regn

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"net"
	"net/url"
	"time"
)

type Client struct {
	// Timeout Connection
	Timeout time.Duration

	// Timeout Reading Response
	TimeoutRead time.Duration

	// Tls Context
	TLSConfig *tls.Config

	// Dialer to create dial connection
	Dialer *net.Dialer

	// Buffer Size of Writer Requsts (default value is 4096)
	WriteBufferSize int

	// Buffer Size of Reader Responses (default value is 4096)
	ReadBufferSize int

	// Net connection of Client
	NetConnection net.Conn

	// Off Nangle
	SetNoDelay bool
	NagleOff   bool

	useProxy      bool
	hostConnected string
	run           bool

	peeker  *bufio.Reader
	flusher *bufio.Writer

	authorization string
	hostProxy     string
	portProxy     string
}

func (c *Client) Status() bool {
	if c.hostConnected != "" {
		return true
	} else {
		return false
	}
}

func (c *Client) connectNet(host string, port string) error {
	if c.Timeout.Seconds() == 0 {
		c.Timeout = time.Duration(20 * time.Second)
	}

	if c.TimeoutRead.Seconds() == 0 {
		c.TimeoutRead = c.Timeout
	}

	if c.Dialer == nil {
		c.Dialer = &net.Dialer{Timeout: c.Timeout}
	}

	var err error

	if port != "443" {
		c.NetConnection, err = c.Dialer.Dial("tcp", host+":"+port)
	} else {
		c.NetConnection, err = tls.DialWithDialer(c.Dialer, "tcp4", host+":"+port, c.TLSConfig)
	}

	if err != nil {
		return &RegnError{Message: "field create connection with '" + host + ":" + port + "' address\n" + err.Error()}
	}

	// c.NetConnection.(*net.TCPConn).SetKeepAlive(true)

	// c.NetConnection.(*net.TCPConn).SetReadBuffer(c.ReadBufferSize)
	// c.NetConnection.(*net.TCPConn).SetWriteBuffer(c.WriteBufferSize)

	c.NetConnection.SetReadDeadline(time.Now().Add(c.TimeoutRead))

	if c.SetNoDelay || c.NagleOff {
		c.NetConnection.(*net.TCPConn).SetNoDelay(true)
	}

	c.createLines()
	return nil
}

func (c *Client) connectHost(address string) error {
	c.flusher.WriteString("CONNECT " + address + " HTTP/1.1\r\n")
	c.flusher.WriteString("Host: " + address + "\r\n")

	if c.authorization != "" {
		c.flusher.WriteString("Proxy-Authorization: Basic " + c.authorization + "\r\n")
	}
	c.flusher.WriteString("\r\n")

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + address + "' address (Flush)"}
	}

	if raw, err := c.peeker.Peek(16); err != nil {
		return &RegnError{Message: "field proxy connection with '" + address + "' address (Peek)"}
	} else {
		if !bytes.Contains(raw, []byte{50, 48, 48}) {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + address + "' address (Contains)"}
		}
		c.peeker.Discard(c.peeker.Buffered())
	}

	return nil
}

func (c *Client) Proxy(Url string) {
	if c.hostConnected != "" {
		panic("can not set proxy after connect with server")
	}

	c.hostConnected = ""
	c.useProxy = true

	Parse, err := url.Parse(Url)
	if err != nil {
		panic("invalid proxy format")
	}

	if Parse.Hostname() == "" {
		panic("no hostname proxy url supplied")
	} else if Parse.Port() == "" {
		panic("no port proxy url supplied")
	}

	c.hostProxy = Parse.Hostname()
	c.portProxy = Parse.Port()

	if Parse.User.Username() != "" {
		password, _ := Parse.User.Password()
		credentials := Parse.User.Username() + ":" + password
		c.authorization = base64.StdEncoding.EncodeToString([]byte(credentials))
	} else {
		c.authorization = ""
	}
}

func (c *Client) Close() {
	if new, ok := c.NetConnection.(*tls.Conn); ok {
		if new != nil {
			new.Close()
			c.NetConnection.Close()
			c.NetConnection = nil
		}
	} else {
		if c.NetConnection != nil {
			c.NetConnection.Close()
			c.NetConnection = nil
		}
	}

	c.closeLines()
	c.hostConnected = ""
	c.run = false
}

func (c *Client) closeLines() {
	if c.peeker != nil {
		peekerPool.Put(c.peeker)
		c.peeker = nil
	}

	if c.flusher != nil {
		flusherPool.Put(c.flusher)
		c.flusher = nil
	}
}

func (c *Client) createLines() {
	c.closeLines()

	if c.ReadBufferSize == 0 {
		c.ReadBufferSize = 4096
	}

	if c.WriteBufferSize == 0 {
		c.WriteBufferSize = 4096
	}

	if new, ok := c.NetConnection.(*tls.Conn); ok {
		if new != nil {
			c.peeker = genPeeker(new, c.ReadBufferSize)
			c.flusher = genFlusher(new, c.WriteBufferSize)
		}
	} else {
		if c.NetConnection != nil {
			c.peeker = genPeeker(c.NetConnection, c.ReadBufferSize)
			c.flusher = genFlusher(c.NetConnection, c.WriteBufferSize)
		}
	}
}

func (c *Client) Connect(REQ *RequestType) error {
	if c.hostConnected != REQ.Header.myhost && c.hostConnected != "" {
		c.Close()
	}

	if c.TLSConfig == nil {
		c.TLSConfig = &tls.Config{}
		c.TLSConfig.InsecureSkipVerify = false
	}

	if c.run {
		c.Close()
		panic("concurrent client goroutines")
	}

	if c.hostConnected == "" {
		c.TLSConfig.ServerName = REQ.Header.myhost

		if c.useProxy {
			if err := c.connectNet(c.hostProxy, c.portProxy); err != nil {
				c.Close()
				return err
			}

			if err := c.connectHost(REQ.Header.myhost + ":" + REQ.Header.myport); err != nil {
				c.Close()
				return err
			}

			if REQ.Header.myport == "443" || REQ.Header.mytls {
				c.NetConnection = tls.Client(c.NetConnection, c.TLSConfig)
				c.createLines()
			}

		} else {
			if err := c.connectNet(REQ.Header.myhost, REQ.Header.myport); err != nil {
				c.Close()
				return err
			}

			if REQ.Header.mytls {
				c.NetConnection = tls.Client(c.NetConnection, c.TLSConfig)
				c.createLines()
			}
		}

		c.hostConnected = REQ.Header.myhost
	}

	return nil
}

// Not support goroutine-safe
func (c *Client) Do(REQ *RequestType, RES *ResponseType) error {

	if err := c.Connect(REQ); err != nil {
		return err
	}

	c.run = true
	if _, err := c.flusher.Write(REQ.Header.raw[:REQ.Header.position]); err != nil {
		c.Close()
		return err
	}

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return err
	}

	RES.Header.position = 0
	RES.Header.theBuffer = RES.Header.theBuffer[:0]
	RES.Header.theBuffer = RES.Header.theBuffer[:RES.Header.bufferSize]

	var indexB int
	var bufferd int
	var contentLength int = -1
	for contentLength != 0 {
		if contentLength == -1 {
			if _, err := c.peeker.Peek(1); err != nil {
				c.Close()
				return err
			}
			bufferd = c.peeker.Buffered()
		} else {
			bufferd = []int{contentLength, c.ReadBufferSize}[intToBool(contentLength > c.ReadBufferSize)]
		}

		raw, _ := c.peeker.Peek(bufferd)
		c.peeker.Discard(bufferd)

		if RES.Header.position+bufferd < RES.Header.bufferSize {
			copy(RES.Header.theBuffer[RES.Header.position:], raw)
			RES.Header.position += bufferd
		}

		if indexB == 0 && contentLength == -1 {
			indexB = bytes.Index(raw, lines[3:])
			if indexB == -1 {
				continue
			}
			indexL := bytes.Index(RES.Header.theBuffer, contentLengthKey) + 16
			if indexL == 15 {
				if raw[len(raw)-1] == 125 {
					break
				}
				continue
			}
			indexRN := bytes.Index(RES.Header.theBuffer[indexL:], lines[5:]) + indexL
			contentLength = bToInt(RES.Header.theBuffer[indexL:indexRN])
			contentLength -= len(raw[indexB+4:])
		} else if contentLength > 0 {
			contentLength -= len(raw)
		} else if raw[len(raw)-1] == 125 {
			break
		} else if bytes.Contains(raw, lines) {
			break
		}
	}

	c.run = false
	return nil
}
