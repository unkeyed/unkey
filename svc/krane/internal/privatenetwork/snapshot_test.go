package privatenetwork

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/krane/internal/testutil"
	"google.golang.org/protobuf/proto"
)

const snapshotChunkSize = 2

type snapshotServer struct {
	mu                  sync.Mutex
	snapshotConnections []*ctrlv1.PrivateNetworkConnection
	served              int
	client              *connect.Client[ctrlv1.StreamPrivateNetworkStateRequest, ctrlv1.PrivateNetworkStateChunk]
}

func newSnapshotServer(t *testing.T) *snapshotServer {
	t.Helper()
	s := &snapshotServer{mu: sync.Mutex{}, snapshotConnections: nil, served: 0, client: nil}
	server := httptest.NewServer(connect.NewServerStreamHandler("/snapshot", func(_ context.Context, _ *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
		s.mu.Lock()
		snapshotConnections := s.snapshotConnections
		s.served++
		version, snapshotID := fmt.Sprintf("v%d", s.served), fmt.Sprintf("snapshot-%d", s.served)
		s.mu.Unlock()
		for start := 0; start < len(snapshotConnections); start += snapshotChunkSize {
			if err := stream.Send(&ctrlv1.PrivateNetworkStateChunk{Version: version, SnapshotId: snapshotID, Connections: snapshotConnections[start:min(start+snapshotChunkSize, len(snapshotConnections))]}); err != nil {
				return err
			}
		}
		return stream.Send(&ctrlv1.PrivateNetworkStateChunk{Version: version, SnapshotId: snapshotID, Complete: true, Total: uint64(len(snapshotConnections)), Certified: true})
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

func identified(snapshotID string, chunks ...*ctrlv1.PrivateNetworkStateChunk) []*ctrlv1.PrivateNetworkStateChunk {
	for _, chunk := range chunks {
		chunk.Version, chunk.SnapshotId = "v-"+snapshotID, snapshotID
		chunk.Certified = chunk.GetCertified() || chunk.GetComplete()
	}
	return chunks
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

func failedChunkStream(t *testing.T, chunks ...*ctrlv1.PrivateNetworkStateChunk) func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
	t.Helper()
	server := httptest.NewServer(connect.NewServerStreamHandler("/chunks", func(_ context.Context, _ *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
		for _, chunk := range chunks {
			if err := stream.Send(chunk); err != nil {
				return err
			}
		}
		return connect.NewError(connect.CodeUnavailable, nil)
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
		{name: "stream ends before complete", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Connections: []*ctrlv1.PrivateNetworkConnection{newTestConnection()}}}},
		{name: "topology on intermediate chunk", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Topology: testTopology()}, {Complete: true}}},
		{name: "certification on intermediate chunk", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Certified: true}, {Complete: true}}},
		{name: "complete count mismatch", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Connections: []*ctrlv1.PrivateNetworkConnection{newTestConnection()}}, {Complete: true, Total: 2}}},
		{name: "complete chunk contains connections", chunks: []*ctrlv1.PrivateNetworkStateChunk{{Complete: true, Total: 1, Connections: []*ctrlv1.PrivateNetworkConnection{newTestConnection()}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Reconciler{clock: clock.NewTestClock(), cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, identified("snapshot-1", tt.chunks...)...)}, clusterKey: &ctrlv1.ClusterKey{}}
			_, err := r.snapshot(t.Context())
			require.Error(t, err)
		})
	}
}

func TestSnapshotReturnsCompleteTopologyAndConnections(t *testing.T) {
	connection := newTestConnection()
	topology := testTopology()
	r := &Reconciler{clock: clock.NewTestClock(), cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, identified("snapshot-1",
		&ctrlv1.PrivateNetworkStateChunk{Connections: []*ctrlv1.PrivateNetworkConnection{connection}},
		&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: 1, Topology: topology},
	)...)}, clusterKey: &ctrlv1.ClusterKey{}}

	snapshot, err := r.snapshot(t.Context())
	require.NoError(t, err)
	require.Len(t, snapshot.GetConnections(), 1)
	require.True(t, proto.Equal(connection, snapshot.GetConnections()[0]))
	require.True(t, proto.Equal(topology, snapshot.GetTopology()))
}

