package discovery

import (
	"context"
	"errors"
	"net/netip"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
	kubetesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
)

func TestCatalogResolveIsolatesConnectionsAndFollowsPromotion(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID, deploymentA, deploymentB := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	otherWorkspace, otherProject := identity, identity
	otherWorkspace.Workspace = uid.New(uid.WorkspacePrefix)
	otherProject.Project = uid.New(uid.ProjectPrefix)
	otherDeployment := uid.New(uid.DeploymentPrefix)
	addConnection(t, c, "connection-a", identity, appID, "payments", deploymentA, "service-a", "1")
	addConnection(t, c, "connection-other-workspace", otherWorkspace, uid.New(uid.AppPrefix), "payments", otherDeployment, "service-other", "1")
	addConnection(t, c, "connection-other-project", otherProject, uid.New(uid.AppPrefix), "payments", otherDeployment, "service-other", "1")
	addServiceAndSlice(t, c, identity, "service-a", deploymentA, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.1", true)
	addServiceAndSlice(t, c, identity, "service-b", deploymentB, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.2", true)

	addresses, exists, err := c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.1")}, addresses)

	deleteConnection(t, c, "connection-a")
	addConnection(t, c, "connection-a", identity, appID, "payments", deploymentB, "service-b", "2")
	addresses, exists, err = c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.2")}, addresses)

	deleteConnection(t, c, "connection-a")
	addConnection(t, c, "connection-a", identity, appID, "payments", deploymentA, "service-a", "3")
	addresses, exists, err = c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.1")}, addresses)

	deleteConnection(t, c, "connection-a")
	_, exists, err = c.Resolve(identity, "payments")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestDirectedConnectionsIsolateCallerDeployments(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	appID := uid.New(uid.AppPrefix)
	callerA, callerB, callerC := identity.Deployment, uid.New(uid.DeploymentPrefix), uid.New(uid.DeploymentPrefix)
	unknownCaller := uid.New(uid.DeploymentPrefix)
	for _, tc := range []struct{ callerDeployment, name, kind, address string }{
		{callerA, "production", "production", "10.0.0.1"},
		{callerB, "preview-a", "production", "10.0.0.2"},
		{callerC, "preview-b", "preview", "10.0.0.3"},
	} {
		caller := identity
		caller.Deployment = tc.callerDeployment
		targetDeployment := uid.New(uid.DeploymentPrefix)
		addConnection(t, c, tc.name, caller, appID, "payments", targetDeployment, tc.name, "1")
		service := addServiceAndSlice(t, c, caller, tc.name, targetDeployment, appID, types.UID(uid.New(uid.TestPrefix)), tc.address, true).DeepCopy()
		object, found, err := c.connections.GetStore().GetByKey("default/" + tc.name)
		require.NoError(t, err)
		require.True(t, found)
		connection := object.(*corev1.ConfigMap).DeepCopy()
		connection.Labels[privatenetwork.EnvironmentKindLabel] = tc.kind
		connection.Labels[labels.LabelKeyCallerDeploymentID] = tc.callerDeployment
		require.NoError(t, c.connections.GetStore().Update(connection))
		service.Labels[privatenetwork.EnvironmentKindLabel] = tc.kind
		require.NoError(t, c.services.GetStore().Update(service))
	}

	for _, tc := range []struct{ kind, callerDeployment, address string }{
		{"production", callerA, "10.0.0.1"},
		{"production", callerB, "10.0.0.2"},
		{"production", callerC, "10.0.0.3"},
		{"production", unknownCaller, ""},
		{"production", "", ""},
		{"preview", callerB, "10.0.0.2"},
		{"preview", callerC, "10.0.0.3"},
		{"preview", unknownCaller, ""},
		{"preview", "", ""},
	} {
		caller := identity
		caller.Deployment = tc.callerDeployment
		addresses, found, err := c.Resolve(caller, "payments")
		require.NoError(t, err)
		require.Equal(t, tc.address != "", found)
		if tc.address != "" {
			require.Equal(t, []netip.Addr{netip.MustParseAddr(tc.address)}, addresses)
		}
	}

	object, found, err := c.services.GetStore().GetByKey("default/preview-a")
	require.NoError(t, err)
	require.True(t, found)
	service := object.(*corev1.Service).DeepCopy()
	service.Labels[labels.LabelKeyAppID] = uid.New(uid.AppPrefix)
	require.NoError(t, c.services.GetStore().Update(service))
	identity.Deployment = callerB
	_, _, err = c.Resolve(identity, "payments")
	require.Error(t, err)
}

func TestCatalogEndpointsRequireReadyPrivateControllerOwnedAddresses(t *testing.T) {
	c := catalogForTest()
	service := addServiceAndSlice(t, c, testCaller(), "service-a", uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix), types.UID(uid.New(uid.TestPrefix)), "10.0.0.1", true)
	service.UID = types.UID(uid.New(uid.TestPrefix))
	require.NoError(t, c.services.GetStore().Update(service))
	_, err := c.endpoints(service, "")
	require.Error(t, err)

	addSlice(t, c, service, "imported", "10.0.0.2", true)
	addresses, err := c.endpoints(service, "")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.2")}, addresses)

	addSlice(t, c, service, "not-ready", "10.0.0.3", false)
	addSlice(t, c, service, "public", "192.0.2.1", true)
	addresses, err = c.endpoints(service, "")
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.2")}, addresses)
}

func TestConnectionSchemaAndUnresolvedTargets(t *testing.T) {
	c := catalogForTest()
	identity := testCaller()
	addConnection(t, c, "connection", identity, uid.New(uid.AppPrefix), "api", "", "", "1")

	_, found, err := c.Resolve(identity, "api")
	require.True(t, found)
	requireReason(t, err, ReasonConnectionUnresolved)

	object, found, err := c.connections.GetStore().GetByKey("default/connection")
	require.NoError(t, err)
	require.True(t, found)
	connection := object.(*corev1.ConfigMap).DeepCopy()
	connection.Data["schemaVersion"] = "2"
	require.NoError(t, c.connections.GetStore().Update(connection))

	_, found, err = c.Resolve(identity, "api")
	require.True(t, found, "a valid alias must remain indexed when its body is malformed")
	requireReason(t, err, ReasonConnectionInvalid)
}

func TestCatalogRejectsForgedServiceIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*corev1.Service)
	}{
		{"workspace", func(service *corev1.Service) {
			service.Labels[labels.LabelKeyWorkspaceID] = uid.New(uid.WorkspacePrefix)
		}},
		{"project", func(service *corev1.Service) { service.Labels[labels.LabelKeyProjectID] = uid.New(uid.ProjectPrefix) }},
		{"app", func(service *corev1.Service) { service.Labels[labels.LabelKeyAppID] = uid.New(uid.AppPrefix) }},
		{"caller deployment", func(service *corev1.Service) {
			service.Labels[labels.LabelKeyCallerDeploymentID] = uid.New(uid.DeploymentPrefix)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := catalogForTest()
			identity := testCaller()
			appID, deploymentID := uid.New(uid.AppPrefix), uid.New(uid.DeploymentPrefix)
			addConnection(t, c, "connection", identity, appID, "api", deploymentID, "target-service", "1")
			service := addServiceAndSlice(t, c, identity, "target-service", deploymentID, appID, types.UID(uid.New(uid.TestPrefix)), "10.0.0.1", true).DeepCopy()
			tc.mutate(service)
			require.NoError(t, c.services.GetStore().Update(service))

			_, found, err := c.Resolve(identity, "api")
			require.True(t, found)
			require.Error(t, err)
		})
	}
}

