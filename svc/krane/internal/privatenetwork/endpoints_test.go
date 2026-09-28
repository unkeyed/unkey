package privatenetwork

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestReconcileOmitsIneligibleSourceAddressesBeforeMeshExport(t *testing.T) {
	app := testApp("dep_a")
	ready := endpointPod(app, "ready", "10.72.0.87")
	unready := endpointPod(app, "unready", "10.72.0.94")
	unready.Status.Conditions[0].Status = corev1.ConditionFalse
	draining := endpointPod(app, "draining", "10.72.0.84")
	now := metav1.Now()
	draining.DeletionTimestamp = &now
	client := fake.NewClientset(ready, unready, draining)
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
		return []*ctrlv1.PrivateNetworkApp{app}, nil
	})
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{}}
	require.NoError(t, r.reconcile(t.Context()))
	services, err := client.CoreV1().Services(app.GetK8SNamespace()).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, services.Items, 1)
	require.Empty(t, services.Items[0].Spec.Selector)
	slices, err := client.DiscoveryV1().EndpointSlices(app.GetK8SNamespace()).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, slices.Items, 1)
	require.Len(t, slices.Items[0].Endpoints, 1)
	require.Equal(t, []string{"10.72.0.87"}, slices.Items[0].Endpoints[0].Addresses)
}

func TestEndpointRefreshWithdrawsBeforeAddingReplacementAndKeepsImports(t *testing.T) {
	app := testApp("dep_a")
	a := endpointPod(app, "a", "10.72.0.84")
	b := endpointPod(app, "b", "10.72.0.87")
	c := endpointPod(app, "c", "10.72.0.253")
	replacement := endpointPod(app, "replacement", "10.72.0.94")
	replacement.Status.Conditions[0].Status = corev1.ConditionFalse
	client := fake.NewClientset(a, b, c, replacement)
	r := &Reconciler{client: client}
	service, err := r.ensureService(t.Context(), app, discoveryName(app.GetDeploymentId(), app.GetPort()), nil)
	require.NoError(t, err)
	imported := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "remote", Namespace: service.Namespace, Labels: map[string]string{
			discoveryv1.LabelServiceName: service.Name,
			discoveryv1.LabelManagedBy:   "endpointslice-mesh-controller.cilium.io",
		}},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"10.73.0.7"}}},
	}
	imported, err = client.DiscoveryV1().EndpointSlices(service.Namespace).Create(t.Context(), imported, metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpoints(t.Context()))
	require.Equal(t, []string{"10.72.0.84", "10.72.0.87", "10.72.0.253"}, sourceAddresses(t, client, service))
	now := metav1.Now()
	a.DeletionTimestamp = &now
	_, err = client.CoreV1().Pods(a.Namespace).Update(t.Context(), a, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpoints(t.Context()))
	require.Equal(t, []string{"10.72.0.87", "10.72.0.253"}, sourceAddresses(t, client, service))
	replacement.Status.Conditions[0].Status = corev1.ConditionTrue
	_, err = client.CoreV1().Pods(replacement.Namespace).UpdateStatus(t.Context(), replacement, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpoints(t.Context()))
	require.Equal(t, []string{"10.72.0.87", "10.72.0.94", "10.72.0.253"}, sourceAddresses(t, client, service))
	for _, pod := range []*corev1.Pod{b, c, replacement} {
		pod.Status.Conditions[0].Status = corev1.ConditionFalse
		_, err = client.CoreV1().Pods(pod.Namespace).UpdateStatus(t.Context(), pod, metav1.UpdateOptions{})
		require.NoError(t, err)
	}
	require.NoError(t, r.reconcileEndpoints(t.Context()))
	require.Empty(t, sourceAddresses(t, client, service))
	remaining, err := client.DiscoveryV1().EndpointSlices(service.Namespace).Get(t.Context(), imported.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, imported, remaining)
}

