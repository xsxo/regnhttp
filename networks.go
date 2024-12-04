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

	"golang.org/x/net/http2"
)

type Client struct {
	Timeout     int
	TimeoutRead int
	TlsConfig   *tls.Config

	// confgiuration
	use_proxy      bool
	host_connected string
	connection     net.Conn
	run            bool

	peeker  *bufio.Reader
	flusher *bufio.Writer

	authorization string
	host_proxy    string
	port_proxy    string
	upgraded      bool
	theFrame      *http2.Framer
	// streamId      uint32

	framChannel chan http2.Frame
}

func (e *RegnError) Error() string {
	return "REGNHTTP Error: " + e.Message
}

func (c *Client) ReturnFrame() *http2.Framer {
	return c.theFrame
}

// func (c *Client) Http2GenId() uint32 {
// 	if c.streamId == 0 {
// 		c.streamId++
// 		return c.streamId
// 	}
// 	c.streamId += 2

// 	return c.streamId
// }

func (c *Client) Status() string {
	if c.run {
		return "Runned"
	} else if c.connection != nil {
		return "Opned"
	}
	return "Closed"
}

func (c *Client) HttpVesrion() string {
	if c.upgraded {
		return "HTTP/2"
	} else {
		return "HTTP/1.1"
	}
}

// func (c *Client) HttpDowngrade() {
// 	c.Close()
// 	c.upgraded = false
// }

// func (c *Client) Http2Upgrade() {
// 	c.Close()
// 	c.upgraded = true
// }

func (c *Client) connectNet(host string, port string) error {
	if c.Timeout == 0 {
		c.Timeout = 10
	}

	var err error

	if port != "443" {
		c.connection, err = net.DialTimeout("tcp", host+":"+port, time.Duration(c.Timeout)*time.Second)
	} else {
		c.connection, err = tls.Dial("tcp4", host+":"+port, c.TlsConfig)
	}

	if err != nil {
		return &RegnError{Message: "Field create connection with '" + host + ":" + port + "' address"}
	}

	if c.TimeoutRead == 0 {
		c.connection.SetDeadline(time.Now().Add(time.Duration(c.Timeout) * time.Second))
	}

	c.createLines()
	err = nil
	return nil
}

func (c *Client) connectHost(host_port string) error {
	therequest := bytes_pool.Get()
	therequest.Reset()
	therequest.WriteString("CONNECT " + host_port + " HTTP/1.1\r\nHost: " + host_port + "\r\nConnection: keep-Alive")

	if c.authorization != "" {
		therequest.WriteString("Authorization: " + c.authorization)
	}
	therequest.WriteString("\r\n\r\n")

	if _, err := c.flusher.Write(therequest.B); err != nil {
		c.Close()
		return &RegnError{Message: "Field proxy connection with '" + host_port + "' address"}
	}

	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: "Field proxy connection with '" + host_port + "' address"}
	}
	therequest.Reset()
	bytes_pool.Put(therequest)

	buffer := make([]byte, 4096)
	if _, err := c.peeker.Read(buffer); err != nil {
		c.Close()
		return &RegnError{Message: "Field proxy connection with '" + host_port + "' address"}
	}

	readed := status_code_regexp.FindSubmatch(buffer)
	buffer = nil

	if len(readed) == 0 {
		c.Close()
		return &RegnError{Message: "Field proxy connection with '" + host_port + "' addr"}
	}

	readed[0] = nil
	return nil
}

func (c *Client) Proxy(Url string) {
	if c.connection != nil {
		c.Close()
	}

	c.host_connected = ""
	c.use_proxy = true

	Parse, err := url.Parse(Url)
	if err != nil {
		panic("REGNHTTP: Invalid proxy format.")
	}

	if Parse.Hostname() == "" {
		panic("REGNHTTP: No host proxy url supplied.")
	} else if Parse.Port() == "" {
		panic("REGNHTTP: No port proxy url supplied.")
	}

	c.host_proxy = Parse.Hostname()
	c.port_proxy = Parse.Port()

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
			c.host_connected = ""
		}
	} else {
		if c.connection != nil {
			c.connection.Close()
			c.host_connected = ""
			c.connection = nil
		}
	}

	if c.theFrame == nil {
		c.closeLines()
	} else {
		c.theFrame = nil
	}

	if c.framChannel != nil {
		close(c.framChannel)
		c.framChannel = nil
	}

	c.run = false
}

func (c *Client) closeLines() {
	if c.peeker != nil {
		nrpool.Put(c.peeker)
		c.peeker = nil
	}

	if c.flusher != nil {
		nwpool.Put(c.flusher)
		c.flusher = nil
	}
}

func (c *Client) createLines() {
	c.closeLines()
	c.peeker = get_reader(c.connection)
	c.flusher = get_writer(c.connection)
}

func (c *Client) Connect(REQ *RequestType) error {
	if c.host_connected != REQ.Header.myhost && c.host_connected != "" {
		c.Close()
	}

	if c.TlsConfig == nil {
		c.TlsConfig = &tls.Config{}
		c.TlsConfig.InsecureSkipVerify = true
	}

	if c.run {
		c.Close()
		panic("REGNHTTP: The client struct isn't support pool connections\ncreate a client for each connection || use sync.Pool for pool connections")
	}

	if c.host_connected == "" {
		c.TlsConfig.ServerName = REQ.Header.myhost

		if c.upgraded {
			c.TlsConfig.NextProtos = []string{"h2"}
		}

		if c.use_proxy {
			if err := c.connectNet(c.host_proxy, c.port_proxy); err != nil {
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
				panic("REGNHTTP/2: Can not use HTT2 Protocol without tls (https://)")
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
				{ID: http2.SettingMaxConcurrentStreams, Val: uint32(64)},
				{ID: http2.SettingInitialWindowSize, Val: uint32(65535)},
				{ID: http2.SettingHeaderTableSize, Val: uint32(4096)},
			}...); err != nil {
				c.Close()
				return &RegnError{Message: "Field send HTTP2 settings to the server"}
			}

			if frame, err := c.theFrame.ReadFrame(); err != nil {
				return &RegnError{Message: "Field read HTTP2 settings from the server"}
			} else if _, ok := frame.(*http2.SettingsFrame); !ok {
				return &RegnError{Message: "Field foramt HTTP2 settings from the server"}
			}

			// go c.readBackground()
		}
		c.host_connected = REQ.Header.myhost
	}

	return nil
}

