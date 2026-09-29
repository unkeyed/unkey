package undns

import (
	"context"
	"log/slog"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/deploy/appbinding"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPrivateFailuresReportDistinctReasons(t *testing.T) {
	for _, tc := range []struct {
		name         string
		source       string
		qtype        uint16
		qname        string
		mutate       func(t *testing.T, c *catalog)
		rcode        uint16
		reason       reason
		bindingState reason
		bindings     int
	}{
		{name: "answer", rcode: dnswire.RcodeSuccess, reason: reasonAnswer, bindingState: stateActive},
		{name: "ipv6", qtype: dnswire.TypeAAAA, rcode: dnswire.RcodeSuccess, reason: reasonNoData, bindingState: stateActive},
		{name: "unknown alias", qname: "ledger.unkey.internal.", rcode: dnswire.RcodeNameError, reason: reasonUnknownName, bindingState: stateActive},
		{name: "unknown source", source: "127.0.0.9", rcode: dnswire.RcodeRefused, reason: reasonUnknownCaller, bindingState: stateActive},
		{
			name: "reused source IP",
			mutate: func(t *testing.T, c *catalog) {
				other := callerPod("127.0.0.1", "production")
				other.Name, other.UID = "other-caller", "other-caller"
				require.NoError(t, c.pods.GetStore().Add(other))
			},
			rcode: dnswire.RcodeRefused, reason: reasonAmbiguousCaller, bindingState: stateActive,
		},
		{
			name: "caller without deployment",
			mutate: func(t *testing.T, c *catalog) {
				pod := callerPod("127.0.0.1", "production")
				delete(pod.Labels, labels.LabelKeyDeploymentID)
				require.NoError(t, c.pods.GetStore().Update(pod))
			},
			rcode: dnswire.RcodeRefused, reason: reasonIneligibleCaller, bindingState: stateActive,
		},
		{
			name:   "pod watch stale",
			mutate: func(_ *testing.T, c *catalog) { c.pods.lastContact.Store(0) },
			rcode:  dnswire.RcodeServerFailure, reason: reasonIdentityUnavailable, bindingState: stateActive,
		},
		{
			name:   "endpoint watch stale",
			mutate: func(_ *testing.T, c *catalog) { c.slices.lastContact.Store(0) },
			rcode:  dnswire.RcodeServerFailure, reason: reasonDiscoveryNotReady, bindingState: stateActive,
		},
		{
			name: "no selected target",
			mutate: func(t *testing.T, c *catalog) {
				updateStoredBinding(t, c, map[string]string{"deploymentId": "", "serviceName": "", "revision": "2"})
			},
			rcode: dnswire.RcodeServerFailure, reason: reasonBindingUnresolved, bindingState: reasonBindingUnresolved,
		},
		{
			name:   "corrupt revision",
			mutate: func(t *testing.T, c *catalog) { updateStoredBinding(t, c, map[string]string{"revision": "latest"}) },
			rcode:  dnswire.RcodeServerFailure, reason: reasonBindingInvalid, bindingState: reasonBindingInvalid,
		},
		{
			name: "duplicate binding",
			mutate: func(t *testing.T, c *catalog) {
				duplicate := storedBinding(t, c).DeepCopy()
				duplicate.Name, duplicate.UID = "binding-copy", "binding-copy"
				require.NoError(t, c.bindings.GetStore().Add(duplicate))
			},
			rcode: dnswire.RcodeServerFailure, reason: reasonBindingAmbiguous, bindingState: reasonBindingAmbiguous, bindings: 2,
		},
		{
			name: "discovery service deleted",
			mutate: func(t *testing.T, c *catalog) {
				require.NoError(t, c.services.GetStore().Delete(storedService(t, c)))
			},
			rcode: dnswire.RcodeServerFailure, reason: reasonServiceMissing, bindingState: reasonServiceMissing,
		},
		{
			name: "discovery service publishes unready pods",
			mutate: func(t *testing.T, c *catalog) {
				service := storedService(t, c).DeepCopy()
				service.Spec.PublishNotReadyAddresses = true
				require.NoError(t, c.services.GetStore().Update(service))
			},
			rcode: dnswire.RcodeServerFailure, reason: reasonServiceRejected, bindingState: reasonServiceRejected,
		},
		{
			name: "discovery service retired",
			mutate: func(t *testing.T, c *catalog) {
				service := storedService(t, c).DeepCopy()
				service.Annotations = map[string]string{appbinding.RetireAfterAnnotation: time.Now().Add(-time.Minute).Format(time.RFC3339Nano)}
				require.NoError(t, c.services.GetStore().Update(service))
			},
			rcode: dnswire.RcodeServerFailure, reason: reasonServiceRetired, bindingState: reasonServiceRetired,
		},
		{
			name: "target lost its endpoints",
			mutate: func(t *testing.T, c *catalog) {
				object, exists, err := c.slices.GetStore().GetByKey("default/imported")
				require.NoError(t, err)
				require.True(t, exists)
				slice := object.(*discoveryv1.EndpointSlice).DeepCopy()
				for i := range slice.Endpoints {
					slice.Endpoints[i].Conditions.Ready = new(false)
				}
				require.NoError(t, c.slices.GetStore().Update(slice))
			},
			rcode: dnswire.RcodeServerFailure, reason: reasonNoReadyEndpoints, bindingState: reasonNoReadyEndpoints,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := startCatalog(t)
			registry := prometheus.NewRegistry()
			h := newHandler(c, Config{TTLSeconds: 5, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1}, registry)
			if tc.mutate != nil {
				tc.mutate(t, c)
			}
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

			response := serveQuery(t, h, source, qname, qtype)
			require.Equal(t, dnsutil.RcodeToString(tc.rcode), dnsutil.RcodeToString(response.Rcode), "rcode for %s", tc.name)
			require.Equal(t, map[string]float64{queryLabels(pathPrivate, tc.reason, tc.rcode, "udp"): 1}, metricValues(t, registry, "unkey_dns_queries_total"),
				"unkey_dns_queries_total after %s", tc.name)

			c.activate()
			want := map[string]map[reason]int{kindBinding: {tc.bindingState: max(tc.bindings, 1)}, kindReplica: {}}
			require.Equal(t, want, c.bindingCounts(), "binding states after %s", tc.name)
		})
	}
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
			c := startCatalog(t)
			registry := prometheus.NewRegistry()
			h := newHandler(c, Config{Upstream: tc.upstream, ForwardTimeout: 200 * time.Millisecond, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1}, registry)

			response := serveQueryOver(t, h, "127.0.0.1", tc.qname, dnswire.TypeA, tc.transport)
			require.Equal(t, dnsutil.RcodeToString(tc.rcode), dnsutil.RcodeToString(response.Rcode), "rcode for %s", tc.name)
			require.Equal(t, map[string]float64{queryLabels(pathPublic, tc.reason, tc.rcode, tc.transport): 1}, metricValues(t, registry, "unkey_dns_queries_total"),
				"unkey_dns_queries_total after %s", tc.name)
			require.Equal(t, uint64(1), latencySamples(t, registry, pathPublic), "public latency samples after %s", tc.name)
		})
	}
}

