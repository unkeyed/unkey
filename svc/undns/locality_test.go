package undns

import (
	"context"
	"net/netip"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	"github.com/unkeyed/unkey/svc/undns/internal/discovery"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
)

func TestServerLocalityUDPAndTCP(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	objects := discoveryObjects(t)
	remote := objects[3].(*discoveryv1.EndpointSlice)
	local := remote.DeepCopy()
	local.Name = "local"
	delete(local.Labels, "multicluster.kubernetes.io/source-cluster")
	local.Labels[discoveryv1.LabelManagedBy] = "private-dns.unkey.com"
	local.Endpoints = []discoveryv1.Endpoint{{Addresses: []string{"10.9.0.1"}, Conditions: discoveryv1.EndpointConditions{Ready: new(true)}}}
	client := fake.NewClientset(append(objects, local, topologyForTest())...)
	clk := clock.NewTestClock()
	c := newTestCatalogWithClock(t, client, time.Minute, clk)
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
	waitForTopology(t, c)

	for _, transport := range []string{"udp", "tcp"} {
		for _, tc := range []struct {
			name  string
			rcode uint16
		}{
			{"local-first.payments", dnswire.RcodeSuccess},
			{"US-EAST-1.Payments", dnswire.RcodeSuccess},
			{"ap-south-1.payments", dnswire.RcodeServerFailure},
			{"local-first.unknown", dnswire.RcodeNameError},
			{"us-east-1.local-first.payments", dnswire.RcodeNameError},
		} {
			t.Run(transport+"/"+tc.name, func(t *testing.T) {
				response, _, err := testDNSClient(time.Second).Exchange(t.Context(), dnswire.NewMsg(tc.name+".unkey.internal.", dnswire.TypeA), transport, cfg.ListenAddress)
				require.NoError(t, err)
				require.Equal(t, tc.rcode, response.Rcode)
				if tc.rcode == dnswire.RcodeSuccess {
					require.Equal(t, []string{"10.9.0.1"}, answerAddresses(response))
					require.Equal(t, uint32(5), response.Answer[0].Header().TTL)
				}
			})
		}
	}
	local.Endpoints[0].Conditions.Ready = new(false)
	_, err = client.DiscoveryV1().EndpointSlices("default").Update(t.Context(), local, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		addresses, _, err := c.Resolve(testCaller(t, c), "local-first.payments")
		return err == nil && len(addresses) == 40
	}, time.Second, time.Millisecond)
	for _, name := range []string{"local-first.payments", "eu-west-1.payments", "payments"} {
		t.Run("fallback/"+name, func(t *testing.T) {
			question := dnswire.NewMsg(name+".unkey.internal.", dnswire.TypeA)
			response, _, err := testDNSClient(time.Second).Exchange(t.Context(), question, "udp", cfg.ListenAddress)
			require.NoError(t, err)
			require.True(t, response.Truncated)
			response, _, err = testDNSClient(time.Second).Exchange(t.Context(), question, "tcp", cfg.ListenAddress)
			require.NoError(t, err)
			require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
			require.False(t, response.Truncated)
			require.Len(t, response.Answer, 40)
			require.NotContains(t, answerAddresses(response), "10.9.0.1")
		})
	}
	clk.Tick(3 * time.Minute)
	response, _, err := testDNSClient(time.Second).Exchange(t.Context(), dnswire.NewMsg("local-first.payments.unkey.internal.", dnswire.TypeA), "tcp", cfg.ListenAddress)
	require.NoError(t, err)
	require.Equal(t, uint16(dnswire.RcodeServerFailure), response.Rcode)
}

func TestServerLocalityTracksImportedEndpointSliceRemovalAndRestore(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	objects := discoveryObjects(t)
	remote := objects[3].(*discoveryv1.EndpointSlice).DeepCopy()
	local := remote.DeepCopy()
	local.Name = "local"
	delete(local.Labels, "multicluster.kubernetes.io/source-cluster")
	local.Labels[discoveryv1.LabelManagedBy] = "private-dns.unkey.com"
	local.Endpoints = []discoveryv1.Endpoint{{Addresses: []string{"10.9.0.1"}, Conditions: discoveryv1.EndpointConditions{Ready: new(true)}}}
	client := fake.NewClientset(append(objects, local, topologyForTest())...)
	c := newTestCatalog(t, client, time.Minute)
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
	waitForTopology(t, c)
	waitForFakeDiscoveryWatches(t, client)

	query := func(name string) *dnswire.Msg {
		t.Helper()
		response, _, queryErr := testDNSClient(time.Second).Exchange(t.Context(), dnswire.NewMsg(name+".unkey.internal.", dnswire.TypeA), "tcp", cfg.ListenAddress)
		require.NoError(t, queryErr)
		return response
	}

	require.NoError(t, client.DiscoveryV1().EndpointSlices("default").Delete(t.Context(), remote.Name, metav1.DeleteOptions{}))
	require.Eventually(t, func() bool {
		addresses, _, err := c.Resolve(testCaller(t, c), "payments")
		return err == nil && len(addresses) == 1 && addresses[0].String() == "10.9.0.1"
	}, time.Second, 5*time.Millisecond)
	for _, name := range []string{"payments", "local-first.payments"} {
		response := query(name)
		require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
		require.Equal(t, []string{"10.9.0.1"}, answerAddresses(response))
	}
	require.Equal(t, uint16(dnswire.RcodeServerFailure), query("eu-west-1.payments").Rcode)

	remote.ResourceVersion = ""
	_, err = client.DiscoveryV1().EndpointSlices("default").Create(t.Context(), remote, metav1.CreateOptions{})
	require.NoError(t, err)
	require.Eventually(t, func() bool {
		addresses, _, err := c.Resolve(testCaller(t, c), "eu-west-1.payments")
		return err == nil && len(addresses) == 40
	}, time.Second, 5*time.Millisecond)
	response := query("eu-west-1.payments")
	require.Equal(t, uint16(dnswire.RcodeSuccess), response.Rcode)
	require.Len(t, response.Answer, 40)
	require.NotContains(t, answerAddresses(response), "10.9.0.1")
}

func topologyForTest() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: privatenetwork.TopologyConfigMap, Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: privatenetwork.TopologyComponent},
		},
		Data: map[string]string{
			"version": "1", "platform": "aws", "cell": "home",
			"region.home": "us-east-1", "region.neighbor": "us-east-1", "region.remote": "eu-west-1",
		},
	}
}

func waitForTopology(t *testing.T, c *discovery.Catalog) {
	t.Helper()
	require.Eventually(t, func() bool {
		caller, err := c.Identify(netip.MustParseAddr("127.0.0.1"))
		if err != nil {
			return false
		}
		addresses, _, err := c.Resolve(caller, "us-east-1.payments")
		return c.Ready() && err == nil && len(addresses) == 1 && addresses[0].String() == "10.9.0.1"
	}, 5*time.Second, time.Millisecond)
}
