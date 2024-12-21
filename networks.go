package regn

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/valyala/bytebufferpool"
	"golang.org/x/net/http2"
)

type Client struct {
	// Timeout Connection
	Timeout int

	// Timeout Reading Response
	TimeoutRead int

	// Tls Context
	TlsConfig *tls.Config

	// Dialer to create dial connection
	Dialer *net.Dialer

	h2MaxIds uint32

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
	theFrame      *http2.Framer
	streamId      uint32

	theBuffer *bytebufferpool.ByteBuffer
}

func (c *Client) ReturnFrame() *http2.Framer {
	return c.theFrame
}

func (c *Client) Http2MaxIds() uint32 {
	return c.h2MaxIds
}

func (c *Client) Http2GenId() uint32 {
	if c.streamId <= c.h2MaxIds {
		c.streamId = 0
	}
	c.streamId += 1
	return c.streamId
}

func (c *Client) Status() bool {
	if c.hostConnected != "" {
		return true
	} else {
		return false
	}
}

func (c *Client) HttpVesrion() string {
	if c.upgraded {
		return "HTTP/2"
	} else {
		return "HTTP/1.1"
	}
}

func (c *Client) HttpDowngrade() {
	if c.upgraded {
		c.Close()
		c.upgraded = false
		// c.prmPool = nil
		// c.secPool = nil
	}
}

func (c *Client) Http2Upgrade() {
	c.Close()
	c.upgraded = true

	if c.theBuffer == nil {
		c.theBuffer = bufferPool.Get()
	}
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
		c.connection, err = tls.DialWithDialer(c.Dialer, "tcp4", host+":"+port, c.TlsConfig)
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

	if c.theBuffer != nil {
		c.theBuffer.Reset()
	}

	c.closeLines()
	c.upgraded = false
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

	if c.TlsConfig == nil {
		c.TlsConfig = &tls.Config{}
		c.TlsConfig.InsecureSkipVerify = false
	}

	if c.run {
		c.Close()
		panic("concurrent client writes")
	}

	if c.hostConnected == "" {
		c.TlsConfig.ServerName = REQ.Header.myhost

		if c.upgraded {
			c.TlsConfig.NextProtos = []string{"h2"}
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
				c.connection = tls.Client(c.connection, c.TlsConfig)
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

			if _, err := c.flusher.Write([]byte(http2.ClientPreface)); err != nil {
				c.Close()
				return err
			}

			if err := c.flusher.Flush(); err != nil {
				c.Close()
				return err
			}

			// c.closeLines()
			c.theFrame = http2.NewFramer(c.connection, c.connection)

			if err := c.theFrame.WriteSettings([]http2.Setting{
				{ID: http2.SettingMaxConcurrentStreams, Val: 12263},
				{ID: http2.SettingInitialWindowSize, Val: 65535},
				{ID: http2.SettingHeaderTableSize, Val: 4096},
				{ID: http2.SettingMaxHeaderListSize, Val: 4096},
			}...); err != nil {
				c.Close()
				return &RegnError{Message: "field send HTTP2 settings to the server"}
			}

			if frame, err := c.theFrame.ReadFrame(); err != nil {
				return &RegnError{Message: "field read HTTP2 settings from the server"}
			} else if f, ok := frame.(*http2.SettingsFrame); ok {
				f.ForeachSetting(func(s http2.Setting) error {
					if s.ID.String() == "MAX_CONCURRENT_STREAMS" {
						c.h2MaxIds = s.Val
					}
					return nil
				})
			} else {
				return &RegnError{Message: "field foramt HTTP2 settings from the server"}
			}
		}
		c.hostConnected = REQ.Header.myhost
	}

	return nil
}

func (c *Client) Http2SendRequest(REQ *RequestType, ID uint32) error {
	if !c.upgraded {
		c.Http2Upgrade()
	}

	if ID%2 == 0 {
		panic("id is not odd")
	}

	if err := c.Connect(REQ); err != nil {
		return err
	}

	c.run = true
	if c.TimeoutRead != 0 {
		c.Dialer.Deadline = time.Now().Add(time.Duration(c.TimeoutRead) * time.Second)
		c.TimeoutRead = 0
	}

	if REQ.Header.hpackHeaders == nil {
		REQ.Http2Upgrade()
	}

	bufHeaders := bytes.Buffer{}
	flagsHeaders := byte(0)
	flagsHeaders |= 0x4 // end headers (no end stream)

	payloadLengthHeaders := REQ.Header.raw.Len()
	bufHeaders.Write([]byte{
		byte(payloadLengthHeaders >> 16),
		byte(payloadLengthHeaders >> 8),
		byte(payloadLengthHeaders),
	})

	bufHeaders.WriteByte(0x1) // type of headers
	bufHeaders.WriteByte(flagsHeaders)
	binary.Write(&bufHeaders, binary.BigEndian, ID&0x7FFFFFFF)
	bufHeaders.Write(REQ.Header.raw.Bytes())

	// إعداد إطار الجسم (DATA)
	bufBody := bytes.Buffer{}
	flagsBody := byte(0)
	flagsBody |= 0x1 // end stream

	payloadLengthBody := REQ.Header.rawBody.Len()
	bufBody.Write([]byte{
		byte(payloadLengthBody >> 16),
		byte(payloadLengthBody >> 8),
		byte(payloadLengthBody),
	})
	bufBody.WriteByte(0x0) // type of body
	bufBody.WriteByte(flagsBody)
	binary.Write(&bufBody, binary.BigEndian, ID&0x7FFFFFFF)
	bufBody.Write(REQ.Header.rawBody.Bytes())

	// إرسال الإطارات
	c.flusher.Write(bufHeaders.Bytes())
	c.flusher.Write(bufBody.Bytes())
	c.flusher.Flush()

	c.run = false
	return nil
}

