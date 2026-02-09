package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Request is a builder for a single HTTP request. It is not reusable.
type Request struct {
	client      *Client
	method      string
	path        string
	body        any // marshaled to JSON on execute; nil = no body
	headers     http.Header
	query       url.Values
	acceptCodes map[int]bool // nil = accept 2xx range
}

func newRequest(c *Client, method, path string, body any) *Request {
	return &Request{
		client:      c,
		method:      method,
		path:        path,
		body:        body,
		headers:     make(http.Header),
		query:       make(url.Values),
		acceptCodes: nil,
	}
}

// Header sets a per-request header, overriding any client-level default.
func (r *Request) Header(key, value string) *Request {
	r.headers.Set(key, value)
	return r
}

// Query adds a query parameter.
func (r *Request) Query(key, value string) *Request {
	r.query.Add(key, value)
	return r
}

// AcceptStatus overrides the default 2xx success check. Only the listed status
// codes are considered successful; all others produce an *Error.
func (r *Request) AcceptStatus(codes ...int) *Request {
	r.acceptCodes = make(map[int]bool, len(codes))
	for _, code := range codes {
		r.acceptCodes[code] = true
	}
	return r
}

// Send executes the request without decoding a response body.
func (r *Request) Send(ctx context.Context) (Response[struct{}], error) {
	raw, err := r.execute(ctx)
	resp := Response[struct{}]{
		StatusCode: raw.statusCode,
		Headers:    raw.headers,
		Body:       struct{}{},
		RawBody:    raw.body,
	}
	if err != nil {
		return resp, err
	}
	return resp, nil
}

// Do executes the request and JSON-decodes the response body into T.
func Do[Res any](ctx context.Context, req *Request) (Response[Res], error) {
	raw, err := req.execute(ctx)

	var zero Res
	resp := Response[Res]{
		StatusCode: raw.statusCode,
		Headers:    raw.headers,
		Body:       zero,
		RawBody:    raw.body,
	}
	if err != nil {
		return resp, err
	}

	if len(raw.body) > 0 {
		if err := json.Unmarshal(raw.body, &resp.Body); err != nil {
			return resp, fmt.Errorf("json decode: %w", err)
		}
	}

	return resp, nil
}

type rawResponse struct {
	statusCode int
	headers    http.Header
	body       []byte
}

func (r *Request) execute(ctx context.Context) (rawResponse, error) {
	// Marshal body
	var bodyReader io.Reader
	if r.body != nil {
		data, err := json.Marshal(r.body)
		if err != nil {
			return rawResponse{}, fmt.Errorf("json encode body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	// Build URL
	rawURL := r.client.baseURL + r.path
	if len(r.query) > 0 {
		rawURL += "?" + r.query.Encode()
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, r.method, rawURL, bodyReader)
	if err != nil {
		return rawResponse{}, fmt.Errorf("create request: %w", err)
	}

	// Merge headers: client defaults first, then per-request overrides
	for key, vals := range r.client.headers {
		for _, v := range vals {
			req.Header.Add(key, v)
		}
	}
	for key, vals := range r.headers {
		req.Header[key] = vals
	}

	// Auto-set Content-Type if body present and not already set
	if r.body != nil && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Execute
	resp, err := r.client.httpClient.Do(req)
	if err != nil {
		return rawResponse{}, fmt.Errorf("execute request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return rawResponse{}, fmt.Errorf("read response body: %w", err)
	}

	raw := rawResponse{
		statusCode: resp.StatusCode,
		headers:    resp.Header,
		body:       respBody,
	}

	// Check status
	if !r.isAcceptable(resp.StatusCode) {
		return raw, &Error{
			StatusCode: resp.StatusCode,
			Body:       respBody,
		}
	}

	return raw, nil
}

func (r *Request) isAcceptable(code int) bool {
	if r.acceptCodes != nil {
		return r.acceptCodes[code]
	}
	return code >= 200 && code < 300
}
