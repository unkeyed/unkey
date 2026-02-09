package httpclient

import "net/http"

// Response holds the result of an HTTP request.
// Body contains the JSON-decoded response when using Do[T].
// RawBody is always populated with the raw response bytes.
type Response[T any] struct {
	StatusCode int
	Headers    http.Header
	Body       T
	RawBody    []byte
}
