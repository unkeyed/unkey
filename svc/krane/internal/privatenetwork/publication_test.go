package privatenetwork

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/deploy/appbinding"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPublicationRetainsBindingUntilRemoteDiscoveryIsReady(t *testing.T) {
	ctx := t.Context()
	selected := testApp("dep_a")
	client := fake.NewClientset(endpointPod(selected, "a", "10.72.0.11"))
	other := testApp("other_a")
	other.AppId, other.AppSlug = "app_2", "metrics"
	other.BindingId, other.BindingName, other.CallerDeploymentId = "binding_2", "metrics-api", "caller_2"
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error) {
		return []*ctrlv1.PrivateNetworkApp{selected, other}, nil
	})}
	dynamic := testDynamicClient()
	r := &Reconciler{client: client, dynamic: dynamic, cluster: control}
	require.NoError(t, r.reconcile(ctx))
	bindingName := resourceName("unkey-pn-binding", "binding_1/caller_1")
	original, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, bindingName, metav1.GetOptions{})
	require.NoError(t, err)

	selected = testApp("dep_b")
	other.DeploymentId = "other_b"
	_, err = client.CoreV1().Pods("customer-1").Create(ctx, endpointPod(other, "other-b", "10.72.0.33"), metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcile(ctx))
	staged, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, bindingName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, original, staged, "B's empty local slice must not replace the durable A binding")
	otherBinding, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, resourceName("unkey-pn-binding", "binding_2/caller_2"), metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "other_b", otherBinding.Data["deploymentId"])
	require.Equal(t, "2", otherBinding.Data["revision"])

	r = &Reconciler{client: client, dynamic: dynamic, cluster: control, now: func() time.Time { return time.Now().Add(time.Hour) }}
	require.NoError(t, r.reconcile(ctx))

	a, err := client.CoreV1().Services("customer-1").Get(ctx, original.Data["serviceName"], metav1.GetOptions{})
	require.NoError(t, err)
	require.NotContains(t, a.Annotations, appbinding.RetireAfterAnnotation)
	require.Equal(t, []string{"10.72.0.11"}, sourceAddresses(t, client, a))

	b, err := client.CoreV1().Services("customer-1").Get(ctx, discoveryName("dep_b", selected.GetPort()), metav1.GetOptions{})
	require.NoError(t, err)
	ready := false
	remote := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "remote-b", Namespace: b.Namespace,
			Labels:          map[string]string{discoveryv1.LabelServiceName: b.Name, discoveryv1.LabelManagedBy: "endpointslice-mesh-controller.cilium.io"},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(b, corev1.SchemeGroupVersion.WithKind("Service"))},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"10.73.0.22"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}}},
	}
	remote, err = client.DiscoveryV1().EndpointSlices(b.Namespace).Create(ctx, remote, metav1.CreateOptions{})
	require.NoError(t, err)

	require.NoError(t, r.reconcile(ctx))
	staged, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, bindingName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, original, staged)

	ready = true
	remote.Endpoints[0].Conditions.Ready = &ready

	for _, tc := range []struct {
		name   string
		change func(*discoveryv1.EndpointSlice)
	}{
		{"wrong-owner", func(s *discoveryv1.EndpointSlice) { s.OwnerReferences[0].UID = "another-service" }},
		{"terminating", func(s *discoveryv1.EndpointSlice) { s.Endpoints[0].Conditions.Terminating = &ready }},
		{"public-address", func(s *discoveryv1.EndpointSlice) { s.Endpoints[0].Addresses = []string{"192.0.2.22"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := remote.DeepCopy()
			tc.change(invalid)
			_, err := client.DiscoveryV1().EndpointSlices(b.Namespace).Update(ctx, invalid, metav1.UpdateOptions{})
			require.NoError(t, err)
			require.NoError(t, r.reconcile(ctx))
			staged, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, bindingName, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, original.Data, staged.Data)
		})
	}

	_, err = client.DiscoveryV1().EndpointSlices(b.Namespace).Update(ctx, remote, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcile(ctx))

	published, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, bindingName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "dep_b", published.Data["deploymentId"])
	require.Equal(t, "2", published.Data["revision"])

	a, err = client.CoreV1().Services("customer-1").Get(ctx, original.Data["serviceName"], metav1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, a.Annotations, appbinding.RetireAfterAnnotation)
}
