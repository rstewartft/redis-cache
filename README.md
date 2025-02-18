# redis-proxy

A proxy that sits between the client and the redis server. It is a simple proxy that forwards the request to the redis server and returns the response to the client.

Both http and RESP (REdis Serialization Protocol) are supported for a call to get a simple string value based on the string key.

## Architecture
![alt text](architecture.png)

## What the proxy does
- HTTP request:
  - The proxy listens on a port specified in the HTTP_ADDRESS env var for incoming requests. Note that the echo/built-in http packages in go support concurrent requests. It also supports rate limiting. It is set to 20 requests per second.
  - It checks if the key is present in the local LRU cache.
  - If the key is present in the cache, it returns the value from the cache.
  - If the key is not present in the cache, it looks for the request in redis.
  - If the key is present in redis, it returns the value from redis and updates the cache, otherwise it returns an error.

- RESP request:
  - The proxy listens on a port specified in the RESP_ADDRESS env var for incoming requests.
  - It parses the request based on the RESP protocol and returns an error if the request is not valid.
  - It then follows the steps above to find they value associated with the key.
  - It will return the response in RESP format.

## Algorithmic Complexity
Getting values from both the LRU cache and redis is O(1) time complexity.

## How to run tests
- Run `make test` to run the tests.
  - this will build the docker image and run the tests in the docker container.
- To run the proxy locally without docker, run `go build redis-proxy`

## How long did it take to complete the assignment?
- It took me about an hour to have the HTTP server working.
- It took me about 2-3 hours to have the RESP server working.
- It took me about 3 hours to get tests working for both the HTTP and RESP servers.
- Documentation took me about 30 mins.

## What requirements were not implemented?
- All the bonus requirements were completed, but both the rate limiter and concurrent processing were trivial thanks to Go's built-in libraries and echo.