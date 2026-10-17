package privatenetwork

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	corev1 "k8s.io/api/core/v1"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPublicationRetainsConnectionUntilRemoteDiscoveryIsReady(t *testing.T) {
	ctx := t.Context()
	selected := newTestConnection()
	client := fake.NewClientset(endpointPod(selected, "a", "10.72.0.11"))
	other := newTestConnection()
	other.WorkspaceId, other.ProjectId = selected.WorkspaceId, selected.ProjectId
	other.TargetAppSlug, other.ConnectionName = "metrics", "metrics-api"
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: snapshotFunc(t, func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error) {
		return []*ctrlv1.PrivateNetworkConnection{selected, other}, nil
	})}
	dynamic := testDynamicClient()
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: dynamic, cluster: control}
	require.NoError(t, r.reconcile(ctx))
	connectionName := connectionResourceName(selected)
	original, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
	require.NoError(t, err)

	selected.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	other.TargetDeploymentId = uid.New(uid.DeploymentPrefix)
	_, err = client.CoreV1().Pods("customer-1").Create(ctx, endpointPod(other, "other-b", "10.72.0.33"), metav1.CreateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcile(ctx))
	staged, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, original, staged, "B's empty local slice must not replace the durable A connection")
	otherConnection, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionResourceName(other), metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, other.TargetDeploymentId, otherConnection.Data["deploymentId"])
	require.Equal(t, "2", otherConnection.Data["revision"])

	r = &Reconciler{client: client, dynamic: dynamic, cluster: control, clock: clock.NewTestClock(time.Now().Add(time.Hour))}
	require.NoError(t, r.reconcile(ctx))

	a, err := client.CoreV1().Services("customer-1").Get(ctx, original.Data["serviceName"], metav1.GetOptions{})
	require.NoError(t, err)
	require.NotContains(t, a.Annotations, privatenetwork.RetireAfterAnnotation)
	require.Equal(t, []string{"10.72.0.11"}, sourceAddresses(t, client, a))

	b, err := client.CoreV1().Services("customer-1").Get(ctx, discoveryName(selected.TargetDeploymentId, selected.GetTargetPort()), metav1.GetOptions{})
	require.NoError(t, err)
	ready := false
	remote := &discoveryv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{Name: "remote-b", Namespace: b.Namespace,
			Labels:          map[string]string{discoveryv1.LabelServiceName: b.Name, discoveryv1.LabelManagedBy: meshSliceManager},
			OwnerReferences: []metav1.OwnerReference{*metav1.NewControllerRef(b, corev1.SchemeGroupVersion.WithKind("Service"))},
		},
		AddressType: discoveryv1.AddressTypeIPv4,
		Endpoints:   []discoveryv1.Endpoint{{Addresses: []string{"10.73.0.22"}, Conditions: discoveryv1.EndpointConditions{Ready: &ready}}},
	}
	remote, err = client.DiscoveryV1().EndpointSlices(b.Namespace).Create(ctx, remote, metav1.CreateOptions{})
	require.NoError(t, err)

	require.NoError(t, r.reconcile(ctx))
	staged, err = client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, original, staged)

	ready = true
	remote.Endpoints[0].Conditions.Ready = &ready

	for _, tc := range []struct {
		name   string
		change func(*discoveryv1.EndpointSlice)
	}{
		{"wrong-owner", func(s *discoveryv1.EndpointSlice) { s.OwnerReferences[0].UID = types.UID(uid.New(uid.TestPrefix)) }},
		{"terminating", func(s *discoveryv1.EndpointSlice) { s.Endpoints[0].Conditions.Terminating = &ready }},
		{"public-address", func(s *discoveryv1.EndpointSlice) { s.Endpoints[0].Addresses = []string{"192.0.2.22"} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			invalid := remote.DeepCopy()
			tc.change(invalid)
			_, err := client.DiscoveryV1().EndpointSlices(b.Namespace).Update(ctx, invalid, metav1.UpdateOptions{})
			require.NoError(t, err)
			require.NoError(t, r.reconcile(ctx))
			staged, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, original.Data, staged.Data)
		})
	}

	_, err = client.DiscoveryV1().EndpointSlices(b.Namespace).Update(ctx, remote, metav1.UpdateOptions{})
	require.NoError(t, err)
	require.NoError(t, r.reconcile(ctx))

	published, err := client.CoreV1().ConfigMaps("customer-1").Get(ctx, connectionName, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, selected.TargetDeploymentId, published.Data["deploymentId"])
	require.Equal(t, "2", published.Data["revision"])

	require.NoError(t, r.reconcile(ctx))
	a, err = client.CoreV1().Services("customer-1").Get(ctx, original.Data["serviceName"], metav1.GetOptions{})
	require.NoError(t, err)
	require.Contains(t, a.Annotations, privatenetwork.RetireAfterAnnotation)
}
