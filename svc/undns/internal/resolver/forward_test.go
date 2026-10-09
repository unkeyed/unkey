package resolver

import (
	"context"
	"math"
	"net"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
)

func TestForwardServesExpiredPositiveAfterUpstreamSERVFAIL(t *testing.T) {
	workspaceCaller := newWorkspaceCaller()
	listener, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	var fail atomic.Bool
	writes := make(chan error, 2)
	server := &dnswire.Server{PacketConn: listener, Handler: dnswire.HandlerFunc(func(_ context.Context, writer dnswire.ResponseWriter, query *dnswire.Msg) {
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, query)
		if fail.Load() {
			response.Rcode = dnswire.RcodeServerFailure
		} else {
			response.Answer = cachedAResponse(query.Question[0].Header().Name, 1).Answer
		}
		writes <- writeDNSResponse(writer, response)
	})}
	started := make(chan struct{})
	done := make(chan error, 1)
	server.NotifyStartedFunc = func(context.Context) { close(started) }
	go func() { done <- server.ListenAndServe() }()
	<-started
	t.Cleanup(func() {
		server.Shutdown(t.Context())
		require.NoError(t, <-done)
	})

	clk := clock.NewTestClock()
	h := newTestResolver(t, nil, Config{Upstream: listener.LocalAddr().String(), ForwardTimeout: time.Second, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardCacheEntries: MinForwardCacheEntries, Clock: clk})
	request := dnswire.NewMsg("example.com.", dnswire.TypeA)
	response, result := h.forward(t.Context(), request, "udp", workspaceCaller)
	require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
	require.Equal(t, reasonUpstream, result.reason)
	require.Equal(t, 1.0, gaugeValue(t, "unkey_dns_upstream_healthy"))

	clk.Tick(2 * time.Second)
	fail.Store(true)
	request.ID++
	response, result = h.forward(t.Context(), request, "udp", workspaceCaller)
	require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
	require.Equal(t, uint32(0), response.Answer[0].Header().TTL)
	require.Equal(t, request.ID, response.ID)
	require.Equal(t, reasonCacheStale, result.reason)
	require.Zero(t, gaugeValue(t, "unkey_dns_upstream_healthy"))
	require.NoError(t, <-writes)
	require.NoError(t, <-writes)
}

func TestForwardStaleTimeoutBoundaries(t *testing.T) {
	for _, tt := range []struct {
		name       string
		age        time.Duration
		negative   bool
		cancel     bool
		shed       bool
		wantRcode  uint16
		wantReason reason
	}{
		{name: "positive after timeout", age: 2 * time.Second, wantRcode: dnswire.RcodeSuccess, wantReason: reasonCacheStale},
		{name: "expired stale window", age: 32 * time.Second, wantRcode: dnswire.RcodeServerFailure, wantReason: reasonUpstreamTimeout},
		{name: "negative after timeout", age: 2 * time.Second, negative: true, wantRcode: dnswire.RcodeServerFailure, wantReason: reasonUpstreamTimeout},
		{name: "caller cancellation", age: 2 * time.Second, cancel: true, wantRcode: dnswire.RcodeServerFailure, wantReason: reasonUpstreamError},
		{name: "quota reached", age: 2 * time.Second, shed: true, wantRcode: dnswire.RcodeServerFailure, wantReason: reasonShed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			workspaceCaller := newWorkspaceCaller()
			listener, err := net.ListenPacket("udp", "127.0.0.1:0")
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, listener.Close()) })
			clk := clock.NewTestClock()
			h := newTestResolver(t, nil, Config{Upstream: listener.LocalAddr().String(), ForwardTimeout: 10 * time.Millisecond, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardCacheEntries: MinForwardCacheEntries, Clock: clk})
			request := dnswire.NewMsg("example.com.", dnswire.TypeA)
			cached := cachedAResponse("example.com.", 1)
			if tt.negative {
				cached.Answer = nil
				cached.Rcode = dnswire.RcodeNameError
				cached.Ns = []dnswire.RR{&dnswire.SOA{Hdr: dnswire.Header{Name: "example.com.", Class: dnswire.ClassINET, TTL: 1}, SOA: rdata.SOA{Minttl: 1}}}
			}
			require.NoError(t, h.cache.set(t.Context(), request, cached))
			clk.Tick(tt.age)
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.cancel {
				cancel()
			}
			if tt.shed {
				h.forwards <- struct{}{}
			}
			response, result := h.forward(ctx, request, "udp", workspaceCaller)
			require.Equal(t, tt.wantRcode, response.Rcode)
			require.Equal(t, tt.wantReason, result.reason)
			if tt.wantRcode == dnswire.RcodeSuccess {
				require.Len(t, response.Answer, 1)
				require.Equal(t, uint32(0), response.Answer[0].Header().TTL)
				require.Equal(t, netip.MustParseAddr("192.0.2.1"), response.Answer[0].(*dnswire.A).Addr)
			} else {
				require.Empty(t, response.Answer)
			}
		})
	}
}

