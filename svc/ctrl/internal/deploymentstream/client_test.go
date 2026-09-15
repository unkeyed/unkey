package deploymentstream

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
	"vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/proto/vtgate"
	"vitess.io/vitess/go/vt/proto/vtgateservice"
)

func TestWatch_DeliveryPrecedesCheckpointAcrossResponses(t *testing.T) {
	position := &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: "unkey", Shard: "0", Gtid: "position-7"}}}
	for _, fail := range []bool{false, true} {
		name := "successful delivery"
		if fail {
			name = "failed delivery"
		}
		t.Run(name, func(t *testing.T) {
			server := &scriptedServer{responses: []*vtgate.VStreamResponse{
				{Events: []*binlog.VEvent{
					{Type: binlog.VEventType_BEGIN},
					{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{RowChanges: []*binlog.RowChange{
						{After: &query.Row{Lengths: []int64{8}, Values: []byte("deploy_a")}},
						{After: &query.Row{Lengths: []int64{8}, Values: []byte("deploy_b")}},
					}}},
					{Type: binlog.VEventType_VGTID, Vgtid: position},
				}},
				{Events: []*binlog.VEvent{{Type: binlog.VEventType_COMMIT}}},
			}}
			var delivered []string
			var token []byte
			failure := errors.New("apply failed")
			err := testClient(t, server).Watch(t.Context(), "region_a", nil, func(id string) error {
				if fail && id == "deploy_b" {
					return failure
				}
				delivered = append(delivered, id)
				return nil
			}, func(next []byte) error {
				require.Equal(t, []string{"deploy_a", "deploy_b"}, delivered)
				token = next
				return nil
			})
			if fail {
				require.ErrorIs(t, err, failure)
				require.Equal(t, []string{"deploy_a"}, delivered)
				require.Empty(t, token)
			} else {
				require.ErrorIs(t, err, io.EOF)
				require.NotEmpty(t, token)
			}
		})
	}
}

func TestWatch_CoalescesOnlyCommittedCheckpoints(t *testing.T) {
	position := func(gtid string) *binlog.VGtid {
		return &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: "unkey", Shard: "0", Gtid: gtid}}}
	}
	server := &scriptedServer{responses: []*vtgate.VStreamResponse{{Events: []*binlog.VEvent{
		{Type: binlog.VEventType_VGTID, Vgtid: position("first")},
		{Type: binlog.VEventType_COMMIT},
		{Type: binlog.VEventType_VGTID, Vgtid: position("idle")},
		{Type: binlog.VEventType_COMMIT},
		{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{RowChanges: []*binlog.RowChange{{After: &query.Row{Lengths: []int64{8}, Values: []byte("deploy_a")}}}}},
		{Type: binlog.VEventType_VGTID, Vgtid: position("pending")},
		{Type: binlog.VEventType_HEARTBEAT},
		{Type: binlog.VEventType_COMMIT},
	}}}}
	client := testClient(t, server)
	controlled := clock.NewTestClock()
	client.clock = controlled
	var checkpoints []string
	err := client.Watch(t.Context(), "region_a", nil, func(string) error {
		require.Equal(t, []string{"first"}, checkpoints, "unrelated commits must not flood Krane")
		controlled.Tick(31 * time.Second)
		return nil
	}, func(token []byte) error {
		decoded, err := client.position("region_a", token)
		require.NoError(t, err)
		checkpoints = append(checkpoints, decoded.ShardGtids[0].Gtid)
		return nil
	})
	require.ErrorIs(t, err, io.EOF)
	require.Equal(t, []string{"first", "idle", "pending"}, checkpoints, "heartbeat may publish only the last committed position")
}

func TestWatch_HeartbeatDoesNotCommitPendingPosition(t *testing.T) {
	client := testClient(t, &scriptedServer{responses: []*vtgate.VStreamResponse{
		{Events: []*binlog.VEvent{{Type: binlog.VEventType_VGTID, Vgtid: &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: "unkey", Shard: "0", Gtid: "uncommitted"}}}}}},
		{Events: []*binlog.VEvent{{Type: binlog.VEventType_HEARTBEAT}}},
	}})
	err := client.Watch(t.Context(), "region_a", nil, func(string) error { return errors.New("unexpected row") }, func([]byte) error { return errors.New("checkpoint before commit") })
	require.ErrorIs(t, err, io.EOF)
}

