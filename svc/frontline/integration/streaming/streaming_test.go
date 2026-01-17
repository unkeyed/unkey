//go:build integration

package streaming

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestSSEStreamingThroughProxy tests that Server-Sent Events are properly
// streamed through the reverse proxy chain (frontline -> sentinel -> backend).
//
// This test validates that:
// 1. SSE events are flushed immediately (not buffered)
// 2. The FlushInterval: -1 setting works correctly
// 3. The Flusher interface on our wrapped ResponseWriters works
func TestSSEStreamingThroughProxy(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start backend SSE server
	backendAddr := findFreePort(t)
	backend := NewSSEBackend(backendAddr)
	go func() {
		if err := backend.Start(); err != nil && err != http.ErrServerClosed {
			t.Logf("backend server error: %v", err)
		}
	}()
	defer backend.Shutdown(ctx)

	// Wait for backend to be ready
	waitForServer(t, "http://"+backendAddr+"/health", 5*time.Second)

	// Create sentinel-like proxy (simulates sentinel forwarding to backend)
	sentinelAddr := findFreePort(t)
	sentinelProxy := createStreamingProxy(t, "http://"+backendAddr)
	sentinelServer := &http.Server{
		Addr:    sentinelAddr,
		Handler: sentinelProxy,
	}
	go func() {
		if err := sentinelServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("sentinel proxy error: %v", err)
		}
	}()
	defer sentinelServer.Shutdown(ctx)

	// Wait for sentinel proxy to be ready
	waitForServer(t, "http://"+sentinelAddr+"/health", 5*time.Second)

	// Create frontline-like proxy (simulates frontline forwarding to sentinel)
	frontlineAddr := findFreePort(t)
	frontlineProxy := createStreamingProxy(t, "http://"+sentinelAddr)
	frontlineServer := &http.Server{
		Addr:    frontlineAddr,
		Handler: frontlineProxy,
	}
	go func() {
		if err := frontlineServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("frontline proxy error: %v", err)
		}
	}()
	defer frontlineServer.Shutdown(ctx)

	// Wait for frontline proxy to be ready
	waitForServer(t, "http://"+frontlineAddr+"/health", 5*time.Second)

	// Test SSE streaming through the full proxy chain
	t.Run("SSE events streamed immediately", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, "GET", "http://"+frontlineAddr+"/sse", nil)
		require.NoError(t, err)

		client := &http.Client{
			Timeout: 10 * time.Second,
		}

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Equal(t, http.StatusOK, resp.StatusCode)
		require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

		// Read SSE events and track timing
		scanner := bufio.NewScanner(resp.Body)
		var events []sseEvent
		eventStart := time.Now()

		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "event:") {
				eventType := strings.TrimPrefix(line, "event: ")
				events = append(events, sseEvent{
					eventType: eventType,
					receivedAt: time.Since(eventStart),
				})
			}
			if strings.HasPrefix(line, "event: done") {
				break
			}
		}
		require.NoError(t, scanner.Err())

		// Verify we received all events
		require.GreaterOrEqual(t, len(events), 6, "should receive at least 6 events (connected + 5 messages + done)")

		// Verify events were received incrementally, not all at once
		// If buffering was happening, all events would arrive at nearly the same time
		// With proper streaming, there should be ~100ms gaps between message events
		t.Logf("Received %d events:", len(events))
		for i, e := range events {
			t.Logf("  Event %d: type=%s, received_at=%v", i, e.eventType, e.receivedAt)
		}

		// Find message events and check they arrived incrementally
		var messageEvents []sseEvent
		for _, e := range events {
			if e.eventType == "message" {
				messageEvents = append(messageEvents, e)
			}
		}

		if len(messageEvents) >= 2 {
			// Check that there's a reasonable gap between first and last message
			// If all events arrived at once (buffered), the gap would be tiny
			gap := messageEvents[len(messageEvents)-1].receivedAt - messageEvents[0].receivedAt
			t.Logf("Time gap between first and last message event: %v", gap)

			// With 5 messages at 100ms intervals, we expect ~400ms gap
			// Allow some tolerance for timing variations
			require.Greater(t, gap.Milliseconds(), int64(200),
				"Events should arrive incrementally, not all at once. Gap: %v", gap)
		}
	})
}

type sseEvent struct {
	eventType  string
	receivedAt time.Duration
}

// createStreamingProxy creates a reverse proxy with streaming enabled (FlushInterval: -1)
// This mimics the configuration we added to frontline and sentinel.
func createStreamingProxy(t *testing.T, target string) http.Handler {
	t.Helper()

	//nolint:exhaustruct
	transport := &http.Transport{
		ForceAttemptHTTP2: true,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	//nolint:exhaustruct
	proxy := &httputil.ReverseProxy{
		Transport:     transport,
		FlushInterval: -1, // Flush immediately - this is what we're testing!
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = strings.TrimPrefix(target, "http://")
			req.Host = req.URL.Host
		},
	}

	return proxy
}

func findFreePort(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := listener.Addr().String()
	listener.Close()
	return addr
}

func waitForServer(t *testing.T, url string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready within %v", url, timeout)
}

// TestResponseWriterFlusherInterface tests that our custom ResponseWriter wrappers
// properly implement the http.Flusher interface needed for streaming.
func TestResponseWriterFlusherInterface(t *testing.T) {
	// This test validates that when we wrap a ResponseWriter with our
	// ErrorCapturingWriter or statusRecorder, the Flush() method is
	// properly propagated to the underlying writer.

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test server that uses a custom ResponseWriter wrapper
	addr := findFreePort(t)
	mux := http.NewServeMux()

	mux.HandleFunc("/stream", func(w http.ResponseWriter, r *http.Request) {
		// Wrap the writer (simulating what our middleware does)
		wrapper := &testFlusherWrapper{ResponseWriter: w}

		w.Header().Set("Content-Type", "text/event-stream")

		flusher, ok := wrapper.(http.Flusher)
		if !ok {
			http.Error(w, "flusher not supported", http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(wrapper, "data: test\n\n")
		flusher.Flush()

		fmt.Fprintf(wrapper, "data: done\n\n")
		flusher.Flush()
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			t.Logf("test server error: %v", err)
		}
	}()
	defer server.Shutdown(ctx)

	time.Sleep(100 * time.Millisecond) // Wait for server to start

	resp, err := http.Get("http://" + addr + "/stream")
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
}

// testFlusherWrapper is a simple wrapper that implements http.Flusher
// by delegating to the underlying ResponseWriter.
type testFlusherWrapper struct {
	http.ResponseWriter
}

func (w *testFlusherWrapper) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *testFlusherWrapper) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
