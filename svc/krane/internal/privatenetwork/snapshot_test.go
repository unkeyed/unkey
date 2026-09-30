package privatenetwork

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
)

const snapshotChunkSize = 2

type snapshotServer struct {
	mu               sync.Mutex
	snapshotBindings []*ctrlv1.PrivateNetworkBinding
	client           *connect.Client[ctrlv1.StreamPrivateNetworkStateRequest, ctrlv1.PrivateNetworkStateChunk]
}

func newSnapshotServer(t *testing.T) *snapshotServer {
	t.Helper()
	s := &snapshotServer{mu: sync.Mutex{}, snapshotBindings: nil, client: nil}
	server := httptest.NewServer(connect.NewServerStreamHandler("/snapshot", func(_ context.Context, _ *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
		s.mu.Lock()
		snapshotBindings := s.snapshotBindings
		s.mu.Unlock()
		for start := 0; start < len(snapshotBindings); start += snapshotChunkSize {
			if err := stream.Send(&ctrlv1.PrivateNetworkStateChunk{Bindings: snapshotBindings[start:min(start+snapshotChunkSize, len(snapshotBindings))]}); err != nil {
				return err
			}
		}
		return stream.Send(&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: uint64(len(snapshotBindings))})
	}))
	t.Cleanup(server.Close)
	s.client = connect.NewClient[ctrlv1.StreamPrivateNetworkStateRequest, ctrlv1.PrivateNetworkStateChunk](server.Client(), server.URL+"/snapshot")
	return s
}

func (s *snapshotServer) serve(ctx context.Context, snapshotBindings []*ctrlv1.PrivateNetworkBinding, req *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
	s.mu.Lock()
	s.snapshotBindings = snapshotBindings
	s.mu.Unlock()
	return s.client.CallServerStream(ctx, connect.NewRequest(req))
}

func snapshotFunc(t *testing.T, snapshot func(context.Context) ([]*ctrlv1.PrivateNetworkBinding, error)) func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
	t.Helper()
	server := newSnapshotServer(t)
	return func(ctx context.Context, req *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
		snapshotBindings, err := snapshot(ctx)
		if err != nil {
			return nil, err
		}
		return server.serve(ctx, snapshotBindings, req)
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
