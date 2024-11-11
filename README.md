## FiberHTTP (GoLang) - Beta Version
FiberHTTP is a lightweight Packge designed solely for High Performance</br>

## Fiberhttp might not for you
FiberHTTP it may lack many features, but this is intentional, FiberHTTP excludes them to maintain a lightweight codebase & maximum performance.
</br>
</br>
</br>**Missing Features**
- Streaming Requests
- Redirects Requests
- Pool Connection (you can create one by yourself using sync.Pool)

## Features (What makes FiberHTTP)
- `Connect` function (create connection with server before send requests)
- Auto Reconnecting when the server Disconnect
- Reducing pressure on the garbage collector
- used sync.Pool to avoid duplicating vars

## How to use
insatll the package
```bash
go get github.com/xsxo/fiberhttp-go
```

Take a lock in Examples folder: [Examples]([https://github.com/xsxo/fiberhttp/tree/main/benchmarks](https://github.com/xsxo/go-fiberhttp/tree/master/examples))