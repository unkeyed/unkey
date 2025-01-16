package session

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"

	apierrors "github.com/unkeyed/unkey/go/pkg/api/errors"
	"github.com/unkeyed/unkey/go/pkg/api/validation"
	"github.com/unkeyed/unkey/go/pkg/uid"
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
	r     *http.Request
	query url.Values
}

var _ Request[any] = &request[any]{}

func (r *request[TRequest]) Query(name string) string {
	return r.query.Get(name)
}
func (r *request[TRequest]) Queries(name string) []string {
	values, ok := r.query[name]
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

type Session[TRequest any, TResponse any] interface {
	// The underlying go context
	Context() context.Context

	// The unique request id for this session.
	RequestID() string

	Request() Request[TRequest]

	// Send a response
	Send(status int, body []byte)

	// Send a json response
	JSON(body TResponse)

	// Returns the underlying response writer as an escape hatch.
	ResponseWriter() http.ResponseWriter

	// Send an error
	Error(apierrors.Error)

	Close()
}

type session[TRequest any, TResponse any] struct {
	requestID string

	validator validation.OpenAPIValidator
	w         http.ResponseWriter
	r         *http.Request
}

func New[TRequest any, TResponse any](w http.ResponseWriter, r *http.Request) Session[TRequest, TResponse] {

	sess := &session[TRequest, TResponse]{
		requestID: uid.Request(),
		w:         w,
		r:         r,
	}

	return sess

}

func (s *session[TRequest, TResponse]) Context() context.Context {
	return s.r.Context()
}
func (s *session[TRequest, TResponse]) RequestID() string {
	return s.requestID
}

func (s *session[TRequest, TResponse]) ResponseWriter() http.ResponseWriter {
	return s.w
}

func (s *session[TRequest, TResponse]) Request() Request[TRequest] {
	panic("IMPLEMENT ME")
}

func (s *session[TRequest, TResponse]) WriteHeader(key, val string) {
	s.w.Header().Add(key, val)
}
func (s *session[TRequest, TResponse]) Send(status int, body []byte) {
	s.w.WriteHeader(status)
	_, err := s.w.Write(body)
	if err != nil {
		panic("HANDLE ME")
	}
}

func (s *session[TRequest, TResponse]) JSON(body TResponse) {
	panic("IMPLEMENT ME")
}
func (s *session[TRequest, TResponse]) Error(e apierrors.Error) {
	b, err := e.Marshal()
	if err != nil {
		panic("HANDLE ME")
	}
	_, err = s.w.Write(b)
	if err != nil {
		panic("HANDLE ME")
	}
}
func (s *session[TRequest, TResponse]) Close() {
	panic("IMPLEMENT ME")
}
