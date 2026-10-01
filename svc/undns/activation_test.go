package undns

import (
	"fmt"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/deploy/appconnection"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestActivationHandlesIndependentDiscoveryObservations(t *testing.T) {
	for _, order := range [][]string{
		{"connection", "service", "slice"}, {"connection", "slice", "service"},
		{"service", "connection", "slice"}, {"service", "slice", "connection"},
		{"slice", "connection", "service"}, {"slice", "service", "connection"},
	} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			c := catalogForTest()
			identity := testCaller()
			addConnection(t, c, "connection", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
			addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
			addresses, _, err := c.resolve(identity, "payments")
			require.NoError(t, err)
			require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)

			staged := catalogForTest()
			addConnection(t, staged, "connection", identity.workspace, identity.project, "app-a", "payments", "b", "b", "2")
			service := addServiceAndSlice(t, staged, "b", "b", "app-a", "uid-b", "10.0.0.22", true)

			connection, found, err := staged.connections.GetStore().GetByKey("default/connection")
			require.NoError(t, err)
			require.True(t, found)
			slice, found, err := staged.slices.GetStore().GetByKey("default/b")
			require.NoError(t, err)
			require.True(t, found)

			for i, event := range order {
				switch event {
				case "connection":
					require.NoError(t, c.connections.GetStore().Update(connection))
				case "service":
					require.NoError(t, c.services.GetStore().Add(service))
				case "slice":
					require.NoError(t, c.slices.GetStore().Add(slice))
				}
				addresses, _, err = c.resolve(identity, "payments")
				require.NoError(t, err)
				want := "10.0.0.11"
				if i == 2 {
					want = "10.0.0.22"
				}
				require.Equal(t, []netip.Addr{netip.MustParseAddr(want)}, addresses)
			}

			require.NoError(t, c.slices.GetStore().Delete(slice))
			_, _, err = c.resolve(identity, "payments")
			require.Error(t, err)
		})
	}
}

func TestActivationSkipsUnreadyRevisionAndNeverImplicitlyRollsBack(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addConnection(t, c, "connection", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)

	updateConnection(t, c, "connection", "b", "missing-b", "2")
	updateConnection(t, c, "connection", "c", "missing-c", "3")
	addresses, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
	addServiceAndSlice(t, c, "missing-c", "c", "app-a", "uid-c", "10.0.0.33", true)
	addresses, _, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.33")}, addresses)

	updateConnection(t, c, "connection", "a", "a", "1")
	addresses, _, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.33")}, addresses)
}

func TestBackgroundActivationChecksReadinessAndAllowsExplicitRollback(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addConnection(t, c, "connection", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	c.activate()
	updateConnection(t, c, "connection", "b", "b", "2")
	b := addServiceAndSlice(t, c, "b", "b", "app-a", "uid-b", "10.0.0.22", false)
	c.activate()
	addresses, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)

	addSlice(t, c, b, "b", "10.0.0.22", true)
	c.activate()
	addSlice(t, c, b, "b", "10.0.0.22", false)
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err, "B activated without a query, so its failure must not revive A")
	updateConnection(t, c, "connection", "a", "a", "2")
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err, "rollback requires a new revision")
	updateConnection(t, c, "connection", "a", "a", "3")
	addresses, _, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
}

func updateConnection(t *testing.T, c *catalog, name, deployment, service, revision string) {
	t.Helper()
	object, found, err := c.connections.GetStore().GetByKey("default/" + name)
	require.NoError(t, err)
	require.True(t, found)
	config := object.(*corev1.ConfigMap).DeepCopy()
	config.Data["deploymentId"] = deployment
	config.Data["serviceName"] = service
	config.Data["revision"] = revision
	require.NoError(t, c.connections.GetStore().Update(config))
}

func TestActivationConnectionIncarnationDoesNotInheritActiveTarget(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addConnection(t, c, "connection", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	object, found, err := c.connections.GetStore().GetByKey("default/connection")
	require.NoError(t, err)
	require.True(t, found)
	recreated := object.(*corev1.ConfigMap).DeepCopy()
	recreated.UID = types.UID("recreated")
	recreated.Data["serviceName"] = "missing"
	require.NoError(t, c.connections.GetStore().Update(recreated))
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err)
}

func TestUnresolvedConnectionRevokesActiveTarget(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addConnection(t, c, "connection", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)

	updateConnection(t, c, "connection", "", "", "2")
	_, found, err := c.resolve(identity, "payments")
	require.True(t, found)
	require.Error(t, err)
	require.Empty(t, c.active)
}

func TestActivationExpiryAndColdRestart(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addConnection(t, c, "connection", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	service := addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true).DeepCopy()
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	addServiceAndSlice(t, c, "missing-b", "b", "app-a", "uid-b", "10.0.0.22", false)

	restarted := catalogForTest()
	restarted.connections, restarted.services, restarted.slices = c.connections, c.services, c.slices
	addresses, found, err := restarted.resolve(identity, "payments")
	require.True(t, found)
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
	updateConnection(t, c, "connection", "b", "missing-b", "2")

	deadline := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	service.Annotations = map[string]string{appconnection.RetireAfterAnnotation: deadline.Format(time.RFC3339Nano)}
	require.NoError(t, c.services.GetStore().Update(service))
	c.now = func() time.Time { return deadline.Add(-time.Nanosecond) }
	addresses, found, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
	c.now = func() time.Time { return deadline }
	_, found, err = c.resolve(identity, "payments")
	require.True(t, found)
	require.Error(t, err, "expiry is enforced without Krane deleting the Service")
	addServiceAndSlice(t, c, "missing-b", "b", "app-a", "uid-b", "10.0.0.22", true)
	addresses, found, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.22")}, addresses)
}

func TestActivationRejectsAmbiguousConnectionsAndChangedAppIdentity(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addConnection(t, c, "connection", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	object, found, err := c.connections.GetStore().GetByKey("default/connection")
	require.NoError(t, err)
	require.True(t, found)
	changed := object.(*corev1.ConfigMap).DeepCopy()
	changed.Labels[labels.LabelKeyAppID] = "another-app"
	changed.Data["revision"] = "2"
	require.NoError(t, c.connections.GetStore().Update(changed))
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err)

	require.NoError(t, c.connections.GetStore().Update(object))
	addConnection(t, c, "duplicate", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	c.activate()
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err)
	require.Empty(t, c.active)
	deleteConnection(t, c, "duplicate")
	_, _, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	deleteConnection(t, c, "connection")
	_, found, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, c.active)
}

func TestActivationConcurrentQueriesAndRefresh(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addConnection(t, c, "connection", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)

	updateConnection(t, c, "connection", "b", "missing-b", "2")

	var group sync.WaitGroup
	errors := make(chan error, 9)
	for range 8 {
		group.Go(func() {
			for range 20 {
				addresses, found, err := c.resolve(identity, "payments")
				if err != nil || !found || len(addresses) != 1 || addresses[0].String() != "10.0.0.11" {
					errors <- fmt.Errorf("unexpected answer: %v, found %v, error %v", addresses, found, err)
					return
				}
			}
		})
	}
	group.Go(func() {
		for range 20 {
			c.activate()
		}
	})

	group.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
}
