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
	mu     sync.Mutex
	apps   []*ctrlv1.PrivateNetworkApp
	client *connect.Client[ctrlv1.StreamPrivateNetworkStateRequest, ctrlv1.PrivateNetworkStateChunk]
}

func newSnapshotServer(t *testing.T) *snapshotServer {
	t.Helper()
	s := &snapshotServer{mu: sync.Mutex{}, apps: nil, client: nil}
	server := httptest.NewServer(connect.NewServerStreamHandler("/snapshot", func(_ context.Context, _ *connect.Request[ctrlv1.StreamPrivateNetworkStateRequest], stream *connect.ServerStream[ctrlv1.PrivateNetworkStateChunk]) error {
		s.mu.Lock()
		apps := s.apps
		s.mu.Unlock()
		for start := 0; start < len(apps); start += snapshotChunkSize {
			if err := stream.Send(&ctrlv1.PrivateNetworkStateChunk{Apps: apps[start:min(start+snapshotChunkSize, len(apps))]}); err != nil {
				return err
			}
		}
		return stream.Send(&ctrlv1.PrivateNetworkStateChunk{Complete: true, Total: uint64(len(apps))})
	}))
	t.Cleanup(server.Close)
	s.client = connect.NewClient[ctrlv1.StreamPrivateNetworkStateRequest, ctrlv1.PrivateNetworkStateChunk](server.Client(), server.URL+"/snapshot")
	return s
}

func (s *snapshotServer) serve(ctx context.Context, apps []*ctrlv1.PrivateNetworkApp, req *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
	s.mu.Lock()
	s.apps = apps
	s.mu.Unlock()
	return s.client.CallServerStream(ctx, connect.NewRequest(req))
}

func snapshotFunc(t *testing.T, snapshot func(context.Context) ([]*ctrlv1.PrivateNetworkApp, error)) func(context.Context, *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
	t.Helper()
	server := newSnapshotServer(t)
	return func(ctx context.Context, req *ctrlv1.StreamPrivateNetworkStateRequest) (*connect.ServerStreamForClient[ctrlv1.PrivateNetworkStateChunk], error) {
		apps, err := snapshot(ctx)
		if err != nil {
			return nil, err
		}
		return server.serve(ctx, apps, req)
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