func TestEndpointRefreshDoesNotWaitForControlPlane(t *testing.T) {
	app := testApp("dep_a")
	pod := endpointPod(app, "a", "10.72.0.84")
	client := fake.NewClientset(pod)
	control := &testutil.MockClusterClient{}
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control}
	service, err := r.ensureService(t.Context(), app, discoveryName(app.GetDeploymentId(), app.GetPort()), nil)
	require.NoError(t, err)
	require.NoError(t, r.reconcileEndpoints(t.Context()))
	require.Len(t, sourceAddresses(t, client, service), 1)
	pod.Status.Conditions[0].Status = corev1.ConditionFalse
	_, err = client.CoreV1().Pods(pod.Namespace).UpdateStatus(t.Context(), pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	entered := make(chan struct{})
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(ctx context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); r.run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("control-plane request did not start")
	}
	require.Eventually(t, func() bool {
		items, err := client.DiscoveryV1().EndpointSlices(service.Namespace).List(ctx, metav1.ListOptions{})
		return err == nil && len(items.Items) == 1 && len(items.Items[0].Endpoints) == 0
	}, 2*time.Second, 10*time.Millisecond)
}

func TestSourceSliceChunksShrinkAndRejectForeignOwnership(t *testing.T) {
	app := testApp("dep_a")
	client := fake.NewClientset()
	r := &Reconciler{client: client}
	service, err := r.ensureService(t.Context(), app, "source", nil)
	require.NoError(t, err)
	service.UID = "new-service"
	pods := make([]corev1.Pod, 101)
	for i := range pods {
		pods[i] = *endpointPod(app, fmt.Sprintf("pod-%d", i), fmt.Sprintf("10.72.0.%d", i+1))
	}
	require.NoError(t, ensureEndpointsNow(t, r, service, pods))
	items, err := client.DiscoveryV1().EndpointSlices(service.Namespace).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, items.Items, 2)
	require.Len(t, items.Items[0].Endpoints, 100)
	require.Len(t, items.Items[1].Endpoints, 1)
	service.UID = "recreated-service"
	slices.Reverse(pods)
	require.NoError(t, ensureEndpointsNow(t, r, service, pods[:2]))
	items, err = client.DiscoveryV1().EndpointSlices(service.Namespace).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, items.Items, 1)
	require.Equal(t, service.UID, metav1.GetControllerOf(&items.Items[0]).UID)
	require.Equal(t, []string{"10.72.0.100", "10.72.0.101"}, sourceAddresses(t, client, service))
	client.ClearActions()
	require.NoError(t, ensureEndpointsNow(t, r, service, pods[:2]))
	for _, action := range client.Actions() {
		require.Equal(t, "list", action.GetVerb())
	}
	foreign := items.Items[0].DeepCopy()
	foreign.Labels[labels.LabelKeyAppID] = "other-app"
	_, err = client.DiscoveryV1().EndpointSlices(service.Namespace).Update(t.Context(), foreign, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.ErrorContains(t, ensureEndpointsNow(t, r, service, nil), "foreign source EndpointSlice")
}

func TestSourceMigrationPublishesSlicesBeforeBinding(t *testing.T) {
	app := testApp("dep_a")
	client := fake.NewClientset(endpointPod(app, "a", "10.72.0.87"))
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
		return []*ctrlv1.PrivateNetworkApp{app}, nil
	})
	r := &Reconciler{client: client, dynamic: testDynamicClient(), cluster: control}
	oldName := resourceName("unkey-pn", app.GetDeploymentId())
	oldService, err := r.ensureService(t.Context(), app, oldName, nil)
	require.NoError(t, err)
	oldService.Spec.Selector = appLabels(app).DeploymentID(app.GetDeploymentId())
	_, err = client.CoreV1().Services(app.GetK8SNamespace()).Update(t.Context(), oldService, metav1.UpdateOptions{})
	require.NoError(t, err)
	bindingName := resourceName("unkey-pn-binding", "binding_1/caller_1")
	_, err = r.ensureBinding(t.Context(), app, bindingName, oldService, nil)
	require.NoError(t, err)
	deny := true
	client.PrependReactor("create", "endpointslices", func(clienttesting.Action) (bool, runtime.Object, error) {
		if deny {
			return true, nil, fmt.Errorf("endpoint write denied")
		}
		return false, nil, nil
	})
	require.ErrorContains(t, r.reconcile(t.Context()), "endpoint write denied")
	binding, err := client.CoreV1().ConfigMaps(app.GetK8SNamespace()).Get(t.Context(), bindingName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, oldName, binding.Data["serviceName"])
	deny = false
	require.NoError(t, r.reconcile(t.Context()))
	binding, err = client.CoreV1().ConfigMaps(app.GetK8SNamespace()).Get(t.Context(), bindingName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, discoveryName(app.GetDeploymentId(), app.GetPort()), binding.Data["serviceName"])
	require.Equal(t, "2", binding.Data["revision"])
}

