package undns

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/undns/internal/discovery"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSlowForwardsDoNotStarveLocalAnswers(t *testing.T) {
	s := startLimitedServer(t, 2, 2)
	for i := range 2 {
		s.queryAsync("127.0.0.1", fmt.Sprintf("slow-%d.example.", i))
		s.waitHeld()
	}

	require.Equal(t, uint16(dnswire.RcodeServerFailure), s.query("127.0.0.2", "fast.example.").Rcode)
	require.Equal(t, uint16(dnswire.RcodeServerFailure), s.query("127.0.0.3", "fast.example.").Rcode,
		"unidentified callers must also obey the full global quota")
	private := s.query("127.0.0.1", "payments.unkey.internal.")
	require.Equal(t, uint16(dnswire.RcodeSuccess), private.Rcode)
	require.Len(t, private.Answer, 40)

	close(s.release)
	require.Eventually(t, func() bool {
		return s.query("127.0.0.2", "fast.example.").Rcode == dnswire.RcodeSuccess
	}, 5*time.Second, 10*time.Millisecond)
}

func TestForwardsPerWorkspaceKeepOtherWorkspacesServed(t *testing.T) {
	s := startLimitedServer(t, 2, 1)
	s.queryAsync("127.0.0.1", "slow.example.")
	s.waitHeld()

	require.Equal(t, uint16(dnswire.RcodeServerFailure), s.query("127.0.0.1", "fast.example.").Rcode)
	require.Equal(t, uint16(dnswire.RcodeSuccess), s.query("127.0.0.2", "fast.example.").Rcode)
	close(s.release)
}

func TestUnidentifiedForwardsShareQuotaWithoutStarvingKnownCallers(t *testing.T) {
	s := startLimitedServer(t, 3, 2)
	s.queryAsync("127.0.0.3", "slow.example.")
	s.waitHeld()

	require.Equal(t, uint16(dnswire.RcodeServerFailure), s.query("127.0.0.4", "fast.example.").Rcode)
	require.Equal(t, uint16(dnswire.RcodeSuccess), s.query("127.0.0.1", "fast.example.").Rcode)
	require.Equal(t, uint16(dnswire.RcodeSuccess), s.query("127.0.0.1", "payments.unkey.internal.").Rcode)
	require.Equal(t, uint16(dnswire.RcodeRefused), s.query("127.0.0.3", "payments.unkey.internal.").Rcode)

	s.clock.Tick(3 * s.watchTimeout)
	require.Equal(t, uint16(dnswire.RcodeServerFailure), s.query("127.0.0.1", "fast.example.").Rcode,
		"stale pod identity must use the same full unidentified quota")
	require.Equal(t, uint16(dnswire.RcodeServerFailure), s.query("127.0.0.1", "payments.unkey.internal.").Rcode)
	close(s.release)
	require.Eventually(t, func() bool {
		return s.query("127.0.0.4", "fast.example.").Rcode == dnswire.RcodeSuccess
	}, 5*time.Second, 10*time.Millisecond, "the unidentified slot must be released")
}

type limitedServer struct {
	t            *testing.T
	address      string
	catalog      *discovery.Catalog
	clock        *clock.TestClock
	watchTimeout time.Duration
	received     chan struct{}
	release      chan struct{}
	pending      sync.WaitGroup
}

func startLimitedServer(t *testing.T, forwardsInFlight, forwardsPerWorkspace int) *limitedServer {
	t.Helper()
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	other := callerPod("127.0.0.2", "production")
	other.Name = "caller-b"
	other.Labels[labels.LabelKeyWorkspaceID] = uid.New(uid.WorkspacePrefix)
	client := fake.NewSimpleClientset(append(discoveryObjects(t), other)...)
	clk := clock.NewTestClock()
	const watchTimeout = 30 * time.Second
	c := newTestCatalogWithClock(t, client, watchTimeout, clk)

	s := &limitedServer{t: t, address: "", catalog: c, clock: clk, watchTimeout: watchTimeout, received: make(chan struct{}, 8), release: make(chan struct{}), pending: sync.WaitGroup{}}
	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	upstream := &dnswire.Server{Listener: upstreamListener, Handler: dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, query *dnswire.Msg) {
		if strings.HasPrefix(query.Question[0].Header().Name, "slow") {
			s.received <- struct{}{}
			<-s.release
		}
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, query)
		if err := writeDNSResponse(w, response); err != nil {
			t.Errorf("write upstream response: %v", err)
		}
	})}

	started := make(chan struct{})
	upstream.NotifyStartedFunc = func(context.Context) { close(started) }
	upstreamDone := make(chan error, 1)
	go func() { upstreamDone <- upstream.ListenAndServe() }()
	<-started

	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(t, err)
	cfg.Upstream = upstreamListener.Addr().String()
	tcp, udp, health := testListeners(t, &cfg)
	cfg.ForwardTimeout = 5 * time.Second
	cfg.QueriesInFlight, cfg.ForwardsInFlight, cfg.ForwardsPerWorkspace = 1, forwardsInFlight, forwardsPerWorkspace
	cfg.ForwardsUnidentified = 1
	s.address = cfg.ListenAddress

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- serveListeners(ctx, cfg, c, tcp, udp, health) }()
	t.Cleanup(func() {
		select {
		case <-s.release:
		default:
			close(s.release)
		}
		s.pending.Wait()
		cancel()
		require.NoError(t, <-done)
		upstream.Shutdown(t.Context())
		require.NoError(t, <-upstreamDone)
	})
	require.Eventually(t, c.Ready, 5*time.Second, 10*time.Millisecond)
	return s
}

func (s *limitedServer) queryAsync(source, name string) {
	s.pending.Go(func() {
		if response := s.query(source, name); response.Rcode != dnswire.RcodeSuccess {
			s.t.Errorf("held query %s from %s: %s", name, source, dnswire.RcodeToString[response.Rcode])
		}
	})
}

func (s *limitedServer) waitHeld() {
	s.t.Helper()
	select {
	case <-s.received:
	case <-time.After(5 * time.Second):
		s.t.Fatal("upstream never received the held query")
	}
}

func (s *limitedServer) query(source, name string) *dnswire.Msg {
	question := dnswire.NewMsg(name, dnswire.TypeA)
	client := &dnswire.Client{Transport: &dnswire.Transport{Dialer: &net.Dialer{Timeout: 10 * time.Second, LocalAddr: &net.TCPAddr{IP: net.ParseIP(source)}}, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}}
	response, _, err := client.Exchange(context.Background(), question, "tcp", s.address)
	if err != nil {
		s.t.Errorf("query %s from %s: %v", name, source, err)
		return new(dnswire.Msg)
	}
	return response
}

func answerAddresses(response *dnswire.Msg) []string {
	addresses := make([]string, 0, len(response.Answer))
	for _, answer := range response.Answer {
		addresses = append(addresses, answer.(*dnswire.A).A.Addr.String())
	}
	return addresses
}
