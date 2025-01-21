package session

import (
	"context"
	"encoding/json"
	"net/http"

	apierrors "github.com/unkeyed/unkey/go/pkg/api/errors"
	"github.com/unkeyed/unkey/go/pkg/api/validation"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Session is a thin wrapper on top of go's standard library net/http
// It offers convenience methods to parse requests and send responses.
//
// Session structs are reused to ease the load on the GC.
// All references to sessions, request bodies or anything within must not be
// used outside of the handler. Make a copy of them if you need to.
type Session[TRequest Redacter, TResponse Redacter] interface {
	// Initialize the empty or reused session
	Init(w http.ResponseWriter, r *http.Request)

	// The underlying go context
	Context() context.Context

	// Validate ensures the request conforms to the openapi specs.
	//
	// If an issue is encountered a proper response is sent to the client
	// and a non-nil error is returned.
	// You should return this error as is from the handler.
	//
	// Example:
	//
	// func handle(s session.Session[Req, Res]) error {
	// err := s.Validate()
	// 	if err != nil {
	//  	return err
	// 	}
	// }
	Validate() error

	// The unique request id for this session.
	RequestID() string

	Request() Request[TRequest]

	// Send a json response
	JSON(status int, body TResponse) error

	// Returns the underlying response writer as an escape hatch.
	ResponseWriter() http.ResponseWriter

	// Send an error
	Error(apierrors.Error) error

	Close()

	// Reset sets all fields to their nil values to free up any references
	//
	// You do not need to call this manually.
	Reset()

	// Flush writes the statuscode, headers and body to the responseWriter
	//
	// Do not call this yourself, it'll be called automatically.
	Flush() error

	// Summary redacts sensitive data and returns all relevant io.
	// This must never be called before or during the main handler.
	//
	// It should only be called in the metrics middleware.
	Summary() (Summary, error)
}

type session[TRequest Redacter, TResponse Redacter] struct {
	requestID string

	validator validation.OpenAPIValidator
	w         http.ResponseWriter
	r         *http.Request

	responseStatus int
	responseHeader http.Header
	responseBody   Redacter
}

func New[TRequest Redacter, TResponse Redacter](validator validation.OpenAPIValidator) *session[TRequest, TResponse] {
	return &session[TRequest, TResponse]{
		requestID: "",
		validator: validator,
		w:         nil,
		r:         nil,
	}
}

func (s *session[TRequest, TResponse]) Init(w http.ResponseWriter, r *http.Request) {

	s.requestID = uid.Request()
	s.w = w
	s.r = r

}

func (s *session[TRequest, TResponse]) Context() context.Context {
	return s.r.Context()
}

func (s *session[TRequest, TResponse]) Validate() error {
	error, valid := s.validator.Validate(s.r)
	if !valid {
		error.RequestId = s.requestID
		b, err := json.Marshal(error)
		if err != nil {
			return err
		}
		s.w.WriteHeader(400)
		_, err = s.w.Write(b)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *session[TRequest, TResponse]) RequestID() string {
	return s.requestID
}

func (s *session[TRequest, TResponse]) ResponseWriter() http.ResponseWriter {
	return s.w
}

func (s *session[TRequest, TResponse]) Request() *request[TRequest] {
	return &request[TRequest]{
		r: s.r,
	}
}

func (s *session[TRequest, TResponse]) WriteHeader(key, val string) {
	s.responseHeader.Add(key, val)
}

func (s *session[TRequest, TResponse]) JSON(status int, body TResponse) error {

	s.responseStatus = status
	s.responseBody = body

	return nil

}
func (s *session[TRequest, TResponse]) Error(e apierrors.Error) error {
	s.responseStatus = e.HTTPStatus()
	s.responseBody = e
	return nil
}
func (s *session[TRequest, TResponse]) Close() {
	panic("IMPLEMENT ME")
}

func (s *session[TRequest, TResponse]) Summary() (Summary, error) {

	req := s.Request()
	requestHeader := req.Headers()
	for k := range requestHeader {
		if k == "authorization" {
			requestHeader.Del(k)
		}
	}

	req.body.Redact()
	requestBody, err := json.Marshal(req.body)
	if err != nil {
		return Summary{}, err
	}

	responseHeader := s.w.Header()

	s.responseBody.Redact()
	responseBody, err := json.Marshal(s.responseBody)
	if err != nil {
		return Summary{}, err
	}

	return Summary{
		Host:           s.r.Host,
		Method:         s.r.Method,
		Path:           s.r.URL.Path,
		RequestHeader:  requestHeader,
		RequestBody:    requestBody,
		ResponseStatus: s.responseStatus,
		ResponseHeader: responseHeader,
		ResponseBody:   responseBody,
	}, nil

}

func (s *session[TRequest, TResponse]) Reset() {
	s.requestID = ""

	s.validator = nil
	s.w = nil
	s.r = nil

	s.responseStatus = 0
	s.responseHeader = http.Header{}
	s.responseBody = nil
}

func (s *session[TRequest, TResponse]) Flush() error {
	s.w.WriteHeader(s.responseStatus)
	for k, vv := range s.responseHeader {
		for _, v := range vv {
			s.w.Header().Add(k, v)
		}
	}

	b, err := json.Marshal(s.responseBody)
	if err != nil {
		return err
	}
	_, err = s.w.Write(b)
	if err != nil {
		return err
	}

	return nil
}
