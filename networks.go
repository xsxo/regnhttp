package regn

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/valyala/bytebufferpool"
)

type Client struct {
	// Timeout Connection
	Timeout int

	// Timeout Reading Response
	TimeoutRead int

	// Tls Context
	TLSConfig *tls.Config

	// Dialer to create dial connection
	Dialer *net.Dialer

	h2streams   uint32
	h2winserver uint32
	h2winclient uint32

	useProxy      bool
	hostConnected string
	connection    net.Conn
	run           bool

	peeker  *bufio.Reader
	flusher *bufio.Writer

	authorization string
	hostProxy     string
	portProxy     string
	upgraded      bool

	theBuffer *bytebufferpool.ByteBuffer
}

func (c *Client) Http2MaxStreams() uint32 {
	return c.h2streams
}

func (c *Client) Status() bool {
	if c.hostConnected != "" {
		return true
	} else {
		return false
	}
}

func (c *Client) HttpVesrion() int {
	if c.upgraded {
		return 2
	} else {
		return 1
	}
}

func (c *Client) HttpDowngrade() {
	if c.upgraded {
		c.Close()
		c.upgraded = false
	}
}

func (c *Client) Http2Upgrade() {
	c.Close()
	c.upgraded = true
}

func (c *Client) connectNet(host string, port string) error {
	if c.Timeout == 0 {
		c.Timeout = 10
	}
	if c.TimeoutRead == 0 {
		c.TimeoutRead = c.Timeout
	}

	if c.Dialer == nil {
		c.Dialer = &net.Dialer{Timeout: time.Duration(c.Timeout) * time.Second, Deadline: time.Now().Add(time.Duration(c.Timeout) * time.Second)}
	}

	var err error

	if port != "443" {
		c.connection, err = c.Dialer.Dial("tcp", host+":"+port)
	} else {
		c.connection, err = tls.DialWithDialer(c.Dialer, "tcp4", host+":"+port, c.TLSConfig)
	}

	if err != nil {
		return &RegnError{Message: "field create connection with '" + host + ":" + port + "' address\n" + err.Error()}
	}

	c.createLines()
	err = nil
	return nil
}

func (c *Client) connectHost(address string) error {
	therequest := bufferPool.Get()
	therequest.Reset()
	therequest.WriteString("CONNECT " + address + " HTTP/1.1\r\nHost: " + address + "\r\nConnection: keep-Alive")

	if c.authorization != "" {
		therequest.WriteString("Authorization: " + c.authorization)
	}
	therequest.WriteString("\r\n\r\n")

	if _, err := c.flusher.Write(therequest.B); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + address + "' address"}
	}

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + address + "' address"}
	}
	therequest.Reset()
	bufferPool.Put(therequest)

	buffer := make([]byte, 4096)
	if _, err := c.peeker.Read(buffer); err != nil {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + address + "' address"}
	}

	readed := statusRegex.FindSubmatch(buffer)
	buffer = nil

	if len(readed) == 0 {
		c.Close()
		return &RegnError{Message: "field proxy connection with '" + address + "' address"}
	}

	readed[0] = nil
	return nil
}

func (c *Client) Proxy(Url string) {
	if c.hostConnected != "" {
		panic("can not set proxy after send request")
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
	if new, ok := c.connection.(*tls.Conn); ok {
		if new != nil {
			new.Close()
			c.connection = nil
			c.hostConnected = ""
		}
	} else {
		if c.connection != nil {
			c.connection.Close()
			c.hostConnected = ""
			c.connection = nil
		}
	}

	c.closeLines()
	if c.theBuffer != nil {
		c.theBuffer.Reset()
		bufferPool.Put(c.theBuffer)
		c.theBuffer = nil
	}

	c.h2streams = 0
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
	c.peeker = genPeeker(c.connection)
	c.flusher = genFlusher(c.connection)
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

		if c.upgraded {
			c.TLSConfig.NextProtos = []string{"h2"}
		}

		if c.useProxy {
			if err := c.connectNet(c.hostProxy, c.portProxy); err != nil {
				c.Close()
				return err
			}

			if err := c.connectHost(REQ.Header.myhost + ":" + REQ.Header.myport); err != nil {
				c.Close()
				return err
			}

			if REQ.Header.myport == "443" {
				c.connection = tls.Client(c.connection, c.TLSConfig)
				c.createLines()
			}

		} else {
			if err := c.connectNet(REQ.Header.myhost, REQ.Header.myport); err != nil {
				c.Close()
				return err
			}
		}

		if c.upgraded {
			if REQ.Header.myport != "443" {
				c.Close()
				panic("http2 protocol support https requests only; use https://")
			}

			if c.theBuffer == nil {
				c.theBuffer = bufferPool.Get()
			}

			if _, err := c.flusher.WriteString("PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"); err != nil {
				c.Close()
				return err
			}

			if err := c.flusher.Flush(); err != nil {
				c.Close()
				return err
			}

			if _, err := c.peeker.Peek(1); err != nil {
				c.Close()
				return err
			}

			buffred := c.peeker.Buffered()
			raw, err := c.peeker.Peek(buffred)
			c.peeker.Discard(buffred)
			indexIds := bytes.IndexByte(raw, 0x03)
			indexWin := bytes.IndexByte(raw, 0x05)

			if err != nil || raw[3] != 0x4 || indexIds == -1 || indexWin == -1 {
				c.Close()
				return &RegnError{Message: "field foramt HTTP2 settings from the server"}
			}
			c.h2streams = binary.BigEndian.Uint32(raw[indexIds+1 : indexIds+5])
			// c.h2winserver = binary.BigEndian.Uint32(raw[indexWin+1 : indexWin+5])
			c.h2winserver = 65535
			c.h2winclient = 65535

			// response settings frame
			c.flusher.Write([]byte{0x00, 0x00, 0x00}) // payloadlenght
			c.flusher.WriteByte(0x4)                  // settings frame
			c.flusher.WriteByte(0x0)                  // end stream (true)
			binary.Write(c.flusher, binary.BigEndian, uint32(0))
			// binary.Write(c.flusher, binary.BigEndian, uint32(0x03))     // SETTINGS_MAX_CONCURRENT_STREAMS (2 bytes)
			// binary.Write(c.flusher, binary.BigEndian, uint16(0x04))
			// binary.Write(c.flusher, binary.BigEndian, uint32(90021324)) // Value
			c.flusher.Flush()
		}
		c.hostConnected = REQ.Header.myhost
	}

	return nil
}

