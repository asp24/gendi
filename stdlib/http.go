package stdlib

import (
	"net/http"
	"time"
)

// NewHTTPClient creates an *http.Client with the specified timeout.
// This is the simplest factory for HTTP clients.
//
// Example:
//
//	parameters:
//	  http_timeout:
//	    type: time.Duration
//	    value: "30s"
//
//	services:
//	  http_client:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewHTTPClient"
//	      args:
//	        - "%http.timeout%"
//	    public: true
func NewHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
	}
}

// NewHTTPClientWithTransport creates an *http.Client with custom timeout and transport.
// Use this when you need connection pooling or other transport-level configuration.
//
// Example:
//
//	services:
//	  http_transport:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewHTTPTransport"
//	      args: [100, 10, 90s]
//
//	  http_client:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewHTTPClientWithTransport"
//	      args:
//	        - "%http.timeout%"
//	        - "@http.transport"
//	    public: true
func NewHTTPClientWithTransport(timeout time.Duration, transport http.RoundTripper) *http.Client {
	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

// NewHTTPTransport creates an *http.Transport with connection pool settings.
//
// Parameters:
//   - maxIdleConns: maximum number of idle connections across all hosts
//   - maxIdleConnsPerHost: maximum idle connections per host
//   - idleConnTimeout: how long idle connections remain in the pool
//
// Example:
//
//	services:
//	  http_transport:
//	    constructor:
//	      func: "github.com/asp24/gendi/stdlib.NewHTTPTransport"
//	      args: [100, 10, 90s]
func NewHTTPTransport(maxIdleConns, maxIdleConnsPerHost int, idleConnTimeout time.Duration) *http.Transport {
	return &http.Transport{
		MaxIdleConns:        maxIdleConns,
		MaxIdleConnsPerHost: maxIdleConnsPerHost,
		IdleConnTimeout:     idleConnTimeout,
	}
}
