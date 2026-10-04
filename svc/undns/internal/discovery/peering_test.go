package discovery

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReplicaDiscoveryResolvesReadyPeersAcrossRegions(t *testing.T) {
	c := catalogForTest()
	require.NoError(t, c.topology.GetStore().Add(topologyForTest()))
	identity := testCaller()
	appID := uid.New(uid.AppPrefix)
	addConnection(t, c, "self-"+identity.Deployment, identity, appID, "caller", identity.Deployment, "self-service", "1")
	service := addServiceAndSlice(t, c, identity, "self-service", identity.Deployment, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.2", true)
	addSlice(t, c, service, "imported-from-other-region", "10.1.0.7", true)
	addSlice(t, c, service, "unready-peer", "10.1.0.8", false)
	locateSlice(t, c, "self-service", "")
	locateSlice(t, c, "imported-from-other-region", "remote")

	addresses, exists, err := c.Resolve(identity, "caller")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.2"), netip.MustParseAddr("10.1.0.7")}, addresses)

	otherRevision := identity
	otherRevision.Deployment = uid.New(uid.DeploymentPrefix)
	_, exists, err = c.Resolve(otherRevision, "caller")
	require.NoError(t, err)
	require.False(t, exists, "another deployment of the same app must not discover these peers")
}

func TestDiscoveryAnswersDoNotDependOnServicePorts(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID, deploymentID := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection-a", identity, appID, "gossip", deploymentID, "service-a", "1")
	service := addServiceAndSlice(t, c, identity, "service-a", deploymentID, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.1", true)

	before, _, err := c.Resolve(identity, "gossip")
	require.NoError(t, err)
	updated := service.DeepCopy()
	updated.Spec.Ports = []corev1.ServicePort{{Name: "app", Port: 7946, Protocol: corev1.ProtocolUDP}}
	require.NoError(t, c.services.GetStore().Update(updated))
	after, _, err := c.Resolve(identity, "gossip")
	require.NoError(t, err)
	require.Equal(t, before, after)
}
