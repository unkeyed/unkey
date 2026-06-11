package handler_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	handler "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
)

// TestRetry_DialFailureAdvancesToNextInstance proves that a dial-phase
// failure on the first candidate instance causes the handler to advance
// to the next instance in LocalInstances and serve the response from
// there. The happy path of the retry loop — if this regresses, single
// dead pods will fail user requests outright.
func TestRetry_DialFailureAdvancesToNextInstance(t *testing.T) {
	t.Parallel()

	deadAddr := unreachableAddr(t)
	aliveAddr, stopAlive := startBackend(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, "served-by-second")
	})
	t.Cleanup(stopAlive)

	decision := localDecision(deadAddr, aliveAddr)
	frontlineAddr, stop := startFrontlineWith(t, decision, nil)
	t.Cleanup(stop)

	resp := mustGet(t, frontlineAddr)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "served-by-second", string(body))
}

// TestRetry_MidStreamFailureDoesNotRetry is the safety property that
// drives the dial-vs-stream distinction. When an upstream accepts the
// TCP connection (so the proxy has started writing the request — the
// app may have already executed side-effects) and then dies without
// producing a response, the handler must NOT replay against the next
// instance, or non-idempotent endpoints execute twice.
//
// We assert two things: the client gets a 5xx, and the second backend's
// request counter stays at zero.
func TestRetry_MidStreamFailureDoesNotRetry(t *testing.T) {
	t.Parallel()

	midCloseAddr, stopMidClose := startMidStreamCloseBackend(t)
	t.Cleanup(stopMidClose)

	var secondHits int64
	secondAddr, stopSecond := startBackend(t, func(_ http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&secondHits, 1)
	})
	t.Cleanup(stopSecond)

	decision := localDecision(midCloseAddr, secondAddr)
	frontlineAddr, stop := startFrontlineWith(t, decision, nil)
	t.Cleanup(stop)

	resp := mustGet(t, frontlineAddr)
	defer func() { _ = resp.Body.Close() }()

	require.GreaterOrEqual(t, resp.StatusCode, 500, "expected 5xx after mid-stream upstream failure")
	require.Equal(t, int64(0), atomic.LoadInt64(&secondHits), "second instance must not be retried after a mid-stream failure — would risk double-execute")
}

// TestRetry_AllLocalDeadFallsThroughToRegion proves the region fallback
// path: once every local instance dial-fails AND the decision carries a
// peer-region standby, the handler invokes ForwardToRegion with that
// address. The real proxy service is stubbed because the region path
// would otherwise dial out over TLS to a public-DNS apex.
//
// We assert the handler walked every local entry (so we know we didn't
// fall through early) and called ForwardToRegion with the standby.
func TestRetry_AllLocalDeadFallsThroughToRegion(t *testing.T) {
	t.Parallel()

	stub := &recordingProxy{
		instanceErr: &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED},
		regionBody:  "served-by-peer-region",
	}
	decision := router.RouteDecision{
		Destination:      router.DestinationLocalInstance,
		DeploymentID:     "dep_test",
		EnvironmentID:    "env_test",
		WorkspaceID:      "ws_test",
		ProjectID:        "proj_test",
		UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
		LocalInstances: []db.FindInstancesByDeploymentIDRow{
			{ID: "a", Address: "127.0.0.1:1"},
			{ID: "b", Address: "127.0.0.1:2"},
		},
		RemoteRegionAddress: "us-west-2.aws",
	}
	frontlineAddr, stop := startFrontlineWith(t, decision, stub)
	t.Cleanup(stop)

	resp := mustGet(t, frontlineAddr)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, "served-by-peer-region", string(body))

	require.Equal(t, []string{"a", "b"}, stub.instanceCalls(), "handler must try every local instance before falling through")
	require.Equal(t, "us-west-2.aws", stub.regionTarget(), "ForwardToRegion called with the standby address")
}

