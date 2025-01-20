package session

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
)

type Request[TBody any] interface {

	// Get a value of a query
	// Returns an empty string if the query does not exist
	Query(name string) string

	// Get multiple query parameter values
	// eg: /?x=a&x=b
	Queries(name string) []string

	// Get a value of a Header
	// Returns an empty string if the header does not exist.
	Header(name string) string

	// Return all request headers.
	Headers() http.Header

	// Parse and validate the incoming body as json according to the openapi
	// validation rules.
	Parse() (TBody, error)

	// Reads the body as bytes
	//
	// If an error is encountered, it is handled and the session ends.
	//
	// You must not modify the returned slice, make a copy if you have to.
	Read() ([]byte, error)

	// The url of the request
	URL() *url.URL

	// The HTTP
	Method() string

	// Returns the underlying request as an escape hatch.
	//
	// Body.Close() is called automatically. You do not need to close it manually.
	Raw() *http.Request
}

type request[TRequest any] struct {
	r *http.Request
}

var _ Request[any] = &request[any]{}

func (r *request[TRequest]) Query(name string) string {
	return r.URL().Query().Get(name)
}
func (r *request[TRequest]) Queries(name string) []string {
	values, ok := r.URL().Query()[name]
	if !ok {
		return []string{}
	}
	return values
}

func (r *request[TRequest]) Headers() http.Header {
	return r.r.Header
}

func (r *request[TRequest]) Header(name string) string {
	return r.Headers().Get(name)
}

func (r *request[TRequest]) Method() string {
	return r.r.Method
}

func (r *request[TRequest]) Parse() (TRequest, error) {
	var body TRequest
	err := json.NewDecoder(r.r.Body).Decode(&body)

	return body, err
}
func (r *request[TRequest]) URL() *url.URL {
	return r.r.URL
}

func (r *request[TRequest]) Raw() *http.Request {
	return r.r
}

func (r *request[TRequest]) Read() ([]byte, error) {
	return io.ReadAll(r.r.Body)
}
