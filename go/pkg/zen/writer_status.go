package zen

import "net/http"

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
