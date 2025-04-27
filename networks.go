package regn

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"net"
	"net/url"
	"sync"
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

	lock  sync.Mutex
	h2Map map[uint32]*ResponseType
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
	if !c.upgraded {
		c.Close()
		if c.h2PrvStreams == 0 {
			c.h2PrvStreams++
		}
		c.upgraded = true
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
	c.NetConnection.SetReadDeadline(time.Now().Add(c.TimeoutRead))

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

	if c.upgraded {
		for key := range c.h2Map {
			c.h2Map[key].Header.StreamId = 0
			delete(c.h2Map, key)
		}
	}

	c.closeLines()
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

			if c.h2Map == nil {
				c.h2Map = make(map[uint32]*ResponseType)
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

// Support goroutine-safe
func (c *Client) Http2WriteRequest(REQ *RequestType, RES *ResponseType) error {
	if !c.upgraded {
		c.Http2Upgrade()
	}

	if REQ.Header.hpackHeaders == nil {
		REQ.Http2Upgrade()
	}

	if !RES.Header.upgraded {
		RES.Http2Upgrade()
	}

	payloadLengthHeaders := uint32(REQ.Header.raw.Len())
	payloadLengthBody := uint32(REQ.Header.rawBody.Len())

	if c.h2WinServer < payloadLengthBody {
		return &RegnError{"data > window server size"}
	} else if RES.Header.StreamId != 0 && !RES.Header.completed {
		return &RegnError{"the respone object is associated with a uncompleted request boject"}
	}

	c.lock.Lock()
	if err := c.Connect(REQ); err != nil {
		c.lock.Unlock()
		return err
	} else if c.h2Streams == 0 {
		c.lock.Unlock()
		return &RegnError{"concurrent streams id"}
	}

	c.h2PrvStreams += 2
	StreamID := c.h2PrvStreams
	c.h2Map[StreamID] = RES

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
	c.lock.Unlock()

	RES.Header.completed = false
	RES.Header.theHeader.Reset()
	RES.Header.theBuffer.Reset()
	RES.Header.StreamId = StreamID

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

// Support goroutine-safe
func (c *Client) Http2ReadRespone(RES *ResponseType) error {
	RES, EX := c.h2Map[RES.Header.StreamId]
	if !EX {
		return &RegnError{"the stream has been closed"}
	}

	RES.Header.streamWindow = 65535
	var streamsWindow uint32 = 65535
	var testStream uint32 = 65535
	var mathed int

	c.lock.Lock()
	for !RES.Header.completed {
		if streamsWindow > RES.Header.streamWindow || streamsWindow == 0 {
			RES.Header.streamWindow = 6553555
			streamsWindow = 6553555
			c.flusher.Write([]byte{0x00, 0x00, 0x04, 0x8, 0x00})
			binary.Write(c.flusher, binary.BigEndian, RES.Header.StreamId)
			binary.Write(c.flusher, binary.BigEndian, uint32(6553555))
			c.h2WindowUpdate()
		} else if c.h2WinClient > c.h2PrvWinC || c.h2WinClient == 0 {
			c.h2WindowUpdate()
		}

		rawPlayload, err := c.peeker.Peek(9)
		if err != nil {
			c.Close()
			c.lock.Unlock()
			return err
		}

		payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, rawPlayload[0:3]...))) + 9
		mathed = []int{payloadLength, 4096}[intToBool(payloadLength > 4096)]
		testStream -= uint32(payloadLength)
		if payloadLength > mathed {
			c.h2WinClient -= uint32(payloadLength - 9)
			streamsWindow -= uint32(payloadLength - 9)
			if streamsWindow > RES.Header.streamWindow || streamsWindow == 0 {
				RES.Header.streamWindow = 6553555
				streamsWindow = 6553555
				c.flusher.Write([]byte{0x00, 0x00, 0x04, 0x8, 0x00})
				binary.Write(c.flusher, binary.BigEndian, RES.Header.StreamId)
				binary.Write(c.flusher, binary.BigEndian, uint32(6553555))
				c.h2WindowUpdate()
			} else if c.h2WinClient > c.h2PrvWinC || c.h2WinClient == 0 {
				c.h2WindowUpdate()
			}

			Stream := binary.BigEndian.Uint32(rawPlayload[5:9]) & 0x7FFFFFFF
			other := c.h2Map[Stream]
			if rawPlayload[4] == 0x1 {
				c.h2Streams++
				other.Header.completed = true
			}

			for payloadLength != 0 {
				raw, err := c.peeker.Peek(mathed)
				if err != nil {
					c.Close()
					c.lock.Unlock()
					return err
				}
				c.peeker.Discard(mathed)
				other.Header.theBuffer.Write(raw)
				payloadLength -= mathed
				mathed = []int{payloadLength, 4096}[intToBool(payloadLength > 4096)]
			}
			other.Header.theBuffer.B = other.Header.theBuffer.B[3:]
			continue
		}

		raw, err := c.peeker.Peek(payloadLength)
		if err != nil {
			c.Close()
			c.lock.Unlock()
			return err
		}
		c.peeker.Discard(payloadLength)

		switch raw[3] {
		case 0x0:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			other := c.h2Map[Stream]
			c.h2WinClient -= uint32(payloadLength - 9)
			streamsWindow -= uint32(payloadLength - 9)
			other.Header.theBuffer.Write(raw[9:payloadLength])
			if raw[4] == 0x1 { // || raw[4] == 0x0
				c.h2Streams++
				other.Header.completed = true
			}
		case 0x1:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			other := c.h2Map[Stream]
			other.Header.theHeader.Write(raw[9:payloadLength])
			if raw[4] == 0x1 || raw[4] == 0x5 { // || raw[4] == 0x0
				c.h2Streams++
				other.Header.completed = true
			}
		case 0x3:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			c.h2Streams++
			other := c.h2Map[Stream]
			other.Header.theBuffer.Write(raw[:payloadLength])
			// other.Header.theBuffer.WriteString("closed stream id by server side")
			other.Header.completed = true
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
			c.lock.Unlock()
			return &RegnError{Message: "the connection has been closed by the server 'http2.GoAwayFrame'"}
		case 0x8:
			Stream := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
			if Stream == 0 {
				winsize := binary.BigEndian.Uint32(raw[9:13])
				c.h2WinServer += winsize
			}
		}
	}

	c.lock.Unlock()
	// RES.Header.StreamId = 0
	return nil
}

