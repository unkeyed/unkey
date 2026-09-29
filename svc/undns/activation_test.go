package undns

import (
	"fmt"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	privatecontract "github.com/unkeyed/unkey/pkg/privatenetwork"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestActivationHandlesIndependentDiscoveryObservations(t *testing.T) {
	for _, order := range [][]string{
		{"binding", "service", "slice"}, {"binding", "slice", "service"},
		{"service", "binding", "slice"}, {"service", "slice", "binding"},
		{"slice", "binding", "service"}, {"slice", "service", "binding"},
	} {
		t.Run(fmt.Sprint(order), func(t *testing.T) {
			c := catalogForTest()
			identity := testCaller()
			addBinding(t, c, "binding", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
			addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
			addresses, _, err := c.resolve(identity, "payments")
			require.NoError(t, err)
			require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)

			staged := catalogForTest()
			addBinding(t, staged, "binding", identity.workspace, identity.project, "app-a", "payments", "b", "b", "2")
			service := addServiceAndSlice(t, staged, "b", "b", "app-a", "uid-b", "10.0.0.22", true)

			binding, found, err := staged.bindings.GetStore().GetByKey("default/binding")
			require.NoError(t, err)
			require.True(t, found)
			slice, found, err := staged.slices.GetStore().GetByKey("default/b")
			require.NoError(t, err)
			require.True(t, found)

			for i, event := range order {
				switch event {
				case "binding":
					require.NoError(t, c.bindings.GetStore().Update(binding))
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
	addBinding(t, c, "binding", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)

	updateBinding(t, c, "binding", "b", "missing-b", "2")
	updateBinding(t, c, "binding", "c", "missing-c", "3")
	addresses, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
	addServiceAndSlice(t, c, "missing-c", "c", "app-a", "uid-c", "10.0.0.33", true)
	addresses, _, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.33")}, addresses)

	updateBinding(t, c, "binding", "a", "a", "1")
	addresses, _, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.33")}, addresses)
}

func TestBackgroundActivationChecksReadinessAndAllowsExplicitRollback(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addBinding(t, c, "binding", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	c.activate()
	updateBinding(t, c, "binding", "b", "b", "2")
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
	updateBinding(t, c, "binding", "a", "a", "2")
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err, "rollback requires a new revision")
	updateBinding(t, c, "binding", "a", "a", "3")
	addresses, _, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
}

func updateBinding(t *testing.T, c *catalog, name, deployment, service, revision string) {
	t.Helper()
	object, found, err := c.bindings.GetStore().GetByKey("default/" + name)
	require.NoError(t, err)
	require.True(t, found)
	config := object.(*corev1.ConfigMap).DeepCopy()
	config.Data["deploymentId"] = deployment
	config.Data["serviceName"] = service
	config.Data["revision"] = revision
	require.NoError(t, c.bindings.GetStore().Update(config))
}

func TestActivationBindingIncarnationDoesNotInheritActiveTarget(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addBinding(t, c, "binding", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	object, found, err := c.bindings.GetStore().GetByKey("default/binding")
	require.NoError(t, err)
	require.True(t, found)
	recreated := object.(*corev1.ConfigMap).DeepCopy()
	recreated.UID = types.UID("recreated")
	recreated.Data["serviceName"] = "missing"
	require.NoError(t, c.bindings.GetStore().Update(recreated))
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err)
}

func TestUnresolvedBindingRevokesActiveTarget(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addBinding(t, c, "binding", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)

	updateBinding(t, c, "binding", "", "", "2")
	_, found, err := c.resolve(identity, "payments")
	require.True(t, found)
	require.Error(t, err)
	require.Empty(t, c.active)
}

func TestActivationExpiryAndColdRestart(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addBinding(t, c, "binding", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	service := addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true).DeepCopy()
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	addServiceAndSlice(t, c, "missing-b", "b", "app-a", "uid-b", "10.0.0.22", false)

	restarted := catalogForTest()
	restarted.bindings, restarted.services, restarted.slices = c.bindings, c.services, c.slices
	addresses, found, err := restarted.resolve(identity, "payments")
	require.True(t, found)
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.11")}, addresses)
	updateBinding(t, c, "binding", "b", "missing-b", "2")

	deadline := time.Date(2026, 9, 21, 12, 30, 0, 0, time.UTC)
	service.Annotations = map[string]string{privatecontract.RetireAfterAnnotation: deadline.Format(time.RFC3339Nano)}
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

func TestActivationRejectsAmbiguousBindingsAndChangedAppIdentity(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addBinding(t, c, "binding", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	object, found, err := c.bindings.GetStore().GetByKey("default/binding")
	require.NoError(t, err)
	require.True(t, found)
	changed := object.(*corev1.ConfigMap).DeepCopy()
	changed.Labels[labels.LabelKeyAppID] = "another-app"
	changed.Data["revision"] = "2"
	require.NoError(t, c.bindings.GetStore().Update(changed))
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err)

	require.NoError(t, c.bindings.GetStore().Update(object))
	addBinding(t, c, "duplicate", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	c.activate()
	_, _, err = c.resolve(identity, "payments")
	require.Error(t, err)
	require.Empty(t, c.active)
	deleteBinding(t, c, "duplicate")
	_, _, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	deleteBinding(t, c, "binding")
	_, found, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.False(t, found)
	require.Empty(t, c.active)
}

func TestActivationConcurrentQueriesAndRefresh(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addBinding(t, c, "binding", identity.workspace, identity.project, "app-a", "payments", "a", "a", "1")
	addServiceAndSlice(t, c, "a", "a", "app-a", "uid-a", "10.0.0.11", true)
	_, _, err := c.resolve(identity, "payments")
	require.NoError(t, err)

	updateBinding(t, c, "binding", "b", "missing-b", "2")

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
