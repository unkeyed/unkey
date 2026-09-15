package deploymentstream

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
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
					{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{RowChanges: []*binlog.RowChange{{After: &query.Row{Lengths: []int64{8}, Values: []byte("deploy_a")}}}}},
					{Type: binlog.VEventType_VGTID, Vgtid: position},
				}},
				{Events: []*binlog.VEvent{{Type: binlog.VEventType_COMMIT}}},
			}}
			var delivered []string
			var token []byte
			failure := errors.New("apply failed")
			err := testClient(t, server).Watch(t.Context(), "region_a", nil, func(id string) error {
				if fail {
					return failure
				}
				delivered = append(delivered, id)
				return nil
			}, func(next []byte) error {
				require.Equal(t, []string{"deploy_a"}, delivered)
				token = next
				return nil
			})
			if fail {
				require.ErrorIs(t, err, failure)
				require.Empty(t, token)
			} else {
				require.ErrorIs(t, err, io.EOF)
				require.NotEmpty(t, token)
			}
		})
	}
}

type scriptedServer struct {
	vtgateservice.UnimplementedVitessServer
	responses []*vtgate.VStreamResponse
	err       error
}

func (s *scriptedServer) VStream(_ *vtgate.VStreamRequest, stream vtgateservice.Vitess_VStreamServer) error {
	for _, response := range s.responses {
		if err := stream.Send(response); err != nil {
			return err
		}
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
	return &Client{connection: connection, client: vtgateservice.NewVitessClient(connection), keyspace: "unkey"}
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