func TestSnapshotAcceptsConsistentProducerIdentityAcrossChunks(t *testing.T) {
	first, second := newTestConnection(), newTestConnection()
	r := &Reconciler{clock: clock.NewTestClock(), cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "snapshot-1", Connections: []*ctrlv1.PrivateNetworkConnection{first}},
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "snapshot-1", Connections: []*ctrlv1.PrivateNetworkConnection{second}},
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "snapshot-1", Complete: true, Total: 2},
	)}, clusterKey: &ctrlv1.ClusterKey{}}

	snapshot, err := r.snapshot(t.Context())
	require.NoError(t, err)
	require.Equal(t, "v1", snapshot.GetVersion())
	require.Equal(t, "snapshot-1", snapshot.GetSnapshotId())
	require.Len(t, snapshot.GetConnections(), 2)
}

func TestSnapshotRejectsMalformedProducerMetadata(t *testing.T) {
	tests := map[string][]*ctrlv1.PrivateNetworkStateChunk{
		"missing snapshot ID": {{Version: "v1"}, {Version: "v1", Complete: true}},
		"version changes":     {{Version: "v1", SnapshotId: "id"}, {Version: "v2", SnapshotId: "id", Complete: true}},
		"snapshot ID changes": {{Version: "v1", SnapshotId: "id-1"}, {Version: "v1", SnapshotId: "id-2", Complete: true}},
		"trailing data":       {{Version: "v1", SnapshotId: "id", Complete: true}, {Version: "v1", SnapshotId: "id"}},
	}
	for name, chunks := range tests {
		t.Run(name, func(t *testing.T) {
			r := &Reconciler{clock: clock.NewTestClock(), cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, chunks...)}, clusterKey: &ctrlv1.ClusterKey{}}
			_, err := r.snapshot(t.Context())
			require.Error(t, err)
		})
	}
}

func TestSnapshotRejectsChunksWithoutIdentity(t *testing.T) {
	tests := map[string][]*ctrlv1.PrivateNetworkStateChunk{
		"complete without identity":     {{Complete: true}},
		"complete without version":      {{SnapshotId: "id", Complete: true}},
		"identity dropped on complete":  {{Version: "v1", SnapshotId: "id"}, {Complete: true}},
		"identity missing before final": {{}, {Version: "v1", SnapshotId: "id", Complete: true}},
	}
	for name, chunks := range tests {
		t.Run(name, func(t *testing.T) {
			r := &Reconciler{clock: clock.NewTestClock(), cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, chunks...)}, clusterKey: &ctrlv1.ClusterKey{}}
			_, err := r.snapshot(t.Context())
			require.ErrorContains(t, err, "missing its snapshot identity")
			require.Nil(t, r.lastSnapshot)
		})
	}
}

func TestSnapshotUnchangedSuccessAndTrailingStreamFailure(t *testing.T) {
	cached := &ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "old", Complete: true, Total: 1, Certified: true, Connections: []*ctrlv1.PrivateNetworkConnection{newTestConnection()}}
	r := &Reconciler{clock: clock.NewTestClock(), cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "new", Complete: true, Unchanged: true, Total: 1},
	)}, clusterKey: &ctrlv1.ClusterKey{}, lastSnapshot: cached}
	snapshot, err := r.snapshot(t.Context())
	require.NoError(t, err)
	require.Equal(t, "new", snapshot.GetSnapshotId())
	require.False(t, snapshot.GetCertified(), "an unchanged response carries its own certification")
	require.Len(t, snapshot.GetConnections(), 1)

	r.cluster = &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: failedChunkStream(t,
		&ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "newer", Complete: true, Unchanged: true, Total: 1},
	)}
	_, err = r.snapshot(t.Context())
	require.Error(t, err)
}

func TestSnapshotUnchangedRequiresMatchingCachedVersionAndTotals(t *testing.T) {
	connection := newTestConnection()
	cached := &ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "old", Complete: true, Total: 1, Connections: []*ctrlv1.PrivateNetworkConnection{connection}}
	for _, tc := range []struct {
		name string
		last *ctrlv1.PrivateNetworkStateChunk
		end  *ctrlv1.PrivateNetworkStateChunk
	}{
		{name: "no prior", end: &ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "new", Complete: true, Unchanged: true}},
		{name: "wrong version", last: cached, end: &ctrlv1.PrivateNetworkStateChunk{Version: "v2", SnapshotId: "new", Complete: true, Unchanged: true, Total: 1}},
		{name: "wrong total", last: cached, end: &ctrlv1.PrivateNetworkStateChunk{Version: "v1", SnapshotId: "new", Complete: true, Unchanged: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Reconciler{clock: clock.NewTestClock(), cluster: &testutil.MockClusterClient{StreamPrivateNetworkStateFunc: chunkStream(t, tc.end)}, clusterKey: &ctrlv1.ClusterKey{}, lastSnapshot: tc.last}
			_, err := r.snapshot(t.Context())
			require.Error(t, err)
		})
	}
}
