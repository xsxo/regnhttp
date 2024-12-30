## Faster than `net/http`
REGN HTTP pkg focuses on leveraging all the features of `HTTP/2` and `HTTP/3` that are not available in net/http to achieve high performance</br>
`net/http` uses a transport layer to switch between different HTTP versions, meaning you won't notice any difference when switching between versions in `net/http`.
#### Missing feauters of HTTP/2 & HTTP/3 in `net/http`
- Send & Read Multi Requests in same connection
- Read the response later after Send the Request
- Header Compression used Hpack & Qpack Algorithms
- Support HTTP/2 & HTTP/3 Directly (Without Tansport)

net/http uses a `transport` layer to handle the transition between different HTTP versions (creating a layer that converts requests from HTTP/1.1 to other versions). This means that, essentially, you are using multiple HTTP versions to send a request, leading to greater resource consumption and not fully leveraging the advantages of newer HTTP versions.

## Features
- `Connect` Function (create connection with server before send requests)
- Reuse Request & Response object instead of creating a new one
- Reducing pressure on The Garbage Collector
- Bulit in sync.Pool to Avoid Duplicating vars
- No Thread Race | No Data Lose
- Mulit Requests on Same Connection
- Read the response later after Send the Request
- Header Compression used Hpack & Qpack Algorithms
- Support Proxy Directly (Without Tansport)
- Support HTTP/2 Directly (Without Tansport)
- Support HTTP/3 Directly (Without Tansport) (soon...)
- Support Tansport (soon...)