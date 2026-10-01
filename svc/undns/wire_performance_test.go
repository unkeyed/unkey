package undns

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kuberuntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
)

type connectionLoad struct {
	connections, endpoints, readers int
	churn                           bool
}

func BenchmarkDNSWire(b *testing.B) {
	featuretesting.SetFeatureDuringTest(b, features.WatchListClient, false)
	for _, test := range []struct {
		transport string
		load      connectionLoad
	}{
		{"udp", connectionLoad{connections: 1, endpoints: 2, readers: 8}},
		{"udp", connectionLoad{connections: 1, endpoints: 8, readers: 8}},
		{"udp", connectionLoad{connections: 1, endpoints: 32, readers: 8}},
		{"udp", connectionLoad{connections: 1, endpoints: 8, readers: 64, churn: true}},
		{"udp", connectionLoad{connections: 1000, endpoints: 32, readers: 64}},
		{"udp", connectionLoad{connections: 1000, endpoints: 8, readers: 64, churn: true}},
		{"tcp", connectionLoad{connections: 1000, endpoints: 32, readers: 64}},
		{"tcp", connectionLoad{connections: 1, endpoints: 8, readers: 64, churn: true}},
		{"fallback", connectionLoad{connections: 1000, endpoints: 32, readers: 64}},
	} {
		name := fmt.Sprintf("%s/connections=%d/endpoints=%d/readers=%d/churn=%t", test.transport, test.load.connections, test.load.endpoints, test.load.readers, test.load.churn)
		b.Run(name, func(b *testing.B) { benchmarkDNSWire(b, test.transport, test.load) })
	}
}

