package zen

import (
	"bufio"
	"net"
	"net/http"
)

// statusRecorder wraps http.ResponseWriter to capture the status code
type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	if !r.written {
		r.statusCode = statusCode
		r.written = true
		r.ResponseWriter.WriteHeader(statusCode)
	}
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if !r.written {
		// If WriteHeader hasn't been called, default to 200
		r.statusCode = 200
		r.written = true
	}

	return r.ResponseWriter.Write(b)
}

// Flush implements http.Flusher for streaming responses.
// Marks headers as written so metrics/logging reflect that headers were sent.
func (r *statusRecorder) Flush() {
	if !r.written {
		r.written = true
		if r.statusCode == 0 {
			r.statusCode = http.StatusOK
		}
	}
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// Hijack implements http.Hijacker for WebSocket and connection takeover.
// Delegates to underlying ResponseWriter if it supports hijacking.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hijacker, ok := r.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, ErrHijackNotSupported
}

// Push implements http.Pusher for HTTP/2 server push.
// Delegates to underlying ResponseWriter if it supports push.
func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	if pusher, ok := r.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return ErrPushNotSupported
}

// Unwrap returns underlying ResponseWriter for http.ResponseController.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}