// func (c *Client) Http2SendRequest(REQ *RequestType, ID uint32) error {
// 	if err := c.Connect(REQ); err != nil {
// 		return err
// 	}

// 	if c.TimeoutRead != 0 {
// 		c.connection.SetDeadline(time.Now().Add(time.Duration(c.TimeoutRead) * time.Second))
// 		c.TimeoutRead = 0
// 	}

// 	if REQ.Header.hpackHeaders == nil {
// 		REQ.upgradeH2c()
// 	}

// 	if err := c.theFrame.WriteHeaders(http2.HeadersFrameParam{
// 		StreamID:      ID,
// 		BlockFragment: REQ.Header.raw.B,
// 		EndHeaders:    true,
// 		EndStream:     false,
// 	}); err != nil {
// 		c.Close()
// 		return &RegnError{Message: "error writing http2 request\n" + err.Error()}
// 	}

// 	if err := c.theFrame.WriteData(ID, true, REQ.Header.rawBody.B); err != nil {
// 		c.Close()
// 		return &RegnError{Message: "error writing http2 request\n" + err.Error()}
// 	}

// 	return nil
// }

// func (c *Client) readBackground() {
// 	c.framChannel = make(chan http2.Frame)
// 	for c.theFrame != nil {
// 		if frame, err := c.theFrame.ReadFrame(); err != nil {
// 			c.framChannel <- frame
// 			frame = nil
// 		}
// 	}
// }

// func (c *Client) Http2ReadRespone(RES *ResponseType, ID uint32) error {
// 	if !c.upgraded {
// 		c.Http2Upgrade()

// 		return &RegnError{Message: "there's no connection to read response; use Client.Connect(request) function to create connection"}
// 	} else if RES.Header.decoder == nil {
// 		RES.upgradeH2c()
// 	}

// 	RES.Header.headers = []hpack.HeaderField{}
// 	RES.Header.thebuffer.Reset()

// 	for frame := range c.framChannel {
// 		switch f := frame.(type) {
// 		case *http2.HeadersFrame:
// 			if f.StreamID != ID {
// 				go func() {
// 					c.framChannel <- frame
// 				}()
// 				continue
// 			}

// 			_, err := RES.Header.decoder.Write(f.HeaderBlockFragment())
// 			if err != nil {
// 				return &RegnError{Message: "Field decode response headers; err: " + err.Error()}
// 			}

// 			if f.StreamEnded() {
// 				return nil
// 			}

// 		case *http2.DataFrame:
// 			if f.StreamID != ID {
// 				go func() {
// 					c.framChannel <- frame
// 				}()
// 				continue
// 			}

// 			RES.Header.thebuffer.Write(f.Data())

// 			if err := c.theFrame.WriteWindowUpdate(1, uint32(len(f.Data()))); err != nil {
// 				return &RegnError{Message: "Failed to send window update; err: " + err.Error()}
// 			}

// 			if f.StreamEnded() {
// 				return nil
// 			}

// 		case *http2.GoAwayFrame:
// 			c.Close()
// 			return &RegnError{Message: "The Connection has been closed"}
// 		}
// 		frame = nil

// 	}
// 	return nil
// }

func (c *Client) Do(REQ *RequestType, RES *ResponseType) error {
	if err := c.Connect(REQ); err != nil {
		return err
	}

	if c.TimeoutRead != 0 {
		c.connection.SetDeadline(time.Now().Add(time.Duration(c.TimeoutRead) * time.Second))
		c.TimeoutRead = 0
	}

	c.run = true

	c.flusher.Write(REQ.Header.raw.B)
	if err := c.flusher.Flush(); err != nil {
		c.Close()
		return &RegnError{Message: " writing error\n" + err.Error()}
	}

	RES.Header.thebuffer.Reset()
	for {
		c.peeker.Peek(1)
		le := c.peeker.Buffered()
		if le == 0 {
			c.Close()
			return &RegnError{Message: " timeout reading"}
		}

		peeked, _ := c.peeker.Peek(le)
		c.peeker.Discard(le)
		RES.Header.thebuffer.Write(peeked)
		peeked = nil

		if bytes.Contains(RES.Header.thebuffer.B, tow_lines) {
			contentLengthMatch := contetre.FindSubmatch(RES.Header.thebuffer.B) // changed form (*RES.Header.thebuffer).B
			if len(contentLengthMatch) > 1 {
				contentLength, _ := strconv.Atoi(string(contentLengthMatch[1]))
				contentLengthMatch[0] = nil
				contentLengthMatch[1] = nil

				if len(bytes.SplitN(RES.Header.thebuffer.B, tow_lines, 2)[1]) >= contentLength {
					break
				}
			} else if bytes.Contains(RES.Header.thebuffer.B, zero_lines) {
				break
			}
		}
	}

	c.run = false
	return nil
}
