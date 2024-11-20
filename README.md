## REGN HTTP - Beta Version
RegnHTTP is a lightweight Packge designed solely for Maximum Performance GoLang</br>

## REGN HTTP might not for you
RegnHTTP it may lack many features, but this is intentional, REGN HTTP excludes them to maintain a lightweight codebase & maximum performance.
</br>
</br>
</br>**Missing Features**
- Streaming Requests
- Redirects Requests
- Pool Connection (you can create one by yourself using sync.Pool)

## Features (What makes REGN HTTP)
- `Connect` Function (create connection with server before send requests)
- Auto Reconnecting when The Server Disconnect
- Reducing pressure on The Garbage Collector
- Bulit sync.Pool to Avoid Duplicating vars
- Support HTTP/2, HTTP/3 (Soon...)
- Mulit Requests on One Connection (soon...)
- No Thread Race | No Lose Data
- Built-in Socket Proxy Connection

## How To Use RegnHTTP
- insatll the package: `go get github.com/xsxo/regnhttp`

- Take a lock at The Examples folder: [Examples](https://github.com/xsxo/regnhttp/tree/master/examples)