func TestWatch_StopsStalledUpstream(t *testing.T) {
	client := testClient(t, &scriptedServer{wait: true})
	controlled := &observedClock{TestClock: clock.NewTestClock(), started: make(chan struct{}, 1)}
	client.clock = controlled
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		done <- client.Watch(ctx, "region_a", nil, func(string) error { return errors.New("unexpected row") }, func([]byte) error { return errors.New("unexpected checkpoint") })
	}()
	select {
	case <-controlled.started:
	case <-ctx.Done():
		t.Fatal("upstream wait did not start its timeout")
	}
	controlled.Tick(29 * time.Second)
	select {
	case err := <-done:
		t.Fatalf("stream stopped before timeout: %v", err)
	default:
	}
	controlled.Tick(time.Second)
	select {
	case err := <-done:
		require.Equal(t, codes.Unavailable, status.Code(err))
		require.Contains(t, err.Error(), "VStream stalled")
	case <-ctx.Done():
		t.Fatal("stalled stream was not stopped")
	}
}

type observedClock struct {
	*clock.TestClock
	started chan struct{}
}

func (c *observedClock) NewTicker(d time.Duration) clock.Ticker {
	ticker := c.TestClock.NewTicker(d)
	c.started <- struct{}{}
	return ticker
}

type scriptedServer struct {
	vtgateservice.UnimplementedVitessServer
	responses []*vtgate.VStreamResponse
	err       error
	wait      bool
}

func (s *scriptedServer) VStream(_ *vtgate.VStreamRequest, stream vtgateservice.Vitess_VStreamServer) error {
	for _, response := range s.responses {
		if err := stream.Send(response); err != nil {
			return err
		}
	}
	if s.wait {
		<-stream.Context().Done()
		return stream.Context().Err()
	}
	return s.err
}

func testClient(t *testing.T, implementation vtgateservice.VitessServer) *Client {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	vtgateservice.RegisterVitessServer(server, implementation)
	finished := make(chan error, 1)
	go func() { finished <- server.Serve(listener) }()
	t.Cleanup(func() { server.Stop(); require.NoError(t, <-finished) })
	connection, err := grpc.NewClient("passthrough:///test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }))
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	return &Client{connection: connection, client: vtgateservice.NewVitessClient(connection), keyspace: "unkey", clock: clock.New()}
}

func TestWatch_OnlyExpiredBinlogsRequireSnapshot(t *testing.T) {
	for _, test := range []struct {
		message string
		expired bool
	}{
		{"Cannot replicate because the source purged required binary logs (errno 1236) (sqlstate HY000)", true},
		{"Could not find first log file name in binary log index file (errno 1236) (sqlstate HY000)", true},
		{"source has purged required GTIDs (errno 1789) (sqlstate HY000)", true},
		{"connection interrupted", false},
	} {
		t.Run(test.message, func(t *testing.T) {
			client := testClient(t, &scriptedServer{err: status.Error(codes.Unknown, test.message)})
			err := client.Watch(t.Context(), "region_a", []byte(`{"region":"region_a","position":{"shardGtids":[{"keyspace":"unkey","shard":"0","gtid":"position-7"}]}}`), func(string) error { return errors.New("unexpected change") }, func([]byte) error { return errors.New("unexpected checkpoint") })
			require.Error(t, err)
			require.Equal(t, test.expired, errors.Is(err, ErrExpired))
		})
	}
}

func TestWatch_RejectsForeignSnapshotTable(t *testing.T) {
	client := testClient(t, &scriptedServer{})
	err := client.Watch(t.Context(), "region_a", []byte(`{"region":"region_a","position":{"shardGtids":[{"keyspace":"unkey","shard":"0","gtid":"position-7","tablePKs":[{"tableName":"workspaces"}]}]}}`), func(string) error { return errors.New("unexpected row") }, func([]byte) error { return errors.New("unexpected checkpoint") })
	require.ErrorIs(t, err, ErrInvalidToken)
}
