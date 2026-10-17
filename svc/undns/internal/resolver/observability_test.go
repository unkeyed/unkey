package resolver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	promclient "github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/undns/internal/discovery"
)

var testRegistry *promclient.Registry

func TestMain(m *testing.M) {
	testRegistry = prometheus.NewServiceRegistry()
	os.Exit(m.Run())
}

func newWorkspaceCaller() discovery.Caller {
	return discovery.Caller{Workspace: uid.New(uid.WorkspacePrefix), Project: uid.New(uid.ProjectPrefix), Deployment: uid.New(uid.DeploymentPrefix), Namespace: "default"}
}

type fakeCatalog struct {
	identityReady bool
	ready         bool
	identifyErr   error
	resolveErr    error
	callers       map[netip.Addr]discovery.Caller
	names         map[string][]netip.Addr
}

func newFakeCatalog() *fakeCatalog {
	addresses := make([]netip.Addr, 0, 40)
	for i := 1; i <= 40; i++ {
		addresses = append(addresses, netip.AddrFrom4([4]byte{10, 0, 0, byte(i)}))
	}
	return &fakeCatalog{
		identityReady: true,
		ready:         true,
		callers:       map[netip.Addr]discovery.Caller{netip.MustParseAddr("127.0.0.1"): newWorkspaceCaller()},
		names:         map[string][]netip.Addr{"payments": addresses},
	}
}

func (f *fakeCatalog) Ready() bool         { return f.ready }
func (f *fakeCatalog) IdentityReady() bool { return f.identityReady }

func (f *fakeCatalog) Identify(ip netip.Addr) (discovery.Caller, error) {
	if f.identifyErr != nil {
		return noCaller(), f.identifyErr
	}
	caller, exists := f.callers[ip]
	if !exists {
		return noCaller(), &discovery.Error{Reason: discovery.ReasonUnknownCaller, Err: errors.New("no pod uses the source IP")}
	}
	return caller, nil
}

func (f *fakeCatalog) Resolve(_ discovery.Caller, name string) ([]netip.Addr, bool, error) {
	if f.resolveErr != nil {
		return nil, true, f.resolveErr
	}
	addresses, exists := f.names[name]
	return slices.Clone(addresses), exists, nil
}

func newTestResolver(t *testing.T, catalog Catalog, cfg Config) *Resolver {
	t.Helper()
	if cfg.Clock == nil {
		cfg.Clock = clock.New()
	}
	if cfg.ForwardCacheEntries == 0 {
		cfg.ForwardCacheEntries = MinForwardCacheEntries
	}
	r, err := New(catalog, cfg)
	require.NoError(t, err)
	t.Cleanup(r.Close)
	return r
}

func discoveryFailure(reason discovery.Reason) func(*fakeCatalog) {
	return func(f *fakeCatalog) {
		f.resolveErr = fmt.Errorf("resolve payments: %w", &discovery.Error{Reason: reason, Err: errors.New("discovery failed")})
	}
}