func (c *Client) Http2ReadRespone(RES *ResponseType, ID uint32) error {
	if c.run {
		panic("concurrent client reader")
	} else if !c.upgraded {
		c.Http2Upgrade()
		return &RegnError{"not found id"}
	}

	if ID%2 == 0 {
		panic("id is not odd")
	} else if c.hostConnected == "" {
		return &RegnError{"not found id"}
	}

	if RES.Header.decoder == nil {
		RES.Http2Upgrade()
	}

	c.run = true
	RES.Header.contectLegnth = -1
	RES.Header.thebuffer.Reset()

	data := make([]byte, 4)
	binary.BigEndian.PutUint32(data, ID)

	indexRaw := bytes.Index(c.theBuffer.B, data) - 5
	for indexRaw > -1 {
		payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, c.theBuffer.B[indexRaw:indexRaw+3]...))) + 9 + indexRaw
		if payloadLength > c.theBuffer.Len() {
			fmt.Println("<<<")
			os.Exit(1)
		}

		// raw := c.theBuffer.B[indexRaw:payloadLength]

		switch c.theBuffer.B[indexRaw+3] {
		case 0x0:
			// fmt.Println("stream id:", ID)
			// fmt.Println("len body from index:", len(raw[9:payloadLength]))
			// RES.Header.thebuffer.Write(raw[9 : payloadLength-9])
			RES.Header.thebuffer.Write(c.theBuffer.B[indexRaw+9 : payloadLength])
		case 0x1:
			// fmt.Println("stream id:", ID)
			// fmt.Println("len headers from index:", len(raw[9:payloadLength]))
			// RES.Header.decoder.Write(raw[9 : payloadLength-9])
			RES.Header.decoder.Write(c.theBuffer.B[indexRaw+9 : payloadLength])
		}

		if c.theBuffer.B[indexRaw+4]&0x1 != 0 {
			c.run = false
		}
		// fmt.Println("len:", c.theBuffer.Len())
		c.theBuffer.B = append(c.theBuffer.B[:indexRaw], c.theBuffer.B[payloadLength:]...)
		indexRaw = bytes.Index(c.theBuffer.B, data) - 5
	}

	// done without any problem
	for c.run {
		if _, err := c.peeker.Peek(9); err != nil {
			return &RegnError{Message: "timeout error"}
		}

		buffred := c.peeker.Buffered()

		if buffred == 0 {
			c.Close()
			return &RegnError{Message: "timeout error"}
		}

		raw, err := c.peeker.Peek(buffred)
		if err != nil {
			c.Close()
			return &RegnError{Message: "timeout error"}
		}

		StreamId := binary.BigEndian.Uint32(raw[5:9]) & 0x7FFFFFFF
		payloadLength := int(binary.BigEndian.Uint32(append([]byte{0}, raw[0:3]...))) + 9
		c.peeker.Discard(payloadLength)

		// if payloadLength == len(raw) {
		// 	fmt.Println("==")
		// } else {
		// 	fmt.Println("len raw:", len(raw))
		// 	fmt.Println("payload legnth:", payloadLength)
		// }

		switch raw[3] {
		case 0x0:
			// fmt.Println("stream id:", StreamId)
			// fmt.Println("len body from peek:", len(raw[9:payloadLength]))
			if StreamId != ID {
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}

			RES.Header.thebuffer.Write(raw[9:payloadLength])

			if raw[4]&0x1 != 0 {
				c.run = false
			}

		case 0x1:
			// fmt.Println("stream id:", ID)
			// fmt.Println("len headers from peek:", len(raw[9:payloadLength]))
			if StreamId != ID {
				c.theBuffer.Write(raw[:payloadLength])
				continue
			}

			RES.Header.decoder.Write(raw[9:payloadLength])

			if raw[4]&0x1 != 0 {
				c.run = false
			}

		case 0x7:
			c.Close()
			return &RegnError{Message: "the connection has been closed by the server 'http2.GoAwayFrame'"}
		case 0x3:
			fmt.Println("from RST")
		case 0x4:
			fmt.Println("from settings")
		case 0x8:
			fmt.Println("window update")

		}
	}

	return nil
}

func (c *Client) Do(REQ *RequestType, RES *ResponseType) error {
	if c.upgraded {
		id := c.Http2GenId()
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
		return &RegnError{Message: "field send request"}
	}

	RES.Header.thebuffer.Reset()
	for {
		c.peeker.Peek(1)
		le := c.peeker.Buffered()
		if le == 0 {
			c.Close()
			return &RegnError{Message: "read timeout"}
		}

		peeked, _ := c.peeker.Peek(le)
		c.peeker.Discard(le)
		RES.Header.thebuffer.Write(peeked)
		peeked = nil

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