// TestRetry_AllLocalDeadNoStandbyReturnsError proves that exhausting
// every local instance WITHOUT a configured standby region surfaces the
// failure to the client instead of silently succeeding. Silent success
// here would mask regional outages from dashboards and SLOs.
func TestRetry_AllLocalDeadNoStandbyReturnsError(t *testing.T) {
	t.Parallel()

	deadA := unreachableAddr(t)
	deadB := unreachableAddr(t)

	decision := localDecision(deadA, deadB)
	decision.RemoteRegionAddress = ""

	frontlineAddr, stop := startFrontlineWith(t, decision, nil)
	t.Cleanup(stop)

	resp := mustGet(t, frontlineAddr)
	defer func() { _ = resp.Body.Close() }()

	require.GreaterOrEqual(t, resp.StatusCode, 500, "expected 5xx when local instances are exhausted and no standby is configured")
}

// TestRetry_PostBodySurvivesDialRetry validates the design's key claim:
// because we only retry on dial-phase failures, the request body is
// guaranteed to be untouched when we move to the next instance — no
// buffering needed. If retry ever expanded to post-dial errors, the
// second backend here would receive an empty body and this test would
// fail loudly.
func TestRetry_PostBodySurvivesDialRetry(t *testing.T) {
	t.Parallel()

	deadAddr := unreachableAddr(t)

	var receivedBody atomic.Value
	receivedBody.Store("")
	aliveAddr, stopAlive := startBackend(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		receivedBody.Store(string(body))
		_, _ = io.WriteString(w, "ack")
	})
	t.Cleanup(stopAlive)

	decision := localDecision(deadAddr, aliveAddr)
	frontlineAddr, stop := startFrontlineWith(t, decision, nil)
	t.Cleanup(stop)

	payload := "hello world, this is the request body"
	req, err := http.NewRequest(http.MethodPost, "http://"+frontlineAddr+"/echo", strings.NewReader(payload))
	require.NoError(t, err)
	req.Host = "retry-test.example.com"
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, payload, receivedBody.Load().(string), "second instance must receive the original body intact")
}

// --- helpers ---

// unreachableAddr returns an address that is guaranteed to refuse TCP
// connections (ECONNREFUSED). The classic listen-then-close trick — the
// kernel keeps the port reserved briefly, but any connect attempt is
// rejected immediately, which is exactly the dial-failure case we want.
func unreachableAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := ln.Addr().String()
	require.NoError(t, ln.Close())
	return addr
}

// startBackend serves the given handler on a localhost listener and
// returns its address plus a cleanup function.
func startBackend(t *testing.T, h http.HandlerFunc) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	//nolint:exhaustruct
	srv := &http.Server{Handler: h}
	go func() { _ = srv.Serve(ln) }()
	return ln.Addr().String(), func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}
}

// startMidStreamCloseBackend accepts each connection, reads enough bytes
// to confirm the request reached us, then closes the connection without
// writing any response. The frontline proxy will see EOF on its read of
// the response — a post-dial error that IsDialError must classify as
// NOT retryable, otherwise the second instance would be hit and the
// safety test would fail.
func startMidStreamCloseBackend(t *testing.T) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer func() { _ = c.Close() }()
				// Drain a single read so the proxy gets past header
				// write before we drop the conn. Errors are expected
				// and ignored — we're closing on purpose.
				_, _ = c.Read(make([]byte, 1024))
			}(conn)
		}
	}()

	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}

// localDecision builds a RouteDecision pointing at the given addresses,
// in order, as local instances. The deployment metadata is fixed —
// individual tests override the RemoteRegionAddress field as needed.
func localDecision(addrs ...string) router.RouteDecision {
	instances := make([]db.FindInstancesByDeploymentIDRow, len(addrs))
	for i, addr := range addrs {
		instances[i] = db.FindInstancesByDeploymentIDRow{
			ID:      fmt.Sprintf("inst_%d", i),
			Address: addr,
		}
	}
	//nolint:exhaustruct
	return router.RouteDecision{
		Destination:      router.DestinationLocalInstance,
		DeploymentID:     "dep_test",
		EnvironmentID:    "env_test",
		WorkspaceID:      "ws_test",
		ProjectID:        "proj_test",
		UpstreamProtocol: db.DeploymentsUpstreamProtocolHttp1,
		LocalInstances:   instances,
	}
}

