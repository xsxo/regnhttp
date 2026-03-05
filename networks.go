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
	// the default value is 20 * time.Second
	// for now this object include proxy established.
	Timeout time.Duration

	// Tls Context // SSL Context
	// for full control use Client.TLSConfig = &tls.Config{...}
	TLSConfig *tls.Config

	// Dialer of the raw connection to get full control objects of connection
	Dialer *net.Dialer

	// Buffer Size of Writer Requsts (default value is 4096)
	WriteBufferSize int

	// Buffer Size of Reader Responses (default value is 4096)
	ReadBufferSize int

	// Raw net connection of Client object
	// if this object is defined by the user before establishing the connection, the object itself will not be created on the client side.
	// if the Client closed or return any err the connection will closed also
	// if use https or 443 port need to use NetConnection.(*tls.Conn) to can use this object
	RawConnection net.Conn

	// Off Nagle algorithm
	// Nagle algorithm: https://en.wikipedia.org/wiki/Nagle%27s_algorithm
	SetNoDelay bool
	NagleOff   bool

	// Ipv6 option need to hostname Ipv6
	// Support Ipv6 proxies (socks4 dose not support)
	// support DNS cache (if use t his option the hostname will converted to Ipv6 and will saved in cache)
	Ipv6 bool

	writer *bufio.Writer
	reader *bufio.Reader

	boolCustomConnection bool
	boolPreRequst        bool
	boolProxy            bool
	run                  bool
	hostConnected        string

	authorization string
	schemeProxy   string
	hostProxy     string
	portProxy     string
	userProxy     string
	passProxy     string
}

// check status connection
func (c *Client) Status() bool {
	if new, ok := c.RawConnection.(*tls.Conn); ok {
		if new != nil {
			return true
		}
	} else if c.RawConnection != nil {
		return true
	}
	return false
}

func (c *Client) connectNet(host string, port string) error {
	if new, ok := c.RawConnection.(*tls.Conn); ok {
		if new != nil {
			c.boolCustomConnection = true
		}
	} else if c.RawConnection != nil {
		c.boolCustomConnection = true
	}

	if c.boolCustomConnection {
		c.createLines()
		return nil
	}

	if c.Timeout.Seconds() == 0 {
		c.Timeout = time.Duration(20 * time.Second)
	}

	if c.Dialer == nil {
		c.Dialer = &net.Dialer{Timeout: c.Timeout}
	}

	var err error
	if port != "443" {
		if c.Ipv6 && !c.boolProxy {
			c.RawConnection, err = c.Dialer.Dial("tcp6", host+":"+port)
		} else {
			c.RawConnection, err = c.Dialer.Dial("tcp4", host+":"+port)
		}

	} else {
		if c.Ipv6 && !c.boolProxy {
			c.RawConnection, err = tls.DialWithDialer(c.Dialer, "tcp6", host+":"+port, c.TLSConfig)
		} else {
			c.RawConnection, err = tls.DialWithDialer(c.Dialer, "tcp4", host+":"+port, c.TLSConfig)
		}
	}

	if err != nil {
		return &RegnError{Message: "field create connection with '" + host + ":" + port + "' address\n" + err.Error()}
	}

	if c.SetNoDelay || c.NagleOff {
		if new, ok := c.RawConnection.(*tls.Conn); ok {
			new.NetConn().(*net.TCPConn).SetNoDelay(true)
		} else {
			c.RawConnection.(*net.TCPConn).SetNoDelay(true)
		}
	}
	c.createLines()
	return nil
}

func (c *Client) SetDeadline(timer time.Time) {
	if new, ok := c.RawConnection.(*tls.Conn); ok {
		if new != nil {
			new.SetDeadline(timer)
		}
	} else if c.RawConnection != nil {
		c.RawConnection.SetDeadline(timer)
	}
}

func (c *Client) SetWriteDeadline(timer time.Time) {
	if new, ok := c.RawConnection.(*tls.Conn); ok {
		if new != nil {
			new.SetWriteDeadline(timer)
		}
	} else if c.RawConnection != nil {
		c.RawConnection.SetWriteDeadline(timer)
	}
}

func (c *Client) SetReadDeadline(timer time.Time) {
	if new, ok := c.RawConnection.(*tls.Conn); ok {
		if new != nil {
			new.SetReadDeadline(timer)
		}
	} else if c.RawConnection != nil {
		c.RawConnection.SetReadDeadline(timer)
	}
}

func (c *Client) connectHTTP(host string, port string) error {
	c.writer.WriteString("CONNECT " + host + ":" + port + " HTTP/1.1\r\nHost: " + host + ":" + port + "\r\n")
	if c.authorization != "" {
		c.writer.WriteString("Proxy-Authorization: Basic " + c.authorization + "\r\n")
	}
	c.writer.Write(line)

	if err := c.writer.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
	}

	if raw, err := c.reader.Peek(16); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
	} else {
		if !bytes.Contains(raw, []byte{50, 48, 48}) {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Contains)"}
		}
		c.reader.Discard(c.reader.Buffered())
	}

	return nil
}

