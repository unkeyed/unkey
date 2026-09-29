package undns

import (
	"context"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/features"
	featuretesting "k8s.io/client-go/features/testing"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/tools/cache"
)

func TestCatalogResolveIsolatesBindingsAndFollowsPromotion(t *testing.T) {
	c := catalogForTest()
	identity := caller{workspace: "workspace-a", project: "project-a", kind: "production", deployment: "caller-deployment-a", namespace: "default"}

	addBinding(t, c, "binding-a", "workspace-a", "project-a", "app-a", "payments", "deployment-a", "service-a", "1")
	addBinding(t, c, "binding-other-workspace", "workspace-b", "project-a", "app-b", "payments", "deployment-other", "service-other", "1")
	addBinding(t, c, "binding-other-project", "workspace-a", "project-b", "app-c", "payments", "deployment-other", "service-other", "1")
	addServiceAndSlice(t, c, "service-a", "deployment-a", "app-a", types.UID("service-a-uid"), "10.0.0.1", true)
	addServiceAndSlice(t, c, "service-b", "deployment-b", "app-a", types.UID("service-b-uid"), "10.0.0.2", true)

	addresses, exists, err := c.resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.1")}, addresses)

	deleteBinding(t, c, "binding-a")
	addBinding(t, c, "binding-a", "workspace-a", "project-a", "app-a", "payments", "deployment-b", "service-b", "2")
	addresses, exists, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.2")}, addresses)

	deleteBinding(t, c, "binding-a")
	addBinding(t, c, "binding-a", "workspace-a", "project-a", "app-a", "payments", "deployment-a", "service-a", "3")
	addresses, exists, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.True(t, exists)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.1")}, addresses)

	deleteBinding(t, c, "binding-a")
	_, exists, err = c.resolve(identity, "payments")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestDirectedBindingsIsolateCallerDeployments(t *testing.T) {
	c := catalogForTest()
	for _, tc := range []struct{ callerDeployment, name, kind, address string }{
		{"caller-deployment-a", "production", "production", "10.0.0.1"},
		{"caller-deployment-b", "preview-a", "production", "10.0.0.2"},
		{"caller-deployment-c", "preview-b", "preview", "10.0.0.3"},
	} {
		addBinding(t, c, tc.name, "workspace-a", "project-a", "app-a", "payments", tc.name, tc.name, "1")
		service := addServiceAndSlice(t, c, tc.name, tc.name, "app-a", types.UID(tc.name), tc.address, true).DeepCopy()
		object, found, err := c.bindings.GetStore().GetByKey("default/" + tc.name)
		require.NoError(t, err)
		require.True(t, found)
		binding := object.(*corev1.ConfigMap).DeepCopy()
		binding.Labels[environmentKindLabel] = tc.kind
		binding.Labels[labels.LabelKeyCallerDeploymentID] = tc.callerDeployment
		require.NoError(t, c.bindings.GetStore().Update(binding))
		service.Labels[environmentKindLabel] = tc.kind
		require.NoError(t, c.services.GetStore().Update(service))
	}

	for _, tc := range []struct{ kind, callerDeployment, address string }{
		{"production", "caller-deployment-a", "10.0.0.1"},
		{"production", "caller-deployment-b", "10.0.0.2"},
		{"production", "caller-deployment-c", "10.0.0.3"},
		{"production", "unknown", ""},
		{"production", "", ""},
		{"preview", "caller-deployment-b", "10.0.0.2"},
		{"preview", "caller-deployment-c", "10.0.0.3"},
		{"preview", "unknown", ""},
		{"preview", "", ""},
	} {
		addresses, found, err := c.resolve(caller{workspace: "workspace-a", project: "project-a", kind: tc.kind, deployment: tc.callerDeployment, namespace: "default"}, "payments")
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
	service.Labels[labels.LabelKeyAppID] = "forged-app"
	require.NoError(t, c.services.GetStore().Update(service))
	_, _, err = c.resolve(caller{workspace: "workspace-a", project: "project-a", kind: "preview", deployment: "caller-deployment-b", namespace: "default"}, "payments")
	require.Error(t, err)
}

func TestCatalogEndpointsRequireReadyPrivateControllerOwnedAddresses(t *testing.T) {
	c := catalogForTest()
	service := addServiceAndSlice(t, c, "service-a", "deployment-a", "app-a", types.UID("old-uid"), "10.0.0.1", true)
	service.UID = types.UID("new-uid")
	require.NoError(t, c.services.GetStore().Update(service))
	_, err := c.endpoints(service)
	require.Error(t, err)

	addSlice(t, c, service, "imported", "10.0.0.2", true)
	addresses, err := c.endpoints(service)
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.2")}, addresses)

	addSlice(t, c, service, "not-ready", "10.0.0.3", false)
	addSlice(t, c, service, "public", "192.0.2.1", true)
	addresses, err = c.endpoints(service)
	require.NoError(t, err)
	require.Equal(t, []netip.Addr{netip.MustParseAddr("10.0.0.2")}, addresses)
}

func TestCatalogRejectsForgedServiceIdentity(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*corev1.Service)
	}{
		{"workspace", func(service *corev1.Service) { service.Labels[labels.LabelKeyWorkspaceID] = "forged" }},
		{"project", func(service *corev1.Service) { service.Labels[labels.LabelKeyProjectID] = "forged" }},
		{"app", func(service *corev1.Service) { service.Labels[labels.LabelKeyAppID] = "forged" }},
		{"caller deployment", func(service *corev1.Service) { service.Labels[labels.LabelKeyCallerDeploymentID] = "forged" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := catalogForTest()
			identity := testCaller()
			addBinding(t, c, "binding", identity.workspace, identity.project, "target-app", "api", "target-deployment", "target-service", "1")
			service := addServiceAndSlice(t, c, "target-service", "target-deployment", "target-app", "service-uid", "10.0.0.1", true).DeepCopy()
			tc.mutate(service)
			require.NoError(t, c.services.GetStore().Update(service))

			_, found, err := c.resolve(identity, "api")
			require.True(t, found)
			require.Error(t, err)
		})
	}
}

