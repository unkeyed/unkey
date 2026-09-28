package undns

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net"
	"net/netip"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"codeberg.org/miekg/dns/dnsutil"
	"codeberg.org/miekg/dns/rdata"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
	cachetesting "k8s.io/client-go/tools/cache/testing"
)

func TestServerUDPTruncationTCPAndCallerDeploymentIsolation(t *testing.T) {
	// The fake API does not send the initial-events bookmark required by WatchList.
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	client := fake.NewSimpleClientset(discoveryObjects(t)...)
	c, err := newCatalog(client, 30*time.Second)
	require.NoError(t, err)
	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(t, err)
	cfg.ListenAddress = unusedTCPAddress(t)
	cfg.HealthAddress = unusedTCPAddress(t)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, c) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("DNS server did not shut down")
		}
	})
	require.Eventually(t, c.ready, 5*time.Second, 10*time.Millisecond)

	question := dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA)
	dns := testDNSClient(time.Second)
	response, _, err := dns.Exchange(t.Context(), question, "udp", cfg.ListenAddress)
	require.NoError(t, err)
	require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
	require.True(t, response.Truncated)
	require.Len(t, response.Answer, 29)

	response, _, err = dns.Exchange(t.Context(), question, "tcp", cfg.ListenAddress)
	require.NoError(t, err)
	require.False(t, response.Truncated)
	require.Len(t, response.Answer, 40)
	require.Contains(t, answerAddresses(response), "10.0.0.1")
	require.Equal(t, uint32(5), response.Answer[0].Header().TTL)

	for _, tc := range []struct {
		kind, deployment string
		rcode            uint16
		answers          int
	}{
		{"production", "caller-deployment-canary", dnswire.RcodeNameError, 0},
		{"production", "", dnswire.RcodeRefused, 0},
		{"preview", "caller-deployment-a", dnswire.RcodeSuccess, 40},
		{"preview", "", dnswire.RcodeRefused, 0},
	} {
		pod := callerPod("127.0.0.1", tc.kind)
		pod.Labels[labels.LabelKeyDeploymentID] = tc.deployment
		_, err = client.CoreV1().Pods("default").Update(ctx, pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			caller, ok := c.identify(netip.MustParseAddr("127.0.0.1"))
			return ok == (tc.deployment != "") && (!ok || caller.kind == tc.kind && caller.deployment == tc.deployment)
		}, time.Second, 10*time.Millisecond)
		response, _, err = dns.Exchange(t.Context(), question, "tcp", cfg.ListenAddress)
		require.NoError(t, err)
		require.Equal(t, tc.rcode, response.Rcode)
		require.Len(t, response.Answer, tc.answers)
	}
}

