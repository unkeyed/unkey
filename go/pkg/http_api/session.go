package httpApi

import (
	"context"
	"encoding/json"
	"net/http"

	apierrors "github.com/unkeyed/unkey/go/pkg/http_api/errors"
	"github.com/unkeyed/unkey/go/pkg/http_api/validation"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Session is a thin wrapper on top of go's standard library net/http
// It offers convenience methods to parse requests and send responses.
//
// Session structs are reused to ease the load on the GC.
// All references to sessions, request bodies or anything within must not be
// used outside of the handler. Make a copy of them if you need to.
type Session[TRequest Redacter, TResponse Redacter] struct {
	requestID string

	validator validation.OpenAPIValidator
	w         http.ResponseWriter
	r         *http.Request

	responseStatus int
	responseHeader http.Header
	responseBody   Redacter
}

func (s *Session[TRequest, TResponse]) Init(w http.ResponseWriter, r *http.Request) {

	s.requestID = uid.Request()
	s.w = w
	s.r = r

}

func (s *Session[TRequest, TResponse]) Context() context.Context {
	return s.r.Context()
}

func (s *Session[TRequest, TResponse]) Validate() error {
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

func (s *Session[TRequest, TResponse]) RequestID() string {
	return s.requestID
}

func (s *Session[TRequest, TResponse]) ResponseWriter() http.ResponseWriter {
	return s.w
}

func (s *Session[TRequest, TResponse]) Request() *request[TRequest] {
	return &request[TRequest]{
		r: s.r,
	}
}

func (s *Session[TRequest, TResponse]) WriteHeader(key, val string) {
	s.responseHeader.Add(key, val)
}

func (s *Session[TRequest, TResponse]) JSON(status int, body TResponse) error {

	s.responseStatus = status
	s.responseBody = body

	return nil

}
func (s *Session[TRequest, TResponse]) Error(e apierrors.Error) error {
	s.responseStatus = e.HTTPStatus()
	s.responseBody = e
	return nil
}

func (s *Session[TRequest, TResponse]) Reset() {
	s.requestID = ""

	s.validator = nil
	s.w = nil
	s.r = nil

	s.responseStatus = 0
	s.responseHeader = http.Header{}
	s.responseBody = nil
}

func (s *Session[TRequest, TResponse]) Flush() error {
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
