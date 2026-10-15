package handler_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/paseto"
	"github.com/unkeyed/unkey/svc/frontline/internal/db"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/peer"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
)

const peerRegion = "far.test"

type peerLab struct {
	frontlineAddr string
	dials         atomic.Int64
}

// startPeerLab runs a frontline whose router sends every request to
// peerRegion, a backhaul pool to one or more relays, and the relays.
func startPeerLab(t *testing.T, decision router.RouteDecision, cfg peer.PoolConfig, relays int) *peerLab {
	t.Helper()
	clk := clock.New()
	metadata, err := meta.New(testMetadataSigningKey)
	require.NoError(t, err)
	transports := proxy.NewTransportRegistry()
	targets, err := peer.ParseCIDRTargets([]string{"127.0.0.0/8"})
	require.NoError(t, err)

	relayAddrs := make([]string, relays)
	for i := range relayAddrs {
		relay := &peer.Relay{
			Metadata: metadata,
			Targets:  targets,
			Transport: func(protocol string) http.RoundTripper {
				return transports.Get(db.DeploymentsUpstreamProtocol(protocol))
			},
			Clock: clk,
		}
		srv := relay.Server()
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		go func() { _ = srv.Serve(ln) }()
		t.Cleanup(func() { _ = srv.Close() })
		relayAddrs[i] = ln.Addr().String()
	}

	lab := &peerLab{frontlineAddr: "", dials: atomic.Int64{}}
	cfg.Regions = []string{peerRegion}
	cfg.Discovery = peer.StaticDiscovery{peerRegion: relayAddrs}
	cfg.Dial = func(ctx context.Context, addr string) (net.Conn, error) {
		lab.dials.Add(1)
		var d net.Dialer
		return d.DialContext(ctx, "tcp", addr)
	}
	pool := peer.NewPool(cfg)
	pool.Start()
	t.Cleanup(pool.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool.WaitReady(ctx)
	require.True(t, pool.Ready(peerRegion))

	//nolint:exhaustruct
	proxySvc, err := proxy.New(proxy.Config{
		InstanceID:         "test-instance",
		Platform:           "test",
		Region:             "test",
		ApexDomain:         "test.local",
		Clock:              clk,
		MaxHops:            3,
		Metadata:           metadata,
		UpstreamTransports: transports,
		Peers:              pool,
	})
	require.NoError(t, err)

	addr, stop := startFrontlineWith(t, decision, proxySvc)
	t.Cleanup(stop)
	lab.frontlineAddr = addr
	return lab
}

func remoteDecision(addrs ...string) router.RouteDecision {
	instances := make([]db.FindInstancesByDeploymentIDRow, len(addrs))
	for i, addr := range addrs {
		instances[i] = db.FindInstancesByDeploymentIDRow{
			ID:      fmt.Sprintf("inst_%d", i),
			Address: addr,
		}
	}
	//nolint:exhaustruct
	return router.RouteDecision{
		Destination:         router.DestinationRemoteRegion,
		DeploymentID:        "dep_test",
		EnvironmentID:       "env_test",
		WorkspaceID:         "ws_test",
		ProjectID:           "proj_test",
		UpstreamProtocol:    db.DeploymentsUpstreamProtocolHttp1,
		RemoteRegionAddress: peerRegion,
		RemoteInstances:     instances,
	}
}

func openWebSocket(t *testing.T, frontlineAddr string) (net.Conn, *bufio.Reader) {
	t.Helper()
	conn, err := net.Dial("tcp", frontlineAddr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, conn.SetDeadline(time.Now().Add(10*time.Second)))
	key := "dGhlIHNhbXBsZSBub25jZQ=="
	require.NoError(t, writeWSHandshake(conn, "ws-test.example.com", "/ws", key))
	br := bufio.NewReader(conn)
	require.NoError(t, expect101(br, key))
	return conn, br
}

// TestPeer_WebSocketsOpenOnWarmConnections is the latency property: a
// cross-region upgrade rides an existing backhaul connection as a stream,
// so no TCP or TLS handshake happens on the request path, even for a burst.
func TestPeer_WebSocketsOpenOnWarmConnections(t *testing.T) {
	t.Parallel()

	backendAddr, stopBackend := startEchoBackend(t)
	t.Cleanup(stopBackend)
	//nolint:exhaustruct
	lab := startPeerLab(t, remoteDecision(backendAddr), peer.PoolConfig{MinConns: 2}, 2)
	warmDials := lab.dials.Load()

	const sockets = 50
	var wg sync.WaitGroup
	errs := make(chan error, sockets)
	for i := range sockets {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, br := openWebSocket(t, lab.frontlineAddr)
			msg := fmt.Sprintf("hello-%d", i)
			if err := writeClientFrame(conn, []byte(msg)); err != nil {
				errs <- err
				return
			}
			payload, err := readServerFrame(br)
			if err != nil {
				errs <- err
				return
			}
			if string(payload) != msg {
				errs <- fmt.Errorf("echo %q, want %q", payload, msg)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	require.Equal(t, warmDials, lab.dials.Load(), "upgrades must not dial new backhaul connections")
}

// TestPeer_WebSocketSurvivesConnectionRotation proves that max-age rotation
// drains a connection instead of cutting the tunnels it carries.
func TestPeer_WebSocketSurvivesConnectionRotation(t *testing.T) {
	t.Parallel()

	backendAddr, stopBackend := startEchoBackend(t)
	t.Cleanup(stopBackend)
	//nolint:exhaustruct
	lab := startPeerLab(t, remoteDecision(backendAddr), peer.PoolConfig{
		MinConns: 1,
		MaxAge:   200 * time.Millisecond,
		Tick:     50 * time.Millisecond,
	}, 1)
	warmDials := lab.dials.Load()

	conn, br := openWebSocket(t, lab.frontlineAddr)
	require.Eventually(t, func() bool { return lab.dials.Load() >= warmDials+3 }, 5*time.Second, 20*time.Millisecond,
		"pool must rotate connections")

	require.NoError(t, writeClientFrame(conn, []byte("still-open")))
	payload, err := readServerFrame(br)
	require.NoError(t, err)
	require.Equal(t, "still-open", string(payload))
}

// TestPeer_UndeliveredAttemptReplaysBody covers the retry rule: the relay
// reports that it could not dial the first instance, so the entry sends the
// request to the next one with the identical body, including the bytes the
// first attempt already sent and bytes past the replay buffer.
func TestPeer_UndeliveredAttemptReplaysBody(t *testing.T) {
	t.Parallel()

	var hits atomic.Int64
	forwardedHost := make(chan string, 1)
	aliveAddr, stopAlive := startBackend(t, func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		forwardedHost <- r.Header.Get("X-Forwarded-Host")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		sum := sha256.Sum256(body)
		_, _ = io.WriteString(w, hex.EncodeToString(sum[:]))
	})
	t.Cleanup(stopAlive)
	//nolint:exhaustruct
	lab := startPeerLab(t, remoteDecision(unreachableAddr(t), aliveAddr), peer.PoolConfig{}, 1)

	payload := make([]byte, 3*peer.DefaultReplayLimit+123)
	_, err := rand.Read(payload)
	require.NoError(t, err)
	want := sha256.Sum256(payload)

	req, err := http.NewRequest(http.MethodPost, "http://"+lab.frontlineAddr+"/upload", io.NopCloser(&slowReader{data: payload}))
	require.NoError(t, err)
	req.Host = "peer-test.example.com"
	req.ContentLength = int64(len(payload))
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	got, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, hex.EncodeToString(want[:]), string(got))
	require.Equal(t, int64(1), hits.Load())
	require.Equal(t, "peer-test.example.com", <-forwardedHost, "forwarding headers set by the entry must reach the instance")
}

// TestPeer_RelayRejectsBeforeDialing checks that the relay dials nothing
// unless the request carries valid, unexpired, single-hop metadata with an
// allowed target.
func TestPeer_RelayRejectsBeforeDialing(t *testing.T) {
	t.Parallel()

	var hits atomic.Int64
	backendAddr, stopBackend := startBackend(t, func(http.ResponseWriter, *http.Request) { hits.Add(1) })
	t.Cleanup(stopBackend)

	clk := clock.New()
	metadata, err := meta.New(testMetadataSigningKey)
	require.NoError(t, err)
	otherKey, err := meta.New("1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100")
	require.NoError(t, err)
	targets, err := peer.ParseCIDRTargets([]string{"127.0.0.0/8"})
	require.NoError(t, err)
	transports := proxy.NewTransportRegistry()
	relay := &peer.Relay{
		Metadata: metadata,
		Targets:  targets,
		Transport: func(protocol string) http.RoundTripper {
			return transports.Get(db.DeploymentsUpstreamProtocol(protocol))
		},
		Clock: clk,
	}

	hop := meta.Hop{Region: "test::test", RequestID: "req", FrontlineID: "fl", TimeUnixMilli: 0}
	target := &meta.Target{DeploymentID: "dep_test", InstanceID: "inst", Address: backendAddr, Protocol: "http1"}
	valid := func() *meta.Metadata {
		//nolint:exhaustruct
		return &meta.Metadata{
			Claims: paseto.Claims{ExpiresAt: clk.Now().Add(time.Minute)},
			Hops:   []meta.Hop{hop},
			Target: target,
		}
	}

	cases := []struct {
		name   string
		codec  *meta.Codec
		mutate func(*meta.Metadata)
		reason string
	}{
		{"wrong_key", otherKey, func(*meta.Metadata) {}, "unauthorized"},
		{"expired", metadata, func(m *meta.Metadata) { m.ExpiresAt = clk.Now().Add(-time.Second) }, "unauthorized"},
		{"two_hops", metadata, func(m *meta.Metadata) { m.Hops = []meta.Hop{hop, hop} }, "hops"},
		{"no_target", metadata, func(m *meta.Metadata) { m.Target = nil }, "bad_request"},
		{"target_outside_cidr", metadata, func(m *meta.Metadata) {
			outside := *target
			outside.Address = "10.1.2.3:8080"
			m.Target = &outside
		}, "instance_unknown"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			md := valid()
			tc.mutate(md)
			token, err := tc.codec.Marshal(md)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodGet, "http://peer-test.example.com/", nil)
			req.Header.Set(peer.HeaderFrontlineMeta, token)
			rec := httptest.NewRecorder()
			relay.ServeHTTP(rec, req)

			require.Equal(t, http.StatusServiceUnavailable, rec.Code)
			require.Equal(t, tc.reason, rec.Header().Get("X-Unkey-Peer-Reject"))
		})
	}
	t.Run("missing_metadata", func(t *testing.T) {
		rec := httptest.NewRecorder()
		relay.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://peer-test.example.com/", nil))
		require.Equal(t, "unauthorized", rec.Header().Get("X-Unkey-Peer-Reject"))
	})
	require.Equal(t, int64(0), hits.Load())
}

// slowReader returns small chunks so the body streams while the relay dials.
type slowReader struct {
	data []byte
}

func (r *slowReader) Read(p []byte) (int, error) {
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p[:min(len(p), 4096)], r.data)
	r.data = r.data[n:]
	return n, nil
}