func TestColdStartServesPublishedBindingWhileReplacementIsStaged(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	objects := discoveryObjects(t)
	binding := objects[1].(*corev1.ConfigMap)
	staged := objects[2].(*corev1.Service).DeepCopy()
	staged.Name, staged.UID = "service-b", "service-b-uid"
	staged.Labels[labels.LabelKeyDeploymentID] = "deployment-b"
	remote := objects[3].(*discoveryv1.EndpointSlice).DeepCopy()
	remote.Name = "imported-b"
	remote.Labels[discoveryv1.LabelServiceName] = staged.Name
	remote.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(staged, corev1.SchemeGroupVersion.WithKind("Service"))}
	remote.Endpoints = nil
	unavailable := binding.DeepCopy()
	unavailable.Name, unavailable.UID = "unavailable", "unavailable"
	unavailable.Data["appSlug"] = "unavailable"
	unavailable.Data["serviceName"] = "missing"
	client := fake.NewSimpleClientset(append(objects, staged, remote, unavailable)...)

	for _, published := range []bool{false, true} {
		t.Run(fmt.Sprintf("published-b=%t", published), func(t *testing.T) {
			if published {
				ready := true
				remote.Endpoints = []discoveryv1.Endpoint{{Addresses: []string{"10.1.0.22"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}}}
				_, err := client.DiscoveryV1().EndpointSlices("default").Update(t.Context(), remote, metav1.UpdateOptions{})
				require.NoError(t, err)
				binding.Data["deploymentId"], binding.Data["serviceName"], binding.Data["revision"] = "deployment-b", "service-b", "2"
				_, err = client.CoreV1().ConfigMaps("default").Update(t.Context(), binding, metav1.UpdateOptions{})
				require.NoError(t, err)
			}
			c, err := newCatalog(client, 30*time.Second)
			require.NoError(t, err)
			cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
			require.NoError(t, err)
			cfg.ListenAddress, cfg.HealthAddress = unusedTCPAddress(t), unusedTCPAddress(t)
			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() { done <- serve(ctx, cfg, c) }()
			t.Cleanup(func() { cancel(); require.NoError(t, <-done) })
			require.Eventually(t, c.ready, 5*time.Second, 10*time.Millisecond)
			for _, transport := range []string{"udp", "tcp"} {
				question := dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA)
				question.UDPSize = 4096
				dns := testDNSClient(time.Second)
				response, _, err := dns.Exchange(t.Context(), question, transport, cfg.ListenAddress)
				require.NoError(t, err)
				require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
				if published {
					require.Len(t, response.Answer, 1)
					require.Equal(t, "10.1.0.22", response.Answer[0].(*dnswire.A).A.Addr.String())
				} else {
					require.Len(t, response.Answer, 40)
					require.Contains(t, answerAddresses(response), "10.0.0.1")
				}
				question = dnswire.NewMsg("unavailable.unkey.internal.", dnswire.TypeA)
				response, _, err = dns.Exchange(t.Context(), question, transport, cfg.ListenAddress)
				require.NoError(t, err)
				require.Equal(t, uint16(dnswire.RcodeServerFailure), response.Rcode)
				require.True(t, c.ready())
			}
		})
	}
}

func TestForwardRetriesTruncatedUDPOverTCPAndStripsClientOptions(t *testing.T) {
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

	h := newHandler(nil, Config{Upstream: tcp.Addr().String(), ForwardTimeout: time.Second, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1}, prometheus.NewRegistry())
	request := dnswire.NewMsg("example.com.", dnswire.TypeA)
	request.UDPSize = 4096
	request.Pseudo = append(request.Pseudo, &dnswire.SUBNET{Family: 1, Netmask: 32, Address: netip.MustParseAddr("10.7.0.1")})
	response := h.forward(t.Context(), request, "udp", "workspace-a")
	require.Equal(t, request.ID, response.ID)
	require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
	require.Len(t, response.Answer, 1)
	require.Equal(t, "192.0.2.7", response.Answer[0].(*dnswire.A).A.Addr.String())
	for range 2 {
		require.Empty(t, (<-queries).Pseudo)
		require.NoError(t, <-writeErrors)
	}
}

// TestForwardStopsWhenServerShutsDown guarantees that shutdown cancels
// in-flight upstream forwards instead of holding their forward slots until
// forward_timeout expires.
func TestForwardStopsWhenServerShutsDown(t *testing.T) {
	upstream, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, upstream.Close()) })
	h := newHandler(nil, Config{Upstream: upstream.LocalAddr().String(), ForwardTimeout: time.Minute, QueriesInFlight: 1, ForwardsInFlight: 1, ForwardsPerWorkspace: 1}, prometheus.NewRegistry())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	started := time.Now()
	response := h.forward(ctx, dnswire.NewMsg("example.com.", dnswire.TypeA), "udp", "workspace-a")
	require.Equal(t, uint16(dnswire.RcodeServerFailure), response.Rcode)
	require.Less(t, time.Since(started), 10*time.Second, "forward ignored server shutdown")
	require.Empty(t, h.workspaceForwards, "forward kept its workspace slot after returning")
}

