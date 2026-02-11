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
	// Timeout Connection (this option make the connection dosen't wait forever)
	Timeout time.Duration

	// Timeout Reading Responses (this option make the read function dosen't wait forever)
	TimeoutRead time.Duration

	// Tls Context // SSL Context
	TLSConfig *tls.Config

	// Raw connection of the client
	Dialer *net.Dialer

	// Buffer Size of Writer Requsts (default value is 4096)
	WriteBufferSize int

	// Buffer Size of Reader Responses (default value is 4096)
	ReadBufferSize int

	// Net connection of Client
	NetConnection net.Conn

	// Ipv6 option need to hostname Ipv6
	// Support Ipv6 proxies (socks4 dose not support)
	// Support DNS cache (if use t his option the hostname will converted to Ipv6 and will saved in cache)
	Ipv6 bool

	// Off Nagle algorithm
	// Nagle algorithm: https://en.wikipedia.org/wiki/Nagle%27s_algorithm
	SetNoDelay bool
	NagleOff   bool

	boolPreRequst bool
	boolProxy     bool
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

// check status connection
func (c *Client) Status() bool {
	if c.hostConnected != "" {
		return true
	} else {
		return false
	}
}

// host & ip of connection
func (c *Client) Host() string {
	return c.hostConnected
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
		if c.Ipv6 && !c.boolProxy {
			c.NetConnection, err = c.Dialer.Dial("tcp6", host+":"+port)
		} else {
			c.NetConnection, err = c.Dialer.Dial("tcp4", host+":"+port)
		}
	} else {
		if c.Ipv6 && !c.boolProxy {
			c.NetConnection, err = tls.DialWithDialer(c.Dialer, "tcp6", host+":"+port, c.TLSConfig)
		} else {
			c.NetConnection, err = tls.DialWithDialer(c.Dialer, "tcp4", host+":"+port, c.TLSConfig)
		}
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

func (c *Client) connectHTTP(host string, port string) error {
	c.flusher.WriteString("CONNECT " + host + ":" + port + " HTTP/1.1\r\nHost: " + host + ":" + port + "\r\n")
	if c.authorization != "" {
		c.flusher.WriteString("Proxy-Authorization: Basic " + c.authorization + "\r\n")
	}
	c.flusher.Write(line)

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
	}

	if raw, err := c.peeker.Peek(16); err != nil {
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
	} else {
		if !bytes.Contains(raw, []byte{50, 48, 48}) {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Contains)"}
		}
		c.peeker.Discard(c.peeker.Buffered())
	}

	return nil
}

func (c *Client) connectSOCKS4(host string, port string) error {
	if c.Ipv6 {
		panic("socks4 proxy dose not support Ipv6")
	}

	c.flusher.Write([]byte{0x04, 0x01}) // ver, meth
	binary.Write(c.flusher, binary.BigEndian, uint16(StringToInt(port)))

	c.flusher.WriteByte(0x00) // userid

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
	}

	raw, err := c.peeker.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x5A {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (raw[1] != 0x5A) 1"}
	}
	c.peeker.Discard(c.peeker.Buffered())

	return nil
}

func (c *Client) connectSOCKS5(host string, port string) error {
	// ver, meth = open, auth
	if c.authorization != "" {
		c.flusher.Write([]byte{0x05, 0x01, 0x02})
	} else {
		c.flusher.Write([]byte{0x05, 0x01, 0x00})
	}

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
	}

	raw, err := c.peeker.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x5A && raw[1] != 0x02 {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (raw[1] != 0x5A) 1"}
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
			return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
		}

		raw, err = c.peeker.Peek(2)
		if err != nil {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
		} else if raw[1] != 0x00 {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (raw[1] != 0x5A) 2"}
		}
		c.peeker.Discard(c.peeker.Buffered())
	}

	// ver, meth = connect, rsv
	c.flusher.Write([]byte{0x05, 0x01, 0x00})

	if c.Ipv6 {
		c.flusher.WriteByte(0x04) // IPv6
		c.flusher.WriteString(host)
		binary.Write(c.flusher, binary.BigEndian, uint16(StringToInt(port)))
	} else {
		c.flusher.WriteByte(0x03) // Domain
		c.flusher.WriteByte(byte(len(host)))
	}

	c.flusher.WriteString(host)
	binary.Write(c.flusher, binary.BigEndian, uint16(StringToInt(port)))

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
	}

	raw, err = c.peeker.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x00 {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (raw[1] != 0x5A) 3"}
	}
	c.peeker.Discard(c.peeker.Buffered())

	return nil
}