func benchmarkDNSWire(b *testing.B, transport string, load connectionLoad) {
	b.Helper()
	c, kube, cfg, names, source := startWireBenchmark(b, load)
	probe := &wireBenchmarkClient{address: cfg.ListenAddress, transport: transport}
	b.Cleanup(func() { require.NoError(b, probe.close()) })
	for i := range 100 {
		name := names[i%len(names)]
		response, _, err := probe.exchange(b.Context(), name)
		require.NoError(b, err)
		require.NoError(b, checkWireAnswer(response, name, load.endpoints))
	}

	stop, done := make(chan struct{}), make(chan struct{})
	var changes []int64
	var updateErr error
	go func() {
		defer close(done)
		if !load.churn {
			return
		}
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()
		for sequence := 1; ; sequence++ {
			select {
			case <-stop:
				return
			case <-ticker.C:
			}
			index := (sequence - 1) % len(source)
			changed := source[index].DeepCopy()
			previous := netip.MustParseAddr(changed.Endpoints[0].Addresses[0])
			next := netip.AddrFrom4([4]byte{172, 16, byte(sequence >> 8), byte(sequence)})
			changed.Endpoints[0].Addresses = []string{next.String()}
			changed.ResourceVersion = strconv.Itoa(sequence)
			start := time.Now()
			if _, updateErr = kube.DiscoveryV1().EndpointSlices("default").Update(b.Context(), changed, metav1.UpdateOptions{}); updateErr != nil {
				return
			}
			for {
				object, exists, err := c.slices.GetStore().GetByKey("default/" + changed.Name)
				if err != nil {
					updateErr = err
					return
				}
				observed := exists && object.(*discoveryv1.EndpointSlice).ResourceVersion == changed.ResourceVersion
				response, _, err := probe.exchange(b.Context(), changed.Labels[discoveryv1.LabelServiceName]+".unkey.internal.")
				if err != nil {
					updateErr = err
					return
				}
				if observed && wireContains(response, previous) {
					updateErr = errors.New("DNS returned removed endpoint after informer observation")
					return
				}
				if response.Rcode == dnswire.RcodeSuccess && wireContains(response, next) && !wireContains(response, previous) {
					break
				}
				if response.Rcode != dnswire.RcodeSuccess || time.Since(start) > 5*time.Second {
					updateErr = fmt.Errorf("DNS update did not converge: rcode=%d", response.Rcode)
					return
				}
			}
			changes = append(changes, time.Since(start).Nanoseconds())
			source[index] = changed
		}
	}()

	var workerID atomic.Uint64
	var mu sync.Mutex
	var success, servfail, transportErrors, fallbacks, tcpReconnects int64
	var latencies []int64
	sampleEvery := max(1, b.N/8192)
	b.SetParallelism(max(1, load.readers/runtime.GOMAXPROCS(0)))
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		client := &wireBenchmarkClient{address: cfg.ListenAddress, transport: transport}
		defer func() {
			if err := client.close(); err != nil {
				b.Error(err)
			}
		}()
		index := int(workerID.Add(1)) * 17
		var good, failed, broken, fallbackCount int64
		var samples []int64
		for pb.Next() {
			name := names[index%len(names)]
			var start time.Time
			if index%sampleEvery == 0 {
				start = time.Now()
			}
			response, fallback, err := client.exchange(b.Context(), name)
			if fallback {
				fallbackCount++
			}
			if err != nil {
				broken++
				b.Errorf("DNS transport: %v", err)
				break
			}
			if response.Rcode == dnswire.RcodeServerFailure {
				failed++
			} else if err := checkWireAnswer(response, name, load.endpoints); err != nil {
				b.Error(err)
				break
			} else {
				good++
				if !start.IsZero() {
					samples = append(samples, time.Since(start).Nanoseconds())
				}
			}
			index++
		}
		mu.Lock()
		success += good
		servfail += failed
		transportErrors += broken
		fallbacks += fallbackCount
		tcpReconnects += client.tcpReconnects
		latencies = append(latencies, samples...)
		mu.Unlock()
	})
	b.StopTimer()
	close(stop)
	<-done
	require.NoError(b, updateErr)
	require.Zero(b, servfail, "healthy discovery updates must not cause SERVFAIL")
	b.ReportMetric(float64(success)/b.Elapsed().Seconds(), "answers/s")
	b.ReportMetric(100*float64(servfail)/float64(b.N), "SERVFAIL-%")
	b.ReportMetric(float64(servfail), "SERVFAIL-count")
	b.ReportMetric(float64(transportErrors), "transport-errors")
	b.ReportMetric(float64(tcpReconnects), "TCP-reconnects")
	b.ReportMetric(100*float64(fallbacks)/float64(b.N), "TCP-fallback-%")
	reportLatency(b, latencies, "query")
	if load.churn {
		b.ReportMetric(float64(len(changes))/b.Elapsed().Seconds(), "updates/s")
		reportLatency(b, changes, "change")
	}
	response, err := http.Get("http://" + cfg.HealthAddress + "/metrics")
	require.NoError(b, err)
	body, err := io.ReadAll(response.Body)
	require.NoError(b, errors.Join(err, response.Body.Close()))
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, "unkey_dns_queries_total{") && strings.Contains(line, `rcode="SERVFAIL"`) {
			b.Log(line)
		}
	}
}

type wireBenchmarkClient struct {
	address, transport string
	udp, tcp           net.Conn
	tcpQueries         int
	tcpReconnects      int64
}

func (c *wireBenchmarkClient) exchange(ctx context.Context, name string) (*dnswire.Msg, bool, error) {
	query := dnswire.NewMsg(name, dnswire.TypeA)
	query.UDPSize = 1232
	if c.transport == "fallback" {
		query.UDPSize = 0
	}
	network := c.transport
	if network == "fallback" {
		network = "udp"
	}
	response, err := c.send(ctx, query, network)
	if err == nil && response.Truncated && c.transport == "fallback" {
		response, err = c.send(ctx, query, "tcp")
		return response, true, err
	}
	return response, false, err
}

func (c *wireBenchmarkClient) send(ctx context.Context, query *dnswire.Msg, network string) (*dnswire.Msg, error) {
	connection := &c.udp
	if network == "tcp" {
		if c.tcpQueries == dnswire.MaxTCPQueries {
			if err := c.tcp.Close(); err != nil {
				return nil, err
			}
			c.tcp = nil
			c.tcpQueries = 0
			c.tcpReconnects++
		}
		connection = &c.tcp
	}
	if *connection == nil {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		var err error
		*connection, err = dialer.DialContext(ctx, network, c.address)
		if err != nil {
			return nil, err
		}
	}
	response, _, err := testDNSClient(2*time.Second).ExchangeWithConn(ctx, query, *connection)
	if err == nil && network == "tcp" {
		c.tcpQueries++
	}
	return response, err
}

