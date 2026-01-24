## `REGNHTTP`
[![CI](https://github.com/xsxo/regnhttp/actions/workflows/go.yml/badge.svg)](https://github.com/xsxo/regnhttp/actions/workflows/go.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/xsxo/regnhttp)](https://pkg.go.dev/github.com/xsxo/regnhttp)
![Release](https://img.shields.io/github/v/release/xsxo/regnhttp?color=007d9c)
</br>
**regnhttp is high performance http client (low level) [example](https://github.com/xsxo/regnhttp/tree/main/examples/default-example)**</br>

## install package
```
go get -u github.com/xsxo/regnhttp
```

## update
```
go get -u github.com/xsxo/regnhttp@latest
```

## Features
- `Connect` Function (create connection with server before send requests)
- Reuse Request & Response object instead of creating a new one
- Reuse the same buffer to Reducing pressure on The Garbage Collector by `sync.Pool`
- No Thread Race | No Data Lose (all objects operate independently)
- Full control of client buffer `cleint.WriteBufferSize` and `client.ReadBufferSize`
- Full control of connection `client.Connection` & `client.TLSConfig` & `client.NagleOff`
- Full control of objects buffer `Request(bufferSize)` and `Request(Response)`
- Get the request & response as a raw `Request.Raw` & `Response.Raw`

## May not for you
- need to know the response & response buffer size
- no support pool connections (to avoid keep save open dead connections) 
- no support streaming requests
- no support compresser responses

! the regnhttp package is for normal requests & responses, not for full HTTP protocol support.
For other use cases, net/http may be a better choice, as it fully supports the HTTP protocol