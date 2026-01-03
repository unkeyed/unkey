package zen

import "net/http"

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