func (c *Client) Http2SendRequest(REQ *RequestType, StreamID uint32) error {
	if !c.upgraded {
		c.Http2Upgrade()
	}

	payloadLengthHeaders := uint32(REQ.Header.raw.Len())
	payloadLengthBody := uint32(REQ.Header.rawBody.Len())

	if StreamID%2 == 0 {
		panic("id is not odd")
	} else if err := c.Connect(REQ); err != nil {
		return err
	} else if c.h2streams == 0 {
		return &RegnError{"concurrent streams id"}
	} else if c.h2winserver < payloadLengthBody {
		return &RegnError{"the window is full try to read any response to upper the window"}
	}

	c.run = true
	if c.TimeoutRead != 0 {
		c.Dialer.Deadline = time.Now().Add(time.Duration(c.TimeoutRead) * time.Second)
		c.TimeoutRead = 0
	}

	if REQ.Header.hpackHeaders == nil {
		REQ.Http2Upgrade()
	}

	c.flusher.Write([]byte{
		byte(payloadLengthHeaders >> 16), // len payload 3 bytes
		byte(payloadLengthHeaders >> 8),  // len payload 3 bytes
		byte(payloadLengthHeaders),       // len payload 3 bytes
		// 0x1,                              // type of frame
		// 0x4,                              // end stream (false) && end header (true)
	})

	// c.h2winserver -= uint32(payloadLengthHeaders)

	c.flusher.WriteByte(0x1)
	c.flusher.WriteByte(0x4)

	binary.Write(c.flusher, binary.BigEndian, StreamID&0x7FFFFFFF) // stream id
	c.flusher.Write(REQ.Header.raw.B)                              // payload
	if err := c.flusher.Flush(); err != nil {
		c.Close()
		c.run = false
		return err
	}

	c.flusher.Write([]byte{
		byte(payloadLengthBody >> 16), // len payload 3 bytes
		byte(payloadLengthBody >> 8),  // len payload 3 bytes
		byte(payloadLengthBody),       // len payload 3 bytes
		// 0x0,                           // type of frame
		// 0x1,                           // end stream (true)
	})

	c.h2winserver -= payloadLengthBody

	c.flusher.WriteByte(0x0)
	c.flusher.WriteByte(0x1)

	binary.Write(c.flusher, binary.BigEndian, StreamID&0x7FFFFFFF) // stream id
	c.flusher.Write(REQ.Header.rawBody.B)                          // payload
	if err := c.flusher.Flush(); err != nil {
		c.Close()
		c.run = false
		return err
	}

	c.h2streams--
	c.run = false
	return nil
}

