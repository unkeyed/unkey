package undns

import (
	"context"
	"fmt"
	"net/netip"
	"testing"
	"time"

	dnswire "codeberg.org/miekg/dns"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/config"
	"github.com/unkeyed/unkey/pkg/deploy/appconnection"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
)

func topologyForTest() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: appconnection.TopologyConfigMap, Namespace: metav1.NamespaceSystem,
			Labels: map[string]string{labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: appconnection.TopologyComponent},
		},
		Data: map[string]string{
			"version": "1", "platform": "aws", "cell": "home",
			"region.home": "us-east-1", "region.neighbor": "us-east-1", "region.remote": "eu-west-1",
		},
	}
}

func locateSlice(t *testing.T, c *catalog, name, cell string) *discoveryv1.EndpointSlice {
	t.Helper()
	object, exists, err := c.slices.GetStore().GetByKey("default/" + name)
	require.NoError(t, err)
	require.True(t, exists)
	slice := object.(*discoveryv1.EndpointSlice).DeepCopy()
	slice.Labels[discoveryv1.LabelManagedBy] = "private-dns.unkey.com"
	if cell != "" {
		slice.Labels[discoveryv1.LabelManagedBy] = "endpointslice-mesh-controller.cilium.io"
		slice.Labels["multicluster.kubernetes.io/source-cluster"] = cell
	}
	require.NoError(t, c.slices.GetStore().Update(slice))
	return slice
}

func TestLocalitySelectsOnlyReadyAuthorizedEndpoints(t *testing.T) {
	c := catalogForTest()
	require.NoError(t, c.topology.GetStore().Add(topologyForTest()))
	addConnection(t, c, "connection-a", "workspace-a", "project-a", "app-a", "payments", "deployment-a", "service-a", "1")
	service := addServiceAndSlice(t, c, "service-a", "deployment-a", "app-a", "uid-a", "10.0.0.1", true)
	local := locateSlice(t, c, "service-a", "")
	addSlice(t, c, service, "neighbor", "10.0.1.1", true)
	neighbor := locateSlice(t, c, "neighbor", "neighbor")
	addSlice(t, c, service, "unknown", "10.0.2.1", true)
	locateSlice(t, c, "unknown", "unmapped")
	wantAll := []netip.Addr{netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.0.1.1"), netip.MustParseAddr("10.0.2.1")}
	var wantRemote []netip.Addr
	for i := 1; i <= 9; i++ {
		address := fmt.Sprintf("10.1.0.%d", i)
		name := fmt.Sprintf("remote-%d", i)
		addSlice(t, c, service, name, address, true)
		locateSlice(t, c, name, "remote")
		wantRemote = append(wantRemote, netip.MustParseAddr(address))
	}
	wantAll = append(wantAll, wantRemote...)
	for _, tc := range []struct {
		name string
		want []netip.Addr
	}{
		{"payments", wantAll},
		{"local-first.payments", wantAll[:2]},
		{"us-east-1.payments", wantAll[:2]},
		{"eu-west-1.payments", wantRemote},
	} {
		t.Run(tc.name, func(t *testing.T) {
			addresses, exists, err := c.resolve(testCaller(), tc.name)
			require.NoError(t, err)
			require.True(t, exists)
			require.Equal(t, tc.want, addresses)
		})
	}
	for _, name := range []string{"local-first.payments", "us-east-1.payments", "eu-west-1.payments"} {
		t.Run("unauthorized/"+name, func(t *testing.T) {
			other := testCaller()
			other.deployment = "other-deployment"
			_, exists, err := c.resolve(other, name)
			require.NoError(t, err)
			require.False(t, exists)
		})
	}

	local.Endpoints[0].Conditions.Ready = new(false)
	neighbor.Endpoints[0].Conditions.Terminating = new(true)
	require.NoError(t, c.slices.GetStore().Update(local))
	require.NoError(t, c.slices.GetStore().Update(neighbor))
	addresses, exists, err := c.resolve(testCaller(), "local-first.payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, wantAll[2:], addresses)
	_, exists, err = c.resolve(testCaller(), "us-east-1.payments")
	require.True(t, exists)
	require.ErrorIs(t, err, errNoReadyEndpoints)
}

func TestLocalityMissingMetadataAndRevisionActivation(t *testing.T) {
	c := catalogForTest()
	addConnection(t, c, "connection-a", "workspace-a", "project-a", "app-a", "payments", "deployment-a", "service-a", "1")
	service := addServiceAndSlice(t, c, "service-a", "deployment-a", "app-a", "uid-a", "10.0.0.1", true)
	locateSlice(t, c, "service-a", "")
	addSlice(t, c, service, "remote", "10.1.0.1", true)
	locateSlice(t, c, "remote", "remote")
	invalid := topologyForTest()
	invalid.Data["version"] = "unknown"
	for _, topology := range []*corev1.ConfigMap{nil, invalid} {
		if topology != nil {
			require.NoError(t, c.topology.GetStore().Update(topology))
		}
		addresses, exists, err := c.resolve(testCaller(), "local-first.payments")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.1")}, addresses)
		_, exists, err = c.resolve(testCaller(), "eu-west-1.payments")
		require.True(t, exists)
		require.ErrorIs(t, err, errNoReadyEndpoints)
	}
	local := locateSlice(t, c, "service-a", "unmapped")
	addresses, _, err := c.resolve(testCaller(), "local-first.payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.1.0.1")}, addresses)
	delete(local.Labels, "multicluster.kubernetes.io/source-cluster")
	local.Labels[discoveryv1.LabelManagedBy] = "private-dns.unkey.com"
	require.NoError(t, c.slices.GetStore().Update(local))
	require.NoError(t, c.topology.GetStore().Update(topologyForTest()))

	addServiceAndSlice(t, c, "service-b", "deployment-b", "app-a", "uid-b", "10.2.0.1", true)
	locateSlice(t, c, "service-b", "remote")
	deleteConnection(t, c, "connection-a")
	addConnection(t, c, "connection-a", "workspace-a", "project-a", "app-a", "payments", "deployment-b", "service-b", "2")
	_, exists, err := c.resolve(testCaller(), "us-east-1.payments")
	require.True(t, exists)
	require.ErrorIs(t, err, errNoReadyEndpoints, "a region selector must not retain an older local revision")
	addresses, exists, err = c.resolve(testCaller(), "local-first.payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.2.0.1")}, addresses)
	deleteConnection(t, c, "connection-a")
	_, exists, err = c.resolve(testCaller(), "local-first.payments")
	require.NoError(t, err)
	require.False(t, exists)
}

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
	c, err := newCatalog(client, time.Minute)
	require.NoError(t, err)
	cfg, err := config.LoadBytes[Config]([]byte(`upstream = "10.96.0.10:53"`))
	require.NoError(t, err)
	cfg.ListenAddress, cfg.HealthAddress = unusedTCPAddress(t), unusedTCPAddress(t)
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
	require.Eventually(t, func() bool { return c.ready() && c.topology.HasSynced() }, 5*time.Second, time.Millisecond)
	c.topology.lastContact.Store(0)

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
		addresses, _, err := c.resolve(testCaller(), "local-first.payments")
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
	c.pods.lastContact.Store(0)
	response, _, err := testDNSClient(time.Second).Exchange(t.Context(), dnswire.NewMsg("local-first.payments.unkey.internal.", dnswire.TypeA), "tcp", cfg.ListenAddress)
	require.NoError(t, err)
	require.Equal(t, uint16(dnswire.RcodeServerFailure), response.Rcode)
}
