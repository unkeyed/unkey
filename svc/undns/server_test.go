package undns

import (
	"context"
	"errors"
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
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/undns/internal/discovery"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func TestServerUDPTruncationTCPAndCallerDeploymentIsolation(t *testing.T) {
	// The fake API does not send the initial-events bookmark required by WatchList.
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	client := fake.NewSimpleClientset(discoveryObjects(t)...)
	c := newTestCatalog(t, client, 30*time.Second)
	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(t, err)
	tcp, udp, health := testListeners(t, &cfg)

	ctx, cancel := context.WithCancel(context.Background())
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
	require.Eventually(t, c.Ready, 5*time.Second, 10*time.Millisecond)

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

	caller, err := client.CoreV1().Pods("default").Get(ctx, "caller", metav1.GetOptions{})
	require.NoError(t, err)
	for _, tc := range []struct {
		kind, deployment string
		rcode            uint16
		answers          int
	}{
		{"production", uid.New(uid.DeploymentPrefix), dnswire.RcodeNameError, 0},
		{"production", "", dnswire.RcodeRefused, 0},
		{"preview", caller.Labels[labels.LabelKeyDeploymentID], dnswire.RcodeSuccess, 40},
		{"preview", "", dnswire.RcodeRefused, 0},
	} {
		pod := caller.DeepCopy()
		pod.Labels[privatenetwork.EnvironmentKindLabel] = tc.kind
		pod.Labels[labels.LabelKeyDeploymentID] = tc.deployment
		_, err = client.CoreV1().Pods("default").Update(ctx, pod, metav1.UpdateOptions{})
		require.NoError(t, err)
		require.Eventually(t, func() bool {
			caller, err := c.Identify(netip.MustParseAddr("127.0.0.1"))
			return (err == nil) == (tc.deployment != "") && (err != nil || caller.Deployment == tc.deployment)
		}, time.Second, 10*time.Millisecond)

		response, _, err = dns.Exchange(t.Context(), question, "tcp", cfg.ListenAddress)
		require.NoError(t, err)
		require.Equal(t, tc.rcode, response.Rcode)
		require.Len(t, response.Answer, tc.answers)
	}
}

func TestServerReturnsServfailWhenAnswerExceedsWireLimit(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	for _, count := range []int{4093, 4094} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			objects := discoveryObjects(t)
			slice := objects[3].(*discoveryv1.EndpointSlice)
			slice.Endpoints = nil
			for i := range count {
				slice.Endpoints = append(slice.Endpoints, discoveryv1.Endpoint{
					Addresses:  []string{netip.AddrFrom4([4]byte{10, 1, byte(i >> 8), byte(i)}).String()},
					Conditions: discoveryv1.EndpointConditions{Ready: new(true)},
				})
			}

			c := newTestCatalog(t, fake.NewSimpleClientset(objects...), 30*time.Second)

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

			require.Eventually(t, c.Ready, 5*time.Second, 10*time.Millisecond)

			for _, transport := range []string{"udp", "tcp"} {
				for _, udpSize := range []uint16{0, 1232} {
					t.Run(fmt.Sprintf("%s/edns-%d", transport, udpSize), func(t *testing.T) {
						question := dnswire.NewMsg("payments.unkey.internal.", dnswire.TypeA)
						question.UDPSize = udpSize
						response, _, err := testDNSClient(time.Second).Exchange(t.Context(), question, transport, cfg.ListenAddress)
						require.NoError(t, err)
						require.Equal(t, question.ID, response.ID)
						require.Equal(t, question.Question[0].Header().Name, response.Question[0].Header().Name)

						if count == 4094 {
							require.Equal(t, uint16(dnswire.RcodeServerFailure), response.Rcode)
							require.Empty(t, response.Answer)
							require.Empty(t, response.Ns)
							require.Empty(t, response.Extra)
							require.False(t, response.Truncated)
							require.False(t, response.Authoritative)
							return
						}

						require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
						want := 4093
						if transport == "udp" {
							want = 29
							if udpSize != 0 {
								want = 74
							}
						}
						require.Len(t, response.Answer, want)
						require.Equal(t, transport == "udp", response.Truncated)
					})
				}
			}
		})
	}
}