func TestShedQueriesReportLimitAndPath(t *testing.T) {
	c := startCatalog(t)
	registry := prometheus.NewRegistry()
	h := newHandler(c, Config{Upstream: "127.0.0.1:1", ForwardTimeout: time.Second, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1}, registry)

	h.slots <- struct{}{}
	require.Equal(t, uint16(dnswire.RcodeServerFailure), serveQuery(t, h, "127.0.0.1", "payments.unkey.internal.", dnswire.TypeA).Rcode)
	<-h.slots
	h.forwards <- struct{}{}
	require.Equal(t, uint16(dnswire.RcodeServerFailure), serveQuery(t, h, "127.0.0.1", "example.com.", dnswire.TypeA).Rcode)

	require.Equal(t, map[string]float64{
		queryLabels(pathPrivate, reasonShed, dnswire.RcodeServerFailure, "udp"): 1,
		queryLabels(pathPublic, reasonShed, dnswire.RcodeServerFailure, "udp"):  1,
	}, metricValues(t, registry, "unkey_dns_queries_total"))
	require.Equal(t, map[string]float64{"limit=queries": 1, "limit=forwards": 1}, metricValues(t, registry, "unkey_dns_shed_total"))
}

func TestMalformedQueriesUseTheInvalidPath(t *testing.T) {
	c := startCatalog(t)
	registry := prometheus.NewRegistry()
	h := newHandler(c, Config{QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1}, registry)

	request := dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA)
	request.Response = true
	require.NoError(t, request.Pack())
	w := &recordingWriter{remote: &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 5300}}
	h.ServeDNS(t.Context(), w, request)

	require.Equal(t, map[string]float64{queryLabels(pathInvalid, reasonMalformed, dnswire.RcodeFormatError, "udp"): 1},
		metricValues(t, registry, "unkey_dns_queries_total"))
}

