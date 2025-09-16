package server

import (
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// Session encapsulates the state and utilities for handling a single HTTP request
// in the gateway. It provides request ID tracking and workspace association.
type Session struct {
	requestID string
	startTime time.Time

	w http.ResponseWriter
	r *http.Request

	// The workspace making the request.
	// This can be extracted from headers or authentication.
	WorkspaceID string

	requestBody    []byte
	responseStatus int32
	responseBody   []byte
}

// init initializes the session with a new request and response writer.
func (s *Session) init(w http.ResponseWriter, r *http.Request) {
	s.requestID = uid.New(uid.RequestPrefix)
	s.startTime = time.Now()
	s.w = w
	s.r = r
	s.WorkspaceID = ""
}

// RequestID returns the unique request ID for this session.
func (s *Session) RequestID() string {
	return s.requestID
}

// Latency returns the time elapsed since the request started.
func (s *Session) Latency() time.Duration {
	return time.Since(s.startTime)
}

// Request returns the underlying http.Request.
func (s *Session) Request() *http.Request {
	return s.r
}

// ResponseWriter returns the underlying http.ResponseWriter.
func (s *Session) ResponseWriter() http.ResponseWriter {
	return s.w
}

// UserAgent returns the User-Agent header from the request.
func (s *Session) UserAgent() string {
	return s.r.UserAgent()
}

// Location returns the client's IP address, checking True-Client-IP header first,
// then falling back to RemoteAddr.
func (s *Session) Location() string {
	location := s.r.Header.Get("True-Client-Ip")
	if location == "" {
		host, _, err := net.SplitHostPort(s.r.RemoteAddr)
		if err == nil {
			location = host
		} else {
			location = s.r.RemoteAddr
		}
	}

	return location
}

// ReadBody reads and returns the request body.
// The body is cached in the session for subsequent reads.
func (s *Session) ReadBody() ([]byte, error) {
	if s.requestBody != nil {
		return s.requestBody, nil
	}

	var err error
	s.requestBody, err = io.ReadAll(s.r.Body)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("unable to read request body"))
	}
	defer s.r.Body.Close()

	return s.requestBody, nil
}

// JSON sends a JSON response with the given status code.
func (s *Session) JSON(status int, body any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return fault.Wrap(err, fault.Internal("json marshal failed"))
	}

	s.w.Header().Set("Content-Type", "application/json")

	return s.send(status, b)
}

// HTML sends an HTML response with the given status code.
func (s *Session) HTML(status int, body []byte) error {
	s.w.Header().Set("Content-Type", "text/html")

	return s.send(status, body)
}

// Plain sends a plain text response with the given status code.
func (s *Session) Plain(status int, body []byte) error {
	s.w.Header().Set("Content-Type", "text/plain")

	return s.send(status, body)
}

// Send sends a raw response with the given status code.
func (s *Session) Send(status int, body []byte) error {
	return s.send(status, body)
}

// send is the internal method for sending responses.
func (s *Session) send(status int, body []byte) error {
	// Store for middleware use
	s.responseStatus = int32(status)
	s.responseBody = body

	s.w.WriteHeader(status)
	log.Printf("gateway error: %d urn: %s", status, s.RequestID())

	_, err := s.w.Write(body)
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to send bytes"))
	}

	return nil
}

// reset clears the session for reuse.
func (s *Session) reset() {
	s.requestID = ""
	s.w = nil
	s.r = nil
	s.WorkspaceID = ""
	s.requestBody = nil
	s.responseStatus = 0
	s.responseBody = nil
}

// wrapResponseWriter wraps http.ResponseWriter to capture the status code.
type wrapResponseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (w *wrapResponseWriter) WriteHeader(code int) {
	if w.written {
		return // Already written, don't write again
	}

	w.statusCode = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *wrapResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}

	return w.ResponseWriter.Write(b)
}
