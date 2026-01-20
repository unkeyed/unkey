package zen

import (
	"bufio"
	"errors"
	"net"
	"net/http"
)

// ErrorCapturingWriter wraps a ResponseWriter to capture proxy errors
// without writing them to the client. This allows errors to be returned
// to the middleware for consistent error handling.
//
// This is useful when using httputil.ReverseProxy where you want to handle
// proxy errors in your handler instead of letting the proxy write directly
// to the client.
type ErrorCapturingWriter struct {
	http.ResponseWriter
	capturedError error
	headerWritten bool
}

// NewErrorCapturingWriter creates a new error capturing writer that wraps
// the given ResponseWriter.
func NewErrorCapturingWriter(w http.ResponseWriter) *ErrorCapturingWriter {
	return &ErrorCapturingWriter{
		ResponseWriter: w,
		capturedError:  nil,
		headerWritten:  false,
	}
}

// Error returns any error that was captured during proxy, or nil if no error occurred.
func (w *ErrorCapturingWriter) Error() error {
	return w.capturedError
}

// SetError captures an error. This is typically called by httputil.ReverseProxy's ErrorHandler.
func (w *ErrorCapturingWriter) SetError(err error) {
	w.capturedError = err
}

// WriteHeader implements http.ResponseWriter. If an error was captured, the header
// write is discarded to prevent partial responses from being sent to the client.
func (w *ErrorCapturingWriter) WriteHeader(statusCode int) {
	if w.capturedError != nil {
		// Discard header writes if we captured an error
		w.headerWritten = true
		return
	}
	w.ResponseWriter.WriteHeader(statusCode)
	w.headerWritten = true
}

// Write implements http.ResponseWriter. If an error was captured, the body write
// is discarded to prevent partial responses from being sent to the client.
func (w *ErrorCapturingWriter) Write(b []byte) (int, error) {
	if w.capturedError != nil {
		// Discard body writes if we captured an error
		return len(b), nil
	}

	if !w.headerWritten {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(b)
}

// Flush implements http.Flusher for streaming responses.
// No-op when error captured (discarding response anyway).
// Ensures headers are written before flushing to support streaming.
func (w *ErrorCapturingWriter) Flush() {
	if w.capturedError != nil {
		return
	}
	if !w.headerWritten {
		w.ResponseWriter.WriteHeader(http.StatusOK)
		w.headerWritten = true
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// ErrHijackNotSupported is returned when the underlying ResponseWriter does not support hijacking.
var ErrHijackNotSupported = errors.New("hijack not supported")

// ErrHijackAfterError is returned when hijacking is attempted after an error was captured.
var ErrHijackAfterError = errors.New("hijack not allowed after error captured")

// ErrPushNotSupported is returned when the underlying ResponseWriter does not support HTTP/2 push.
var ErrPushNotSupported = errors.New("push not supported")

// Hijack implements http.Hijacker for WebSocket and connection takeover.
// Returns ErrHijackAfterError if an error was captured, as the connection state is undefined.
// Returns ErrHijackNotSupported if the underlying ResponseWriter doesn't support hijacking.
func (w *ErrorCapturingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if w.capturedError != nil {
		return nil, nil, ErrHijackAfterError
	}
	if hijacker, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hijacker.Hijack()
	}
	return nil, nil, ErrHijackNotSupported
}

// Push implements http.Pusher for HTTP/2 server push.
// No-op returning ErrPushNotSupported when error captured or underlying writer doesn't support push.
func (w *ErrorCapturingWriter) Push(target string, opts *http.PushOptions) error {
	if w.capturedError != nil {
		return ErrPushNotSupported
	}
	if pusher, ok := w.ResponseWriter.(http.Pusher); ok {
		return pusher.Push(target, opts)
	}
	return ErrPushNotSupported
}

// Unwrap returns underlying ResponseWriter for http.ResponseController.
func (w *ErrorCapturingWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