// startFrontlineWith spins up a frontline that always returns the given
// decision from its router. When proxySvc is nil, a real proxy.Service
// is used — tests exercising actual TCP/transport behavior want the real
// one. When proxySvc is non-nil, the stub stands in (used by the
// region-fallback test, which would otherwise need TLS infrastructure).
func startFrontlineWith(t *testing.T, decision router.RouteDecision, proxySvc proxy.Service) (string, func()) {
	t.Helper()

	if proxySvc == nil {
		//nolint:exhaustruct
		ps, err := proxy.New(proxy.Config{
			InstanceID:         "test-instance",
			Platform:           "test",
			Region:             "test",
			ApexDomain:         "test.local",
			Clock:              clock.New(),
			MaxHops:            3,
			UpstreamTransports: proxy.NewTransportRegistry(),
		})
		require.NoError(t, err)
		proxySvc = ps
	}

	h := &handler.Handler{
		RouterService: &stubRouter{decision: decision},
		ProxyService:  proxySvc,
		Engine:        nil,
		Clock:         clock.New(),
	}

	//nolint:exhaustruct
	zenSrv, err := zen.New(zen.Config{
		ReadTimeout:                 -1,
		WriteTimeout:                -1,
		MaxRequestBodySize:          0,
		DisableRequestBodyBuffering: true, // mirrors production (run.go)
	})
	require.NoError(t, err)
	// Mirror the production middleware chain. Observability is load-
	// bearing for the error-path tests: without it, a fault-wrapped error
	// from the handler never gets translated into an HTTP status code and
	// the client just sees a hung connection.
	mws := []zen.Middleware{
		zen.WithPanicRecovery(),
		middleware.WithReservedHeaderStrip(),
		middleware.WithObservability(errorpage.NewRenderer()),
	}
	zenSrv.RegisterRoute(mws, h)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = zenSrv.Serve(ctx, ln) }()

	waitForListener(t, ln.Addr().String())

	return ln.Addr().String(), func() {
		cancel()
		shutdownCtx, sc := context.WithTimeout(context.Background(), 2*time.Second)
		defer sc()
		_ = zenSrv.Shutdown(shutdownCtx)
	}
}

func mustGet(t *testing.T, addr string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, "http://"+addr+"/foo", nil)
	require.NoError(t, err)
	req.Host = "retry-test.example.com"
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// recordingProxy is a proxy.Service stub that records which instances
// were attempted and which region target ForwardToRegion saw. Used by
// the region-fallback test to avoid standing up TLS for the peer hop.
//
// The frontline handler runs on zen's request goroutine while the test
// reads back state from the test goroutine after http.DefaultClient.Do
// returns. The TCP I/O between them is NOT a Go memory-model
// synchronization point, so the mutex is required to satisfy -race even
// though contention in practice is zero (one request in flight at a time).
type recordingProxy struct {
	instanceErr error
	regionBody  string

	mu       sync.Mutex
	calls    []string
	regionTo string
}

// instanceCalls returns the IDs of instances ForwardToInstance was
// called with, in order.
func (r *recordingProxy) instanceCalls() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]string, len(r.calls))
	copy(out, r.calls)
	return out
}

// regionTarget returns the address ForwardToRegion was called with, or
// "" if it was never called.
func (r *recordingProxy) regionTarget() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.regionTo
}

func (r *recordingProxy) ForwardToInstance(_ context.Context, _ *zen.Session, _ db.DeploymentsUpstreamProtocol, inst db.FindInstancesByDeploymentIDRow) error {
	r.mu.Lock()
	r.calls = append(r.calls, inst.ID)
	r.mu.Unlock()
	return r.instanceErr
}

func (r *recordingProxy) ForwardToRegion(_ context.Context, sess *zen.Session, target string) error {
	r.mu.Lock()
	r.regionTo = target
	r.mu.Unlock()
	_, _ = io.WriteString(sess.ResponseWriter(), r.regionBody)
	return nil
}

var _ proxy.Service = (*recordingProxy)(nil)
