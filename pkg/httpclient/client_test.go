package httpclient_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/httpclient"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestHeadersApplyToRetriesWithoutMutatingRequest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-API-Key") != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if strings.Join(r.Header.Values("X-Feature"), ",") != "one,two" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	headers := http.Header{
		"authorization": {"Bearer test-token"},
		"X-Feature":     {"one", "two"},
	}
	option := httpclient.WithHeaders(headers)
	headers["authorization"][0] = "Bearer changed"
	headers["X-Feature"][0] = "changed"
	client := httpclient.New("headers-test",
		httpclient.WithHeaders(http.Header{
			"Authorization": {"Bearer replaced"},
			"X-API-Key":     {"test-key"},
		}),
		option,
	)
	t.Cleanup(client.CloseIdleConnections)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer original")
	res, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Equal(t, "Bearer original", req.Header.Get("Authorization"))
	require.Empty(t, req.Header.Get("X-API-Key"))
}

func TestConfiguredHeadersStayOnOriginalOrigin(t *testing.T) {
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" || r.Header.Get("X-API-Key") != "" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.URL.Path == "/first" {
			http.Redirect(w, r, "/second", http.StatusFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(target.Close)
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" || r.Header.Get("X-API-Key") != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/same-origin", http.StatusFound)
			return
		}
		http.Redirect(w, r, target.URL+"/first", http.StatusFound)
	}))
	t.Cleanup(source.Close)
	client := httpclient.New("redirect-test",
		httpclient.WithHeaders(http.Header{
			"Authorization": {"Bearer test-token"},
			"X-API-Key":     {"test-key"},
		}),
	)
	t.Cleanup(client.CloseIdleConnections)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, source.URL+"/start", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer caller-token")
	req.Header.Set("X-API-Key", "caller-key")
	res, err := client.Do(req)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, http.StatusNoContent, res.StatusCode)
}

func TestTracingPropagatesWithoutMutatingRequest(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	ctx, parent := provider.Tracer("test").Start(context.Background(), "parent")
	t.Cleanup(func() { parent.End() })
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		remote := propagation.TraceContext{}.Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		if trace.SpanContextFromContext(remote).TraceID() != parent.SpanContext().TraceID() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	client := httpclient.New("tracing-test", httpclient.WithTracing(otelhttp.WithTracerProvider(provider)))
	t.Cleanup(client.CloseIdleConnections)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)
	res, err := client.Do(req)
	require.NoError(t, err)
	require.Len(t, recorder.Ended(), 1)
	_, err = io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())
	require.Equal(t, http.StatusOK, res.StatusCode)
	require.Empty(t, req.Header.Get("traceparent"))
	spans := recorder.Ended()
	require.Len(t, spans, 2)
	for _, span := range spans {
		require.Equal(t, trace.SpanKindClient, span.SpanKind())
		require.Equal(t, parent.SpanContext().SpanID(), span.Parent().SpanID())
	}
}

func TestRetryAfterWaitRespectsOverallTimeout(t *testing.T) {
	for _, value := range []string{"120", time.Now().Add(2 * time.Minute).UTC().Format(http.TimeFormat)} {
		t.Run(value, func(t *testing.T) {
			recorder := tracetest.NewSpanRecorder()
			provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
			t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if requests.Add(1) == 1 {
					w.Header().Set("Retry-After", value)
					w.WriteHeader(http.StatusTooManyRequests)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)
			client := httpclient.New("retry-after-test",
				httpclient.WithTimeout(100*time.Millisecond),
				httpclient.WithBackoff(func(int) time.Duration { return 0 }),
				httpclient.WithTracing(otelhttp.WithTracerProvider(provider)),
			)
			t.Cleanup(client.CloseIdleConnections)
			res, err := client.Get(server.URL)
			if res != nil {
				require.NoError(t, res.Body.Close())
			}
			require.ErrorIs(t, err, context.DeadlineExceeded)
			spans := recorder.Ended()
			require.Len(t, spans, 1)
			require.Equal(t, trace.SpanKindClient, spans[0].SpanKind())
		})
	}
}

func TestTimeoutIncludesResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	client := httpclient.New("timeout-test", httpclient.WithTimeout(100*time.Millisecond))
	t.Cleanup(client.CloseIdleConnections)
	res, err := client.Get(server.URL)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, res.Body.Close()) })
	_, err = res.Body.Read(make([]byte, 1))
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestCircuitRejectsCallsAfterRepeatedServerFailures(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	t.Cleanup(server.Close)
	client := httpclient.New("breaker-test", httpclient.WithAttempts(1), httpclient.WithTracing(otelhttp.WithTracerProvider(provider)))
	t.Cleanup(client.CloseIdleConnections)
	for range 5 {
		res, err := client.Get(server.URL)
		require.NoError(t, err)
		require.Equal(t, http.StatusServiceUnavailable, res.StatusCode)
		require.NoError(t, res.Body.Close())
	}
	res, err := client.Get(server.URL)
	require.ErrorIs(t, err, circuitbreaker.ErrTripped)
	require.Nil(t, res)
	spans := recorder.Ended()
	require.Len(t, spans, 5)
	for _, span := range spans {
		require.Equal(t, codes.Error, span.Status().Code)
	}
}

func TestRetryBudgetPreservesFinalResponse(t *testing.T) {
	for _, tt := range []struct {
		status   int
		attempts int
	}{
		{status: http.StatusTooManyRequests, attempts: 3},
		{status: http.StatusBadGateway, attempts: 3},
		{status: http.StatusServiceUnavailable, attempts: 3},
		{status: http.StatusGatewayTimeout, attempts: 3},
		{status: http.StatusBadRequest, attempts: 1},
		{status: http.StatusInternalServerError, attempts: 1},
	} {
		t.Run(http.StatusText(tt.status), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempt := requests.Add(1)
				w.WriteHeader(tt.status)
				_, err := io.WriteString(w, strconv.Itoa(int(attempt)))
				if err != nil {
					t.Error(err)
				}
			}))
			t.Cleanup(server.Close)
			client := httpclient.New("budget-test", httpclient.WithBackoff(func(int) time.Duration { return 0 }))
			t.Cleanup(client.CloseIdleConnections)
			res, err := client.Get(server.URL)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, res.Body.Close()) })
			body, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			require.Equal(t, tt.status, res.StatusCode)
			require.Equal(t, strconv.Itoa(tt.attempts), string(body))
		})
	}
}

func TestRetriesOnlyReplayableIdempotentRequests(t *testing.T) {
	for _, tt := range []struct {
		name   string
		method string
		stream bool
		status int
	}{
		{name: "get", method: http.MethodGet, stream: false, status: http.StatusOK},
		{name: "put body", method: http.MethodPut, stream: false, status: http.StatusOK},
		{name: "post", method: http.MethodPost, stream: false, status: http.StatusServiceUnavailable},
		{name: "stream", method: http.MethodPut, stream: true, status: http.StatusServiceUnavailable},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, err := io.ReadAll(r.Body)
				if err != nil || string(body) != "payload" {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				if requests.Add(1) == 1 {
					w.WriteHeader(http.StatusServiceUnavailable)
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			t.Cleanup(server.Close)
			var body io.Reader = strings.NewReader("payload")
			if tt.stream {
				body = io.NopCloser(body)
			}
			req, err := http.NewRequestWithContext(context.Background(), tt.method, server.URL, body)
			require.NoError(t, err)
			client := httpclient.New("retry-test")
			t.Cleanup(client.CloseIdleConnections)
			res, err := client.Do(req)
			require.NoError(t, err)
			require.NoError(t, res.Body.Close())
			require.Equal(t, tt.status, res.StatusCode)
		})
	}
}
