package privatenetwork

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"google.golang.org/protobuf/proto"
)

const snapshotChunkSize = 2

type snapshotServer struct {
	mu                  sync.Mutex
	snapshotConnections []*ctrlv1.PrivateNetworkConnection
	client              *connect.Client[ctrlv1.StreamPrivateNetworkStateRequest, ctrlv1.PrivateNetworkStateChunk]
}

func newSnapshotServer(t *testing.T) *snapshotServer {
	t.Helper()
	s := &snapshotServer{mu: sync.Mutex{}, snapshotConnections: nil, client: nil}
	server := httptest.NewServer(connect.NewServerStreamHandler("/snapshot", func(_ context.Context, _ *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
		s.mu.Lock()
		snapshotConnections := s.snapshotConnections
		s.mu.Unlock()
		for start := 0; start < len(snapshotConnections); start += snapshotChunkSize {
			if err := stream.Send(&ctrlv1.PrivateNetworkStateChunk{Connections: snapshotConnections[start:min(start+snapshotChunkSize, len(snapshotConnections))]}); err != nil {
				return err
			}
		}
		return stream.Send(&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: uint64(len(snapshotConnections))})
	}))
	t.Cleanup(server.Close)
	s.client = connect.NewClient[ctrlv1.StreamPrivateNetworkStateRequest, ctrlv1.PrivateNetworkStateChunk](server.Client(), server.URL+"/snapshot")
	return s
}

func (s *snapshotServer) serve(ctx context.Context, snapshotConnections []*ctrlv1.PrivateNetworkConnection, req *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
	s.mu.Lock()
	s.snapshotConnections = snapshotConnections
	s.mu.Unlock()
	return s.client.CallServerStream(ctx, connect.NewRequest(req))
}

func snapshotFunc(t *testing.T, snapshot func(context.Context) ([]*ctrlv1.PrivateNetworkConnection, error)) func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
	t.Helper()
	server := newSnapshotServer(t)
	return func(ctx context.Context, req *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
		snapshotConnections, err := snapshot(ctx)
		if err != nil {
			return nil, err
		}
		return server.serve(ctx, snapshotConnections, req)
	}
}

func chunkStream(t *testing.T, chunks ...*ctrlv1.PrivateNetworkStateChunk) func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
	t.Helper()
	server := httptest.NewServer(connect.NewServerStreamHandler("/chunks", func(_ context.Context, _ *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
		for _, chunk := range chunks {
			if err := stream.Send(chunk); err != nil {
				return err
			}
		}
		return nil
	}))
	t.Cleanup(server.Close)
	client := connect.NewClient[ctrlv1.StreamPrivateNetworkStateRequest, ctrlv1.PrivateNetworkStateChunk](server.Client(), server.URL+"/chunks")
	return func(ctx context.Context, req *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
		return client.CallServerStream(ctx, connect.NewRequest(req))
	}
}

func TestSnapshotRequiresCompleteStreamAndFinalTopology(t *testing.T) {
	tests := []struct {
		name   string
		chunks []*ctrlv1.PrivateNetworkStateChunk
	}{
		{name: "stream ends before complete", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Connections: []*ctrlv1.PrivateNetworkConnection{testConnection("dep_a")}}}},
		{name: "topology on intermediate chunk", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Topology: testTopology()}, {Complete: true}}},
		{name: "complete count mismatch", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Connections: []*ctrlv1.PrivateNetworkConnection{testConnection("dep_a")}}, {Complete: true, Total: 2}}},
		{name: "complete chunk contains connections", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Complete: true, Total: 1, Connections: []*ctrlv1.PrivateNetworkConnection{testConnection("dep_a")}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, tt.chunks...)}, clusterKey: &ctrlv1.ClusterKey{}}
			_, err := r.snapshot(t.Context())
			require.Error(t, err)
		})
	}
}

func TestSnapshotReturnsCompleteTopologyAndConnections(t *testing.T) {
	connection := testConnection("dep_a")
	topology := testTopology()
	r := &Reconciler{cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Connections: []*ctrlv1.PrivateNetworkConnection{connection}},
		&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: 1, Topology: topology},
	)}, clusterKey: &ctrlv1.ClusterKey{}}

	snapshot, err := r.snapshot(t.Context())
	require.NoError(t, err)
	require.Len(t, snapshot.GetConnections(), 1)
	require.True(t, proto.Equal(connection, snapshot.GetConnections()[0]))
	require.True(t, proto.Equal(topology, snapshot.GetTopology()))
}
