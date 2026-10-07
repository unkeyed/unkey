package watcher

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
)

func TestWatch_ResetsTokenAfterThreeFailuresWithoutCheckpoint(t *testing.T) {
	server := &recoveryServer{tokens: make(chan []byte, 6)}
	_, handler := ctrlv1connect.NewClusterServiceHandler(server)
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	w := New(Config{Cluster: ctrl.NewConnectClusterServiceClient(ctrlv1connect.NewClusterServiceClient(httpServer.Client(), httpServer.URL))})
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- w.Watch(ctx) }()
	for _, want := range [][]byte{nil, []byte("first"), []byte("first"), []byte("second"), []byte("second"), nil} {
		select {
		case got := <-server.tokens:
			require.Equal(t, want, got)
		case <-ctx.Done():
			t.Fatal("watch did not reconnect")
		}
	}
	cancel()
	require.NoError(t, <-done)
}

type recoveryServer struct {
	ctrlv1connect.UnimplementedClusterServiceHandler
	connections atomic.Int32
	tokens      chan []byte
}

func (s *recoveryServer) WatchDeploymentChanges(ctx context.Context, req *connect.Request[ctrlv1.WatchDeploymentChangesRequest], stream *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
	s.tokens <- req.Msg.GetResumeToken()
	switch s.connections.Add(1) {
	case 1:
		if err := stream.Send(&ctrlv1.DeploymentChangeEvent{ResumeToken: []byte("first")}); err != nil {
			return err
		}
	case 3:
		if err := stream.Send(&ctrlv1.DeploymentChangeEvent{ResumeToken: []byte("second")}); err != nil {
			return err
		}
	case 6:
		<-ctx.Done()
		return ctx.Err()
	}
	return connect.NewError(connect.CodeInternal, errors.New("upstream stream failed"))
}

func (s *recoveryServer) SyncDesiredState(context.Context, *connect.Request[ctrlv1.SyncDesiredStateRequest], *connect.ServerStream[ctrlv1.DeploymentChangeEvent]) error {
	return nil
}