func (c *wireBenchmarkClient) close() error {
	var errs []error
	for _, connection := range []net.Conn{c.udp, c.tcp} {
		if connection != nil {
			errs = append(errs, connection.Close())
		}
	}
	return errors.Join(errs...)
}

func checkWireAnswer(response *dnswire.Msg, name string, endpoints int) error {
	if response.Rcode != dnswire.RcodeSuccess || response.Truncated || !response.Authoritative || len(response.Question) != 1 || response.Question[0].Header().Name != name || len(response.Answer) != endpoints {
		return fmt.Errorf("DNS answer: name=%s rcode=%d truncated=%t count=%d", name, response.Rcode, response.Truncated, len(response.Answer))
	}
	for _, rr := range response.Answer {
		a, ok := rr.(*dnswire.A)
		if !ok || a.Hdr.TTL != 5 || !a.A.Addr.Is4() || !a.A.Addr.IsPrivate() {
			return errors.New("DNS answer has invalid type, TTL or address")
		}
	}
	return nil
}

func wireContains(response *dnswire.Msg, address netip.Addr) bool {
	for _, rr := range response.Answer {
		if a, ok := rr.(*dnswire.A); ok && a.A.Addr == address {
			return true
		}
	}
	return false
}

func startWireBenchmark(b *testing.B, load connectionLoad) (*catalog, *fake.Clientset, Config, []string, []*discoveryv1.EndpointSlice) {
	b.Helper()
	template := discoveryObjects(b)
	objects := []kuberuntime.Object{template[0]}
	var names []string
	var source []*discoveryv1.EndpointSlice
	for id := range load.connections {
		name := fmt.Sprintf("app-%06d", id)
		names = append(names, name+".unkey.internal.")
		connection := template[1].(*corev1.ConfigMap).DeepCopy()
		connection.Name, connection.UID = name, types.UID(name)
		connection.Data["appSlug"], connection.Data["serviceName"] = name, name
		connection.Labels[labels.LabelKeyConnectionID] = name
		service := template[2].(*corev1.Service).DeepCopy()
		service.Name, service.UID = name, types.UID(name)
		objects = append(objects, connection, service)
		for region := range min(3, load.endpoints) {
			slice := template[3].(*discoveryv1.EndpointSlice).DeepCopy()
			slice.Name = fmt.Sprintf("%s-region-%d", name, region)
			slice.Labels = ciliumImportedSliceLabels(service)
			slice.Labels["multicluster.kubernetes.io/source-cluster"] = fmt.Sprint(region)
			slice.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(service, corev1.SchemeGroupVersion.WithKind("Service"))}
			slice.Endpoints = nil
			for endpoint := load.endpoints - 1; endpoint >= 0; endpoint-- {
				if endpoint%min(3, load.endpoints) != region {
					continue
				}
				address := netip.AddrFrom4([4]byte{10, byte(id >> 8), byte(id), byte(endpoint + 1)})
				slice.Endpoints = append(slice.Endpoints, discoveryv1.Endpoint{Addresses: []string{address.String()}, Conditions: discoveryv1.EndpointConditions{Ready: new(true)}})
			}
			objects = append(objects, slice)
			source = append(source, slice)
		}
	}
	kube := fake.NewSimpleClientset(objects...)
	c, err := newCatalog(kube, 30*time.Second)
	require.NoError(b, err)
	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(b, err)
	cfg.ListenAddress, cfg.HealthAddress = unusedTCPAddress(b), unusedTCPAddress(b)
	ctx, cancel := context.WithCancel(b.Context())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, c) }()
	b.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(b, err)
		case <-time.After(5 * time.Second):
			b.Error("DNS benchmark server did not shut down")
		}
	})
	require.Eventually(b, c.ready, 5*time.Second, time.Millisecond)
	return c, kube, cfg, names, source
}

func reportLatency(b *testing.B, samples []int64, name string) {
	b.Helper()
	if len(samples) == 0 {
		return
	}
	slices.Sort(samples)
	b.ReportMetric(float64(samples[(len(samples)-1)*50/100]), name+"-p50-ns")
	b.ReportMetric(float64(samples[(len(samples)-1)*99/100]), name+"-p99-ns")
}
