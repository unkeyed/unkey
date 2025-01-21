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

	// Send a response
	Send(status int, body []byte) error

	// Send a json response
	JSON(body TResponse) error

	// Returns the underlying response writer as an escape hatch.
	ResponseWriter() http.ResponseWriter

	// Send an error
	Error(apierrors.Error) error

	Close()
}

type session[TRequest any, TResponse any] struct {
	requestID string

	validator validation.OpenAPIValidator
	w         http.ResponseWriter
	r         *http.Request
}

func New[TRequest any, TResponse any](validator validation.OpenAPIValidator) *session[TRequest, TResponse] {
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

func (s *session[TRequest, TResponse]) Request() Request[TRequest] {
	return &request[TRequest]{
		r: s.r,
	}
}

func (s *session[TRequest, TResponse]) WriteHeader(key, val string) {
	s.w.Header().Add(key, val)
}
func (s *session[TRequest, TResponse]) Send(status int, body []byte) error {
	s.w.WriteHeader(status)
	_, err := s.w.Write(body)
	if err != nil {
		return err
	}
	return nil
}

func (s *session[TRequest, TResponse]) JSON(body TResponse) error {
	panic("IMPLEMENT ME")
}
func (s *session[TRequest, TResponse]) Error(e apierrors.Error) error {
	b, err := e.Marshal()
	if err != nil {
		return err
	}
	_, err = s.w.Write(b)
	if err != nil {
		return err
	}
	return nil
}
func (s *session[TRequest, TResponse]) Close() {
	panic("IMPLEMENT ME")
}
