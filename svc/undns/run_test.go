package undns

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
)

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

	c, err := newCatalog(client, time.Minute)
	require.NoError(t, err)

	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	udp, err := net.ListenPacket("udp", tcp.Addr().String())
	require.NoError(t, err)

	var forwarded atomic.Int64
	upstream := dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, query *dnswire.Msg) {
		forwarded.Add(1)
		if strings.HasSuffix(query.Question[0].Header().Name, privateZone) {
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
	cfg.ListenAddress, cfg.HealthAddress = unusedTCPAddress(t), unusedTCPAddress(t)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, c) }()
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

	check := func(phase string, privateCode uint16, discoveryMetric string) {
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
		require.Contains(t, string(body), "unkey_dns_discovery_ready "+discoveryMetric+"\n", phase)

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

	require.False(t, c.ready())
	check("cold", dnswire.RcodeServerFailure, "0")

	apiUnavailable.Store(false)
	require.Eventually(t, c.ready, 10*time.Second, 10*time.Millisecond)
	check("synchronized", dnswire.RcodeSuccess, "1")

	for name, informer := range map[string]*trackedInformer{
		"pods": c.pods, "bindings": c.bindings, "services": c.services, "slices": c.slices,
	} {
		informer.lastContact.Store(0)
		check(name+"-failed", dnswire.RcodeServerFailure, "0")
		informer.lastContact.Store(time.Now().Add(-3 * time.Minute).UnixNano())
		check(name+"-stale", dnswire.RcodeServerFailure, "0")
		informer.lastContact.Store(time.Now().UnixNano())
		check(name+"-recovered", dnswire.RcodeSuccess, "1")
	}
}
