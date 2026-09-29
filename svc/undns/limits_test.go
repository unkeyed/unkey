package undns

import (
	"context"
	"fmt"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPrivateAnswersShuffleAddresses(t *testing.T) {
	c := catalogForTest()
	addBinding(t, c, "binding-a", "workspace-a", "project-a", "app-a", "payments", "deployment-a", "service-a", "1")
	service := addServiceAndSlice(t, c, "service-a", "deployment-a", "app-a", types.UID("service-a-uid"), "10.0.0.1", true)
	want := []string{"10.0.0.1"}
	for i := 2; i <= 8; i++ {
		address := fmt.Sprintf("10.0.0.%d", i)
		addSlice(t, c, service, fmt.Sprintf("service-a-%d", i), address, true)
		want = append(want, address)
	}
	h := newHandler(c, Config{TTLSeconds: 5, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1}, prometheus.NewRegistry())

	question := dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA).Question[0]
	firsts := map[string]bool{}
	for range 64 {
		response := new(dnswire.Msg)
		h.answerPrivate(response, question, testCaller())
		got := answerAddresses(response)
		firsts[got[0]] = true
		slices.Sort(got)
		require.Equal(t, want, got)
	}
	require.Greater(t, len(firsts), 1, "64 answers all started with the same address")
}

func TestSlowForwardsDoNotStarveLocalAnswers(t *testing.T) {
	s := startLimitedServer(t, 2, 2)
	for range 2 {
		s.queryAsync("127.0.0.1", "slow.example.")
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

	s.catalog.pods.lastContact.Store(0)
	require.Equal(t, uint16(dnswire.RcodeServerFailure), s.query("127.0.0.1", "fast.example.").Rcode,
		"stale pod identity must use the same full unidentified quota")
	require.Equal(t, uint16(dnswire.RcodeServerFailure), s.query("127.0.0.1", "payments.unkey.internal.").Rcode)
	close(s.release)
	require.Eventually(t, func() bool {
		return s.query("127.0.0.4", "fast.example.").Rcode == dnswire.RcodeSuccess
	}, 5*time.Second, 10*time.Millisecond, "the unidentified slot must be released")
}

type limitedServer struct {
	t        *testing.T
	address  string
	catalog  *catalog
	received chan struct{}
	release  chan struct{}
	pending  sync.WaitGroup
}

func startLimitedServer(t *testing.T, forwardsInFlight, forwardsPerWorkspace int) *limitedServer {
	t.Helper()
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	other := callerPod("127.0.0.2", "production")
	other.Name, other.UID = "caller-b", "caller-b"
	other.Labels[labels.LabelKeyWorkspaceID] = "workspace-b"
	client := fake.NewSimpleClientset(append(discoveryObjects(t), other)...)
	c, err := newCatalog(client, 30*time.Second)
	require.NoError(t, err)

	s := &limitedServer{t: t, address: "", catalog: c, received: make(chan struct{}, 8), release: make(chan struct{}), pending: sync.WaitGroup{}}
	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	upstream := &dnswire.Server{Listener: upstreamListener, Handler: dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, query *dnswire.Msg) {
		if query.Question[0].Header().Name == "slow.example." {
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
	cfg.ListenAddress, cfg.HealthAddress = unusedTCPAddress(t), unusedTCPAddress(t)
	cfg.ForwardTimeout = 5 * time.Second
	cfg.QueriesInFlight, cfg.ForwardsInFlight, cfg.ForwardsPerWorkspace = 1, forwardsInFlight, forwardsPerWorkspace
	cfg.ForwardsUnidentified = 1
	s.address = cfg.ListenAddress

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, c) }()
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
	require.Eventually(t, c.ready, 5*time.Second, 10*time.Millisecond)
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
