package regn

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
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

	// Ipv6 option need to hostname and proxy server supported Ipv6
	// Its work with only sock5 proxy right now
	Ipv6 bool

	// Off Nangle
	SetNoDelay bool
	NagleOff   bool

	// Off DNS cache (use hostname directly)
	OFF_DNS_CACHE bool

	useProxy      bool
	hostConnected string
	run           bool

	peeker  *bufio.Reader
	flusher *bufio.Writer

	authorization string
	schemeProxy   string
	hostProxy     string
	portProxy     string
	userProxy     string
	passProxy     string
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
	if !c.OFF_DNS_CACHE {
		ip, err := HostToIp(host, c.Ipv6)
		if err != nil {
			return err
		}
		host = ip.String()
	}

	if port != "443" {
		c.NetConnection, err = c.Dialer.Dial("tcp", host+":"+port)
	} else {
		c.NetConnection, err = tls.DialWithDialer(c.Dialer, "tcp4", host+":"+port, c.TLSConfig)
	}

	if err != nil {
		return &RegnError{Message: "field create connection with '" + host + ":" + port + "' address\n" + err.Error()}
	}

	if (c.SetNoDelay || c.NagleOff) && port != "443" {
		c.NetConnection.(*net.TCPConn).SetNoDelay(true)
	}

	c.NetConnection.SetReadDeadline(time.Now().Add(c.TimeoutRead))

	c.createLines()
	return nil
}

func (c *Client) connectHTTP(address string) error {
	c.flusher.WriteString("CONNECT " + address + " HTTP/1.1\r\nHost: " + address + "\r\n")
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

func (c *Client) connectSOCKS4(ip net.IP, port string) error {
	c.flusher.Write([]byte{0x04, 0x01}) // ver, meth
	_ = binary.Write(c.flusher, binary.BigEndian, uint16(StringToInt(port)))
	c.flusher.Write(ip)
	c.flusher.WriteByte(0x00) // userid

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (Flush)"}
	}

	raw, err := c.peeker.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x5A {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (raw[1] != 0x5A) 1"}
	}
	c.peeker.Discard(c.peeker.Buffered())

	return nil
}

func (c *Client) connectSOCKS5(ip net.IP, port string) error {
	// ver, meth = open, auth
	if c.authorization != "" {
		c.flusher.Write([]byte{0x05, 0x01, 0x02})
	} else {
		c.flusher.Write([]byte{0x05, 0x01, 0x00})
	}

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (Flush)"}
	}

	raw, err := c.peeker.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x5A && raw[1] != 0x02 {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (raw[1] != 0x5A) 1"}
	}
	c.peeker.Discard(c.peeker.Buffered())

	if c.authorization != "" {
		c.flusher.WriteByte(0x01)
		c.flusher.WriteByte(byte(len(c.userProxy)))
		c.flusher.Write([]byte(c.userProxy))
		c.flusher.WriteByte(byte(len(c.passProxy)))
		c.flusher.Write([]byte(c.passProxy))
		if err := c.flusher.Flush(); err != nil {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (Flush)"}
		}

		raw, err = c.peeker.Peek(2)
		if err != nil {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (Peek)"}
		} else if raw[1] != 0x00 {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (raw[1] != 0x5A) 2"}
		}
		c.peeker.Discard(c.peeker.Buffered())
	}

	// ver, meth = connect, rsv
	c.flusher.Write([]byte{0x05, 0x01, 0x00})
	if c.Ipv6 {
		c.flusher.WriteByte(0x04) // IPv6
	} else {
		c.flusher.WriteByte(0x01) // IPv4
	}

	c.flusher.Write(ip)
	_ = binary.Write(c.flusher, binary.BigEndian, uint16(StringToInt(port)))

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (Flush)"}
	}

	raw, err = c.peeker.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x00 {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + ip.String() + ":" + port + "' address (raw[1] != 0x5A) 3"}
	}
	c.peeker.Discard(c.peeker.Buffered())

	return nil
}

