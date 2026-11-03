package privatenetwork

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/cilium"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestReconcileOmitsIneligibleSourceAddressesBeforeMeshExport(t *testing.T) {
	connectionSpec := newTestConnection()
	ready := endpointPod(connectionSpec, "ready", "10.72.0.87")
	unready := endpointPod(connectionSpec, "unready", "10.72.0.94")
	unready.Status.Conditions[0].Status = corev1.ConditionFalse
	draining := endpointPod(connectionSpec, "draining", "10.72.0.84")
	now := metav1.Now()
	draining.DeletionTimestamp = &now

	client := fake.NewClientset(ready, unready, draining)
	control := &testutil.MockClusterClient{}
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
	})
	r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})

	require.NoError(t, converge(t, r))

	require.Empty(t, resolvedTarget(t, client, connectionSpec).Spec.Selector)
	require.Equal(t, []string{"10.72.0.87"}, resolvedAddresses(t, client, connectionSpec))

	endpointSlices, err := client.DiscoveryV1().EndpointSlices(connectionSpec.GetK8SNamespace()).List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, endpointSlices.Items, 1)
	require.Len(t, endpointSlices.Items[0].Endpoints, 1)
	require.Equal(t, []string{"10.72.0.87"}, endpointSlices.Items[0].Endpoints[0].Addresses)
}

func TestDiscoveryPublishesConnectionsWhileEndpointsRefresh(t *testing.T) {
	first, second := newTestConnection(), newTestConnection()
	second.WorkspaceId, second.ProjectId = first.WorkspaceId, first.ProjectId
	second.TargetAppId, second.TargetEnvironmentId = first.TargetAppId, first.TargetEnvironmentId
	unready := endpointPod(second, "b", "10.72.0.12")
	client := fake.NewClientset(endpointPod(first, "a", "10.72.0.11"), unready, endpointPod(second, "c", "10.72.0.13"))
	assignServiceUIDs(client)
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{first, second}, nil
	})}
	policies := testDynamicClient()
	r := newTestReconciler(t, Config{Client: client, Dynamic: policies, Cluster: control})
	for _, connectionSpec := range []*ctrlv1.PrivateNetworkConnection{first, second} {
		_, err := r.ensureService(t.Context(), connectionSpec, nil)
		require.NoError(t, err)
	}

	refreshed := false
	policies.PrependReactor("create", cilium.NetworkPolicyResource.Resource, func(clienttesting.Action) (bool, runtime.Object, error) {
		if !refreshed {
			refreshed = true
			unready.Status.Conditions[0].Status = corev1.ConditionFalse
			if _, err := client.CoreV1().Pods(unready.Namespace).UpdateStatus(t.Context(), unready, metav1.UpdateOptions{}); err != nil {
				return true, nil, err
			}
			syncKubernetesCaches(t, r)
			if err := r.reconcileEndpoints(t.Context()); err != nil {
				return true, nil, err
			}
		}
		return false, nil, nil
	})

	require.NoError(t, converge(t, r))
	require.True(t, refreshed)
	require.Equal(t, []string{"10.72.0.11"}, resolvedAddresses(t, client, first))
	require.Equal(t, []string{"10.72.0.13"}, resolvedAddresses(t, client, second))
}

func TestEndpointRefreshWithdrawsBeforeAddingReplacementAndKeepsImports(t *testing.T) {
	connectionSpec := newTestConnection()
	a := endpointPod(connectionSpec, "a", "10.72.0.84")
	b := endpointPod(connectionSpec, "b", "10.72.0.87")
	c := endpointPod(connectionSpec, "c", "10.72.0.253")
	replacement := endpointPod(connectionSpec, "replacement", "10.72.0.94")
	replacement.Status.Conditions[0].Status = corev1.ConditionFalse
	client := fake.NewClientset(a, b, c, replacement)
	r := newTestReconciler(t, Config{Client: client})
	service, err := r.ensureService(t.Context(), connectionSpec, nil)
	require.NoError(t, err)

	imported := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "remote", Namespace: service.Namespace, Labels: map[string]string{
			discoveryv1.LabelServiceName: service.Name,
			discoveryv1.LabelManagedBy:   meshSliceManager,
		}},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"10.73.0.7"}}},
	}
	imported, err = client.DiscoveryV1().EndpointSlices(service.Namespace).Create(t.Context(), imported, metav1.CreateOptions{})
	require.NoError(t, err)

	require.NoError(t, publishEndpoints(t, r))
	require.Equal(t, []string{"10.72.0.84", "10.72.0.87", "10.72.0.253"}, sourceAddresses(t, client, service))

	now := metav1.Now()
	a.DeletionTimestamp = &now
	_, err = client.CoreV1().Pods(a.Namespace).Update(t.Context(), a, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, publishEndpoints(t, r))
	require.Equal(t, []string{"10.72.0.87", "10.72.0.253"}, sourceAddresses(t, client, service))

	replacement.Status.Conditions[0].Status = corev1.ConditionTrue
	_, err = client.CoreV1().Pods(replacement.Namespace).UpdateStatus(t.Context(), replacement, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, publishEndpoints(t, r))
	require.Equal(t, []string{"10.72.0.87", "10.72.0.94", "10.72.0.253"}, sourceAddresses(t, client, service))

	for _, pod := range []*corev1.Pod{b, c, replacement} {
		pod.Status.Conditions[0].Status = corev1.ConditionFalse
		_, err = client.CoreV1().Pods(pod.Namespace).UpdateStatus(t.Context(), pod, metav1.UpdateOptions{})
		require.NoError(t, err)
	}
	require.NoError(t, publishEndpoints(t, r))
	require.Empty(t, sourceAddresses(t, client, service))

	remaining, err := client.DiscoveryV1().EndpointSlices(service.Namespace).Get(t.Context(), imported.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, imported, remaining)
}

