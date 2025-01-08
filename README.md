## Faster than `net/http`
REGN HTTP pkg focuses on leveraging all the features of `HTTP/2` and `HTTP/3` that are not available in net/http to achieve high performance</br>
`net/http` uses a transport layer to switch between different HTTP versions, meaning you won't notice any difference when switching between versions in `net/http`.
#### Missing feauters of HTTP/2 & HTTP/3 in `net/http`
- Send & Read Multi Requests in same connection
- Read the response later after Send the Request
- Header Compression used Hpack & Qpack Algorithms
- Support HTTP/2 & HTTP/3 Directly (Without Tansport)

net/http uses a `transport layer` to handle the transition between different HTTP versions (creating a layer that converts requests from HTTP/1.1 to other versions). This means that, essentially, you are using multiple HTTP versions to send a request, leading to greater resource consumption and not fully leveraging the advantages of newer HTTP versions.

## Benchmarks multi streams
regn client:
```bash
goos: linux
goarch: amd64
pkg: github.com/xsxo/regnhttp/benchmark-regnhttp
cpu: AMD EPYC 7763 64-Core Processor                
BenchmarkRegnhttp-4   	       1	   4972462 ns/op
PASS
ok  	github.com/xsxo/regnhttp/benchmark-regnhttp	0.013s
```

fasthttp client:
```bash
goos: linux
goarch: amd64
pkg: benchmark
cpu: AMD EPYC 7763 64-Core Processor                
BenchmarkFasthttp-4   	       1	19938180508 ns/op
PASS
ok  	benchmark	19.943s
```

net/http client:
```bash
goos: linux
goarch: amd64
pkg: benchmark
cpu: AMD EPYC 7763 64-Core Processor                
BenchmarkNethttp-4   	       1	20989279354 ns/op
PASS
ok  	benchmark	20.995s
```

- The performance of regnhttp and fasthttp is very similar when using HTTP/1.1 only. This is because both libraries are built on the same performance optimization concepts.
<br>
- There is no performance difference between HTTP versions on local servers. The difference will only be noticeable when sending requests to a remote server.


## Features
- `Connect` Function (create connection with server before send requests)
- Reuse Request & Response object instead of creating a new one
- Reducing pressure on The Garbage Collector
- Bulit in sync.Pool to Avoid Duplicating vars
- No Thread Race | No Data Lose
- Mulit Requests on Same Connection
- Read the response later after Send the Request
- Header Compression used Hpack & Qpack Algorithms
- Support Proxy  Directly (Without Tansport)
- Support Socks  Directly (Without Tansport) (soon...)
- Support HTTP/2 Directly (Without Tansport)
- Support HTTP/3 Directly (Without Tansport) (soon...)
- Support Tansport (soon...)