func (c *Client) Http2ReadRespone(RES *ResponseType, StreamID uint32) error {
	if c.run {
		panic("concurrent client goroutines")
	} else if !c.upgraded {
		c.Http2Upgrade()
		return &RegnError{"http version is not http2"}
	}

	if StreamID%2 == 0 {
		panic("stream id is not odd")
	} else if c.hostConnected == "" {
		return &RegnError{"the connection has been closed"}
	} else if RES.Header.decoder == nil {
		RES.Http2Upgrade()
	}

	c.run = true
	RES.Header.contectLegnth = -1
	RES.Header.thebuffer.Reset()

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, StreamID)

	indexRaw := bytes.Index(c.theBuffer.B, data) - 5
	for indexRaw > -1 {
		payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, c.theBuffer.B[indexRaw:indexRaw+3]...))) + 9 + indexRaw

		switch c.theBuffer.B[indexRaw+3] {
		case 0x0:
			RES.Header.thebuffer.Write(c.theBuffer.B[indexRaw+9 : payloadLength])
		case 0x1:
			RES.Header.decoder.Write(c.theBuffer.B[indexRaw+9 : payloadLength])
		case 0x3:
			c.run = false
			c.theBuffer.B = append(c.theBuffer.B[:indexRaw], c.theBuffer.B[payloadLength:]...)
			return &RegnError{"the stream id has been canceled by the server"}
		}

		if c.theBuffer.B[indexRaw+4]&0x1 != 0 {
			c.run = false
		}
		c.theBuffer.B = append(c.theBuffer.B[:indexRaw], c.theBuffer.B[payloadLength:]...)
		indexRaw = bytes.Index(c.theBuffer.B, data) - 5
	}

	for c.run {
		if _, err := c.peeker.Peek(1); err != nil {
			c.Close()
			return err
		}
		buffred := c.peeker.Buffered()
		if buffred < 8 {
			c.Close()
			return &RegnError{Message: "PROTOCOL_ERROR"}
		}

		raw, _ := c.peeker.Peek(buffred)

		payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, raw[0:3]...))) + 9
		if payloadLength > buffred {
			c.Close()
			return &RegnError{Message: "PROTOCOL_ERROR"}
		}
		c.peeker.Discard(payloadLength)

		switch raw[3] {
		case 0x0:
			c.h2winclient -= uint32(len(raw[:payloadLength]) - 9)
			increment := 65535 - c.h2winclient
			if c.h2winclient < 10000 {
				c.flusher.Write([]byte{0x00, 0x00, 0x04, 0x8, 0x00})
				// c.flusher.WriteByte(0x8)
				// c.flusher.WriteByte(0x00)
				binary.Write(c.flusher, binary.BigEndian, uint32(0))
				binary.Write(c.flusher, binary.BigEndian, uint32(increment))
				c.flusher.Flush()
				c.h2winclient += increment
			}

			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			if Stream != StreamID {
				c.h2streams++
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}

			RES.Header.thebuffer.Write(raw[9:payloadLength])

			if raw[4]&0x1 != 0 {
				c.run = false
			}

		case 0x1:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF

			if Stream != StreamID {
				c.h2streams++
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}

			RES.Header.decoder.Write(raw[9:payloadLength])

			if raw[4]&0x1 != 0 {
				c.run = false
			}
		case 0x3:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			if Stream != StreamID {
				c.h2streams++
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}
			c.h2streams++
			return &RegnError{"the stream id has been canceled by the server"}
		case 0x4:
			indexIds := bytes.IndexByte(raw, 0x03)
			indexWin := bytes.IndexByte(raw, 0x5)

			if indexIds == -1 || indexWin == -1 {
				continue
			}
			c.h2streams = binary.BigEndian.Uint32(raw[indexIds+1:indexIds+5]) - c.h2streams
			// c.h2winserver = binary.BigEndian.Uint32(raw[indexWin+1 : indexWin+5])
		case 0x7:
			c.Close()
			return &RegnError{Message: "the connection has been closed by the server 'http2.GoAwayFrame'"}
		case 0x8:
			winsize := binary.BigEndian.Uint32(raw[9:13]) // + c.h2winserver
			// winsize = winsize / 15
			c.h2winserver = winsize
		}
	}
	c.h2streams++
	return nil
}

func (c *Client) Do(REQ *RequestType, RES *ResponseType) error {
	if c.upgraded {
		id := c.h2streams
		if id%2 == 0 {
			id++
		}
		if err := c.Http2SendRequest(REQ, id); err != nil {
			return err
		}
		if err := c.Http2ReadRespone(RES, id); err != nil {
			return err
		}

		return nil
	}

	if err := c.Connect(REQ); err != nil {
		return err
	}

	if c.TimeoutRead != 0 {
		c.Dialer.Deadline = time.Now().Add(time.Duration(c.TimeoutRead) * time.Second)
		c.TimeoutRead = 0
	}

	c.run = true

	if REQ.Header.hpackHeaders != nil {
		REQ.HttpDowngrade()
	}

	if RES.Header.decoder != nil {
		RES.HttpDowngrade()
	}

	c.flusher.Write(REQ.Header.raw.B)
	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return err
	}

	RES.Header.thebuffer.Reset()
	for {
		if _, err := c.peeker.Peek(1); err != nil {
			c.Close()
			return err
		}
		le := c.peeker.Buffered()
		raw, _ := c.peeker.Peek(le)

		c.peeker.Discard(le)
		RES.Header.thebuffer.Write(raw)
		raw = nil

		if bytes.Contains(RES.Header.thebuffer.B, lines[1:]) {
			contentLengthMatch := lenRegex.FindSubmatch(RES.Header.thebuffer.B)
			if len(contentLengthMatch) > 1 {
				contentLength, _ := strconv.Atoi(string(contentLengthMatch[1]))
				contentLengthMatch[0] = nil
				contentLengthMatch[1] = nil

				if len(bytes.SplitN(RES.Header.thebuffer.B, lines[1:], 2)[1]) >= contentLength {
					break
				}
			} else if bytes.Contains(RES.Header.thebuffer.B, lines) {
				break
			}
		}
	}

	c.run = false
	return nil
}
