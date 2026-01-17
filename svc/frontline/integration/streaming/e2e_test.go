//go:build integration

// Package streaming provides integration tests for HTTP/2 and streaming support
// through the frontline -> sentinel -> backend proxy chain.
//
// These tests require the docker-compose stack to be running:
//
//	cd dev && docker compose up -d mysql streaming-backend sentinel frontline
//
// Then run the tests:
//
//	bazel test --test_tag_filters=integration //svc/frontline/integration/streaming:streaming_test
package streaming

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// getFrontlineURL returns the frontline URL, checking for environment override
func getFrontlineURL() string {
	if url := os.Getenv("FRONTLINE_URL"); url != "" {
		return url
	}
	return "http://localhost:7445"
}

// getStreamingBackendURL returns the streaming backend URL for direct testing
func getStreamingBackendURL() string {
	if url := os.Getenv("STREAMING_BACKEND_URL"); url != "" {
		return url
	}
	return "http://localhost:8085"
}

// TestStreamingBackendDirectly tests that the streaming backend works directly
func TestStreamingBackendDirectly(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	backendURL := getStreamingBackendURL()

	// Check health first
	healthResp, err := http.Get(backendURL + "/health")
	require.NoError(t, err, "streaming backend should be reachable at %s", backendURL)
	defer healthResp.Body.Close()
	require.Equal(t, http.StatusOK, healthResp.StatusCode, "streaming backend should be healthy")

	// Test SSE streaming
	req, err := http.NewRequestWithContext(ctx, "GET", backendURL+"/sse", nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	events := readSSEEvents(t, resp, 3) // Read at least 3 events
	require.GreaterOrEqual(t, len(events), 3, "should receive at least 3 events")
	t.Logf("Received %d events directly from backend", len(events))
}

// TestSSEThroughProxyChain tests SSE streaming through frontline -> sentinel -> backend
func TestSSEThroughProxyChain(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	frontlineURL := getFrontlineURL()

	// Check frontline health first
	healthResp, err := http.Get(frontlineURL + "/internal/health")
	require.NoError(t, err, "frontline should be reachable at %s", frontlineURL)
	defer healthResp.Body.Close()
	require.Equal(t, http.StatusOK, healthResp.StatusCode, "frontline should be healthy")

	// Test SSE through the full proxy chain
	req, err := http.NewRequestWithContext(ctx, "GET", frontlineURL+"/sse", nil)
	require.NoError(t, err)
	req.Host = "streaming.local.dev" // Route to our test deployment

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode, "SSE request through proxy should succeed")
	require.Equal(t, "text/event-stream", resp.Header.Get("Content-Type"))

	// Verify we got the frontline headers
	require.NotEmpty(t, resp.Header.Get("X-Unkey-Request-Id"), "should have request ID header")

	// Read and verify SSE events with timing
	events := readSSEEventsWithTiming(t, resp, 5)
	require.GreaterOrEqual(t, len(events), 5, "should receive at least 5 events through proxy")

	// Verify events were streamed incrementally (not buffered)
	verifyIncrementalStreaming(t, events)
}

// TestSSEThroughProxyChainWithLocalhost tests using localhost hostname
func TestSSEThroughProxyChainWithLocalhost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	frontlineURL := getFrontlineURL()

	req, err := http.NewRequestWithContext(ctx, "GET", frontlineURL+"/sse", nil)
	require.NoError(t, err)
	req.Host = "localhost" // Also configured in seed data

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	events := readSSEEvents(t, resp, 3)
	require.GreaterOrEqual(t, len(events), 3)
	t.Logf("Received %d events through proxy (localhost host)", len(events))
}

type sseEvent struct {
	eventType  string
	data       string
	receivedAt time.Time
}

func readSSEEvents(t *testing.T, resp *http.Response, minEvents int) []sseEvent {
	t.Helper()
	return readSSEEventsWithTiming(t, resp, minEvents)
}

func readSSEEventsWithTiming(t *testing.T, resp *http.Response, minEvents int) []sseEvent {
	t.Helper()

	var events []sseEvent
	scanner := bufio.NewScanner(resp.Body)
	var currentEvent sseEvent

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "event:") {
			currentEvent.eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			currentEvent.receivedAt = time.Now()
		} else if strings.HasPrefix(line, "data:") {
			currentEvent.data = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		} else if line == "" && currentEvent.eventType != "" {
			// Empty line marks end of event
			events = append(events, currentEvent)
			t.Logf("Event %d: type=%s, data=%s", len(events), currentEvent.eventType, currentEvent.data)
			currentEvent = sseEvent{}

			if len(events) >= minEvents {
				break
			}
		}

		// Stop on done event
		if currentEvent.eventType == "done" {
			events = append(events, currentEvent)
			break
		}
	}

	require.NoError(t, scanner.Err())
	return events
}

func verifyIncrementalStreaming(t *testing.T, events []sseEvent) {
	t.Helper()

	// Find message events
	var messageEvents []sseEvent
	for _, e := range events {
		if e.eventType == "message" {
			messageEvents = append(messageEvents, e)
		}
	}

	if len(messageEvents) < 2 {
		t.Log("Not enough message events to verify incremental streaming")
		return
	}

	// Check time gap between first and last message
	firstMsg := messageEvents[0]
	lastMsg := messageEvents[len(messageEvents)-1]
	gap := lastMsg.receivedAt.Sub(firstMsg.receivedAt)

	t.Logf("Time gap between first and last message: %v", gap)

	// With 500ms intervals in the backend, expect at least 200ms gap for 2+ messages
	// If everything was buffered, gap would be near zero
	if len(messageEvents) >= 2 {
		expectedMinGap := time.Duration(len(messageEvents)-1) * 200 * time.Millisecond
		require.Greater(t, gap, expectedMinGap,
			"Events should arrive incrementally. Got %v gap for %d messages, expected at least %v",
			gap, len(messageEvents), expectedMinGap)
	}
}

// TestEchoEndpoint tests that regular requests also work through the proxy
func TestEchoEndpoint(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	frontlineURL := getFrontlineURL()

	req, err := http.NewRequestWithContext(ctx, "GET", frontlineURL+"/echo", nil)
	require.NoError(t, err)
	req.Host = "streaming.local.dev"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Contains(t, resp.Header.Get("Content-Type"), "application/json")

	// Read body
	buf := make([]byte, 1024)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])

	t.Logf("Echo response: %s", body)
	require.Contains(t, body, "streaming-backend", "response should come from streaming backend")
}

// BenchmarkSSELatency measures the latency for receiving the first SSE event
func BenchmarkSSELatency(b *testing.B) {
	frontlineURL := getFrontlineURL()

	// Verify service is up
	healthResp, err := http.Get(frontlineURL + "/internal/health")
	if err != nil {
		b.Skip("frontline not available")
	}
	healthResp.Body.Close()

	client := &http.Client{Timeout: 30 * time.Second}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		req, _ := http.NewRequestWithContext(ctx, "GET", frontlineURL+"/sse", nil)
		req.Host = "streaming.local.dev"

		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			cancel()
			b.Fatal(err)
		}

		// Read until first event
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			if strings.HasPrefix(scanner.Text(), "event:") {
				break
			}
		}

		b.ReportMetric(float64(time.Since(start).Microseconds()), "us/first-event")

		resp.Body.Close()
		cancel()
	}
}

func init() {
	// Print helpful message if tests fail due to missing services
	fmt.Println("Streaming integration tests require docker-compose services:")
	fmt.Println("  cd dev && docker compose up -d mysql streaming-backend sentinel frontline")
	fmt.Println("")
}