func TestReadyEndpointsExcludeWrongIdentityAndUnsafePodStates(t *testing.T) {
	app := testApp("dep_a")
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Namespace: app.GetK8SNamespace(), Labels: appLabels(app).DeploymentID(app.GetDeploymentId()),
	}}
	for name, mutate := range map[string]func(*corev1.Pod){
		"namespace":     func(p *corev1.Pod) { p.Namespace = "other" },
		"uid":           func(p *corev1.Pod) { p.UID = "" },
		"host-network":  func(p *corev1.Pod) { p.Spec.HostNetwork = true },
		"pending":       func(p *corev1.Pod) { p.Status.Phase = corev1.PodPending },
		"missing-ready": func(p *corev1.Pod) { p.Status.Conditions = nil },
		"public-ip":     func(p *corev1.Pod) { p.Status.PodIPs = []corev1.PodIP{{IP: "8.8.8.8"}} },
		"ipv6":          func(p *corev1.Pod) { p.Status.PodIPs = []corev1.PodIP{{IP: "fd00::1"}} },
	} {
		t.Run(name, func(t *testing.T) {
			pod := endpointPod(app, "excluded", "10.72.0.84")
			mutate(pod)
			endpoints := readyEndpoints(service, []corev1.Pod{
				*pod, *endpointPod(app, "survivor", "10.72.0.87"),
			})
			require.Len(t, endpoints, 1)
			require.Equal(t, []string{"10.72.0.87"}, endpoints[0].Addresses)
			require.Empty(t, readyEndpoints(service, []corev1.Pod{*pod}))
		})
	}
	for _, key := range []string{
		labels.LabelKeyManagedBy, labels.LabelKeyComponent, labels.LabelKeyWorkspaceID,
		labels.LabelKeyProjectID, labels.LabelKeyAppID, labels.LabelKeyDeploymentID,
	} {
		t.Run(key, func(t *testing.T) {
			pod := endpointPod(app, "excluded", "10.72.0.84")
			pod.Labels[key] = "other"
			require.Empty(t, readyEndpoints(service, []corev1.Pod{*pod}))
		})
	}
	for _, kind := range []string{"production", "preview"} {
		pod := endpointPod(app, "same-network", "10.72.0.84")
		pod.Labels[labels.LabelKeyEnvironmentKind] = kind
		endpoints := readyEndpoints(service, []corev1.Pod{*pod})
		require.Len(t, endpoints, 1)
		require.Equal(t, []string{"10.72.0.84"}, endpoints[0].Addresses)
	}
}

func sourceAddresses(t *testing.T, client *fake.Clientset, service *corev1.Service) []string {
	t.Helper()
	items, err := client.DiscoveryV1().EndpointSlices(service.Namespace).List(t.Context(), metav1.ListOptions{LabelSelector: labels.Labels{
		discoveryv1.LabelManagedBy: sourceSliceManager, discoveryv1.LabelServiceName: service.Name,
	}.ToString()})
	require.NoError(t, err)
	var addresses []string
	for _, item := range items.Items {
		for _, endpoint := range item.Endpoints {
			require.True(t, *endpoint.Conditions.Ready)
			require.False(t, *endpoint.Conditions.Terminating)
			addresses = append(addresses, endpoint.Addresses...)
		}
	}
	return addresses
}

func endpointPod(app *ctrlv1.PrivateNetworkApp, name, ip string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: app.GetK8SNamespace(), UID: types.UID(name),
			Labels: appLabels(app).DeploymentID(app.GetDeploymentId()).ComponentDeployment(),
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning, PodIP: ip,
			PodIPs:     []corev1.PodIP{{IP: ip}},
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
		},
	}
}

func ensureEndpointsNow(t *testing.T, r *Reconciler, service *corev1.Service, pods []corev1.Pod) error {
	t.Helper()
	published, err := r.sourceSlices(t.Context())
	if err != nil {
		return err
	}
	return r.ensureEndpoints(t.Context(), service, pods, published[service.Namespace+"/"+service.Name])
}
