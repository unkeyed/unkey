package httpclient

import (
	"net/http"
	"time"
)

// Client is an HTTP client configured with base URL, default headers, and
// other options. It is safe for concurrent use — all state is immutable after
// construction and headers are cloned per request.
type Client struct {
	httpClient *http.Client
	baseURL    string
	headers    http.Header
}

// ClientOption configures a Client.
type ClientOption func(*Client)

// New creates a new Client with the given options.
func New(opts ...ClientOption) *Client {
	c := &Client{
		httpClient: http.DefaultClient,
		baseURL:    "",
		headers:    make(http.Header),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// WithTimeout sets the HTTP client timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) {
		// If the caller is still using the shared DefaultClient, clone it.
		if c.httpClient == http.DefaultClient {
			c.httpClient = &http.Client{}
		}
		c.httpClient.Timeout = d
	}
}

// WithBaseURL sets the base URL prepended to every request path.
func WithBaseURL(url string) ClientOption {
	return func(c *Client) {
		c.baseURL = url
	}
}

// WithHTTPClient replaces the underlying *http.Client.
func WithHTTPClient(hc *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = hc
	}
}

// WithHeader sets a default header sent with every request.
func WithHeader(key, value string) ClientOption {
	return func(c *Client) {
		c.headers.Set(key, value)
	}
}

// WithBearerToken sets the Authorization header to "Bearer <token>".
func WithBearerToken(token string) ClientOption {
	return func(c *Client) {
		c.headers.Set("Authorization", "Bearer "+token)
	}
}

// WithUserAgent sets the User-Agent header.
func WithUserAgent(ua string) ClientOption {
	return func(c *Client) {
		c.headers.Set("User-Agent", ua)
	}
}

// Get starts building a GET request.
func (c *Client) Get(path string) *Request {
	return newRequest(c, http.MethodGet, path, nil)
}

// Post starts building a POST request with the given body (JSON-marshaled).
func (c *Client) Post(path string, body any) *Request {
	return newRequest(c, http.MethodPost, path, body)
}

// Put starts building a PUT request with the given body (JSON-marshaled).
func (c *Client) Put(path string, body any) *Request {
	return newRequest(c, http.MethodPut, path, body)
}

// Patch starts building a PATCH request with the given body (JSON-marshaled).
func (c *Client) Patch(path string, body any) *Request {
	return newRequest(c, http.MethodPatch, path, body)
}

// Delete starts building a DELETE request.
func (c *Client) Delete(path string) *Request {
	return newRequest(c, http.MethodDelete, path, nil)
}