func (c *Client) connectSOCKS4(host string, port string) error {
	if c.Ipv6 {
		panic("socks4 proxy dose not support Ipv6")
	}

	c.writer.Write([]byte{0x04, 0x01}) // ver, meth
	binary.Write(c.writer, binary.BigEndian, uint16(StringToInt(port)))

	c.writer.WriteByte(0x00) // userid

	if err := c.writer.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
	}

	raw, err := c.reader.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x5A {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (raw[1] != 0x5A) 1"}
	}
	c.reader.Discard(c.reader.Buffered())

	return nil
}

func (c *Client) connectSOCKS5(host string, port string) error {
	// ver, meth = open, auth
	if c.authorization != "" {
		c.writer.Write([]byte{0x05, 0x01, 0x02})
	} else {
		c.writer.Write([]byte{0x05, 0x01, 0x00})
	}

	if err := c.writer.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
	}

	raw, err := c.reader.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x5A && raw[1] != 0x02 {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (raw[1] != 0x5A) 1"}
	}
	c.reader.Discard(c.reader.Buffered())

	if c.authorization != "" {
		c.writer.WriteByte(0x01)
		c.writer.WriteByte(byte(len(c.userProxy)))
		c.writer.Write([]byte(c.userProxy))
		c.writer.WriteByte(byte(len(c.passProxy)))
		c.writer.Write([]byte(c.passProxy))
		if err := c.writer.Flush(); err != nil {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
		}

		raw, err = c.reader.Peek(2)
		if err != nil {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
		} else if raw[1] != 0x00 {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (raw[1] != 0x5A) 2"}
		}
		c.reader.Discard(c.reader.Buffered())
	}

	// ver, meth = connect, rsv
	c.writer.Write([]byte{0x05, 0x01, 0x00})

	if c.Ipv6 {
		c.writer.WriteByte(0x04) // IPv6
		c.writer.WriteString(host)
		binary.Write(c.writer, binary.BigEndian, uint16(StringToInt(port)))
	} else {
		c.writer.WriteByte(0x03) // Domain
		c.writer.WriteByte(byte(len(host)))
	}

	c.writer.WriteString(host)
	binary.Write(c.writer, binary.BigEndian, uint16(StringToInt(port)))

	if err := c.writer.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Flush)"}
	}

	raw, err = c.reader.Peek(2)
	if err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (Peek)"}
	} else if raw[1] != 0x00 {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + host + ":" + port + "' address (raw[1] != 0x5A) 3"}
	}
	c.reader.Discard(c.reader.Buffered())

	return nil
}

