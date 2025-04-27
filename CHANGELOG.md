# v1.0.0 (24/12/30)
- First Version

# v1.1.0 (24/12/31)
- Fix Set Connection in bufio.Writer & bufio.Reader
- Fix Recuce use buffer when connection with proxy

# v1.2.0 (24/12/31)
- Fix reading http2 responses
- Fix window update frames http2
- Update reader function http2
- Fix proxy "http://" protocol

# v1.3.0 (25/1/1)
- Fix End stream http2
- Fix Payload legnth

# v1.4.0 (25/1/3)
- Fix c.theBuffer when save other streams
- Fix Pending var
- Fix Window Update Frame from client side
- Fix Benchmark regnhttp stopped without reason

# v1.5.0 (25/1/8)
- Fix Http2 Headers Clear after read
- Fix Http2 Decoder & Encoder
- Fix Http2 Response Buffer
- Fix Http2 Flusher in Http2SendRequest func

# v1.6.0 (25/1/9)
- Fix Http2 Read Headers -> Get & GetAll
- Fix Http2 Read StatusCode
- Fix Http2 Read Reason

# v1.7.0 (25/1/11)
- Fix Http2 Tags
- Fix Http2 REQ.HttpDowngrade function
- Improve performance Buffer writer in flush

# v1.8.0 (25/1/12)
- Fix Http2 stream flow control window
- Added `Http2StreamLevelFlowControl` object

# v1.9.0 (25/1/12)
- Removed `Http2StreamLevelFlowControl` object
- Convert `Http2StreamLevelFlowControl` to auto
- Added Http2SendHeaders function
- Added Http2SendBody function

# v1.10.0 (25/1/30)
- Fix established proxy

# v1.11.0 (25/3/26)
- Change raw established request
- Fix ReadDedline (read timeout)
- Change `NetConnection` of Client object to public
- Change Timeout type from int to time.Duration

# v1.12.0 (25/4/23)
- Change TimeoutRead && Timeout to static
- Improve SetBody && SetBodyString algorithm
- Improve Reading HTTP/1.1 response
- Support HTTP/1.0 proxies

# v1.13.0 (25/4/27)
- Added ReadBufferSize && WriteBufferSize
- Imporve `Http2ReadResponse` function
- Change stream id argument to auto stream id
- Change Name function `Http2SendRequest` to `Http2WriteRequest`
- New Tests Files && New Examples Files

# v1.14.0 (25/4/28)
- Fix Buffer Full at bufio.Reader
- Change name Json() function to BodyJson()
- Added benchmark http1.1
- New `README.md` File