func (c *Client) Proxy(Url string) {
	if c.hostConnected != "" {
		panic("can not set proxy after connect with server")
	}

	c.userProxy = ""
	c.passProxy = ""
	c.authorization = ""
	c.hostConnected = ""
	c.boolProxy = true

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
	} else if c.NetConnection != nil {
		c.NetConnection.Close()
		c.NetConnection = nil
	}

	c.closeLines()
	c.hostConnected = ""
	c.boolPreRequst = false
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

	c.peeker = genPeeker(c.ReadBufferSize)
	c.flusher = genFlusher(c.WriteBufferSize)

	if new, ok := c.NetConnection.(*tls.Conn); ok {
		c.flusher.Reset(new)
		c.peeker.Reset(new)
	} else if c.NetConnection != nil {
		c.flusher.Reset(c.NetConnection)
		c.peeker.Reset(c.NetConnection)
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

	if c.Ipv6 && !REQ.Header.myipv6 {
		REQ.Header.myipv6 = true
		ips, err := net.LookupIP(REQ.Header.myhost)
		if err != nil {
			return err
		}
		for _, ip := range ips {
			if ip.To16() != nil {
				REQ.Header.myhost = ip.String()
				break
			}
		}
	}

	if c.hostConnected == "" {
		c.TLSConfig.ServerName = REQ.Header.myhost
		if c.boolProxy {
			if err := c.connectNet(c.hostProxy, c.portProxy); err != nil {
				c.Close()
				return err
			}

			switch c.schemeProxy {
			case "https", "http":
				if err := c.connectHTTP(REQ.Header.myhost, REQ.Header.myport); err != nil {
					c.Close()
					return err
				}
			case "socks4":
				if !REQ.Header.myipv4 {
					REQ.Header.myipv4 = true
					ips, err := net.LookupIP(REQ.Header.myhost)
					if err != nil {
						return err
					}
					for _, ip := range ips {
						if ip.To4() != nil {
							REQ.Header.myhost = ip.String()
							break
						}
					}
				}
				if err := c.connectSOCKS4(REQ.Header.myhost, REQ.Header.myport); err != nil {
					c.Close()
					return err
				}
			case "socks5":
				if err := c.connectSOCKS5(REQ.Header.myhost, REQ.Header.myport); err != nil {
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

func (c *Client) DoPreRequest(REQ *RequestType) error {
	if err := c.Connect(REQ); err != nil {
		c.Close()
		return err
	}

	c.run = true
	c.boolPreRequst = true
	if _, err := c.flusher.Write(REQ.Header.raw[:REQ.Header.position-1]); err != nil {
		c.Close()
		return err
	}

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return err
	}

	if _, err := c.flusher.Write(REQ.Header.raw[REQ.Header.position-1 : REQ.Header.position]); err != nil {
		c.Close()
		return err
	}

	c.run = false
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
	if !c.boolPreRequst {
		if _, err := c.flusher.Write(REQ.Header.raw[:REQ.Header.position]); err != nil {
			RES.Reset()
			c.Close()
			return err
		}
	}

	c.boolPreRequst = false
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
			bufferd = min(c.ReadBufferSize, contentLength)
			contentLength -= bufferd
		} else if chunked > 0 {
			bufferd = min(c.ReadBufferSize, chunked+7)
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
				hex, b := hexBytesToInt(RES.Header.theBuffer[start : indexRNRN+rn])
				if !b || hex == 0 {
					contentLength = 0
					break
				} else if hex == 0 {
					contentLength = 0
					break
				}

				if len(RES.Header.theBuffer[indexRNRN+rn+2:RES.Header.position]) > hex {
					chunked = 0
				} else {
					chunked = hex - len(RES.Header.theBuffer[indexRNRN+rn+2:RES.Header.position])
				}

				indexRNRN += hex + 4 + len(RES.Header.theBuffer[start:indexRNRN+rn])
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
					if RES.Header.position > indexRNRN {
						rn := bytes.Index(RES.Header.theBuffer[indexRNRN:RES.Header.position], line) + indexRNRN
						if rn == indexRNRN-1 {
							continue
						}
						hex, b := hexBytesToInt(RES.Header.theBuffer[indexRNRN:rn])
						if !b || hex == 0 {
							continue
						} else if hex == 0 {
							continue
						}
						chunked = hex - len(RES.Header.theBuffer[rn+2:RES.Header.position])
						indexRNRN += hex + 4 + len(RES.Header.theBuffer[indexRNRN:rn])
					}
				} else {
					contentLength = 0
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