func TestServerAnswersCallerWhoseIPRemainsOnEvictedPod(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	evicted := callerPod("127.0.0.1", "production")
	evicted.Name, evicted.UID = "evicted", "evicted"
	evicted.Status.Phase, evicted.Status.Reason = corev1.PodFailed, "Evicted"
	client := fake.NewSimpleClientset(append(discoveryObjects(t), evicted)...)
	c, err := newCatalog(client, 30*time.Second)
	require.NoError(t, err)

	upstreamListener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	upstream := &dnswire.Server{Listener: upstreamListener, Handler: dnswire.HandlerFunc(func(_ context.Context, w dnswire.ResponseWriter, query *dnswire.Msg) {
		response := new(dnswire.Msg)
		dnsutil.SetReply(response, query)
		response.Answer = []dnswire.RR{&dnswire.A{
			Hdr: dnswire.Header{Name: query.Question[0].Header().Name, Class: dnswire.ClassINET, TTL: 23},
			A:   rdata.A{Addr: netip.MustParseAddr("192.0.2.7")},
		}}
		if err := writeDNSResponse(w, response); err != nil {
			t.Errorf("write upstream response: %v", err)
		}
	})}
	started := make(chan struct{})
	upstream.NotifyStartedFunc = func(context.Context) { close(started) }
	upstreamDone := make(chan error, 1)
	go func() { upstreamDone <- upstream.ListenAndServe() }()
	<-started
	t.Cleanup(func() {
		upstream.Shutdown(t.Context())
		require.NoError(t, <-upstreamDone)
	})

	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(t, err)
	cfg.Upstream = upstreamListener.Addr().String()
	cfg.ListenAddress = unusedTCPAddress(t)
	cfg.HealthAddress = unusedTCPAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- serve(ctx, cfg, c) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(5 * time.Second):
			t.Error("DNS server did not shut down")
		}
	})
	require.Eventually(t, c.ready, 5*time.Second, 10*time.Millisecond)

	dns := testDNSClient(time.Second)
	for _, tc := range []struct {
		name, answer string
		answers      int
	}{
		{"payments.unkey.internal.", "10.0.0.1", 40},
		{"example.com.", "192.0.2.7", 1},
	} {
		question := dnswire.NewMsg(tc.name, dnswire.TypeA)
		response, _, err := dns.Exchange(t.Context(), question, "tcp", cfg.ListenAddress)
		require.NoError(t, err)
		require.Equal(t, "NOERROR", dnswire.RcodeToString[response.Rcode], "A %s from 127.0.0.1", tc.name)
		require.Len(t, response.Answer, tc.answers)
		require.Contains(t, answerAddresses(response), tc.answer)
	}
}

func TestIdentifyRejectsUnknownAndAcceptsPreview(t *testing.T) {
	c := catalogForTest()
	_, ok := c.identify(netipAddress("10.0.0.99"))
	require.False(t, ok)

	pod := callerPod("10.0.0.99", "preview")
	require.NoError(t, c.pods.GetStore().Add(pod))
	identity, ok := c.identify(netipAddress("10.0.0.99"))
	require.True(t, ok)
	require.Equal(t, "preview", identity.kind)

	pod.DeletionTimestamp = &metav1.Time{Time: time.Now()}
	require.NoError(t, c.pods.GetStore().Update(pod))
	_, ok = c.identify(netipAddress("10.0.0.99"))
	require.False(t, ok)
}

