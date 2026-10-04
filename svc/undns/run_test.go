package undns

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/svc/undns/internal/server"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
)

func TestServerDrainsTCPRepliesAtQueryLimit(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	c := newTestCatalog(t, fake.NewSimpleClientset(discoveryObjects(t)...), time.Minute)
	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(t, err)
	tcp, udp, health := testListeners(t, &cfg)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- serveListeners(ctx, cfg, c, tcp, udp, health) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("DNS server did not shut down")
		}
	})
	require.Eventually(t, c.Ready, 5*time.Second, time.Millisecond)

	conn, err := net.DialTimeout("tcp", cfg.ListenAddress, time.Second)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	client := testDNSClient(time.Second)
	for i := range dnswire.MaxTCPQueries {
		query := dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA)
		query.ID = uint16(i)
		response, _, err := client.ExchangeWithConn(t.Context(), query, conn)
		require.NoError(t, err, "request %d/%d", i+1, dnswire.MaxTCPQueries)
		require.Equal(t, query.ID, response.ID)
		require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
		require.False(t, response.Truncated)
		require.Len(t, response.Answer, 40)
	}
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(time.Second)))
	_, err = conn.Read(make([]byte, 1))
	require.ErrorIs(t, err, io.EOF, "the query cap must still close the connection")
}

// TestServeFailsWhenTCPListenerFails guarantees that losing the DNS TCP
// listener without shutdown stops the whole process instead of leaving it
// serving UDP only.
func TestServeFailsWhenTCPListenerFails(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	c := newTestCatalog(t, fake.NewSimpleClientset(discoveryObjects(t)...), time.Minute)
	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(t, err)
	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	udp, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	health, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- serveListeners(t.Context(), cfg, c, tcp, udp, health) }()
	require.Eventually(t, func() bool {
		return server.Probe(t.Context(), "udp", udp.LocalAddr().String()) == nil
	}, 5*time.Second, 10*time.Millisecond)

	require.NoError(t, tcp.Close())
	select {
	case err := <-done:
		require.ErrorIs(t, err, net.ErrClosed)
		require.ErrorContains(t, err, "accept DNS TCP connection")
	case <-time.After(5 * time.Second):
		t.Fatal("serve kept running after its TCP listener failed")
	}
}