func TestColdStartServesPublishedConnectionWhileReplacementIsStaged(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	objects := discoveryObjects(t)
	connection := objects[1].(*corev1.ConfigMap)
	staged := objects[2].(*corev1.Service).DeepCopy()
	staged.Name, staged.UID = "service-b", types.UID(uid.New(uid.TestPrefix))
	staged.Labels[labels.LabelKeyDeploymentID] = uid.New(uid.DeploymentPrefix)
	remote := objects[3].(*discoveryv1.EndpointSlice).DeepCopy()
	remote.Name = "imported-b"
	remote.Labels[discoveryv1.LabelServiceName] = staged.Name
	remote.OwnerReferences = []metav1.OwnerReference{*metav1.NewControllerRef(staged, corev1.SchemeGroupVersion.WithKind("Service"))}
	remote.Endpoints = nil
	unavailable := connection.DeepCopy()
	unavailable.Name, unavailable.UID = "unavailable", types.UID(uid.New(uid.TestPrefix))
	unavailable.Labels[labels.LabelKeyConnectionID] = uid.New(uid.ConnectionPrefix)
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
				connection.Data["deploymentId"], connection.Data["serviceName"], connection.Data["revision"] = staged.Labels[labels.LabelKeyDeploymentID], staged.Name, "2"
				_, err = client.CoreV1().ConfigMaps("default").Update(t.Context(), connection, metav1.UpdateOptions{})
				require.NoError(t, err)
			}

			c := newTestCatalog(t, client, 30*time.Second)

			cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
			require.NoError(t, err)
			tcp, udp, health := testListeners(t, &cfg)

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan error, 1)
			go func() { done <- serveListeners(ctx, cfg, c, tcp, udp, health) }()
			t.Cleanup(func() { cancel(); require.NoError(t, <-done) })

			require.Eventually(t, c.Ready, 5*time.Second, 10*time.Millisecond)
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
				require.True(t, c.Ready())
			}
		})
	}
}

func TestServerAnswersCallerWhoseIPRemainsOnEvictedPod(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	evicted := callerPod("127.0.0.1", "production")
	evicted.Name = "evicted"
	evicted.Status.Phase, evicted.Status.Reason = corev1.PodFailed, "Evicted"
	client := fake.NewSimpleClientset(append(discoveryObjects(t), evicted)...)
	c := newTestCatalog(t, client, 30*time.Second)

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
	tcp, udp, health := testListeners(t, &cfg)

	ctx, cancel := context.WithCancel(context.Background())
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

	require.Eventually(t, c.Ready, 5*time.Second, 10*time.Millisecond)

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

func discoveryObjects(t testing.TB) []runtime.Object {
	t.Helper()
	pod := callerPod("127.0.0.1", "production")
	workspace, project := pod.Labels[labels.LabelKeyWorkspaceID], pod.Labels[labels.LabelKeyProjectID]
	appID, deploymentID := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix)

	connection := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "connection", Namespace: "default", UID: types.UID(uid.New(uid.TestPrefix)), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: privatenetwork.DiscoveryComponent,
			labels.LabelKeyWorkspaceID: workspace, labels.LabelKeyProjectID: project,
			labels.LabelKeyAppID: appID, privatenetwork.EnvironmentKindLabel: "production",
			labels.LabelKeyCallerDeploymentID: pod.Labels[labels.LabelKeyDeploymentID], labels.LabelKeyConnectionID: uid.New(uid.ConnectionPrefix),
		}},
		Data: connectionData(t, "payments", deploymentID, "service-a", 1),
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "service-a", Namespace: "default", UID: types.UID(uid.New(uid.TestPrefix)), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: privatenetwork.DiscoveryComponent, labels.LabelKeyWorkspaceID: workspace,
			labels.LabelKeyProjectID: project, labels.LabelKeyAppID: appID,
			labels.LabelKeyDeploymentID: deploymentID,
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

	return []runtime.Object{pod, connection, service, slice}
}

