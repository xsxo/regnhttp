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
	Timeout time.Duration

	// Timeout Reading Response
	TimeoutRead time.Duration

	// Tls Context
	TLSConfig *tls.Config

	// Dialer to create dial connection
	Dialer *net.Dialer

	// Net connection of Client
	NetConnection net.Conn

	h2Streams   uint32
	h2WinServer uint32
	h2FrameSize uint32

	h2WinClient  uint32
	h2PrvWinC    uint32
	h2PrvStreams uint32

	useProxy      bool
	hostConnected string
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
	if c.Timeout.Seconds() == 0 {
		c.Timeout = time.Duration(10 * time.Second)
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
	c.NetConnection.SetReadDeadline(time.Now().Add(c.TimeoutRead))
	c.Timeout = time.Duration(0 * time.Second)
	c.TimeoutRead = time.Duration(0 * time.Second)
	c.createLines()
	err = nil
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

	if raw, err := c.peeker.Peek(20); err != nil {
		return &RegnError{Message: "field proxy connection with '" + address + "' address (Peek)"}
	} else {
		if !bytes.Contains(raw, []byte("HTTP/1.1 200")) {
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
			c.NetConnection = nil
			c.hostConnected = ""
		}
	} else {
		if c.NetConnection != nil {
			c.NetConnection.Close()
			c.hostConnected = ""
			c.NetConnection = nil
		}
	}

	c.closeLines()
	if c.theBuffer != nil {
		c.theBuffer.Reset()
		bufferPool.Put(c.theBuffer)
		c.theBuffer = nil
	}

	c.h2PrvStreams = 0
	c.h2WinServer = 0
	c.h2FrameSize = 0
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

	if new, ok := c.NetConnection.(*tls.Conn); ok {
		if new != nil {
			c.peeker = genPeeker(new)
			c.flusher = genFlusher(new)
		}
	} else {
		if c.NetConnection != nil {
			c.peeker = genPeeker(c.NetConnection)
			c.flusher = genFlusher(c.NetConnection)
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

			if raw[3] != 0x4 {
				c.Close()
				return &RegnError{Message: "field foramt HTTP2 settings from the server"}
			}

			raw = raw[9:]
			for len(raw) >= 6 {
				switch binary.BigEndian.Uint16(raw[:2]) {
				case 0x03:
					c.h2Streams = binary.BigEndian.Uint32(raw[2:6])
				case 0x04:
					c.h2WinServer = binary.BigEndian.Uint32(raw[2:6])
				case 0x05:
					c.h2FrameSize = binary.BigEndian.Uint32(raw[2:6])
				}
				raw = raw[6:]
			}

			if c.h2Streams == 0 {
				c.h2Streams = 100
			}

			if c.h2WinServer == 0 {
				c.h2WinServer = 65535
			}

			if c.h2FrameSize == 0 {
				c.h2FrameSize = 16384
			}

			c.h2PrvWinC = 65535
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

func (c *Client) Http2SendHeaders(REQ *RequestType, StreamID uint32) error {
	if !c.upgraded {
		c.Http2Upgrade()
	}

	if StreamID%2 == 0 {
		panic("id is not odd")
	} else if err := c.Connect(REQ); err != nil {
		return err
	} else if c.h2Streams == 0 {
		return &RegnError{"concurrent streams id"}
	} else if REQ.Header.hpackHeaders == nil {
		REQ.Http2Upgrade()
	}

	payloadLengthHeaders := uint32(REQ.Header.raw.Len())

	c.flusher.Write([]byte{
		byte(payloadLengthHeaders >> 16), // len payload 3 bytes
		byte(payloadLengthHeaders >> 8),  // len payload 3 bytes
		byte(payloadLengthHeaders),       // len payload 3 bytes
		0x1,                              // type of frame
		0x4,                              // end stream (false) && end header (true)
	})

	binary.Write(c.flusher, binary.BigEndian, StreamID&0x7FFFFFFF) // stream id
	c.flusher.Write(REQ.Header.raw.B)                              // payload

	if err := c.flusher.Flush(); err != nil {
		return err
	}

	return nil
}

func (c *Client) Http2SendBody(REQ *RequestType, StreamID uint32) error {
	if !c.upgraded {
		c.Http2Upgrade()
	}

	if StreamID%2 == 0 {
		panic("id is not odd")
	} else if err := c.Connect(REQ); err != nil {
		return err
	} else if c.h2Streams == 0 {
		return &RegnError{"concurrent streams id"}
	}

	payloadLengthBody := uint32(REQ.Header.rawBody.Len())

	c.flusher.Write([]byte{
		byte(payloadLengthBody >> 16), // len payload 3 bytes
		byte(payloadLengthBody >> 8),  // len payload 3 bytes
		byte(payloadLengthBody),       // len payload 3 bytes
		0x0,                           // type of frame
		0x1,                           // end stream (true)
	})

	binary.Write(c.flusher, binary.BigEndian, StreamID&0x7FFFFFFF) // stream id
	c.flusher.Write(REQ.Header.rawBody.B)                          // payload

	if err := c.flusher.Flush(); err != nil {
		return err
	}

	c.h2WinServer -= payloadLengthBody
	c.h2Streams--
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
	} else if c.h2Streams == 0 {
		return &RegnError{"concurrent streams id"}
	} else if c.h2WinServer < payloadLengthBody {
		return &RegnError{"data > window server size"}
	} else if REQ.Header.hpackHeaders == nil {
		REQ.Http2Upgrade()
	}

	c.flusher.Write([]byte{
		byte(payloadLengthHeaders >> 16), // len payload 3 bytes
		byte(payloadLengthHeaders >> 8),  // len payload 3 bytes
		byte(payloadLengthHeaders),       // len payload 3 bytes
		0x1,                              // type of frame
		0x4,                              // end stream (false) && end header (true)
	})

	binary.Write(c.flusher, binary.BigEndian, StreamID&0x7FFFFFFF) // stream id
	c.flusher.Write(REQ.Header.raw.B)                              // payload

	c.flusher.Write([]byte{
		byte(payloadLengthBody >> 16), // len payload 3 bytes
		byte(payloadLengthBody >> 8),  // len payload 3 bytes
		byte(payloadLengthBody),       // len payload 3 bytes
		0x0,                           // type of frame
		0x1,                           // end stream (true)
	})

	binary.Write(c.flusher, binary.BigEndian, StreamID&0x7FFFFFFF) // stream id
	c.flusher.Write(REQ.Header.rawBody.B)                          // payload

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return err
	}

	c.h2WinServer -= payloadLengthBody
	c.h2Streams--

	return nil
}

func (c *Client) h2WindowUpdate() {
	if c.h2WinClient > c.h2PrvWinC {
		c.h2WinClient = 0
	}

	if c.h2PrvWinC == 65535 {
		c.h2PrvWinC = 6655535
	}

	increment := c.h2PrvWinC - c.h2WinClient

	c.flusher.Write([]byte{0x00, 0x00, 0x04})
	c.flusher.WriteByte(0x8)
	c.flusher.WriteByte(0x00)
	binary.Write(c.flusher, binary.BigEndian, uint32(0))
	binary.Write(c.flusher, binary.BigEndian, uint32(increment))
	c.flusher.Flush()
	c.h2WinClient += increment
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
	} else if !RES.Header.upgraded {
		RES.Http2Upgrade()
	}

	if c.TimeoutRead.Seconds() != 0 {
		c.NetConnection.SetReadDeadline(time.Now().Add(c.TimeoutRead))
		c.TimeoutRead = time.Duration(0 * time.Second)
	}

	c.run = true
	RES.Header.theHeader.Reset()
	RES.Header.theBuffer.Reset()

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, StreamID)

	pending := 0
	var privateWindow uint32 = 65535
	var streamsWindow uint32 = 65535
	var privateStream uint32 = StreamID

	for c.run {
		indexRaw := bytes.Index(c.theBuffer.B, data) - 5

		for indexRaw > -1 {
			payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, c.theBuffer.B[indexRaw:indexRaw+3]...))) + 9 + indexRaw
			if payloadLength > c.theBuffer.Len() {
				break
			}

			switch c.theBuffer.B[indexRaw+3] {
			case 0x0:
				RES.Header.theBuffer.Write(c.theBuffer.B[indexRaw+9 : payloadLength])
			case 0x1:
				RES.Header.theHeader.Write(c.theBuffer.B[indexRaw+9 : payloadLength])
			case 0x3:
				c.run = false
				c.theBuffer.B = append(c.theBuffer.B[:indexRaw], c.theBuffer.B[payloadLength:]...)
				return &RegnError{"the stream id " + strconv.Itoa(int(StreamID)) + " has been canceled by the server"}
			}

			if c.theBuffer.B[indexRaw+4] == 0x1 { // || c.theBuffer.B[indexRaw+4] == 0x0
				c.h2Streams++
				c.theBuffer.B = append(c.theBuffer.B[:indexRaw], c.theBuffer.B[payloadLength:]...)
				c.run = false
				return nil
			}

			c.theBuffer.B = append(c.theBuffer.B[:indexRaw], c.theBuffer.B[payloadLength:]...)
			indexRaw = bytes.Index(c.theBuffer.B, data) - 5
		}

		if streamsWindow > privateWindow || streamsWindow == 0 {
			privateWindow = 6553555
			streamsWindow = 6553555
			c.flusher.Write([]byte{0x00, 0x00, 0x04, 0x8, 0x00})
			binary.Write(c.flusher, binary.BigEndian, uint32(privateStream))
			binary.Write(c.flusher, binary.BigEndian, uint32(6553555))
			c.h2WindowUpdate()
		} else if c.h2WinClient > c.h2PrvWinC || c.h2WinClient == 0 {
			c.h2WindowUpdate()
		}

		if _, err := c.peeker.Peek(9); err != nil {
			c.Close()
			return err
		}

		buffered := c.peeker.Buffered()
		raw, _ := c.peeker.Peek(buffered)
		payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, raw[0:3]...))) + 9

		if pending != 0 {
			if buffered > pending {
				c.peeker.Discard(pending)
				c.theBuffer.Write(raw[:pending])
				pending = 0
				continue
			} else {
				pending -= buffered
				c.theBuffer.Write(raw)
				c.peeker.Discard(buffered)
				continue
			}

		} else if payloadLength > buffered {

			if raw[3] == 0x0 {
				Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
				if privateStream != Stream {
					streamsWindow = 65535
					privateWindow = 65535
					privateStream = Stream
				}

				c.h2WinClient -= uint32(payloadLength - 9)
				streamsWindow -= uint32(payloadLength - 9)
			}

			pending = payloadLength - buffered
			c.theBuffer.Write(raw)
			c.peeker.Discard(buffered)
			continue
		}

		c.peeker.Discard(payloadLength)

		switch raw[3] {
		case 0x0:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			if privateStream != Stream {
				streamsWindow = 65535
				privateWindow = 65535
				privateStream = Stream
			}

			c.h2WinClient -= uint32(payloadLength - 9)
			streamsWindow -= uint32(payloadLength - 9)
			if StreamID != Stream { // || raw[4] != 0x1 && raw[4] != 0x0
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}

			RES.Header.theBuffer.Write(raw[9:payloadLength])
			if raw[4] == 0x1 { // || raw[4] == 0x0
				c.h2Streams++
				c.run = false
			}
		case 0x1:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			if StreamID != Stream { // || raw[4] != 0x4 && raw[4] != 0x1 && raw[4] != 0x0
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}

			RES.Header.theHeader.Write(raw[9:payloadLength])

			if raw[4] == 0x1 || raw[4] == 0x5 { // || raw[4] == 0x0
				c.h2Streams++
				c.run = false
			}
		case 0x3:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			if Stream != StreamID {
				c.h2Streams++
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}

			c.h2Streams++
			c.run = false
			return &RegnError{"the stream id `" + strconv.Itoa(int(StreamID)) + "` has been canceled by the server"}

		case 0x4:
			raw = raw[9:]
			for len(raw) >= 6 {
				switch binary.BigEndian.Uint16(raw[:2]) {
				case 0x03:
					c.h2Streams = binary.BigEndian.Uint32(raw[2:6]) - c.h2Streams
				case 0x04:
					c.h2WinServer = binary.BigEndian.Uint32(raw[2:6]) - c.h2WinServer
				case 0x05:
					c.h2FrameSize = binary.BigEndian.Uint32(raw[2:6]) - c.h2FrameSize
				}
				raw = raw[6:]
			}

		case 0x7:
			c.Close()
			return &RegnError{Message: "the connection has been closed by the server 'http2.GoAwayFrame'"}
		case 0x8:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			if Stream == 0 {
				winsize := binary.BigEndian.Uint32(raw[9:13])
				c.h2WinServer += winsize
			}
			// else {
			// 	c.flusher.Write(raw)
			// 	c.flusher.Flush()
			// }
		}
	}

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

	if c.TimeoutRead.Seconds() != 0 {
		c.NetConnection.SetReadDeadline(time.Now().Add(c.TimeoutRead))
		c.TimeoutRead = time.Duration(0 * time.Second)
	}

	c.run = true

	if RES.Header.upgraded {
		RES.HttpDowngrade()
	}

	if len(REQ.Header.hpackHeaders) != 0 {
		REQ.HttpDowngrade()
	}

	if _, err := c.flusher.Write(REQ.Header.raw.B); err != nil {
		c.Close()
		return err
	}

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return err
	}

	RES.Header.theBuffer.Reset()
	for {
		if _, err := c.peeker.Peek(1); err != nil {
			c.Close()
			return err
		}
		le := c.peeker.Buffered()
		raw, _ := c.peeker.Peek(le)

		c.peeker.Discard(le)
		RES.Header.theBuffer.Write(raw)
		raw = nil

		if bytes.Contains(RES.Header.theBuffer.B, lines[1:]) {
			contentLengthMatch := lenRegex.FindSubmatch(RES.Header.theBuffer.B)
			if len(contentLengthMatch) > 1 {
				contentLength, _ := strconv.Atoi(string(contentLengthMatch[1]))
				contentLengthMatch[0] = nil
				contentLengthMatch[1] = nil

				if len(bytes.SplitN(RES.Header.theBuffer.B, lines[1:], 2)[1]) >= contentLength {
					break
				}
			} else if bytes.Contains(RES.Header.theBuffer.B, lines) {
				break
			}
		}
	}

	c.run = false
	return nil
}
