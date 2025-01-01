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

	h2Streams   uint32
	h2WinServer uint32
	h2FrameSize uint32

	h2WinClient  uint32
	h2PrvStreams uint32

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
	return c.h2Streams
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
	therequest.WriteString("CONNECT " + address + " HTTP/1.1\r\nHost: " + address + "\r\nConnection: Keep-Alive")

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

	if raw, err := c.peeker.Peek(20); err != nil {
		return &RegnError{Message: "field proxy connection with '" + address + "' address"}
	} else {
		readed := statusRegex.FindSubmatch(raw)
		if len(readed) <= 0 {
			c.Close()
			return &RegnError{Message: "field proxy connection with '" + address + "' address"}
		}
		readed[0] = nil
		c.peeker.Discard(c.peeker.Buffered())
	}

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

	c.h2Streams = 0
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

	if new, ok := c.connection.(*tls.Conn); ok {
		if new != nil {
			c.peeker = genPeeker(new)
			c.flusher = genFlusher(new)
		}
	} else {
		if c.connection != nil {
			c.peeker = genPeeker(c.connection)
			c.flusher = genFlusher(c.connection)
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

			if REQ.Header.myport == "443" || REQ.Header.mytls {
				c.connection = tls.Client(c.connection, c.TLSConfig)
				c.createLines()
			}

		} else {
			if err := c.connectNet(REQ.Header.myhost, REQ.Header.myport); err != nil {
				c.Close()
				return err
			}

			if REQ.Header.mytls {
				c.connection = tls.Client(c.connection, c.TLSConfig)
				c.createLines()
			}
		}

		if c.upgraded {
			if REQ.Header.myport != "443" && !REQ.Header.mytls {
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
			raw, _ := c.peeker.Peek(buffred)
			c.peeker.Discard(buffred)
			indexIds := bytes.IndexByte(raw, 0x03)
			indexWin := bytes.IndexByte(raw, 0x04)
			indexMax := bytes.IndexByte(raw, 0x05)

			if raw[3] != 0x4 {
				c.Close()
				return &RegnError{Message: "field foramt HTTP2 settings from the server"}
			}

			if indexIds != -1 {
				c.h2Streams = binary.BigEndian.Uint32(raw[indexIds+1 : indexIds+5])
			} else {
				c.h2Streams = 100
			}

			if indexWin != -1 {
				c.h2WinServer = binary.BigEndian.Uint32(raw[indexWin : indexWin+5])
			} else {
				c.h2WinServer = 65535
			}

			if indexMax != -1 {
				c.h2FrameSize = binary.BigEndian.Uint32(raw[indexMax+1 : indexMax+5])
			} else {
				c.h2FrameSize = 16384
			}

			c.h2WinClient = 65535

			c.flusher.Write([]byte{0x00, 0x00, 0x00})            // payloadlenght
			c.flusher.WriteByte(0x4)                             // settings frame
			c.flusher.WriteByte(0x0)                             // end stream (true)
			binary.Write(c.flusher, binary.BigEndian, uint32(0)) // stream id (0)
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
	}

	if c.h2Streams == 0 {
		return &RegnError{"concurrent streams id"}
	} else if c.h2WinServer < payloadLengthBody {
		return &RegnError{"data > window server size"}
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

	// c.h2WinServer -= uint32(payloadLengthHeaders)

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

	c.h2WinServer -= payloadLengthBody

	c.flusher.WriteByte(0x0)
	c.flusher.WriteByte(0x1)

	binary.Write(c.flusher, binary.BigEndian, StreamID&0x7FFFFFFF) // stream id
	c.flusher.Write(REQ.Header.rawBody.B)                          // payload
	if err := c.flusher.Flush(); err != nil {
		c.Close()
		c.run = false
		return err
	}

	c.h2Streams--
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

	if c.TimeoutRead != 0 {
		c.Dialer.Deadline = time.Now().Add(time.Duration(c.TimeoutRead) * time.Second)
		c.TimeoutRead = 0
	}

	c.run = true
	RES.Header.contectLegnth = -1
	RES.Header.thebuffer.Reset()

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, StreamID)

	pending := 0
	for c.run {
		indexRaw := bytes.Index(c.theBuffer.B, data) - 5
		for indexRaw > -1 {
			payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, c.theBuffer.B[indexRaw:indexRaw+3]...))) + 9 + indexRaw

			if payloadLength > c.theBuffer.Len() {
				break
			}

			switch c.theBuffer.B[indexRaw+3] {
			case 0x0:
				RES.Header.thebuffer.Write(c.theBuffer.B[indexRaw+9 : payloadLength])
			case 0x1:
				RES.Header.decoder.Write(c.theBuffer.B[indexRaw+9 : payloadLength])
			case 0x3:
				c.run = false
				c.h2Streams++
				c.theBuffer.B = append(c.theBuffer.B[:indexRaw], c.theBuffer.B[payloadLength:]...)
				return &RegnError{"the stream id " + strconv.Itoa(int(StreamID)) + " has been canceled by the server"}
			}

			if c.theBuffer.B[indexRaw+4] == 0x1 || c.theBuffer.B[indexRaw+4] == 0x0 {
				c.run = false
			}

			c.theBuffer.B = append(c.theBuffer.B[:indexRaw], c.theBuffer.B[payloadLength:]...)
			indexRaw = bytes.Index(c.theBuffer.B, data) - 5
		}

		if !c.run {
			break
		} else if _, err := c.peeker.Peek(9); err != nil {
			c.Close()
			return err
		}

		buffered := c.peeker.Buffered()
		raw, _ := c.peeker.Peek(buffered)

		payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, raw[0:3]...))) + 9
		if payloadLength > len(raw) {
			pending = payloadLength - len(raw)
			c.theBuffer.Write(raw)
			c.peeker.Discard(buffered)
			continue
		} else if pending != 0 {
			if pending > len(raw) {
				pending -= len(raw)
				c.theBuffer.Write(raw)
				c.peeker.Discard(buffered)
				continue
			} else {
				c.peeker.Discard(pending)
				c.theBuffer.Write(raw[:pending])
				pending = 0
			}
		}
		c.peeker.Discard(payloadLength)
		Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
		switch raw[3] {
		case 0x0:
			if payloadLength-9 >= int(c.h2WinClient) {
				increment := 65535 - c.h2WinClient
				c.flusher.Write([]byte{0x00, 0x00, 0x04})
				c.flusher.WriteByte(0x8)
				c.flusher.WriteByte(0x00)
				binary.Write(c.flusher, binary.BigEndian, uint32(0))
				binary.Write(c.flusher, binary.BigEndian, uint32(increment))
				c.flusher.Flush()
				c.h2WinClient += increment
			} else {
				c.h2WinClient -= uint32(payloadLength - 9)
			}

			if StreamID != Stream || raw[4] != 0x1 || raw[4] != 0x0 {
				c.h2Streams++
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}

			RES.Header.thebuffer.Write(raw[:payloadLength])

			if raw[4] == 0x1 || raw[4] == 0x0 {
				c.h2Streams++
				c.run = false
			}
		case 0x1:
			if StreamID != Stream || raw[4] != 0x4 {
				c.h2Streams++
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}
			RES.Header.decoder.Write(raw[indexRaw+9 : payloadLength])

			if raw[4] == 0x1 || raw[4] == 0x0 {
				c.h2Streams++
				c.run = false
			}
		case 0x3:
			if Stream != StreamID {
				c.h2Streams++
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}
			c.h2Streams++
			return &RegnError{"the stream id " + strconv.Itoa(int(StreamID)) + " has been canceled by the server"}

		case 0x4:
			indexIds := bytes.IndexByte(raw, 0x03)
			indexWin := bytes.IndexByte(raw, 0x5)

			if indexIds == -1 || indexWin == -1 {
				continue
			}
			c.h2Streams = binary.BigEndian.Uint32(raw[indexIds+1:indexIds+5]) - c.h2Streams
			c.h2WinServer = binary.BigEndian.Uint32(raw[indexWin+1 : indexWin+5])
		case 0x7:
			c.Close()
			return &RegnError{Message: "the connection has been closed by the server 'http2.GoAwayFrame'"}
		case 0x8:
			winsize := binary.BigEndian.Uint32(raw[9:13])
			c.h2WinServer += winsize
		default:
			c.theBuffer.Write(raw)
		}
	}

	c.h2Streams++
	return nil
}

func (c *Client) Do(REQ *RequestType, RES *ResponseType) error {
	if c.upgraded {
		if c.h2PrvStreams == 0 {
			c.h2PrvStreams++
		} else {
			c.h2PrvStreams += 2
		}

		if err := c.Http2SendRequest(REQ, c.h2PrvStreams); err != nil {
			return err
		}
		if err := c.Http2ReadRespone(RES, c.h2PrvStreams); err != nil {
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

	if _, err := c.flusher.Write(REQ.Header.raw.B); err != nil {
		c.Close()
		return err
	}

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
