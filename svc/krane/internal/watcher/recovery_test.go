package watcher

import (
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	ctrl "github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
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

func TestWatch_LogsClusterAndFailureCountOnSnapshotRestart(t *testing.T) {
	logs := loggertest.Install(t)
	start := logs.Snapshot()

	server := &recoveryServer{tokens: make(chan []byte, 6)}
	_, handler := ctrlv1connect.NewClusterServiceHandler(server)
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)

	w := New(Config{
		Cluster:  ctrl.NewConnectClusterServiceClient(ctrlv1connect.NewClusterServiceClient(httpServer.Client(), httpServer.URL)),
		CellID:   "cell-test",
		Region:   "us-east-1",
		Platform: "kubernetes",
	})
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Second)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- w.Watch(ctx) }()

	for range 6 {
		select {
		case <-server.tokens:
		case <-ctx.Done():
			t.Fatal("watch did not reconnect")
		}
	}

	var restart *slog.Record
	for _, record := range logs.Since(start) {
		if record.Message == "stream: restarting snapshot after consecutive failures" {
			r := record
			restart = &r
			break
		}
	}
	require.NotNil(t, restart, "expected a snapshot restart log record")

	attrs := loggertest.FlatAttrs(*restart)
	require.Equal(t, "cell-test", attrs["cell_id"])
	require.Equal(t, "us-east-1", attrs["region"])
	require.Equal(t, "kubernetes", attrs["platform"])
	require.EqualValues(t, 3, attrs["failures"])

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
