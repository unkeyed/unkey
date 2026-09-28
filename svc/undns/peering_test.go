package undns

import (
	"net/netip"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestReplicaDiscoveryResolvesReadyPeersAcrossRegions(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addBinding(t, c, "self-"+identity.deployment, "workspace-a", "project-a", "app-caller", "caller", identity.deployment, "self-service", "1")
	service := addServiceAndSlice(t, c, "self-service", identity.deployment, "app-caller", types.UID("self-uid"), "10.0.0.2", true)
	addSlice(t, c, service, "imported-from-other-region", "10.1.0.7", true)
	addSlice(t, c, service, "unready-peer", "10.1.0.8", false)

	addresses, exists, err := c.resolve(identity, "caller")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.2"), netip.MustParseAddr("10.1.0.7")}, addresses)

	otherRevision := identity
	otherRevision.deployment = "caller-deployment-b"
	_, exists, err = c.resolve(otherRevision, "caller")
	require.NoError(t, err)
	require.False(t, exists, "another deployment of the same app must not discover these peers")
}

func TestDiscoveryAnswersDoNotDependOnServicePorts(t *testing.T) {
	c := catalogForTest()
	addBinding(t, c, "binding-a", "workspace-a", "project-a", "app-a", "gossip", "deployment-a", "service-a", "1")
	service := addServiceAndSlice(t, c, "service-a", "deployment-a", "app-a", types.UID("service-a-uid"), "10.0.0.1", true)

	before, _, err := c.resolve(testCaller(), "gossip")
	require.NoError(t, err)
	updated := service.DeepCopy()
	updated.Spec.Ports = []corev1.ServicePort{{Name: "app", Port: 7946, Protocol: corev1.ProtocolUDP}}
	require.NoError(t, c.services.GetStore().Update(updated))
	after, _, err := c.resolve(testCaller(), "gossip")
	require.NoError(t, err)
	require.Equal(t, before, after)
}
