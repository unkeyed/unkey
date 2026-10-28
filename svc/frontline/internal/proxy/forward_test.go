package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
)

func TestForwardLogsTransportFaultsButNotClientCancellation(t *testing.T) {
	var output bytes.Buffer
	logger.AddHandler(slog.NewJSONHandler(&output, nil))
	for _, destination := range []string{"instance", "frontline"} {
		for _, tc := range []struct {
			name     string
			err      error
			canceled bool
			stream   bool
			logged   bool
		}{
			{name: "client_canceled", err: context.Canceled, canceled: true},
			{name: "unexpected_cancellation", err: context.Canceled, logged: true},
			{name: "deadline", err: context.DeadlineExceeded, logged: true},
			{name: "upstream_reset", err: syscall.ECONNRESET, logged: true},
			{name: "reset_after_cancel", err: syscall.ECONNRESET, canceled: true, logged: true},
			{name: "truncated_response", err: io.ErrUnexpectedEOF, stream: true, logged: true},
			{name: "canceled_stream", err: context.Canceled, canceled: true, stream: true},
		} {
			t.Run(destination+"/"+tc.name, func(t *testing.T) {
				transport := loggingTransport{err: tc.err, stream: tc.stream}
				peer := &http.Transport{}
				peer.RegisterProtocol("https", transport)
				t.Cleanup(peer.CloseIdleConnections)
				metadata, err := meta.New(testMetadataSigningKey)
				require.NoError(t, err)
				clk := clock.New()
				svc, err := New(Config{
					Clock: clk, Metadata: metadata, Transport: peer,
					UpstreamTransports: &TransportRegistry{transports: map[db.DeploymentsUpstreamProtocol]http.RoundTripper{db.DeploymentsUpstreamProtocolHttp1: transport}},
				})
				require.NoError(t, err)
				ctx, cancel := context.WithCancel(t.Context())
				t.Cleanup(cancel)
				if tc.canceled {
					cancel()
				}
				ctx = context.WithValue(ctx, http.ServerContextKey, &http.Server{})
				ctx = WithRequestStartTime(ctx, clk.Now())
				req := httptest.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/orders", nil)
				sess := &zen.Session{}
				require.NoError(t, sess.Init(httptest.NewRecorder(), req, 0))
				snapshot := output.Len()
				if destination == "instance" {
					err = svc.ForwardToInstance(ctx, sess, db.DeploymentsUpstreamProtocolHttp1, db.FindInstancesByDeploymentIDRow{ID: uid.New(uid.InstancePrefix), Address: "upstream.test"})
				} else {
					err = svc.ForwardToRegion(ctx, sess, "us-west-2.aws", nil)
				}
				if tc.stream {
					require.NoError(t, err)
				} else {
					require.ErrorIs(t, err, tc.err)
				}
				data := output.Bytes()[snapshot:]
				if !tc.logged {
					require.Empty(t, string(data))
					return
				}
				var record struct {
					Event     string `json:"event"`
					RequestID string `json:"requestId"`
					Target    string `json:"target"`
					Message   string `json:"msg"`
					Error     string `json:"error"`
				}
				require.NoError(t, json.Unmarshal(data, &record))
				if tc.stream {
					require.Equal(t, "frontline.proxy_stream_error", record.Event)
				} else {
					require.Equal(t, "frontline.proxy_attempt_error", record.Event)
				}
				require.Equal(t, sess.RequestID(), record.RequestID)
				require.NotEmpty(t, record.Target)
				require.Contains(t, record.Error+record.Message, tc.err.Error())
			})
		}
	}
}

type loggingTransport struct {
	err    error
	stream bool
}

func (t loggingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	if t.stream {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(t)}, nil
	}
	return nil, t.err
}

func (t loggingTransport) Read([]byte) (int, error) { return 0, t.err }

func TestServeWithAbortRecovery_SwallowsErrAbortHandler(t *testing.T) {
	t.Parallel()

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	// Must not bubble up. If it does, the test process crashes.
	serveWithAbortRecovery(h, w, r)
}

func TestServeWithAbortRecovery_RepanicsOtherValues(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("boom")
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(sentinel)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("expected panic to propagate, got none")
		}
		if rec != sentinel {
			t.Fatalf("expected original panic value, got %v", rec)
		}
	}()

	serveWithAbortRecovery(h, w, r)
}

func TestServeWithAbortRecovery_NoPanicPassthrough(t *testing.T) {
	t.Parallel()

	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusTeapot)
	})

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	serveWithAbortRecovery(h, w, r)

	if !called {
		t.Fatal("handler was not invoked")
	}
	if w.Code != http.StatusTeapot {
		t.Fatalf("expected status %d, got %d", http.StatusTeapot, w.Code)
	}
}
