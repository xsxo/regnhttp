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
)

type __inforamtion__ struct {
	use_proxy      bool
	host_connected string
	connection     net.Conn
	run            bool

	peeker  *bufio.Reader
	flusher *bufio.Writer

	authorization string
	host_proxy    string
	port_proxy    string
	usedhttp2     bool
}

type Client struct {
	confgiuration *__inforamtion__
	Timeout       int
	TimeoutRead   int
	TlsConfig     *tls.Config
}

func (e *RegnError) Error() string {
	return "REGNHTTP Error: " + e.Message
}

func (cn *Client) Http2Upgrade() {
	cn.confgiuration.usedhttp2 = true
	cn.Close()
}

func (cn *Client) connectNet(host string, port string) error {
	if cn.Timeout == 0 {
		cn.Timeout = 10
	}

	var err error

	if port != "443" {
		cn.confgiuration.connection, err = net.DialTimeout("tcp", host+":"+port, time.Duration(cn.Timeout)*time.Second)
	} else {
		cn.confgiuration.connection, err = tls.Dial("tcp4", host+":"+port, cn.TlsConfig)
	}

	if err != nil {
		return &RegnError{Message: "Field connection with '" + host + "' host"}
	}

	// if cn.TimeoutRead == 0 {
	// 	cn.confgiuration.connection.SetDeadline(time.Now().Add(time.Duration(cn.Timeout) * time.Second))
	// }

	cn.create_line()
	err = nil
	return nil
}

func (cn *Client) connectHost(host_port string) error {
	therequest := bytes_pool.Get()
	therequest.Reset()
	therequest.WriteString("CONNECT " + host_port + " HTTP/1.1\r\nHost: " + host_port + "\r\nConnection: keep-Alive")

	if cn.confgiuration.authorization != "" {
		therequest.WriteString("Authorization: " + cn.confgiuration.authorization)
	}
	therequest.WriteString("\r\n\r\n")

	if _, err := cn.confgiuration.connection.Write(therequest.B); err != nil {
		cn.Close()
		return &RegnError{Message: "Field connection with '" + host_port + "' addr"}
	}

	if err := cn.confgiuration.flusher.Flush(); err != nil {
		cn.Close()
		return &RegnError{Message: "Field connection with '" + host_port + "' addr"}
	}
	therequest.Reset()
	bytes_pool.Put(therequest)

	buffer := make([]byte, 4096)
	if _, err := cn.confgiuration.peeker.Read(buffer); err != nil {
		return &RegnError{Message: "Field proxy connection with '" + host_port + "' addr"}
	}

	readed := status_code_regexp.FindSubmatch(buffer)
	buffer = nil

	if len(readed) == 0 {
		cn.Close()
		return &RegnError{Message: "Field proxy connection with '" + host_port + "' addr"}
	}

	readed[0] = nil
	return nil
}

func (cn *Client) Proxy(Url string) {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	} else if cn.confgiuration.connection != nil {
		cn.Close()
	}

	cn.confgiuration.host_connected = ""
	cn.confgiuration.use_proxy = true

	Parse, err := url.Parse(Url)
	if err != nil {
		cn.Close()
		panic("REGNHTTP: Invalid proxy format.")
	}

	if Parse.Host == "" {
		cn.Close()
		panic("REGNHTTP: No host proxy url supplied.")
	} else if Parse.Port() == "" {
		cn.Close()
		panic("REGNHTTP: No port proxy url supplied.")
	}

	cn.confgiuration.host_proxy = Parse.Hostname()
	cn.confgiuration.port_proxy = Parse.Port()

	if Parse.User.Username() != "" {
		password, _ := Parse.User.Password()
		credentials := Parse.User.Username() + ":" + password
		cn.confgiuration.authorization = ""
		cn.confgiuration.authorization = base64.StdEncoding.EncodeToString([]byte(credentials))
	}
}

func (cn *Client) Close() {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	} else if cn.confgiuration.connection != nil {
		cn.confgiuration.connection.Close()
	}

	cn.close_line()
	cn.confgiuration.connection = nil
	cn.confgiuration.host_connected = ""
	cn.confgiuration.run = false
}

func (cn *Client) close_line() {
	if cn.confgiuration.peeker != nil {
		nrpool.Put(cn.confgiuration.peeker)
		cn.confgiuration.peeker = nil
	}

	if cn.confgiuration.flusher != nil {
		nwpool.Put(cn.confgiuration.flusher)
		cn.confgiuration.flusher = nil
	}
}

func (cn *Client) create_line() {
	cn.close_line()
	cn.confgiuration.peeker = get_reader(cn.confgiuration.connection)
	cn.confgiuration.flusher = get_writer(cn.confgiuration.connection)
}

func (cn *Client) Connect(REQ *RequestType) error {
	if cn.confgiuration == nil {
		cn.confgiuration = &__inforamtion__{}
	} else if cn.confgiuration.host_connected != REQ.Header.myhost {
		cn.Close()
	}

	if cn.TlsConfig == nil {
		cn.TlsConfig = &tls.Config{}
		cn.TlsConfig.InsecureSkipVerify = false
	}

	if cn.confgiuration.run {
		panic("REGNHTTP: The client struct isn't support pool connections\ncreate a client for each connection || use sync.Pool for pool connections")
	}

	if cn.confgiuration.use_proxy {
		if cn.confgiuration.connection == nil {
			cn.TlsConfig.ServerName = REQ.Header.myhost
			if err := cn.connectNet(cn.confgiuration.host_proxy, cn.confgiuration.port_proxy); err != nil {
				return err
			}

			if err := cn.connectHost(REQ.Header.myhost + ":" + REQ.Header.myport); err != nil {
				return err
			}

			if REQ.Header.myport == "443" {
				cn.confgiuration.connection = tls.Client(cn.confgiuration.connection, cn.TlsConfig)
				cn.create_line()
			}

			cn.confgiuration.host_connected = REQ.Header.myhost
		}

	} else if cn.confgiuration.connection == nil {
		cn.TlsConfig.ServerName = REQ.Header.myhost
		if err := cn.connectNet(REQ.Header.myhost, REQ.Header.myport); err != nil {
			return err
		}
		cn.confgiuration.host_connected = REQ.Header.myhost
	}

	return nil

}

func (cn *Client) sendOld(REQ *RequestType, RES *ResponseType) error {
	if err := cn.Connect(REQ); err != nil {
		cn.Close()
		return err
	}

	cn.confgiuration.run = true

	if cn.TimeoutRead != 0 {
		cn.confgiuration.connection.SetDeadline(time.Now().Add(time.Duration(cn.TimeoutRead) * time.Second))
		cn.TimeoutRead = 0
	}

	cn.confgiuration.flusher.Write(REQ.Header.raw.B)
	if err := cn.confgiuration.flusher.Flush(); err != nil {
		cn.Close()
		return &RegnError{Message: " Error Writing \n" + err.Error()}
	}

	RES.Header.thebuffer.Reset()
	for {
		cn.confgiuration.peeker.Peek(1)
		le := cn.confgiuration.peeker.Buffered()
		if le == 0 {
			cn.Close()
			return &RegnError{Message: " Timeout Reading"}
		}

		peeked, _ := cn.confgiuration.peeker.Peek(le)
		cn.confgiuration.peeker.Discard(le)
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

	cn.confgiuration.run = false
	return nil

}

func (cn *Client) Do(REQ *RequestType, RES *ResponseType) error {
	return cn.sendOld(REQ, RES)
}