func TestTrackedInformerFailsClosedWhenStale(t *testing.T) {
	source := cachetesting.NewFakeControllerSource()
	informer := &trackedInformer{
		SharedIndexInformer: cache.NewSharedIndexInformer(source, &corev1.Pod{}, 0, nil),
		timeout:             time.Second,
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go informer.RunWithContext(ctx)
	require.Eventually(t, informer.HasSynced, 5*time.Second, 10*time.Millisecond)
	informer.lastContact.Store(time.Now().UnixNano())
	require.True(t, informer.healthy())
	informer.lastContact.Store(time.Now().Add(-3 * time.Second).UnixNano())
	require.False(t, informer.healthy())
}

func discoveryObjects(t *testing.T) []runtime.Object {
	t.Helper()
	pod := callerPod("127.0.0.1", "production")
	binding := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "binding", Namespace: "default", UID: "binding", Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: bindingComponent,
			labels.LabelKeyWorkspaceID: "workspace-a", labels.LabelKeyProjectID: "project-a",
			labels.LabelKeyAppID: "app-a", environmentKindLabel: "production",
			labels.LabelKeyCallerDeploymentID: "caller-deployment-a", labels.LabelKeyBindingID: "binding-id",
		}},
		Data: map[string]string{"appSlug": "payments", "deploymentId": "deployment-a", "serviceName": "service-a", "revision": "1"},
	}
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "service-a", Namespace: "default", UID: "service-uid", Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: bindingComponent, labels.LabelKeyWorkspaceID: "workspace-a",
			labels.LabelKeyProjectID: "project-a", labels.LabelKeyAppID: "app-a",
			labels.LabelKeyDeploymentID: "deployment-a",
		}},
		Spec: corev1.ServiceSpec{ClusterIP: corev1.ClusterIPNone},
	}
	ready := true
	controller := true
	endpoints := make([]discoveryv1.Endpoint, 0, 40)
	for i := 1; i <= 40; i++ {
		endpoints = append(endpoints, discoveryv1.Endpoint{Addresses: []string{fmt.Sprintf("10.0.0.%d", i)}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}})
	}
	slice := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "imported", Namespace: "default",
			Labels:          ciliumImportedSliceLabels(service),
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "Service", Name: service.Name, UID: service.UID, Controller: &controller}},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   endpoints,
	}
	return []runtime.Object{pod, binding, service, slice}
}

// ciliumImportedSliceLabels returns the labels Cilium ClusterMesh puts on an
// EndpointSlice it imports for a global Service: the Service's own labels
// plus its reserved EndpointSlice labels.
func ciliumImportedSliceLabels(service *corev1.Service) map[string]string {
	sliceLabels := maps.Clone(service.Labels)
	sliceLabels[discoveryv1.LabelServiceName] = service.Name
	sliceLabels[discoveryv1.LabelManagedBy] = "endpointslice-mesh-controller.cilium.io"
	sliceLabels["multicluster.kubernetes.io/source-cluster"] = "remote"
	return sliceLabels
}

func callerPod(address, kind string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "caller", Namespace: "default", UID: types.UID("caller-" + address), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: "deployment",
			labels.LabelKeyWorkspaceID: "workspace-a", labels.LabelKeyProjectID: "project-a",
			labels.LabelKeyAppID: "caller-app", labels.LabelKeyEnvironmentID: "environment-a",
			labels.LabelKeyDeploymentID: "caller-deployment-a", environmentKindLabel: kind,
		}},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, PodIP: address, PodIPs: []corev1.PodIP{{IP: address}}},
	}
}

func unusedTCPAddress(t *testing.T) string {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	address := listener.Addr().String()
	require.NoError(t, listener.Close())
	return address
}

func netipAddress(raw string) netip.Addr {
	return netip.MustParseAddr(raw)
}

func testDNSClient(timeout time.Duration) *dnswire.Client {
	return &dnswire.Client{Transport: &dnswire.Transport{
		Dialer: &net.Dialer{Timeout: timeout}, ReadTimeout: timeout, WriteTimeout: timeout,
	}}
}

func writeDNSResponse(w dnswire.ResponseWriter, response *dnswire.Msg) error {
	if err := response.Pack(); err != nil {
		return err
	}
	_, err := io.Copy(w, response)
	return err
}