func (c *Client) Proxy(Url string) {
	if new, ok := c.RawConnection.(*tls.Conn); ok {
		if new != nil {
			panic("can not set proxy after connect with server")
		}
	} else if c.RawConnection != nil {
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
	if new, ok := c.RawConnection.(*tls.Conn); ok {
		if new != nil {
			new.Close()
			c.RawConnection.Close()
			c.RawConnection = nil
		}
	} else if c.RawConnection != nil {
		c.RawConnection.Close()
		c.RawConnection = nil
	}

	c.closeLines()
	c.hostConnected = ""
	c.boolPreRequst = false
	c.boolCustomConnection = false
	c.run = false
}

func (c *Client) closeLines() {
	if c.writer != nil {
		flusherPool.Put(c.writer)
		c.writer = nil
	}

	if c.reader != nil {
		peekerPool.Put(c.reader)
		c.reader = nil
	}
}

func (c *Client) createLines() {
	// c.closeLines()

	if c.ReadBufferSize == 0 {
		c.ReadBufferSize = 4096
	}

	if c.WriteBufferSize == 0 {
		c.WriteBufferSize = 4096
	}

	c.writer = genFlusher(c.WriteBufferSize)
	c.reader = genPeeker(c.ReadBufferSize)

	if new, ok := c.RawConnection.(*tls.Conn); ok {
		c.writer.Reset(new)
		c.reader.Reset(new)
	} else if c.RawConnection != nil {
		c.writer.Reset(c.RawConnection)
		c.reader.Reset(c.RawConnection)
	}
}

func (c *Client) Connect(REQ *RequestType) error {
	if c.boolCustomConnection {
		return nil
	}

	if c.hostConnected != REQ.Header.myhost && c.hostConnected != "" {
		c.Close()
	}

	if c.TLSConfig == nil {
		c.TLSConfig = &tls.Config{}
	}

	if c.run {
		c.Close()
		panic("concurrent client goroutines")
	}

	if c.Ipv6 && c.boolProxy && !REQ.Header.myipv6 {
		REQ.Header.myipv6 = true
		ips, err := net.LookupIP(REQ.Header.sshost)
		if err != nil {
			return err
		}
		for _, ip := range ips {
			if ip.To16() != nil {
				REQ.Header.myhost = "[" + ip.String() + "]"
				break
			}
		}
	}

	if c.hostConnected == "" {
		c.TLSConfig.ServerName = REQ.Header.sshost
		if c.boolProxy {
			if err := c.connectNet(c.hostProxy, c.portProxy); err != nil {
				c.Close()
				return err
			}

			if c.boolCustomConnection {
				return nil
			}

			c.RawConnection.SetDeadline(time.Now().Add(c.Timeout))
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
			c.RawConnection.SetDeadline(time.Time{})

			if REQ.Header.myport == "443" || REQ.Header.mytls || c.schemeProxy == "https" {
				c.RawConnection = tls.Client(c.RawConnection, c.TLSConfig)
				c.createLines()
			}
		} else {
			if err := c.connectNet(REQ.Header.myhost, REQ.Header.myport); err != nil {
				c.Close()
				return err
			}

			if c.boolCustomConnection {
				return nil
			}

			if REQ.Header.mytls {
				c.RawConnection = tls.Client(c.RawConnection, c.TLSConfig)
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
	if _, err := c.writer.Write(REQ.Header.raw[:REQ.Header.position-1]); err != nil {
		c.Close()
		return err
	}

	if err := c.writer.Flush(); err != nil {
		c.Close()
		return err
	}

	if _, err := c.writer.Write(REQ.Header.raw[REQ.Header.position-1 : REQ.Header.position]); err != nil {
		c.Close()
		return err
	}

	c.run = false
	return nil
}

func (c *Client) DoTimeout(REQ *RequestType, RES *ResponseType, Timeout time.Duration) error {
	if err := c.Connect(REQ); err != nil {
		c.Close()
		return err
	}

	c.RawConnection.SetDeadline(time.Now().Add(Timeout))
	if err := c.Do(REQ, RES); err != nil {
		c.Close()
		return err
	}
	c.RawConnection.SetDeadline(time.Time{})
	return nil
}

// not support goroutine-safe (mutli threads)
func (c *Client) Do(REQ *RequestType, RES *ResponseType) error {
	RES.Header.position = 0
	RES.Header.theBuffer = RES.Header.theBuffer[:RES.Header.bufferSize]
	if err := c.Connect(REQ); err != nil {
		c.Close()
		return err
	}

	c.run = true
	if !c.boolPreRequst {
		if _, err := c.writer.Write(REQ.Header.raw[:REQ.Header.position]); err != nil {
			c.Close()
			return err
		}
	}

	c.boolPreRequst = false
	if err := c.writer.Flush(); err != nil {
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
			if _, err := c.reader.Peek(1); err != nil {
				c.Close()
				return err
			}
			bufferd = c.reader.Buffered()
		}

		raw, err := c.reader.Peek(bufferd)
		if err != nil {
			c.Close()
			return err
		}

		_, err = c.reader.Discard(bufferd)
		if err != nil {
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

		if chunkedB {
			for chunked <= 0 {
				rn := bytes.Index(RES.Header.theBuffer[indexRNRN:RES.Header.position], line)
				if rn == -1 {
					break
				}

				start := indexRNRN
				hex, b := hexBytesToInt(RES.Header.theBuffer[start : indexRNRN+rn])
				if !b || hex == 0 {
					contentLength = 0
					break
				}

				if len(RES.Header.theBuffer[indexRNRN+rn+2:RES.Header.position]) > hex {
					chunked = 0
				} else {
					chunked = hex - len(RES.Header.theBuffer[indexRNRN+rn+2:RES.Header.position])
				}
				indexRNRN += hex + 4 + len(RES.Header.theBuffer[start:indexRNRN+rn])
			}
		} else if indexRNRN == -1 {
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
						for chunkedB && chunked <= 0 {
							rn := bytes.Index(RES.Header.theBuffer[indexRNRN:RES.Header.position], line)
							if rn == -1 {
								break
							}

							start := indexRNRN
							hex, b := hexBytesToInt(RES.Header.theBuffer[start : indexRNRN+rn])
							if !b || hex == 0 {
								contentLength = 0
								break
							}

							if len(RES.Header.theBuffer[indexRNRN+rn+2:RES.Header.position]) > hex {
								chunked = 0
							} else {
								chunked = hex - len(RES.Header.theBuffer[indexRNRN+rn+2:RES.Header.position])
							}
							indexRNRN += hex + 4 + len(RES.Header.theBuffer[start:indexRNRN+rn])
						}
					}
				} else {
					break
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
