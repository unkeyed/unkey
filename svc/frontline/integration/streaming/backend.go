package streaming

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// SSEBackend is a simple Server-Sent Events server for testing streaming support.
type SSEBackend struct {
	server *http.Server
	addr   string
}

// NewSSEBackend creates a new SSE backend server.
func NewSSEBackend(addr string) *SSEBackend {
	return &SSEBackend{addr: addr}
}

// Start starts the SSE backend server.
func (b *SSEBackend) Start() error {
	mux := http.NewServeMux()

	// SSE endpoint that streams events
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Backend-Received", "true")

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		// Send initial event immediately
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		flusher.Flush()

		// Send numbered events
		for i := 1; i <= 5; i++ {
			select {
			case <-r.Context().Done():
				return
			case <-time.After(100 * time.Millisecond):
				fmt.Fprintf(w, "event: message\ndata: {\"seq\":%d}\n\n", i)
				flusher.Flush()
			}
		}

		// Send completion event
		fmt.Fprintf(w, "event: done\ndata: {\"status\":\"complete\"}\n\n")
		flusher.Flush()
	})

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Echo endpoint that returns request headers (for debugging)
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, "{\"host\":\"%s\",\"method\":\"%s\",\"path\":\"%s\"}", r.Host, r.Method, r.URL.Path)
	})

	//nolint:exhaustruct
	b.server = &http.Server{
		Addr:    b.addr,
		Handler: mux,
	}

	return b.server.ListenAndServe()
}

// Shutdown gracefully shuts down the SSE backend.
func (b *SSEBackend) Shutdown(ctx context.Context) error {
	if b.server == nil {
		return nil
	}
	return b.server.Shutdown(ctx)
}

// Address returns the address the server is listening on.
func (b *SSEBackend) Address() string {
	return b.addr
}
