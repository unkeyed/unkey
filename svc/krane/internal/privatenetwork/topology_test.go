package privatenetwork

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"k8s.io/client-go/kubernetes/fake"
)

func testTopology() *ctrlv1.PrivateNetworkTopology {
	return &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{
		{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"},
		{Platform: "aws", Region: "eu-west-1", CellId: "aws-eu-west-1"},
	}}
}

func TestReconcileRejectsInvalidTopology(t *testing.T) {
	tests := []struct {
		name     string
		topology *ctrlv1.PrivateNetworkTopology
	}{
		{name: "unsupported version", topology: &ctrlv1.PrivateNetworkTopology{Version: 2}},
		{name: "missing own cell", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "eu-west-1", CellId: "other"}}}},
		{name: "wrong own region", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "us-west-2", CellId: "aws-us-east-1"}}}},
		{name: "wrong platform", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "gcp", Region: "us-east-1", CellId: "aws-us-east-1"}}}},
		{name: "empty platform", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Region: "us-east-1", CellId: "aws-us-east-1"}}}},
		{name: "empty cell", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "us-east-1"}}}},
		{name: "empty region", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", CellId: "aws-us-east-1"}}}},
		{name: "invalid region", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "local-first", CellId: "aws-us-east-1"}}}},
		{name: "duplicate cell", topology: &ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}, {Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := fake.NewClientset()
			control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, identified("snapshot-1", &ctrlv1.PrivateNetworkStateChunk{Complete: true, Topology: tt.topology})...)}
			r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}
			require.Error(t, r.reconcile(t.Context()))
			require.Empty(t, connectionServices(t, client, "customer-1"))
		})
	}
}

func TestReconcileSnapshotFailuresPreserveRegionalAliases(t *testing.T) {
	ctx := t.Context()
	client := fake.NewClientset()
	spec := newTestConnection()
	control := &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, identified("snapshot-1",
		&ctrlv1.PrivateNetworkStateChunk{Connections: []*ctrlv1.PrivateNetworkConnection{spec}},
		&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: 1, Topology: testTopology()},
	)...)}
	r := &Reconciler{clock: clock.NewTestClock(), client: client, dynamic: testDynamicClient(), cluster: control, clusterKey: &ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}
	require.NoError(t, r.reconcile(ctx))
	before := connectionServices(t, client, "customer-1")
	require.Len(t, before, 3)

	tests := []struct {
		name   string
		stream func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error)
	}{
		{name: "partial snapshot", stream: chunkStream(t, identified("snapshot-2", &ctrlv1.PrivateNetworkStateChunk{})...)},
		{name: "invalid topology", stream: chunkStream(t, identified("snapshot-3",
			&ctrlv1.PrivateNetworkStateChunk{Connections: []*ctrlv1.PrivateNetworkConnection{spec}},
			&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: 1, Topology: &ctrlv1.PrivateNetworkTopology{Version: 2}},
		)...)},
		{name: "RPC failure", stream: func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
			return nil, fmt.Errorf("unavailable")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			control.StreamPrivateNetworkStateFunc = tt.stream
			require.Error(t, r.reconcile(ctx))
			require.Equal(t, before, connectionServices(t, client, "customer-1"))
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
	require.Len(t, connectionServices(t, client, "customer-1"), 3, "one global and one regional alias per topology region")

	control.StreamPrivateNetworkStateFunc = chunkStream(t, identified("snapshot-2", &ctrlv1.PrivateNetworkStateChunk{Complete: true, Topology: &ctrlv1.PrivateNetworkTopology{Version: 2}})...)
	require.Error(t, r.reconcile(t.Context()))
	require.Len(t, connectionServices(t, client, "customer-1"), 3, "an invalid topology neither confirms absence nor removes regions")
	control.StreamPrivateNetworkStateFunc = chunkStream(t, identified("snapshot-3", &ctrlv1.PrivateNetworkStateChunk{Complete: true, Topology: &ctrlv1.PrivateNetworkTopology{Version: 2}})...)
	require.Error(t, r.reconcile(t.Context()))
	require.Empty(t, connectionServices(t, client, "customer-1"))
}

func TestRegionsAreDNSLabelsThatFitRegionalServiceNames(t *testing.T) {
	longest := strings.Repeat("r", privatenetwork.MaxRegionLength)
	require.True(t, validRegion(longest))
	require.Len(t, privatenetwork.RegionalConnectionServiceName(uid.New(uid.DeploymentPrefix), strings.Repeat("a", 63), longest), 63)
	for _, region := range []string{"", longest + "r", "US-EAST-1", "us.east", "local-first"} {
		require.False(t, validRegion(region), region)
		_, err := New(Config{
			Client: fake.NewClientset(), Dynamic: testDynamicClient(), Cluster: &testutil.MockClusterClient{},
			ClusterKey: &ctrlv1.ClusterKey{Region: region}, Clock: clock.NewTestClock(),
		})
		require.Error(t, err, region)
	}
	r := &Reconciler{clusterKey: &ctrlv1.ClusterKey{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"}}
	_, err := r.topologyRegions(&ctrlv1.PrivateNetworkTopology{Version: 1, Clusters: []*ctrlv1.ClusterKey{
		{Platform: "aws", Region: "us-east-1", CellId: "aws-us-east-1"},
		{Platform: "aws", Region: longest + "r", CellId: "aws-long"},
	}})
	require.Error(t, err)
}