func TestEndpointRefreshDoesNotWaitForControlPlane(t *testing.T) {
	connectionSpec := newTestConnection()
	pod := endpointPod(connectionSpec, "a", "10.72.0.84")
	client := fake.NewClientset(pod)
	control := &testutil.MockClusterClient{}
	r := newTestReconciler(t, Config{Client: client, Dynamic: testDynamicClient(), Cluster: control})
	service, err := r.ensureService(t.Context(), connectionSpec, nil)
	require.NoError(t, err)
	require.NoError(t, publishEndpoints(t, r))
	require.Len(t, sourceAddresses(t, client, service), 1)

	pod.Status.Conditions[0].Status = corev1.ConditionFalse
	_, err = client.CoreV1().Pods(pod.Namespace).UpdateStatus(t.Context(), pod, metav1.UpdateOptions{})
	require.NoError(t, err)

	entered := make(chan struct{})
	control.StreamPrivateNetworkStateFunc = snapshotFunc(t, func(ctx context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		close(entered)
		<-ctx.Done()
		return nil, ctx.Err()
	})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); r.Run(ctx) }()
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

func TestRunRepublishesEndpointsWhenPodsChange(t *testing.T) {
	connectionSpec := newTestConnection()
	pod := endpointPod(connectionSpec, "a", "10.72.0.84")
	client := fake.NewClientset(pod)
	clk := clock.NewTestClock()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{connectionSpec}, nil
	})}
	r := newTestReconciler(t, Config{Client: client, Cluster: control, Clock: clk})

	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() { defer close(done); r.Run(ctx) }()
	t.Cleanup(func() { cancel(); <-done })

	service := func() *corev1.Service {
		service, err := client.CoreV1().Services(connectionSpec.GetK8SNamespace()).Get(ctx, privatenetwork.ServiceName(connectionSpec.GetTargetDeploymentId(), connectionSpec.GetTargetPort()), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		require.NoError(t, err)
		return service
	}
	addresses := func(want ...string) func() bool {
		return func() bool {
			clk.Tick(delaySettle)
			current := service()
			return current != nil && slices.Equal(want, sourceAddresses(t, client, current))
		}
	}
	require.Eventually(t, addresses("10.72.0.84"), 10*time.Second, 10*time.Millisecond, "a Service discovery publishes gets its endpoints")

	pod.Status.Conditions[0].Status = corev1.ConditionFalse
	_, err := client.CoreV1().Pods(pod.Namespace).UpdateStatus(ctx, pod, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.Eventually(t, addresses(), 10*time.Second, 10*time.Millisecond, "a Pod change withdraws its endpoint")
}

func TestSourceSliceChunksShrinkAndRejectForeignOwnership(t *testing.T) {
	ctx := t.Context()
	connectionSpec := newTestConnection()
	client := fake.NewClientset()
	r := newTestReconciler(t, Config{Client: client})
	service, err := r.ensureService(ctx, connectionSpec, nil)
	require.NoError(t, err)
	service.UID = types.UID(uid.New(uid.TestPrefix))

	pods := make([]*corev1.Pod, 101)
	for i := range pods {
		pods[i], err = client.CoreV1().Pods(connectionSpec.GetK8SNamespace()).Create(ctx, endpointPod(connectionSpec, fmt.Sprintf("pod-%d", i), fmt.Sprintf("10.72.0.%d", i+1)), metav1.CreateOptions{})
		require.NoError(t, err)
	}

	require.NoError(t, ensureEndpointsNow(t, r, service))
	items, err := client.DiscoveryV1().EndpointSlices(service.Namespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, items.Items, 2)
	require.Len(t, items.Items[0].Endpoints, 100)
	require.Len(t, items.Items[1].Endpoints, 1)

	service.UID = types.UID(uid.New(uid.TestPrefix))
	for _, pod := range pods[:99] {
		require.NoError(t, client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{}))
	}
	require.NoError(t, ensureEndpointsNow(t, r, service))
	items, err = client.DiscoveryV1().EndpointSlices(service.Namespace).List(ctx, metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, items.Items, 1)
	require.Equal(t, service.UID, metav1.GetControllerOf(&items.Items[0]).UID)
	require.Equal(t, []string{"10.72.0.100", "10.72.0.101"}, sourceAddresses(t, client, service))

	client.ClearActions()
	require.NoError(t, ensureEndpointsNow(t, r, service))
	require.Empty(t, writeActions(client.Actions()))

	drifted := items.Items[0].DeepCopy()
	drifted.Labels[labels.LabelKeyAppID] = uid.New(uid.AppPrefix)
	_, err = client.DiscoveryV1().EndpointSlices(service.Namespace).Update(ctx, drifted, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, ensureEndpointsNow(t, r, service))
	restored, err := client.DiscoveryV1().EndpointSlices(service.Namespace).Get(ctx, drifted.Name, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, service.Labels[labels.LabelKeyAppID], restored.Labels[labels.LabelKeyAppID])

	foreign := restored.DeepCopy()
	foreign.OwnerReferences[0].Name = "other"
	_, err = client.DiscoveryV1().EndpointSlices(service.Namespace).Update(ctx, foreign, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.ErrorContains(t, ensureEndpointsNow(t, r, service), "foreign source EndpointSlice")
}

func TestReadyEndpointsExcludeWrongIdentityAndUnsafePodStates(t *testing.T) {
	connectionSpec := newTestConnection()
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{
		Namespace: connectionSpec.GetK8SNamespace(), Labels: targetAppLabels(connectionSpec).DeploymentID(connectionSpec.GetTargetDeploymentId()),
	}}

	for name, mutate := range map[string]func(*corev1.Pod){
		"uid":           func(p *corev1.Pod) { p.UID = "" },
		"host-network":  func(p *corev1.Pod) { p.Spec.HostNetwork = true },
		"pending":       func(p *corev1.Pod) { p.Status.Phase = corev1.PodPending },
		"missing-ready": func(p *corev1.Pod) { p.Status.Conditions = nil },
		"ipv6":          func(p *corev1.Pod) { p.Status.PodIPs = []corev1.PodIP{{IP: "fd00::1"}} },
	} {
		t.Run(name, func(t *testing.T) {
			pod := endpointPod(connectionSpec, "excluded", "10.72.0.84")
			mutate(pod)
			endpoints := readyEndpoints(service, []*corev1.Pod{pod, endpointPod(connectionSpec, "survivor", "10.72.0.87")}, "")
			require.Len(t, endpoints, 1)
			require.Equal(t, []string{"10.72.0.87"}, endpoints[0].Addresses)
			require.Empty(t, readyEndpoints(service, []*corev1.Pod{pod}, ""))
		})
	}

	sharedAddressPod := endpointPod(connectionSpec, "shared-address", "100.64.0.9")
	require.Equal(t, []string{"100.64.0.9"}, readyEndpoints(service, []*corev1.Pod{sharedAddressPod}, "")[0].Addresses, "pods may use the shared address space")

	for _, key := range []string{
		labels.LabelKeyManagedBy, labels.LabelKeyComponent, labels.LabelKeyWorkspaceID,
		labels.LabelKeyProjectID, labels.LabelKeyAppID, labels.LabelKeyDeploymentID,
	} {
		t.Run(key, func(t *testing.T) {
			pod := endpointPod(connectionSpec, "excluded", "10.72.0.84")
			pod.Labels[key] = "other"
			require.Empty(t, readyEndpoints(service, []*corev1.Pod{pod}, ""))
		})
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

func endpointPod(connectionSpec *ctrlv1.PrivateNetworkConnection, name, ip string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: connectionSpec.GetK8SNamespace(), UID: types.UID(uid.New(uid.TestPrefix)),
			Labels: targetAppLabels(connectionSpec).DeploymentID(connectionSpec.GetTargetDeploymentId()).ComponentDeployment(),
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning, PodIP: ip,
			PodIPs:     []corev1.PodIP{{IP: ip}},
			Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionTrue}},
		},
	}
}

const meshSliceManager = "endpointslice-mesh-controller.cilium.io"

func readySlice(service *corev1.Service, name, manager string) *discoveryv1.EndpointSlice {
	ready := true
	sliceLabels := make(map[string]string, len(service.Labels)+2)
	for key, value := range service.Labels {
		sliceLabels[key] = value
	}
	sliceLabels[discoveryv1.LabelServiceName] = service.Name
	sliceLabels[discoveryv1.LabelManagedBy] = manager
	return &discoveryv1.EndpointSlice{
		TypeMeta:    metav1.TypeMeta{APIVersion: discoveryv1.SchemeGroupVersion.String(), Kind: "EndpointSlice"},
		ObjectMeta:  metav1.ObjectMeta{Name: name, Namespace: service.Namespace, Labels: sliceLabels, OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(service, corev1.SchemeGroupVersion.WithKind("Service"))}},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"10.72.0.11"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}}},
	}
}

func ensureEndpointsNow(t *testing.T, r *Reconciler, service *corev1.Service) error {
	t.Helper()
	syncCaches(t, r)
	return r.ensureEndpoints(t.Context(), service)
}
