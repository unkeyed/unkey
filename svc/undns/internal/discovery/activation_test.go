package discovery

import (
	"fmt"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
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
			appID, deploymentA, deploymentB := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
			addConnection(t, c, "connection", identity, appID, "payments", deploymentA, "a", "1")
			addServiceAndSlice(t, c, identity, "a", deploymentA, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.11", true)
			addresses, _, err := c.Resolve(identity, "payments")
			require.NoError(t, err)
			require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)

			staged := catalogForTest()
			addConnection(t, staged, "connection", identity, appID, "payments", deploymentB, "b", "2")
			service := addServiceAndSlice(t, staged, identity, "b", deploymentB, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.22", true)

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
				addresses, _, err = c.Resolve(identity, "payments")
				if i == 2 {
					require.NoError(t, err)
					require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.22")}, addresses)
				} else if event == "connection" || order[0] == "connection" {
					require.Error(t, err, "published target is authoritative before its discovery objects arrive")
				} else {
					require.NoError(t, err)
					require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
				}
			}

			require.NoError(t, c.slices.GetStore().Delete(slice))
			_, _, err = c.Resolve(identity, "payments")
			require.Error(t, err)
		})
	}
}

func TestPublishedConnectionSwitchAndRollbackAreImmediate(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID := uid.New(uid.AppPrefix)
	deploymentA, deploymentB, deploymentC := uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection", identity, appID, "payments", deploymentA, "a", "1")
	addServiceAndSlice(t, c, identity, "a", deploymentA, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.11", true)
	_, _, err := c.Resolve(identity, "payments")
	require.NoError(t, err)

	updateConnection(t, c, "connection", deploymentB, "missing-b", "2")
	updateConnection(t, c, "connection", deploymentC, "missing-c", "3")
	_, _, err = c.Resolve(identity, "payments")
	requireReason(t, err, ReasonServiceMissing)
	addServiceAndSlice(t, c, identity, "missing-c", deploymentC, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.33", true)
	addresses, _, err := c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.33")}, addresses)

	updateConnection(t, c, "connection", deploymentA, "a", "1")
	addresses, _, err = c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
}

func TestStatusRefreshDoesNotGatePublishedConnection(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID, deploymentA, deploymentB := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection", identity, appID, "payments", deploymentA, "a", "1")
	addServiceAndSlice(t, c, identity, "a", deploymentA, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.11", true)
	c.updateStatuses()
	updateConnection(t, c, "connection", deploymentB, "b", "2")
	b := addServiceAndSlice(t, c, identity, "b", deploymentB, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.22", false)
	c.updateStatuses()
	_, _, err := c.Resolve(identity, "payments")
	requireReason(t, err, ReasonNoReadyEndpoints)

	addSlice(t, c, b, "b", "10.0.0.22", true)
	c.updateStatuses()
	addSlice(t, c, b, "b", "10.0.0.22", false)
	_, _, err = c.Resolve(identity, "payments")
	require.Error(t, err, "the published target's failure must not revive the previous target")
	updateConnection(t, c, "connection", deploymentA, "a", "2")
	addresses, _, err := c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
}

func updateConnection(t *testing.T, c *Catalog, name, deployment, service, revision string) {
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
	appID, deploymentID := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection", identity, appID, "payments", deploymentID, "a", "1")
	addServiceAndSlice(t, c, identity, "a", deploymentID, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.11", true)
	_, _, err := c.Resolve(identity, "payments")
	require.NoError(t, err)
	object, found, err := c.connections.GetStore().GetByKey("default/connection")
	require.NoError(t, err)
	require.True(t, found)
	recreated := object.(*corev1.ConfigMap).DeepCopy()
	recreated.UID = types.UID(uid.New(uid.TestPrefix))
	recreated.Data["serviceName"] = "missing"
	require.NoError(t, c.connections.GetStore().Update(recreated))
	_, _, err = c.Resolve(identity, "payments")
	require.Error(t, err)
}

func TestUnresolvedConnectionRevokesActiveTarget(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID, deploymentID := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection", identity, appID, "payments", deploymentID, "a", "1")
	addServiceAndSlice(t, c, identity, "a", deploymentID, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.11", true)
	_, _, err := c.Resolve(identity, "payments")
	require.NoError(t, err)

	updateConnection(t, c, "connection", "", "", "2")
	_, found, err := c.Resolve(identity, "payments")
	require.True(t, found)
	require.Error(t, err)
}

func TestActivationExpiryAndColdRestart(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID, deploymentA, deploymentB := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	serviceBUID := types.UID(uid.New(uid.TestPrefix))
	addConnection(t, c, "connection", identity, appID, "payments", deploymentA, "a", "1")
	service := addServiceAndSlice(t, c, identity, "a", deploymentA, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.11", true).DeepCopy()
	_, _, err := c.Resolve(identity, "payments")
	require.NoError(t, err)
	addServiceAndSlice(t, c, identity, "missing-b", deploymentB, appID, serviceBUID, "10.0.0.22", false)

	restarted := catalogForTest()
	restarted.connections, restarted.services, restarted.slices = c.connections, c.services, c.slices
	addresses, found, err := restarted.Resolve(identity, "payments")
	require.True(t, found)
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
	updateConnection(t, c, "connection", deploymentB, "missing-b", "2")

	deadline := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	service.Annotations = map[string]string{privatenetwork.RetireAfterAnnotation: deadline.Format(time.RFC3339Nano)}
	require.NoError(t, c.services.GetStore().Update(service))
	c.clock = clock.NewTestClock(deadline.Add(-time.Nanosecond))
	_, found, err = c.Resolve(identity, "payments")
	require.True(t, found)
	requireReason(t, err, ReasonNoReadyEndpoints)
	c.clock = clock.NewTestClock(deadline)
	_, found, err = c.Resolve(identity, "payments")
	require.True(t, found)
	require.Error(t, err, "expiry is enforced without Krane deleting the Service")
	addServiceAndSlice(t, c, identity, "missing-b", deploymentB, appID, serviceBUID, "10.0.0.22", true)
	addresses, found, err = c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.22")}, addresses)
}

func TestActivationRejectsAmbiguousConnectionsAndChangedAppIdentity(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID, deploymentID := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection", identity, appID, "payments", deploymentID, "a", "1")
	addServiceAndSlice(t, c, identity, "a", deploymentID, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.11", true)
	_, _, err := c.Resolve(identity, "payments")
	require.NoError(t, err)
	object, found, err := c.connections.GetStore().GetByKey("default/connection")
	require.NoError(t, err)
	require.True(t, found)
	changed := object.(*corev1.ConfigMap).DeepCopy()
	changed.Labels[labels.LabelKeyAppID] = uid.New(uid.AppPrefix)
	changed.Data["revision"] = "2"
	require.NoError(t, c.connections.GetStore().Update(changed))
	_, _, err = c.Resolve(identity, "payments")
	require.Error(t, err)

	require.NoError(t, c.connections.GetStore().Update(object))
	addConnection(t, c, "duplicate", identity, appID, "payments", deploymentID, "a", "1")
	c.updateStatuses()
	_, _, err = c.Resolve(identity, "payments")
	require.Error(t, err)
	deleteConnection(t, c, "duplicate")
	_, _, err = c.Resolve(identity, "payments")
	require.NoError(t, err)
	deleteConnection(t, c, "connection")
	_, found, err = c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.False(t, found)
}

func TestActivationConcurrentQueriesAndRefresh(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID, deploymentA, deploymentB := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection", identity, appID, "payments", deploymentA, "a", "1")
	addServiceAndSlice(t, c, identity, "a", deploymentA, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.11", true)
	_, _, err := c.Resolve(identity, "payments")
	require.NoError(t, err)

	updateConnection(t, c, "connection", deploymentB, "missing-b", "2")

	var group sync.WaitGroup
	errors := make(chan error, 9)
	for range 8 {
		group.Go(func() {
			for range 20 {
				addresses, found, err := c.Resolve(identity, "payments")
				if !found || err == nil || ReasonOf(err) != ReasonServiceMissing || len(addresses) != 0 {
					errors <- fmt.Errorf("unexpected answer: %v, found %v, error %v", addresses, found, err)
					return
				}
			}
		})
	}
	group.Go(func() {
		for range 20 {
			c.updateStatuses()
		}
	})

	group.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}
}