func TestCanceledForwardDoesNotPoisonNewCaller(t *testing.T) {
	workspaceCaller := newWorkspaceCaller()
	listener, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	release := make(chan struct{})
	received := make(chan struct{}, 1)
	writeErrors := make(chan error, 2)
	server := &dnswire.Server{PacketConn: listener, Handler: dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, query *dnswire.Msg) {
		select {
		case received <- struct{}{}:
			<-release
		default:
		}
		response := cachedAResponse(query.Question[0].Header().Name, 30)
		dnsutil.SetReply(response, query)
		writeErrors <- writeDNSResponse(w, response)
	})}
	started := make(chan struct{})
	done := make(chan error, 1)
	server.NotifyStartedFunc = func(context.Context) { close(started) }
	go func() { done <- server.ListenAndServe() }()
	<-started
	t.Cleanup(func() {
		server.Shutdown(t.Context())
		require.NoError(t, <-done)
	})

	h := newTestResolver(t, nil, Config{Upstream: listener.LocalAddr().String(), ForwardTimeout: time.Second, QueriesInFlight: 2, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardCacheEntries: MinForwardCacheEntries})
	ctx, cancel := context.WithCancel(t.Context())
	firstDone := make(chan *dnswire.Msg, 1)
	go func() {
		response, _ := h.forward(ctx, dnswire.NewMsg("example.com.", dnswire.TypeA), "udp", workspaceCaller)
		firstDone <- response
	}()
	<-received
	cancel()
	require.Equal(t, uint16(dnswire.RcodeServerFailure), (<-firstDone).Rcode)
	close(release)
	response, _ := h.forward(t.Context(), dnswire.NewMsg("example.com.", dnswire.TypeA), "udp", workspaceCaller)
	require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
	require.NoError(t, <-writeErrors)
	require.NoError(t, <-writeErrors)
}

func TestForwardCacheRestoresRequestQuestionSpelling(t *testing.T) {
	workspaceCaller := newWorkspaceCaller()
	h := newTestResolver(t, nil, Config{ForwardTimeout: time.Second, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardCacheEntries: MinForwardCacheEntries})
	query := dnswire.NewMsg("mixed.example.", dnswire.TypeA)
	require.NoError(t, h.cache.set(t.Context(), query, cachedAResponse("MiXeD.Example.", 30)))
	request := dnswire.NewMsg("mIxEd.ExAmPlE.", dnswire.TypeA)
	got, _ := h.forward(t.Context(), request, "udp", workspaceCaller)
	require.Equal(t, "mIxEd.ExAmPlE.", got.Question[0].Header().Name)
	require.Equal(t, "mIxEd.ExAmPlE.", request.Question[0].Header().Name)
}

func TestForwardRetriesTruncatedUDPOverTCPAndStripsClientOptions(t *testing.T) {
	workspaceCaller := newWorkspaceCaller()
	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	udp, err := net.ListenPacket("udp", tcp.Addr().String())
	require.NoError(t, err)

	queries := make(chan *dnswire.Msg, 2)
	writeErrors := make(chan error, 2)
	upstream := dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, query *dnswire.Msg) {
		queries <- query.Copy()
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, query)
		if _, ok := w.RemoteAddr().(*net.UDPAddr); ok {
			response.Truncated = true
		} else {
			response.Answer = []dnswire.RR{&dnswire.A{
				Hdr: dnswire.Header{Name: query.Question[0].Header().Name, Class: dnswire.ClassINET, TTL: 23},
				A:   rdata.A{Addr: netip.MustParseAddr("192.0.2.7")},
			}}
		}
		writeErrors <- writeDNSResponse(w, response)
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

	h := newTestResolver(t, nil, Config{Upstream: tcp.Addr().String(), ForwardTimeout: time.Second, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1})
	request := dnswire.NewMsg("example.com.", dnswire.TypeA)
	request.UDPSize = 4096
	request.Pseudo = append(request.Pseudo, &dnswire.SUBNET{Family: 1, Netmask: 32, Address: netip.MustParseAddr("10.7.0.1")})
	response, result := h.forward(t.Context(), request, "udp", workspaceCaller)
	require.Equal(t, reasonUpstream, result.reason)
	require.Equal(t, request.ID, response.ID)
	require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
	require.Len(t, response.Answer, 1)
	require.Equal(t, "192.0.2.7", response.Answer[0].(*dnswire.A).A.Addr.String())
	for range 2 {
		require.Empty(t, (<-queries).Pseudo)
		require.NoError(t, <-writeErrors)
	}
}