// Not support goroutine-safe
func (c *Client) Do(REQ *RequestType, RES *ResponseType) error {
	if c.upgraded {
		if err := c.Http2WriteRequest(REQ, RES); err != nil {
			return err
		} else if err = c.Http2ReadRespone(RES); err != nil {
			return err
		}
		return nil
	}

	if err := c.Connect(REQ); err != nil {
		return err
	}

	if RES.Header.upgraded {
		RES.HttpDowngrade()
	}

	if len(REQ.Header.hpackHeaders) != 0 {
		REQ.HttpDowngrade()
	}

	c.run = true
	if _, err := c.flusher.Write(REQ.Header.raw.B); err != nil {
		c.Close()
		return err
	}

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return err
	}

	RES.Header.theBuffer.Reset()
	var bodySize int
	var contentLength int
	for {
		if contentLength == 0 {
			if _, err := c.peeker.Peek(1); err != nil {
				c.Close()
				return err
			}
			le := c.peeker.Buffered()
			raw, _ := c.peeker.Peek(le)
			c.peeker.Discard(le)
			RES.Header.theBuffer.Write(raw)

			idx := bytes.Index(raw, contentLengthKey)
			if idx > 0 {
				i := idx + len(contentLengthKey)
				for ; i < len(RES.Header.theBuffer.B); i++ {
					c := RES.Header.theBuffer.B[i]
					if c < '0' || c > '9' {
						break
					}
					contentLength = contentLength*10 + int(c-'0')
				}
			}

			index := bytes.Index(RES.Header.theBuffer.B, lines[3:])
			if index > 0 {
				bodySize = len(RES.Header.theBuffer.B[index+4:])
			}
		} else {
			raw, err := c.peeker.Peek(contentLength - bodySize)
			if err != nil {
				c.Close()
				return err
			}

			lened := len(raw)
			c.peeker.Discard(lened)
			RES.Header.theBuffer.Write(raw)
			bodySize += lened
		}

		if contentLength != 0 && contentLength <= bodySize {
			break
		} else if bytes.Contains(RES.Header.theBuffer.B, lines) {
			RES.Header.theBuffer.B = RES.Header.theBuffer.B[:len(RES.Header.theBuffer.B)-7]
			break
		}
	}

	c.run = false
	return nil
}