// TestPrivateFailuresReportDistinctReasons guarantees that each discovery
// state maps to the response code clients depend on and to its own reason in
// unkey_dns_queries_total.
func TestPrivateFailuresReportDistinctReasons(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source string
		qtype  uint16
		qname  string
		mutate func(f *fakeCatalog)
		rcode  uint16
		reason reason
	}{
		{name: "answer", rcode: dnswire.RcodeSuccess, reason: reasonAnswer},
		{name: "ipv6", qtype: dnswire.TypeAAAA, rcode: dnswire.RcodeSuccess, reason: reasonNoData},
		{name: "unsupported type", qtype: dnswire.TypeMX, rcode: dnswire.RcodeNotImplemented, reason: reasonUnsupportedType},
		{name: "unknown alias", qname: "ledger.unkey.internal.", rcode: dnswire.RcodeNameError, reason: reasonUnknownName},
		{name: "unknown source", source: "127.0.0.9", rcode: dnswire.RcodeRefused, reason: reason(discovery.ReasonUnknownCaller)},
		{
			name: "reused source IP",
			mutate: func(f *fakeCatalog) {
				f.identifyErr = &discovery.Error{Reason: discovery.ReasonAmbiguousCaller, Err: nil}
			},
			rcode: dnswire.RcodeRefused, reason: reason(discovery.ReasonAmbiguousCaller),
		},
		{
			name: "caller without deployment",
			mutate: func(f *fakeCatalog) {
				f.identifyErr = &discovery.Error{Reason: discovery.ReasonIneligibleCaller, Err: nil}
			},
			rcode: dnswire.RcodeRefused, reason: reason(discovery.ReasonIneligibleCaller),
		},
		{
			name:   "pod watch stale",
			mutate: func(f *fakeCatalog) { f.identityReady, f.ready = false, false },
			rcode:  dnswire.RcodeServerFailure, reason: reasonIdentityUnavailable,
		},
		{
			name:   "endpoint watch stale",
			mutate: func(f *fakeCatalog) { f.ready = false },
			rcode:  dnswire.RcodeServerFailure, reason: reasonDiscoveryNotReady,
		},
		{name: "no selected target", mutate: discoveryFailure(discovery.ReasonConnectionUnresolved), rcode: dnswire.RcodeServerFailure, reason: reason(discovery.ReasonConnectionUnresolved)},
		{name: "corrupt revision", mutate: discoveryFailure(discovery.ReasonConnectionInvalid), rcode: dnswire.RcodeServerFailure, reason: reason(discovery.ReasonConnectionInvalid)},
		{name: "duplicate connection", mutate: discoveryFailure(discovery.ReasonConnectionAmbiguous), rcode: dnswire.RcodeServerFailure, reason: reason(discovery.ReasonConnectionAmbiguous)},
		{name: "discovery service deleted", mutate: discoveryFailure(discovery.ReasonServiceMissing), rcode: dnswire.RcodeServerFailure, reason: reason(discovery.ReasonServiceMissing)},
		{name: "discovery service publishes unready pods", mutate: discoveryFailure(discovery.ReasonServiceRejected), rcode: dnswire.RcodeServerFailure, reason: reason(discovery.ReasonServiceRejected)},
		{name: "discovery service retired", mutate: discoveryFailure(discovery.ReasonServiceRetired), rcode: dnswire.RcodeServerFailure, reason: reason(discovery.ReasonServiceRetired)},
		{name: "target lost its endpoints", mutate: discoveryFailure(discovery.ReasonNoReadyEndpoints), rcode: dnswire.RcodeServerFailure, reason: reason(discovery.ReasonNoReadyEndpoints)},
		{
			name:   "cache failure",
			mutate: func(f *fakeCatalog) { f.resolveErr = errors.New("index failure") },
			rcode:  dnswire.RcodeServerFailure, reason: reason(discovery.ReasonLookupError),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			catalog := newFakeCatalog()
			if tc.mutate != nil {
				tc.mutate(catalog)
			}
			h := newTestResolver(t, catalog, Config{TTLSeconds: 5, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1})
			source, qname, qtype := tc.source, tc.qname, tc.qtype
			if source == "" {
				source = "127.0.0.1"
			}
			if qname == "" {
				qname = "payments.unkey.internal."
			}
			if qtype == 0 {
				qtype = dnswire.TypeA
			}

			before := metricValues(t, "unkey_dns_queries_total")
			response := serveQuery(t, h, source, qname, qtype)
			require.Equal(t, dnsutil.RcodeToString(tc.rcode), dnsutil.RcodeToString(response.Rcode), "rcode for %s", tc.name)
			require.Equal(t, map[string]float64{queryLabels(pathPrivate, tc.reason, tc.rcode, "udp"): 1}, metricDelta(before, metricValues(t, "unkey_dns_queries_total")),
				"unkey_dns_queries_total after %s", tc.name)
		})
	}
}

func TestPrivateAnswersShuffleAddresses(t *testing.T) {
	catalog := newFakeCatalog()
	catalog.names["payments"] = catalog.names["payments"][:8]
	want := make([]string, 0, 8)
	for _, address := range catalog.names["payments"] {
		want = append(want, address.String())
	}
	h := newTestResolver(t, catalog, Config{TTLSeconds: 5, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1})

	question := dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA).Question[0]
	caller := catalog.callers[netip.MustParseAddr("127.0.0.1")]
	firsts := map[string]bool{}
	for range 64 {
		response := new(dnswire.Msg)
		h.answerPrivate(response, question, caller)
		got := answerAddresses(response)
		firsts[got[0]] = true
		slices.Sort(got)
		require.Equal(t, want, got)
	}
	require.Greater(t, len(firsts), 1, "64 answers all started with the same address")
}