func (c *Client) Proxy(Url string) {
	if c.hostConnected != "" {
		panic("can not set proxy after connect with server")
	}

	c.hostConnected = ""
	c.useProxy = true

	pparse, err := url.Parse(Url)
	if err != nil {
		panic("invalid proxy format")
	}
	if len(pparse.Scheme) > 6 {
		pparse.Scheme = pparse.Scheme[:6]
	}
	if pparse.Scheme != "http" && pparse.Scheme != "https" && pparse.Scheme != "socks4" && pparse.Scheme != "socks5" {
		panic("proxy scheme '" + pparse.Scheme + "' not supported")
	}

	c.schemeProxy = pparse.Scheme

	if pparse.Hostname() == "" {
		panic("no hostname proxy url supplied")
	} else if pparse.Port() == "" {
		panic("no port proxy url supplied")
	}

	c.hostProxy = pparse.Hostname()
	c.portProxy = pparse.Port()
	c.userProxy = pparse.User.Username()
	c.passProxy, _ = pparse.User.Password()

	if c.userProxy != "" {
		c.authorization = base64.StdEncoding.EncodeToString([]byte(c.userProxy + ":" + c.passProxy))
		if c.schemeProxy == "socks4" {
			panic("socks4 not support authorization")
		}
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

			ip, err := HostToIp(REQ.Header.myhost, c.Ipv6)
			if err != nil {
				return err
			}

			switch c.schemeProxy {
			case "https", "http":
				if err := c.connectHTTP(ip.String() + ":" + REQ.Header.myport); err != nil {
					c.Close()
					return err
				}
			case "socks4":
				if err := c.connectSOCKS4(ip, REQ.Header.myport); err != nil {
					c.Close()
					return err
				}
			case "socks5":
				if err := c.connectSOCKS5(ip, REQ.Header.myport); err != nil {
					c.Close()
					return err
				}
			}

			if REQ.Header.myport == "443" || REQ.Header.mytls || c.schemeProxy == "https" {
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
	RES.Header.position = 0
	RES.Header.theBuffer = RES.Header.theBuffer[:RES.Header.bufferSize]
	if err := c.Connect(REQ); err != nil {
		RES.Reset()
		c.Close()
		return err
	}

	c.run = true
	if _, err := c.flusher.Write(REQ.Header.raw[:REQ.Header.position]); err != nil {
		RES.Reset()
		c.Close()
		return err
	}

	if err := c.flusher.Flush(); err != nil {
		RES.Reset()
		c.Close()
		return err
	}

	var chunkedB bool
	var chunked int = -1
	var indexRNRN int = -1
	var bufferd int
	var contentLength int = -1
	for contentLength != 0 {
		if contentLength > 0 {
			if c.ReadBufferSize > contentLength {
				bufferd = contentLength
			} else {
				bufferd = c.ReadBufferSize
			}
			contentLength -= bufferd
		} else if chunked > 0 {
			if c.ReadBufferSize > chunked {
				bufferd = chunked
			} else {
				bufferd = c.ReadBufferSize
			}
			chunked -= bufferd
		} else {
			if _, err := c.peeker.Peek(1); err != nil {
				RES.Reset()
				c.Close()
				return err
			}
			bufferd = c.peeker.Buffered()
		}

		raw, err := c.peeker.Peek(bufferd)
		if err != nil {
			RES.Reset()
			c.Close()
			return err
		}

		_, err = c.peeker.Discard(bufferd)
		if err != nil {
			RES.Reset()
			c.Close()
			return err
		}

		if RES.Header.position+bufferd < RES.Header.bufferSize {
			copy(RES.Header.theBuffer[RES.Header.position:], raw)
			RES.Header.position += bufferd
		} else {
			RES.Header.theBuffer = append(RES.Header.theBuffer, raw...)
			RES.Header.position += bufferd
			RES.Header.bufferSize += bufferd
		}

		if chunkedB && chunked <= 0 {
			for {
				if indexRNRN > RES.Header.position {
					break
				}

				rn := bytes.Index(RES.Header.theBuffer[indexRNRN:RES.Header.position], line)
				if rn == -1 {
					break
				}

				start := indexRNRN
				end := indexRNRN + rn

				hex, b := hexBytesToInt(RES.Header.theBuffer[start:end])
				if !b || hex == 0 {
					contentLength = 0
					break
				} else if hex == 0 {
					contentLength = 0
					break
				}

				if len(RES.Header.theBuffer[end+2:RES.Header.position]) > hex {
					chunked = 0
				} else {
					chunked = hex - len(RES.Header.theBuffer[end+2:RES.Header.position])
				}

				indexRNRN += hex + 4 + len(RES.Header.theBuffer[start:end])
				break
			}
			continue
		}

		if indexRNRN == -1 {
			indexRNRN = bytes.Index(RES.Header.theBuffer[:RES.Header.position], lines)
			if indexRNRN == -1 {
				continue
			}

			indexL := bytes.Index(RES.Header.theBuffer[:RES.Header.position], contentLengthKey) + 16
			if indexL == 15 {
				if bytes.Contains(RES.Header.theBuffer[:indexRNRN], chunkedValue) {
					chunkedB = true
					indexRNRN += 4
				}
				continue
			}
			indexRN := bytes.Index(RES.Header.theBuffer[indexL:], line) + indexL
			contentLength = BytesToInt(RES.Header.theBuffer[indexL:indexRN])
			contentLength -= len(RES.Header.theBuffer[indexRNRN+4 : RES.Header.position])
		}
	}

	RES.Header.theBuffer = RES.Header.theBuffer[:RES.Header.position]
	c.run = false
	return nil
}
