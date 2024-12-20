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
	winBlock  uint32
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

	if c.theFrame == nil {
		c.closeLines()
	} else {
		c.theFrame = nil
	}

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

			c.closeLines()
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

	if err := c.theFrame.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      ID,
		BlockFragment: REQ.Header.raw.B,
		EndHeaders:    true,
		EndStream:     false,
	}); err != nil {
		c.Close()
		return &RegnError{Message: "field send http2 request\n" + err.Error()}
	}

	if err := c.theFrame.WriteData(ID, true, REQ.Header.rawBody.B); err != nil {
		c.Close()
		return &RegnError{Message: "field send http2 request\n" + err.Error()}
	}

	// if c.winBlock >= 60000 {
	// 	if err := c.theFrame.WriteWindowUpdate(c.streamId, c.winBlock); err != nil {
	// 		return &RegnError{Message: "field send HTTP2 settings to the server"}
	// 	}
	// }

	c.run = false
	return nil
}

func (c *Client) Http2ReadRespone(RES *ResponseType, ID uint32) error {
	if c.run {
		panic("concurrent client reader")
	} else if !c.upgraded {
		c.Http2Upgrade()
		return nil
	}

	if ID%2 == 0 {
		panic("id is not odd")
	}

	if RES.Header.decoder == nil {
		RES.Http2Upgrade()
	}

	c.run = true
	RES.Header.contectLegnth = -1
	RES.Header.thebuffer.Reset()

	headStart := []byte{byte(ID), 0x11}
	indexHead := bytes.Index(c.theBuffer.B, headStart)

	if indexHead != -1 {
		HeadEnd := bytes.Index(c.theBuffer.B, []byte{byte(ID), 0x14})
		RES.Header.decoder.Write(c.theBuffer.B[indexHead+2 : HeadEnd])
		c.theBuffer.B = append(c.theBuffer.B[:indexHead], c.theBuffer.B[HeadEnd+2:]...)
	}

	bodyStart := []byte{byte(ID), 0x91}
	indexBody := bytes.Index(c.theBuffer.B, bodyStart)

	if indexBody != -1 {
		bodyEnd := bytes.Index(c.theBuffer.B, []byte{byte(ID), 0x94})
		RES.Header.thebuffer.Write(c.theBuffer.B[indexBody+2 : bodyEnd])
		c.theBuffer.B = append(c.theBuffer.B[:indexBody], c.theBuffer.B[bodyEnd+2:]...)
		if RES.Header.contectLegnth == RES.Header.thebuffer.Len() || bytes.Contains(RES.Header.thebuffer.B, lines) {
			c.run = false
		}
		c.run = false
	}

	for c.run {
		frame, _ := c.theFrame.ReadFrame()
		switch f := frame.(type) {
		case *http2.HeadersFrame:
			if f.StreamID != ID {
				c.theBuffer.Write([]byte{byte(f.StreamID), 0x11})
				c.theBuffer.Write(f.HeaderBlockFragment())
				c.theBuffer.Write([]byte{byte(f.StreamID), 0x14})
				continue
			}

			RES.Header.decoder.Write(f.HeaderBlockFragment())

			if f.StreamEnded() {
				c.run = false
			}
			f = nil
		case *http2.DataFrame:
			// if err := c.theFrame.WriteWindowUpdate(c.streamId, uint32(len(f.Data()))); err != nil {
			// 	return &RegnError{Message: "field send HTTP2 settings to the server"}
			// }

			if f.StreamID != ID {
				// bodyStart[0] = byte(f.StreamID)
				c.theBuffer.Write([]byte{byte(f.StreamID), 0x91})
				c.theBuffer.Write(f.Data())
				c.theBuffer.Write([]byte{byte(f.StreamID), 0x94})
				continue
			}

			c.winBlock += f.Length
			RES.Header.thebuffer.Write(f.Data())

			if f.StreamEnded() {
				c.run = false
			}

			f = nil
		case *http2.GoAwayFrame:
			c.Close()
			return &RegnError{Message: "the connection has been closed by the server 'http2.GoAwayFrame'\n" + "response: " + f.ErrCode.String()}

		case *http2.RSTStreamFrame:
			c.run = false
			return &RegnError{Message: "the request id has been cancelled by the server '" + f.ErrCode.String() + "'"}

		case *http2.SettingsFrame:
			f.ForeachSetting(func(s http2.Setting) error {
				if s.ID.String() == "MAX_CONCURRENT_STREAMS" {
					c.h2MaxIds = s.Val
				}

				return nil
			})

		case nil:
			return &RegnError{Message: "the connection has been closed by the server (http2.NilFrame)"}
		}
		frame = nil
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