func TestPublicForwardFailuresReportDistinctReasons(t *testing.T) {
	upstream := startUpstream(t, func(query *dnswire.Msg) *dnswire.Msg {
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, query)
		switch query.Question[0].Header().Name {
		case "broken.example.":
			response.Rcode = dnswire.RcodeServerFailure
		case "mismatch.example.":
			response.Question[0].Header().Name = "other.example."
		case "slow.example.":
			return nil
		}
		return response
	})
	closed := unusedTCPAddress(t)

	for _, tc := range []struct {
		name      string
		qname     string
		upstream  string
		rcode     uint16
		reason    reason
		transport string
	}{
		{name: "answer", qname: "ok.example.", upstream: upstream, rcode: dnswire.RcodeSuccess, reason: reasonUpstream, transport: "tcp"},
		{name: "upstream servfail", qname: "broken.example.", upstream: upstream, rcode: dnswire.RcodeServerFailure, reason: reasonUpstream, transport: "tcp"},
		{name: "wrong question", qname: "mismatch.example.", upstream: upstream, rcode: dnswire.RcodeServerFailure, reason: reasonUpstreamInvalid, transport: "tcp"},
		{name: "no reply", qname: "slow.example.", upstream: upstream, rcode: dnswire.RcodeServerFailure, reason: reasonUpstreamTimeout, transport: "tcp"},
		{name: "refused connection", qname: "ok.example.", upstream: closed, rcode: dnswire.RcodeServerFailure, reason: reasonUpstreamError, transport: "tcp"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestResolver(t, newFakeCatalog(), Config{Upstream: tc.upstream, ForwardTimeout: 200 * time.Millisecond, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1})

			before := metricValues(t, "unkey_dns_queries_total")
			samples := latencySamples(t, pathPublic)
			response := serveQueryOver(t, h, "127.0.0.1", tc.qname, dnswire.TypeA, tc.transport)
			require.Equal(t, dnsutil.RcodeToString(tc.rcode), dnsutil.RcodeToString(response.Rcode), "rcode for %s", tc.name)
			require.Equal(t, map[string]float64{queryLabels(pathPublic, tc.reason, tc.rcode, tc.transport): 1}, metricDelta(before, metricValues(t, "unkey_dns_queries_total")),
				"unkey_dns_queries_total after %s", tc.name)
			require.Equal(t, samples+1, latencySamples(t, pathPublic), "public latency samples after %s", tc.name)
			healthy := 0.0
			if tc.rcode == dnswire.RcodeSuccess {
				healthy = 1
			}
			require.Equal(t, healthy, gaugeValue(t, "unkey_dns_upstream_healthy"), "upstream health after %s", tc.name)
		})
	}
}

func TestShedQueriesReportLimitAndPath(t *testing.T) {
	h := newTestResolver(t, newFakeCatalog(), Config{Upstream: "127.0.0.1:1", ForwardTimeout: time.Second, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1})
	queriesBefore := metricValues(t, "unkey_dns_queries_total")
	shedBefore := metricValues(t, "unkey_dns_shed_total")

	h.slots <- struct{}{}
	require.Equal(t, uint16(dnswire.RcodeServerFailure), serveQuery(t, h, "127.0.0.1", "payments.unkey.internal.", dnswire.TypeA).Rcode)
	<-h.slots
	h.forwards <- struct{}{}
	require.Equal(t, uint16(dnswire.RcodeServerFailure), serveQuery(t, h, "127.0.0.1", "example.com.", dnswire.TypeA).Rcode)

	require.Equal(t, map[string]float64{
		queryLabels(pathPrivate, reasonShed, dnswire.RcodeServerFailure, "udp"): 1,
		queryLabels(pathPublic, reasonShed, dnswire.RcodeServerFailure, "udp"):  1,
	}, metricDelta(queriesBefore, metricValues(t, "unkey_dns_queries_total")))
	require.Equal(t, map[string]float64{"limit=queries": 1, "limit=forwards": 1}, metricDelta(shedBefore, metricValues(t, "unkey_dns_shed_total")))
}