func TestIdleResolverReportsZeroBindingStates(t *testing.T) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(newBindingCollector(catalogForTest()))
	families, err := registry.Gather()
	require.NoError(t, err)
	require.Len(t, families, 1)
	require.Len(t, families[0].GetMetric(), 2*len(bindingStates))
	for _, metric := range families[0].GetMetric() {
		require.Zero(t, metric.GetGauge().GetValue(), "idle binding state %v", metric.GetLabel())
	}
}

func TestReplicaBindingsReportTheirOwnKind(t *testing.T) {
	c := seededCatalog(t)
	replica := storedBinding(t, c).DeepCopy()
	replica.Name, replica.UID = "replica", "replica"
	replica.Labels[labels.LabelKeyCallerDeploymentID] = "deployment-a"
	replica.Labels[labels.LabelKeyBindingID] = "self-deployment-a"
	replica.Data["appSlug"] = "caller-app"
	require.NoError(t, c.bindings.GetStore().Add(replica))

	c.activate()
	counts := c.bindingCounts()
	require.Equal(t, 1, counts[kindBinding][stateActive], "directed bindings: %v", counts)
	require.Equal(t, 1, counts[kindReplica][stateActive], "replica bindings: %v", counts)
}

func TestPendingRevisionReportsTheServedTarget(t *testing.T) {
	c := seededCatalog(t)
	capture := loggertest.Install(t)
	c.activate()
	service := storedService(t, c).DeepCopy()
	service.Name, service.UID = "service-b", "service-b-uid"
	service.Labels[labels.LabelKeyDeploymentID] = "deployment-b"
	require.NoError(t, c.services.GetStore().Add(service))
	updateStoredBinding(t, c, map[string]string{"deploymentId": "deployment-b", "serviceName": "service-b", "revision": "2"})

	since := capture.Snapshot()
	c.activate()
	require.Equal(t, 1, c.bindingCounts()[kindBinding][statePendingRevision])
	record := findRecord(t, capture.Since(since), "private DNS binding serves its previous target until the new target has ready endpoints")
	attrs := loggertest.FlatAttrs(record)
	require.Equal(t, "deployment-a", attrs["serving_deployment_id"])
	require.Equal(t, "deployment-b", attrs["target_deployment_id"])
}

func TestBindingStateChangesAreLoggedOnce(t *testing.T) {
	c := seededCatalog(t)
	capture := loggertest.Install(t)
	since := capture.Snapshot()
	c.activate()
	require.Empty(t, bindingRecords(capture.Since(since)), "healthy bindings seen on startup must not be logged")

	service := storedService(t, c)
	require.NoError(t, c.services.GetStore().Delete(service))
	since = capture.Snapshot()
	c.activate()
	c.activate()
	records := bindingRecords(capture.Since(since))
	require.Len(t, records, 1, "a failing binding logs once per state change")
	require.Equal(t, slog.LevelWarn, records[0].Level)
	attrs := loggertest.FlatAttrs(records[0])
	require.Equal(t, string(reasonServiceMissing), attrs["state"])
	require.Equal(t, string(stateActive), attrs["previous_state"])
	require.Equal(t, "binding-id", attrs["binding_id"])
	require.Equal(t, "caller-deployment-a", attrs["caller_deployment_id"])
	require.Equal(t, "deployment-a", attrs["target_deployment_id"])
	require.Equal(t, "payments", attrs["alias"])

	require.NoError(t, c.services.GetStore().Add(service))
	since = capture.Snapshot()
	c.activate()
	c.activate()
	records = bindingRecords(capture.Since(since))
	require.Len(t, records, 1, "a recovered binding logs once")
	require.Equal(t, slog.LevelInfo, records[0].Level)
	attrs = loggertest.FlatAttrs(records[0])
	require.Equal(t, string(stateActive), attrs["state"])
	require.Equal(t, string(reasonServiceMissing), attrs["previous_state"])
}

