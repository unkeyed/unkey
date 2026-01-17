// Package main provides a standalone SSE streaming server for integration testing.
// This server can be run as a Docker container to test streaming through the proxy chain.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// SSE endpoint that streams events
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("SSE connection from %s", r.RemoteAddr)

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
		log.Printf("Sent connected event")

		// Send numbered events
		for i := 1; i <= 10; i++ {
			select {
			case <-r.Context().Done():
				log.Printf("Client disconnected after %d events", i-1)
				return
			case <-time.After(500 * time.Millisecond):
				fmt.Fprintf(w, "event: message\ndata: {\"seq\":%d,\"time\":\"%s\"}\n\n", i, time.Now().Format(time.RFC3339Nano))
				flusher.Flush()
				log.Printf("Sent message event %d", i)
			}
		}

		// Send completion event
		fmt.Fprintf(w, "event: done\ndata: {\"status\":\"complete\"}\n\n")
		flusher.Flush()
		log.Printf("Sent done event")
	})

	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Echo endpoint that returns request info (for debugging)
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"host":"%s","method":"%s","path":"%s","remote_addr":"%s"}`,
			r.Host, r.Method, r.URL.Path, r.RemoteAddr)
	})

	// Catch-all for debugging routing
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Request: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"message":"streaming-backend","path":"%s"}`, r.URL.Path)
	})

	//nolint:exhaustruct
	server := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("Shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		server.Shutdown(ctx)
	}()

	log.Printf("Starting SSE streaming backend on port %s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