func TestMalformedQueriesUseTheInvalidPath(t *testing.T) {
	h := newTestResolver(t, newFakeCatalog(), Config{QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1})

	request := dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA)
	request.Response = true
	require.NoError(t, request.Pack())
	w := &recordingWriter{remote: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5300}}
	before := metricValues(t, "unkey_dns_queries_total")
	h.ServeDNS(t.Context(), w, request)

	require.Equal(t, map[string]float64{queryLabels(pathInvalid, reasonMalformed, dnswire.RcodeFormatError, "udp"): 1},
		metricDelta(before, metricValues(t, "unkey_dns_queries_total")))
}

func TestFailureLogsAreSampledPerReason(t *testing.T) {
	clk := clock.NewTestClock(time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC))
	h := newTestResolver(t, newFakeCatalog(), Config{QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1, Clock: clk})
	capture := loggertest.Install(t)
	before := metricValues(t, "unkey_dns_queries_total")

	since := capture.Snapshot()
	for range 3 {
		serveQuery(t, h, "127.0.0.9", "payments.unkey.internal.", dnswire.TypeA)
	}
	serveQuery(t, h, "127.0.0.1", "ledger.unkey.internal.", dnswire.TypeA)
	records := queryFailureRecords(capture.Since(since))
	require.Len(t, records, 1, "repeated failures with one reason log once per interval; NXDOMAIN is not a resolver failure")
	attrs := loggertest.FlatAttrs(records[0])
	require.Equal(t, string(discovery.ReasonUnknownCaller), attrs["reason"])
	require.Contains(t, attrs["source"], "127.0.0.9")
	require.NotContains(t, attrs, "name")
	require.Equal(t, 3.0, metricDelta(before, metricValues(t, "unkey_dns_queries_total"))[queryLabels(pathPrivate, reason(discovery.ReasonUnknownCaller), dnswire.RcodeRefused, "udp")],
		"sampling must not drop counts")

	clk.Tick(failureLogInterval)
	since = capture.Snapshot()
	serveQuery(t, h, "127.0.0.9", "payments.unkey.internal.", dnswire.TypeA)
	require.Len(t, queryFailureRecords(capture.Since(since)), 1, "a new interval logs the reason again")
}