func ciliumImportedSliceLabels(service *corev1.Service) map[string]string {
	sliceLabels := maps.Clone(service.Labels)
	sliceLabels[discoveryv1.LabelServiceName] = service.Name
	sliceLabels[discoveryv1.LabelManagedBy] = "endpointslice-mesh-controller.cilium.io"
	sliceLabels["multicluster.kubernetes.io/source-cluster"] = "remote"
	return sliceLabels
}

func callerPod(address, kind string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "caller", Namespace: "default", UID: types.UID(uid.New(uid.TestPrefix)), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: "deployment",
			labels.LabelKeyWorkspaceID: uid.New(uid.WorkspacePrefix), labels.LabelKeyProjectID: uid.New(uid.ProjectPrefix),
			labels.LabelKeyAppID: uid.New(uid.AppPrefix), labels.LabelKeyEnvironmentID: uid.New(uid.EnvironmentPrefix),
			labels.LabelKeyDeploymentID: uid.New(uid.DeploymentPrefix), privatenetwork.EnvironmentKindLabel: kind,
		}},
		Status: corev1.PodStatus{Phase: corev1.PodRunning, PodIP: address, PodIPs: []corev1.PodIP{{IP: address}}},
	}
}

// The fake tracker does not replay changes between List and Watch.
func waitForFakeDiscoveryWatches(t *testing.T, client *fake.Clientset) {
	t.Helper()
	require.Eventually(t, func() bool {
		pending := map[string]bool{"pods": true, "configmaps": true, "services": true, "endpointslices": true}
		for _, action := range client.Actions() {
			if action.GetVerb() == "watch" && action.GetNamespace() == "" {
				delete(pending, action.GetResource().Resource)
			}
		}
		return len(pending) == 0
	}, 5*time.Second, time.Millisecond)
}

func testListeners(t testing.TB, cfg *Config) (net.Listener, net.PacketConn, net.Listener) {
	t.Helper()
	closeAtCleanup := func(closer io.Closer) {
		t.Cleanup(func() {
			if err := closer.Close(); !errors.Is(err, net.ErrClosed) {
				require.NoError(t, err)
			}
		})
	}

	tcp, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	closeAtCleanup(tcp)
	udp, err := net.ListenPacket("udp", tcp.Addr().String())
	require.NoError(t, err)
	closeAtCleanup(udp)
	health, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	closeAtCleanup(health)

	cfg.ListenAddress = tcp.Addr().String()
	cfg.HealthAddress = health.Addr().String()
	return tcp, udp, health
}

func newTestCatalog(t testing.TB, client kubernetes.Interface, watchTimeout time.Duration) *discovery.Catalog {
	t.Helper()
	return newTestCatalogWithClock(t, client, watchTimeout, clock.New())
}

func newTestCatalogWithClock(t testing.TB, client kubernetes.Interface, watchTimeout time.Duration, clk clock.Clock) *discovery.Catalog {
	t.Helper()
	c, err := discovery.New(client, watchTimeout, clk)
	require.NoError(t, err)
	return c
}

func connectionData(t testing.TB, slug, deployment, service string, revision uint64) map[string]string {
	t.Helper()
	data, err := privatenetwork.Encode(privatenetwork.ConnectionData{Alias: slug, DeploymentID: deployment, ServiceName: service, Revision: revision})
	require.NoError(t, err)
	return data
}

func testCaller(t *testing.T, c *discovery.Catalog) discovery.Caller {
	t.Helper()
	caller, err := c.Identify(netip.MustParseAddr("127.0.0.1"))
	require.NoError(t, err)
	return caller
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