func catalogForTest() *catalog {
	return &catalog{
		pods:     informerForTest(&corev1.Pod{}, cache.Indexers{podIPIndex: indexPodIP}),
		bindings: informerForTest(&corev1.ConfigMap{}, cache.Indexers{appIndex: indexBinding}),
		services: informerForTest(&corev1.Service{}, nil),
		slices:   informerForTest(&discoveryv1.EndpointSlice{}, cache.Indexers{serviceIndex: indexSlice}),
		active:   make(map[string]*corev1.ConfigMap),
		now:      time.Now,
	}
}

func testCaller() caller {
	return caller{workspace: "workspace-a", project: "project-a", deployment: "caller-deployment-a", namespace: "default"}
}

func informerForTest(object runtime.Object, indexers cache.Indexers) *trackedInformer {
	return &trackedInformer{SharedIndexInformer: cache.NewSharedIndexInformer(nil, object, 0, indexers)}
}

func addBinding(t *testing.T, c *catalog, name, workspace, project, appID, slug, deployment, service, revision string) {
	t.Helper()
	require.NoError(t, c.bindings.GetStore().Add(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: types.UID(name), Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: bindingComponent,
			labels.LabelKeyWorkspaceID: workspace, labels.LabelKeyProjectID: project,
			labels.LabelKeyAppID: appID, environmentKindLabel: "production",
			labels.LabelKeyCallerDeploymentID: "caller-deployment-a", labels.LabelKeyBindingID: name,
		}},
		Data: map[string]string{"appSlug": slug, "deploymentId": deployment, "serviceName": service, "revision": revision},
	}))
}

func deleteBinding(t *testing.T, c *catalog, name string) {
	t.Helper()
	object, exists, err := c.bindings.GetStore().GetByKey("default/" + name)
	require.NoError(t, err)
	require.True(t, exists)
	require.NoError(t, c.bindings.GetStore().Delete(object))
}

func addServiceAndSlice(t *testing.T, c *catalog, name, deployment, appID string, uid types.UID, address string, ready bool) *corev1.Service {
	t.Helper()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "default", UID: uid, Labels: map[string]string{
			labels.LabelKeyManagedBy: "krane", labels.LabelKeyComponent: bindingComponent, labels.LabelKeyWorkspaceID: "workspace-a",
			labels.LabelKeyProjectID: "project-a", labels.LabelKeyAppID: appID,
			labels.LabelKeyDeploymentID: deployment,
		}},
		Spec: corev1.ServiceSpec{ClusterIP: corev1.ClusterIPNone},
	}
	require.NoError(t, c.services.GetStore().Add(service))
	addSlice(t, c, service, name, address, ready)
	return service
}

func addSlice(t *testing.T, c *catalog, service *corev1.Service, name, address string, ready bool) {
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

// TestCatalogWatchesOnlyKraneObjects guarantees that undns caches only the
// Pods and discovery objects Krane publishes, so its memory tracks private
// networking instead of every object in the cluster.
func TestCatalogWatchesOnlyKraneObjects(t *testing.T) {
	featuretesting.SetFeatureDuringTest(t, features.WatchListClient, false)
	unrelated := metav1.ObjectMeta{Name: "unrelated", Namespace: "default", Labels: map[string]string{"app": "other"}}
	objects := append(discoveryObjects(t),
		&corev1.Pod{ObjectMeta: unrelated, Status: corev1.PodStatus{Phase: corev1.PodRunning, PodIP: "10.9.0.1"}},
		&corev1.ConfigMap{ObjectMeta: unrelated},
		&corev1.Service{ObjectMeta: unrelated},
		&discoveryv1.EndpointSlice{ObjectMeta: unrelated, AddressType: discoveryv1.AddressTypeIPv4},
	)
	c, err := newCatalog(fake.NewClientset(objects...), 30*time.Second)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- c.run(ctx) }()
	t.Cleanup(func() {
		cancel()
		require.NoError(t, <-done)
	})

	require.Eventually(t, c.readyDiscovery, 5*time.Second, 10*time.Millisecond)

	for name, informer := range map[string]*trackedInformer{"pods": c.pods, "bindings": c.bindings, "services": c.services, "endpointslices": c.slices} {
		keys := informer.GetStore().ListKeys()
		require.Len(t, keys, 1, "%s cache keys: %v", name, keys)
		require.NotEqual(t, "default/unrelated", keys[0], "%s cached an object Krane did not publish", name)
	}
}