func catalogForTest() *Catalog {
	return &Catalog{
		pods:        informerForTest("pods", &corev1.Pod{}, cache.Indexers{podIPIndex: indexPodIP}),
		connections: informerForTest("connections", &corev1.ConfigMap{}, cache.Indexers{appIndex: indexConnection}),
		services:    informerForTest("services", &corev1.Service{}, nil),
		slices:      informerForTest("endpointslices", &discoveryv1.EndpointSlice{}, cache.Indexers{serviceIndex: indexSlice}),
		topology:    informerForTest("topology", &corev1.ConfigMap{}, nil),
		clock:       clock.New(),
	}
}

func testCaller() Caller {
	return Caller{Workspace: uid.New(uid.WorkspacePrefix), Project: uid.New(uid.ProjectPrefix), Deployment: uid.New(uid.DeploymentPrefix), Namespace: "default"}
}

func informerForTest(resource string, object runtime.Object, indexers cache.Indexers) *trackedInformer {
	return &trackedInformer{SharedIndexInformer: cache.NewSharedIndexInformer(nil, object, 0, indexers), resource: resource, clock: clock.New()}
}

func connectionData(t testing.TB, slug, deployment, service, revision string) map[string]string {
	t.Helper()
	parsed, err := strconv.ParseUint(revision, 10, 64)
	require.NoError(t, err)
	data, err := privatenetwork.Encode(privatenetwork.ConnectionData{Alias: slug, DeploymentID: deployment, ServiceName: service, Revision: parsed})
	require.NoError(t, err)
	return data
}

func addConnection(t *testing.T, c *Catalog, name string, caller Caller, appID, slug, deployment, service, revision string) {
	t.Helper()
	require.NoError(t, c.connections.GetStore().Add(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID(uid.New(uid.TestPrefix)), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: privatenetwork.DiscoveryComponent,
			labels.LabelKeyWorkspaceID: caller.Workspace, labels.LabelKeyProjectID: caller.Project,
			labels.LabelKeyAppID: appID, privatenetwork.EnvironmentKindLabel: "production",
			labels.LabelKeyCallerDeploymentID: caller.Deployment, labels.LabelKeyConnectionID: uid.New(uid.ConnectionPrefix),
		}},
		Data: connectionData(t, slug, deployment, service, revision),
	}))
}