func TestForwardStopsWhenServerShutsDown(t *testing.T) {
	workspaceCaller := newWorkspaceCaller()
	upstream, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, upstream.Close()) })
	h := newTestResolver(t, nil, Config{Upstream: upstream.LocalAddr().String(), ForwardTimeout: time.Minute, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	started := time.Now()
	response, result := h.forward(ctx, dnswire.NewMsg("example.com.", dnswire.TypeA), "udp", workspaceCaller)
	require.Equal(t, uint16(dnswire.RcodeServerFailure), response.Rcode)
	require.Equal(t, reasonUpstreamError, result.reason, "a canceled forward is not an upstream timeout")
	require.Less(t, time.Since(started), 10*time.Second, "forward ignored server shutdown")
	require.Empty(t, h.workspaceForwards, "forward kept its workspace slot after returning")
}

func TestForwardTreatsOutOfRangeTTLsAsZero(t *testing.T) {
	for _, rcode := range []uint16{dnswire.RcodeSuccess, dnswire.RcodeServerFailure} {
		t.Run(dnswire.RcodeToString[rcode], func(t *testing.T) {
			var upstreamQueries atomic.Int32
			upstream := startUpstream(t, func(query *dnswire.Msg) *dnswire.Msg {
				upstreamQueries.Add(1)
				response := new(dnswire.Msg)
				dnsutil.SetReply(response, query)
				response.Rcode = rcode
				response.Answer = cachedAResponse(query.Question[0].Header().Name, 1<<31).Answer
				return response
			})
			h := newTestResolver(t, newFakeCatalog(), Config{Upstream: upstream, ForwardTimeout: time.Second, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, Clock: clock.NewTestClock()})

			for range 2 {
				response := serveQueryOver(t, h, "127.0.0.1", "example.com.", dnswire.TypeA, "tcp")
				require.Equal(t, rcode, response.Rcode)
				require.Len(t, response.Answer, 1)
				require.Zero(t, response.Answer[0].Header().TTL)
			}
			require.Equal(t, int32(2), upstreamQueries.Load(), "a zero TTL answer is not served from cache")
		})
	}
}

func TestZeroOutOfRangeTTLsKeepsValidTTLsAndEDNSFlags(t *testing.T) {
	response := cachedAResponse("example.", math.MaxInt32)
	response.Answer = append(response.Answer, &dnswire.CNAME{
		Hdr:   dnswire.Header{Name: "alias.example.", Class: dnswire.ClassINET, TTL: 1 << 31},
		CNAME: rdata.CNAME{Target: "example."},
	})
	response.Ns = []dnswire.RR{&dnswire.SOA{
		Hdr: dnswire.Header{Name: "example.", Class: dnswire.ClassINET, TTL: math.MaxUint32},
		SOA: rdata.SOA{Minttl: 1 << 31},
	}}
	response.Extra = []dnswire.RR{&dnswire.OPT{Hdr: dnswire.Header{Name: ".", Class: 1232, TTL: 1 << 31}}}

	zeroOutOfRangeTTLs(response)

	require.Equal(t, uint32(math.MaxInt32), response.Answer[0].Header().TTL)
	require.Zero(t, response.Answer[1].Header().TTL)
	require.Zero(t, response.Ns[0].Header().TTL)
	require.Zero(t, response.Ns[0].(*dnswire.SOA).Minttl)
	require.Equal(t, uint32(1<<31), response.Extra[0].Header().TTL, "OPT carries EDNS flags in its TTL field")
}
