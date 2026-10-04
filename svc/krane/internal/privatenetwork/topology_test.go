package privatenetwork

import (
	"context"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"github.com/unkeyed/unkey/svc/krane/pkg/labels"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func testTopology() *ctrlv1.PrivateNetworkTopology {
	return &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{
		{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"},
		{Platform: "aws", Region: "eu-west-1", CellId: "aws-eu-west-1"},
	}}
}

func TestEnsureTopologyPublishesUpdatesAndNoOps(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	r := &Reconciler{clock: clock.NewTestClock(), client: client, clusterKey: &ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}

	require.NoError(t, r.ensureTopology(ctx, nil))
	require.Empty(t, client.Actions())
	require.NoError(t, r.ensureTopology(ctx, testTopology()))

	configMap, err := client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(ctx, privatenetwork.TopologyConfigMap, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, map[string]string{
		"version": "1", "platform": "aws", "cell": "aws-us-east-1",
		"region.aws-us-east-1": "us-east-1", "region.aws-eu-west-1": "eu-west-1",
	}, configMap.Data)
	require.Equal(t, "krane", configMap.Labels[labels.LabelKeyManagedBy])
	require.Equal(t, privatenetwork.TopologyComponent, configMap.Labels[labels.LabelKeyComponent])

	actions := len(client.Actions())
	require.NoError(t, r.ensureTopology(ctx, testTopology()))
	require.Len(t, client.Actions(), actions+1)
	require.Equal(t, "get", client.Actions()[actions].GetVerb())

	updated := testTopology()
	updated.Clusters = append(updated.Clusters, &ctrlv1.ClusterKey{Platform: "aws", Region: "eu-central-1", CellId: "aws-eu-central-1"})
	require.NoError(t, r.ensureTopology(ctx, updated))
	configMap, err = client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(ctx, privatenetwork.TopologyConfigMap, metav1.GetOptions{})
	require.NoError(t, err)
	require.Equal(t, "eu-central-1", configMap.Data["region.aws-eu-central-1"])
	updated.Clusters[1].Region = "eu-central-1"
	require.Error(t, r.ensureTopology(ctx, updated))
}

func TestEnsureTopologyRejectsInvalidAndForeignData(t *testing.T) {
	tests := []struct {
		name     string
		topology *ctrlv1.PrivateNetworkTopology
		existing *corev1.ConfigMap
	}{
		{name: "unsupported version", topology: &ctrlv1.PrivateNetworkTopology{Version: 2}},
		{name: "missing own cell", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "eu-west-1", CellId: "other"}}}},
		{name: "wrong own region", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "us-west-2", CellId: "aws-us-east-1"}}}},
		{name: "wrong platform", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "gcp", Region: "us-east-1", CellId: "aws-us-east-1"}}}},
		{name: "empty platform", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Region: "us-east-1", CellId: "aws-us-east-1"}}}},
		{name: "empty cell", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "us-east-1"}}}},
		{name: "invalid cell", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "us-east-1", CellId: "cell/one"}}}},
		{name: "empty region", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", CellId: "aws-us-east-1"}}}},
		{name: "invalid region", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "local-first", CellId: "aws-us-east-1"}}}},
		{name: "duplicate cell", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}, {Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}}},
		{name: "foreign map", topology: testTopology(), existing: &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: privatenetwork.TopologyConfigMap, Namespace: metav1.NamespaceSystem, Labels: map[string]string{"owner": "other"}}, Data: map[string]string{"keep": "me"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientset()
			if tt.existing != nil {
				_, err := client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Create(t.Context(), tt.existing, metav1.CreateOptions{})
				require.NoError(t, err)
			}
			r := &Reconciler{clock: clock.NewTestClock(), client: client, clusterKey: &ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}
			require.Error(t, r.ensureTopology(t.Context(), tt.topology))
			if tt.existing != nil {
				got, err := client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(t.Context(), privatenetwork.TopologyConfigMap, metav1.GetOptions{})
				require.NoError(t, err)
				require.Equal(t, map[string]string{"keep": "me"}, got.Data)
			}
		})
	}
}

func TestReconcileSnapshotFailuresPreserveTopology(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, identified("snapshot-1",
		&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: 0, Topology: testTopology()},
	)...)}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}
	require.NoError(t, r.reconcile(ctx))

	tests := []struct {
		name   string
		stream func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error)
	}{
		{name: "partial snapshot", stream: chunkStream(t, identified("snapshot-2", &ctrlv1.PrivateNetworkStateChunk{})...)},
		{name: "invalid topology", stream: chunkStream(t, identified("snapshot-3", &ctrlv1.PrivateNetworkStateChunk{Complete: true, Topology: &ctrlv1.PrivateNetworkTopology{Version: 2}})...)},
		{name: "RPC failure", stream: func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
			return nil, fmt.Errorf("unavailable")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			control.StreamPrivateNetworkStateFunc = tt.stream
			require.Error(t, r.reconcile(ctx))
			configMap, err := client.CoreV1().ConfigMaps(metav1.NamespaceSystem).Get(ctx, privatenetwork.TopologyConfigMap, metav1.GetOptions{})
			require.NoError(t, err)
			require.Equal(t, "us-east-1", configMap.Data["region.aws-us-east-1"])
			require.Equal(t, "eu-west-1", configMap.Data["region.aws-eu-west-1"])
		})
	}
}

func TestTopologyFailureDoesNotPreventConnectionRevocation(t *testing.T) {
	client := fake.NewClientset()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, identified("snapshot-1",
		&ctrlv1.PrivateNetworkStateChunk{Connections: []*ctrlv1.PrivateNetworkConnection{newTestConnection()}},
		&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: 1, Topology: testTopology()},
	)...)}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}
	require.NoError(t, r.reconcile(t.Context()))
	connections, err := client.CoreV1().ConfigMaps("customer-1").List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, connections.Items, 1)

	control.StreamPrivateNetworkStateFunc = chunkStream(t, identified("snapshot-2", &ctrlv1.PrivateNetworkStateChunk{Complete: true, Topology: &ctrlv1.PrivateNetworkTopology{Version: 2}})...)
	require.Error(t, r.reconcile(t.Context()))
	connections, err = client.CoreV1().ConfigMaps("customer-1").List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Len(t, connections.Items, 1)
	control.StreamPrivateNetworkStateFunc = chunkStream(t, identified("snapshot-3", &ctrlv1.PrivateNetworkStateChunk{Complete: true, Topology: &ctrlv1.PrivateNetworkTopology{Version: 2}})...)
	require.Error(t, r.reconcile(t.Context()))
	connections, err = client.CoreV1().ConfigMaps("customer-1").List(t.Context(), metav1.ListOptions{})
	require.NoError(t, err)
	require.Empty(t, connections.Items)
}