// TestPublicDNSAndReadinessSurviveDiscoveryFailure guarantees that a cold,
// failed, or stalled discovery only closes private answers: public forwarding
// and listener readiness keep working, and only public questions reach the
// upstream.
func TestPublicDNSAndReadinessSurviveDiscoveryFailure(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	var apiUnavailable atomic.Bool
	apiUnavailable.Store(true)
	client := fake.NewSimpleClientset(discoveryObjects(t)...)
	client.PrependReactor("list", "*", func(kubetesting.Action) (bool, runtime.Object, error) {
		if apiUnavailable.Load() {
			return true, nil, errors.New("API unavailable")
		}
		return false, nil, nil
	})

	clk := clock.NewTestClock()
	c := newTestCatalogWithClock(t, client, time.Minute, clk)

	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	udp, err := net.ListenPacket("udp", tcp.Addr().String())
	require.NoError(t, err)

	var forwarded atomic.Int64
	upstream := dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, query *dnswire.Msg) {
		forwarded.Add(1)
		if strings.HasSuffix(query.Question[0].Header().Name, "unkey.internal.") {
			t.Error("private query leaked to upstream")
		}
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, query)
		response.Answer = []dnswire.RR{&dnswire.A{
			Hdr: dnswire.Header{Name: query.Question[0].Header().Name, Class: dnswire.ClassINET, TTL: 19},
			A:   rdata.A{Addr: netip.MustParseAddr("203.0.113.7")},
		}}
		if err := writeDNSResponse(w, response); err != nil {
			t.Errorf("write upstream response: %v", err)
		}
	})

	for _, server := range []*dnswire.Server{{Listener: tcp}, {PacketConn: udp}} {
		started := make(chan struct{})
		done := make(chan error, 1)
		server.Handler = upstream
		server.NotifyStartedFunc = func(context.Context) { close(started) }
		go func() { done <- server.ListenAndServe() }()
		<-started
		t.Cleanup(func() {
			server.Shutdown(t.Context())
			require.NoError(t, <-done)
		})
	}

	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(t, err)
	cfg.Upstream = tcp.Addr().String()
	dnsTCP, dnsUDP, health := testListeners(t, &cfg)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- serveListeners(ctx, cfg, c, dnsTCP, dnsUDP, health) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	httpClient := &http.Client{Timeout: time.Second}
	require.EventuallyWithT(t, func(collect *assert.CollectT) {
		response, getErr := httpClient.Get("http://" + cfg.HealthAddress + "/health/ready")
		require.NoError(collect, getErr)
		require.NoError(collect, response.Body.Close())
		require.Equal(collect, http.StatusOK, response.StatusCode)
	}, 5*time.Second, 10*time.Millisecond)

	check := func(phase string, privateCode uint16, discoveryReady float64) {
		t.Helper()
		response, getErr := httpClient.Get("http://" + cfg.HealthAddress + "/health/ready")
		require.NoError(t, getErr)
		require.NoError(t, response.Body.Close())
		require.Equal(t, http.StatusOK, response.StatusCode, phase)

		response, getErr = httpClient.Get("http://" + cfg.HealthAddress + "/metrics")
		require.NoError(t, getErr)
		body, readErr := io.ReadAll(response.Body)
		require.NoError(t, response.Body.Close())
		require.NoError(t, readErr)
		require.Equal(t, http.StatusOK, response.StatusCode, phase)
		require.Contains(t, string(body), "go_goroutines", phase)
		require.Eventually(t, func() bool {
			return gaugeValue(t, "unkey_dns_discovery_ready") == discoveryReady
		}, 5*time.Second, 10*time.Millisecond, "unkey_dns_discovery_ready during %s", phase)

		before := forwarded.Load()
		for _, transport := range []string{"udp", "tcp"} {
			dns := testDNSClient(time.Second)
			public, _, queryErr := dns.Exchange(t.Context(), dnswire.NewMsg(fmt.Sprintf("%s-%s.example.", phase, transport), dnswire.TypeA), transport, cfg.ListenAddress)
			require.NoError(t, queryErr)
			require.Equal(t, uint16(dnswire.RcodeSuccess), public.Rcode, phase)
			require.Equal(t, []string{"203.0.113.7"}, answerAddresses(public), phase)

			private, _, queryErr := dns.Exchange(t.Context(), dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA), transport, cfg.ListenAddress)
			require.NoError(t, queryErr)
			require.Equal(t, privateCode, private.Rcode, phase)
			if privateCode == dnswire.RcodeSuccess {
				require.NotEmpty(t, private.Answer)
			} else {
				require.Empty(t, private.Answer)
			}
		}
		require.Equal(t, before+2, forwarded.Load(), "only public questions reach upstream")
	}

	require.False(t, c.Ready())
	check("cold", dnswire.RcodeServerFailure, 0)

	apiUnavailable.Store(false)
	require.Eventually(t, c.Ready, 10*time.Second, 10*time.Millisecond)
	check("synchronized", dnswire.RcodeSuccess, 1)

	clk.Tick(3 * time.Minute)
	require.False(t, c.Ready())
	check("stale", dnswire.RcodeServerFailure, 0)

	clk.Tick(-3 * time.Minute)
	require.True(t, c.Ready())
	check("recovered", dnswire.RcodeSuccess, 1)
}

func TestMain(m *testing.M) {
	testRegistry = prometheus.NewServiceRegistry()
	os.Exit(m.Run())
}

var testRegistry *promclient.Registry

func gaugeValue(t testing.TB, name string) float64 {
	t.Helper()
	families, err := testRegistry.Gather()
	if err != nil {
		t.Errorf("gather metrics: %v", err)
		return -1
	}
	for _, family := range families {
		if family.GetName() == name && len(family.GetMetric()) == 1 {
			return family.GetMetric()[0].GetGauge().GetValue()
		}
	}
	return -1
}
