package discovery

import (
	"fmt"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

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

func locateSlice(t *testing.T, c *Catalog, name, cell string) *discoveryv1.EndpointSlice {
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
	identity := testCaller()
	appID, deploymentID := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix)
	require.NoError(t, c.topology.GetStore().Add(topologyForTest()))
	addConnection(t, c, "connection-a", identity, appID, "payments", deploymentID, "service-a", "1")
	service := addServiceAndSlice(t, c, identity, "service-a", deploymentID, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.1", true)
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
			addresses, exists, err := c.Resolve(identity, tc.name)
			require.NoError(t, err)
			require.True(t, exists)
			require.Equal(t, tc.want, addresses)
		})
	}
	for _, name := range []string{"local-first.payments", "us-east-1.payments", "eu-west-1.payments"} {
		t.Run("unauthorized/"+name, func(t *testing.T) {
			other := identity
			other.Deployment = uid.New(uid.DeploymentPrefix)
			_, exists, err := c.Resolve(other, name)
			require.NoError(t, err)
			require.False(t, exists)
		})
	}

	local.Endpoints[0].Conditions.Ready = new(false)
	neighbor.Endpoints[0].Conditions.Terminating = new(true)
	require.NoError(t, c.slices.GetStore().Update(local))
	require.NoError(t, c.slices.GetStore().Update(neighbor))
	addresses, exists, err := c.Resolve(identity, "local-first.payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, wantAll[2:], addresses)
	_, exists, err = c.Resolve(identity, "us-east-1.payments")
	require.True(t, exists)
	requireReason(t, err, ReasonNoReadyEndpoints)
}

func TestLocalityMissingMetadataAndRevisionActivation(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID, deploymentA, deploymentB := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection-a", identity, appID, "payments", deploymentA, "service-a", "1")
	service := addServiceAndSlice(t, c, identity, "service-a", deploymentA, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.1", true)
	locateSlice(t, c, "service-a", "")
	addSlice(t, c, service, "remote", "10.1.0.1", true)
	locateSlice(t, c, "remote", "remote")
	invalid := topologyForTest()
	invalid.Data["version"] = "unknown"
	for _, topology := range []*corev1.ConfigMap{nil, invalid} {
		if topology != nil {
			require.NoError(t, c.topology.GetStore().Update(topology))
		}
		addresses, exists, err := c.Resolve(identity, "local-first.payments")
		require.NoError(t, err)
		require.True(t, exists)
		require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.1")}, addresses)
		_, exists, err = c.Resolve(identity, "eu-west-1.payments")
		require.True(t, exists)
		requireReason(t, err, ReasonNoReadyEndpoints)
	}
	local := locateSlice(t, c, "service-a", "unmapped")
	addresses, _, err := c.Resolve(identity, "local-first.payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.1"), netip.MustParseAddr("10.1.0.1")}, addresses)
	delete(local.Labels, "multicluster.kubernetes.io/source-cluster")
	local.Labels[discoveryv1.LabelManagedBy] = "private-dns.unkey.com"
	require.NoError(t, c.slices.GetStore().Update(local))
	require.NoError(t, c.topology.GetStore().Update(topologyForTest()))

	addServiceAndSlice(t, c, identity, "service-b", deploymentB, appID, types.UID(uid.New(uid.TestPrefix)), "10.2.0.1", true)
	locateSlice(t, c, "service-b", "remote")
	deleteConnection(t, c, "connection-a")
	addConnection(t, c, "connection-a", identity, appID, "payments", deploymentB, "service-b", "2")
	_, exists, err := c.Resolve(identity, "us-east-1.payments")
	require.True(t, exists)
	requireReason(t, err, ReasonNoReadyEndpoints, "a region selector must not retain an older local revision")
	addresses, exists, err = c.Resolve(identity, "local-first.payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.2.0.1")}, addresses)
	deleteConnection(t, c, "connection-a")
	_, exists, err = c.Resolve(identity, "local-first.payments")
	require.NoError(t, err)
	require.False(t, exists)
}