func TestQueryTelemetryDoesNotExposeDNSNames(t *testing.T) {
	for _, tc := range []struct {
		path string
		name string
	}{
		{path: pathPrivate, name: "customer-secret.unkey.internal."},
		{path: pathPublic, name: "customer-secret.example.com."},
	} {
		t.Run(tc.path, func(t *testing.T) {
			h := newTestResolver(t, newFakeCatalog(), Config{QueriesInFlight: 1, ForwardsInFlight: 1})
			capture := loggertest.Install(t)
			response := dnswire.NewMsg(tc.name, dnswire.TypeA)
			response.Rcode = dnswire.RcodeServerFailure
			result := outcome{
				reason: reasonUpstreamError,
				err:    fmt.Errorf("lookup %s failed", tc.name),
				caller: noCaller(),
			}

			since := capture.Snapshot()
			h.observe(tc.path, result, response, "udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1)}, time.Millisecond)

			records := queryFailureRecords(capture.Since(since))
			require.Len(t, records, 1)
			attrs := loggertest.FlatAttrs(records[0])
			require.Equal(t, string(reasonUpstreamError), attrs["reason"])
			require.NotContains(t, attrs, "name")
			require.NotContains(t, attrs, "error")
			require.NotContains(t, fmt.Sprint(attrs), "customer-secret")
			families, err := testRegistry.Gather()
			require.NoError(t, err)
			require.NotEmpty(t, families)
			require.NotContains(t, fmt.Sprint(families), "customer-secret")
		})
	}
}

type recordingWriter struct {
	dnswire.ResponseWriter
	remote  net.Addr
	written []byte
}

func (w *recordingWriter) RemoteAddr() net.Addr { return w.remote }

func (w *recordingWriter) Conn() net.Conn { return nil }

func (w *recordingWriter) Write(data []byte) (int, error) {
	w.written = append(w.written, data...)
	return len(data), nil
}

func serveQuery(t *testing.T, h *Resolver, source, name string, qtype uint16) *dnswire.Msg {
	t.Helper()
	return serveQueryOver(t, h, source, name, qtype, "udp")
}

func serveQueryOver(t *testing.T, h *Resolver, source, name string, qtype uint16, transport string) *dnswire.Msg {
	t.Helper()
	request := dnswire.NewMsg(name, qtype)
	require.NoError(t, request.Pack())
	var remote net.Addr = &net.UDPAddr{IP: net.ParseIP(source), Port: 5300}
	if transport == "tcp" {
		remote = &net.TCPAddr{IP: net.ParseIP(source), Port: 5300}
	}
	w := &recordingWriter{remote: remote}
	h.ServeDNS(t.Context(), w, request)

	require.Greater(t, len(w.written), 2, "ServeDNS wrote no length-prefixed response")
	response := new(dnswire.Msg)
	response.Data = w.written[2:]
	require.NoError(t, response.Unpack())
	return response
}

func startUpstream(t *testing.T, reply func(*dnswire.Msg) *dnswire.Msg) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	server := &dnswire.Server{Listener: listener, Handler: dnswire.HandlerFunc(func(ctx context.Context, w dnswire.ResponseWriter, query *dnswire.Msg) {
		response := reply(query)
		if response == nil {
			<-ctx.Done()
			return
		}
		if err := writeDNSResponse(w, response); err != nil {
			t.Errorf("write upstream response: %v", err)
		}
	})}
	started := make(chan struct{})
	server.NotifyStartedFunc = func(context.Context) { close(started) }
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	<-started
	t.Cleanup(func() {
		server.Shutdown(t.Context())
		require.NoError(t, <-done)
	})
	return listener.Addr().String()
}

func unusedTCPAddress(t testing.TB) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return address
}

func writeDNSResponse(w dnswire.ResponseWriter, response *dnswire.Msg) error {
	if err := response.Pack(); err != nil {
		return err
	}
	_, err := io.Copy(w, response)
	return err
}

func answerAddresses(response *dnswire.Msg) []string {
	addresses := make([]string, 0, len(response.Answer))
	for _, answer := range response.Answer {
		addresses = append(addresses, answer.(*dnswire.A).A.Addr.String())
	}
	return addresses
}

func queryLabels(path string, why reason, rcode uint16, transport string) string {
	return "path=" + path + ",rcode=" + dnsutil.RcodeToString(rcode) + ",reason=" + string(why) + ",transport=" + transport
}

func metricValues(t *testing.T, name string) map[string]float64 {
	t.Helper()
	families, err := testRegistry.Gather()
	require.NoError(t, err)
	values := map[string]float64{}
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.GetMetric() {
			pairs := make([]string, 0, len(metric.GetLabel()))
			for _, label := range metric.GetLabel() {
				pairs = append(pairs, label.GetName()+"="+label.GetValue())
			}
			values[strings.Join(pairs, ",")] = metric.GetCounter().GetValue() + metric.GetGauge().GetValue()
		}
	}
	return values
}

func metricDelta(before, after map[string]float64) map[string]float64 {
	changed := map[string]float64{}
	for series, value := range after {
		if value != before[series] {
			changed[series] = value - before[series]
		}
	}
	return changed
}

func gaugeValue(t *testing.T, name string) float64 {
	t.Helper()
	return metricValues(t, name)[""]
}

func latencySamples(t *testing.T, path string) uint64 {
	t.Helper()
	families, err := testRegistry.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != "unkey_dns_query_duration_seconds" {
			continue
		}
		for _, metric := range family.GetMetric() {
			for _, label := range metric.GetLabel() {
				if label.GetName() == "path" && label.GetValue() == path {
					return metric.GetHistogram().GetSampleCount()
				}
			}
		}
	}
	return 0
}

func queryFailureRecords(records []slog.Record) []slog.Record {
	var matched []slog.Record
	for _, record := range records {
		if record.Message == "DNS query failed" {
			matched = append(matched, record)
		}
	}
	return matched
}