func deleteConnection(t *testing.T, c *Catalog, name string) {
	t.Helper()
	object, exists, err := c.connections.GetStore().GetByKey("default/" + name)
	require.NoError(t, err)
	require.True(t, exists)
	require.NoError(t, c.connections.GetStore().Delete(object))
}

func addServiceAndSlice(t *testing.T, c *Catalog, caller Caller, name, deployment, appID string, serviceUID types.UID, address string, ready bool) *corev1.Service {
	t.Helper()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: serviceUID, Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: privatenetwork.DiscoveryComponent, labels.LabelKeyWorkspaceID: caller.Workspace,
			labels.LabelKeyProjectID: caller.Project, labels.LabelKeyAppID: appID,
			labels.LabelKeyDeploymentID: deployment,
		}},
		Spec: corev1.ServiceSpec{ClusterIP: corev1.ClusterIPNone},
	}
	require.NoError(t, c.services.GetStore().Add(service))
	addSlice(t, c, service, name, address, ready)
	return service
}

func addSlice(t *testing.T, c *Catalog, service *corev1.Service, name, address string, ready bool) {
	t.Helper()
	controller := true
	require.NoError(t, c.slices.GetStore().Add(&discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: service.Namespace,
			Labels:          map[string]string{discoveryv1.LabelServiceName: service.Name},
			OwnerReferences: []metav1.OwnerReference{{APIVersion: "v1", Kind: "Service", Name: service.Name, UID: service.UID, Controller: &controller}},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{address}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}}},
	}))
}

func TestCatalogWatchesOnlyKraneObjects(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	unrelated := metav1.ObjectMeta{Name: "unrelated", Namespace: "default", Labels: map[string]string{"app": "other"}}
	objects := append(discoveryObjects(t),
		&corev1.Pod{ObjectMeta: unrelated, Status: corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.9.0.1"}},
		&corev1.ConfigMap{ObjectMeta: unrelated},
		&corev1.Service{ObjectMeta: unrelated},
		&discoveryv1.EndpointSlice{ObjectMeta: unrelated, AddressType: discoveryv1.AddressTypeIPv4},
	)
	c, err := New(fake.NewClientset(objects...), 30*time.Second, clock.New())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	require.Eventually(t, c.Ready, 5*time.Second, 10*time.Millisecond)

	for name, informer := range map[string]*trackedInformer{"pods": c.pods, "connections": c.connections, "services": c.services, "endpointslices": c.slices} {
		keys := informer.GetStore().ListKeys()
		require.Len(t, keys, 1, "%s cache keys: %v", name, keys)
		require.NotEqual(t, "default/unrelated", keys[0], "%s cached an object Krane did not publish", name)
	}
}

// TestRunReportsWatchHealth guarantees that discovery health series exist as 0
// before the caches synchronize, and that a failed or silently stalled watch
// closes private discovery and is reported under its own resource label.
func TestRunReportsWatchHealth(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	var apiUnavailable atomic.Bool
	apiUnavailable.Store(true)
	client := fake.NewSimpleClientset(discoveryObjects(t)...)
	client.PrependReactor("list", "*", func(kubetesting.Action) (bool, runtime.Object, error) {
		if apiUnavailable.Load() {
			return true, nil, errors.New("API unavailable")
		}
		return false, nil, nil
	})
	c, err := New(client, time.Minute, clock.New())
	require.NoError(t, err)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- c.Run(ctx) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	resources := []string{"pods", "connections", "services", "endpointslices", "topology"}
	waitForHealth := func(phase string, ready float64, failed string) {
		t.Helper()
		require.Eventually(t, func() bool {
			if gaugeValues(t, "unkey_dns_discovery_ready")[""] != ready {
				return false
			}
			watches := gaugeValues(t, "unkey_dns_discovery_watch_healthy")
			for _, resource := range resources {
				want := 1.0
				if ready == 0 && (failed == "" || resource == failed) {
					want = 0
				}
				if value, exists := watches["resource="+resource]; !exists || value != want {
					return false
				}
			}
			return true
		}, 10*time.Second, 10*time.Millisecond, phase)
	}

	waitForHealth("cold", 0, "")
	require.False(t, c.Ready())

	apiUnavailable.Store(false)
	waitForHealth("synchronized", 1, "")
	require.True(t, c.Ready())

	for _, informer := range []*trackedInformer{c.pods, c.connections, c.services, c.slices} {
		informer.lastContact.Store(0)
		require.False(t, c.Ready(), "%s watch failed", informer.resource)
		waitForHealth(informer.resource+"-failed", 0, informer.resource)
		informer.lastContact.Store(time.Now().Add(-3 * time.Minute).UnixNano())
		require.False(t, c.Ready(), "%s watch stalled", informer.resource)
		waitForHealth(informer.resource+"-stale", 0, informer.resource)
		informer.lastContact.Store(time.Now().UnixNano())
		require.True(t, c.Ready(), "%s watch recovered", informer.resource)
		waitForHealth(informer.resource+"-recovered", 1, "")
	}
}
