package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
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
	responseStatus int
	responseBody   []byte
	error          error
}

// init initializes the session with a new request and response writer.
func (s *Session) init(w http.ResponseWriter, r *http.Request) error {
	s.requestID = uid.New(uid.RequestPrefix)
	s.startTime = time.Now()
	s.w = w
	s.r = r
	s.WorkspaceID = ""

	// Read and cache the request body so metrics middleware can access it even on early errors.
	// We need to replace r.Body with a fresh reader afterwards so other middleware
	// can still read the body if necessary.
	var err error
	s.requestBody, err = io.ReadAll(s.r.Body)
	closeErr := s.r.Body.Close()

	// Handle read errors
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("unable to read request body"),
			fault.Public("The request body could not be read."),
		)
	}

	// Handle close error
	if closeErr != nil {
		return fault.Wrap(closeErr,
			fault.Internal("failed to close request body"),
			fault.Public("An error occurred processing the request."),
		)
	}

	// Replace body with a fresh reader for subsequent middleware
	s.r.Body = io.NopCloser(bytes.NewReader(s.requestBody))
	return nil
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

// CaptureResponseWriter returns a ResponseWriter that captures the response body.
// It returns the wrapper and a function to retrieve the captured data.
func (s *Session) CaptureResponseWriter() (http.ResponseWriter, func()) {
	wrapper := &captureResponseWriter{
		body:           []byte{},
		written:        false,
		ResponseWriter: s.w,
		statusCode:     http.StatusOK, // Default to 200 if not set
	}

	// Return a function to store captured data back in session
	capture := func() {
		s.responseStatus = wrapper.statusCode
		s.responseBody = wrapper.body
	}

	return wrapper, capture
}

// SetError stores the error for logging purposes.
func (s *Session) SetError(err error) {
	if s.error == nil {
		s.error = err
	}
}

// UserAgent returns the User-Agent header from the request.
func (s *Session) UserAgent() string {
	return s.r.UserAgent()
}

// Location returns the client's IP address, checking X-Forwarded-For header first,
// then falling back to RemoteAddr.
func (s *Session) Location() string {
	xff := s.r.Header.Get("X-Forwarded-For")
	if xff != "" {
		ips := strings.Split(xff, ",")
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				return ip
			}
		}
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(s.r.RemoteAddr)
	if err == nil {
		return host
	}
	return s.r.RemoteAddr
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
	s.responseStatus = status
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
	s.error = nil
}

// captureResponseWriter wraps http.ResponseWriter to capture the status code and response body.
type captureResponseWriter struct {
	http.ResponseWriter
	statusCode int
	body       []byte
	written    bool
}

func (w *captureResponseWriter) WriteHeader(code int) {
	if w.written {
		return // Already written, don't write again
	}

	w.statusCode = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

func (w *captureResponseWriter) Write(b []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}

	// Capture the body
	w.body = append(w.body, b...)

	return w.ResponseWriter.Write(b)
}