func TestFailureLogsAreSampledPerReason(t *testing.T) {
	c := startCatalog(t)
	registry := prometheus.NewRegistry()
	h := newHandler(c, Config{QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1, ForwardsUnidentified: 1}, registry)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	h.failures.now = func() time.Time { return now }
	capture := loggertest.Install(t)

	since := capture.Snapshot()
	for range 3 {
		serveQuery(t, h, "127.0.0.9", "payments.unkey.internal.", dnswire.TypeA)
	}
	serveQuery(t, h, "127.0.0.1", "ledger.unkey.internal.", dnswire.TypeA)
	records := queryFailureRecords(capture.Since(since))
	require.Len(t, records, 1, "repeated failures with one reason log once per interval; NXDOMAIN is not a resolver failure")
	attrs := loggertest.FlatAttrs(records[0])
	require.Equal(t, string(reasonUnknownCaller), attrs["reason"])
	require.Contains(t, attrs["source"], "127.0.0.9")
	require.Equal(t, "payments.unkey.internal.", attrs["name"])
	require.Equal(t, 3.0, metricValues(t, registry, "unkey_dns_queries_total")[queryLabels(pathPrivate, reasonUnknownCaller, dnswire.RcodeRefused, "udp")],
		"sampling must not drop counts")

	now = now.Add(failureLogInterval)
	since = capture.Snapshot()
	serveQuery(t, h, "127.0.0.9", "payments.unkey.internal.", dnswire.TypeA)
	require.Len(t, queryFailureRecords(capture.Since(since)), 1, "a new interval logs the reason again")
}

func startCatalog(t *testing.T) *catalog {
	t.Helper()
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	c, err := newCatalog(fake.NewSimpleClientset(discoveryObjects(t)...), 30*time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	var running sync.WaitGroup
	running.Go(func() { require.NoError(t, c.run(ctx)) })
	t.Cleanup(func() {
		cancel()
		running.Wait()
	})
	require.Eventually(t, c.ready, 5*time.Second, 10*time.Millisecond)
	return c
}

func seededCatalog(t *testing.T) *catalog {
	t.Helper()
	c := catalogForTest()
	for _, object := range discoveryObjects(t) {
		switch typed := object.(type) {
		case *corev1.Pod:
			require.NoError(t, c.pods.GetStore().Add(typed))
		case *corev1.ConfigMap:
			require.NoError(t, c.bindings.GetStore().Add(typed))
		case *corev1.Service:
			require.NoError(t, c.services.GetStore().Add(typed))
		case *discoveryv1.EndpointSlice:
			require.NoError(t, c.slices.GetStore().Add(typed))
		default:
			t.Fatalf("unexpected discovery object %T", object)
		}
	}
	return c
}

func storedBinding(t *testing.T, c *catalog) *corev1.ConfigMap {
	t.Helper()
	object, exists, err := c.bindings.GetStore().GetByKey("default/binding")
	require.NoError(t, err)
	require.True(t, exists)
	return object.(*corev1.ConfigMap)
}

func updateStoredBinding(t *testing.T, c *catalog, data map[string]string) {
	t.Helper()
	binding := storedBinding(t, c).DeepCopy()
	for key, value := range data {
		binding.Data[key] = value
	}
	require.NoError(t, c.bindings.GetStore().Update(binding))
}

func storedService(t *testing.T, c *catalog) *corev1.Service {
	t.Helper()
	object, exists, err := c.services.GetStore().GetByKey("default/service-a")
	require.NoError(t, err)
	require.True(t, exists)
	return object.(*corev1.Service)
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

func serveQuery(t *testing.T, h *handler, source, name string, qtype uint16) *dnswire.Msg {
	t.Helper()
	return serveQueryOver(t, h, source, name, qtype, "udp")
}

func serveQueryOver(t *testing.T, h *handler, source, name string, qtype uint16, transport string) *dnswire.Msg {
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

func queryLabels(path string, why reason, rcode uint16, transport string) string {
	return "path=" + path + ",rcode=" + dnsutil.RcodeToString(rcode) + ",reason=" + string(why) + ",transport=" + transport
}

func metricValues(t *testing.T, registry *prometheus.Registry, name string) map[string]float64 {
	t.Helper()
	families, err := registry.Gather()
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

func latencySamples(t *testing.T, registry *prometheus.Registry, path string) uint64 {
	t.Helper()
	families, err := registry.Gather()
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

func findRecord(t *testing.T, records []slog.Record, message string) slog.Record {
	t.Helper()
	for _, record := range records {
		if record.Message == message {
			return record
		}
	}
	t.Fatalf("no log record %q in %d records", message, len(records))
	return slog.Record{}
}

func bindingRecords(records []slog.Record) []slog.Record {
	var matched []slog.Record
	for _, record := range records {
		switch record.Message {
		case "private DNS binding cannot be served",
			"private DNS binding serves its target",
			"private DNS binding serves its previous target until the new target has ready endpoints":
			matched = append(matched, record)
		}
	}
	return matched
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
