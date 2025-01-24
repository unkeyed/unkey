package zen

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Session is a thin wrapper on top of go's standard library net/http
// It offers convenience methods to parse requests and send responses.
//
// Session structs are reused to ease the load on the GC.
// All references to sessions, request bodies or anything within must not be
// used outside of the handler. Make a copy of them if you need to.
type Session struct {
	requestID string

	w http.ResponseWriter
	r *http.Request

	requestBody    []byte
	responseStatus int
	responseBody   []byte
}

func (s *Session) Init(w http.ResponseWriter, r *http.Request) error {
	s.requestID = uid.Request()
	s.w = w
	s.r = r

	return nil
}

func (s *Session) Context() context.Context {
	return s.r.Context()
}

// Request returns the underlying http.Request.
//
// Do not store references or modify it outside of the handler function.
func (s *Session) Request() *http.Request {
	return s.r
}

func (s *Session) RequestID() string {
	return s.requestID
}

func (s *Session) ResponseWriter() http.ResponseWriter {
	return s.w
}

func (s *Session) BindBody(dst any) error {
	err := json.Unmarshal(s.requestBody, dst)
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("failed to unmarshal request body", "The request body was not valid json."),
		)
	}
	return nil
}

func (s *Session) AddHeader(key, val string) {
	s.w.Header().Add(key, val)
}

func (s *Session) send(status int, body []byte) error {
	// Store the status and body for middleware use
	// Unlike the headers, we can't access it on the responseWriter
	s.responseStatus = status
	s.responseBody = body

	s.w.WriteHeader(status)
	_, err := s.w.Write(body)
	if err != nil {
		return fault.Wrap(err, fault.WithDesc("failed to send bytes", "Unable to send response body."))
	}

	return nil
}

// Send sets the response status and header
// It then marshals the body as JSON and sends it to the client.
func (s *Session) JSON(status int, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fault.Wrap(err,
			fault.WithDesc("json marshal failed", "The response body could not be marshalled to JSON."),
		)
	}
	s.ResponseWriter().Header().Add("Content-Type", "application/json")
	return s.send(status, b)
}

// reset is called automatically before the session is returned to the pool.
// It resets all fields to their null value to prevent leaking data between
// requests.
func (s *Session) reset() {
	s.requestID = ""

	s.w = nil
	s.r = nil

	s.requestBody = nil
	s.responseStatus = 0
	s.responseBody = nil
}
