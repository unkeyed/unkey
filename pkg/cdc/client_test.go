package cdc

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/proto"
	binlog "vitess.io/vitess/go/vt/proto/binlogdata"
	"vitess.io/vitess/go/vt/proto/query"
	"vitess.io/vitess/go/vt/proto/vtgate"
	"vitess.io/vitess/go/vt/proto/vtgateservice"
)

func TestNewConnection_ValidatesSettings(t *testing.T) {
	for _, test := range []struct {
		name    string
		config  ConnectionConfig
		invalid bool
	}{
		{name: "missing address", config: ConnectionConfig{Keyspace: "unkey"}, invalid: true},
		{name: "missing keyspace", config: ConnectionConfig{Address: "localhost:33575"}, invalid: true},
		{name: "username only", config: ConnectionConfig{Address: "localhost:33575", Keyspace: "unkey", Username: "user"}, invalid: true},
		{name: "password only", config: ConnectionConfig{Address: "localhost:33575", Keyspace: "unkey", Password: "test-password"}, invalid: true},
		{name: "credentials without TLS", config: ConnectionConfig{Address: "localhost:33575", Keyspace: "unkey", Username: "user", Password: "test-password", Insecure: true}, invalid: true},
		{name: "TLS without credentials", config: ConnectionConfig{Address: "localhost:33575", Keyspace: "unkey"}},
		{name: "TLS with credentials", config: ConnectionConfig{Address: "localhost:33575", Keyspace: "unkey", Username: "user", Password: "test-password"}},
		{name: "local plaintext", config: ConnectionConfig{Address: "localhost:33575", Keyspace: "unkey", Insecure: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			client, err := NewConnection(test.config)
			if client != nil {
				t.Cleanup(func() { require.NoError(t, client.Close()) })
			}
			if test.invalid {
				require.Error(t, err)
				require.Nil(t, client)
				_, tagged := fault.GetCode(err)
				require.True(t, tagged, "invalid settings must return a tagged assertion error")
				return
			}
			require.NoError(t, err)
			require.NotNil(t, client)
		})
	}
}

func TestWatch_PreservesFieldsAndBeforeAfterImages(t *testing.T) {
	rules := []Rule{{Table: "records", Query: "select id, value from records"}, {Table: "settings", Query: "select name from settings"}}
	events := []*binlog.VEvent{
		{Type: binlog.VEventType_FIELD, FieldEvent: &binlog.FieldEvent{TableName: "records", Fields: []*query.Field{{Name: "id", Type: query.Type_INT64}, {Name: "value", Type: query.Type_VARBINARY}}}},
		{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{TableName: "records", RowChanges: []*binlog.RowChange{
			{After: &query.Row{Lengths: []int64{1, -1}, Values: []byte("7")}},
			{Before: &query.Row{Lengths: []int64{1, -1}, Values: []byte("7")}, After: &query.Row{Lengths: []int64{1, 3}, Values: []byte{'7', 0, 1, 2}}},
			{Before: &query.Row{Lengths: []int64{1, 3}, Values: []byte{'7', 0, 1, 2}}},
		}}},
		{Type: binlog.VEventType_FIELD, FieldEvent: &binlog.FieldEvent{TableName: "settings", Fields: []*query.Field{{Name: "name", Type: query.Type_VARCHAR}}}},
		{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{TableName: "settings", RowChanges: []*binlog.RowChange{{After: &query.Row{Lengths: []int64{5}, Values: []byte("theme")}}}}},
	}
	server := &scriptedServer{responses: []*vtgate.VStreamResponse{{Events: events}}, requests: make(chan *vtgate.VStreamRequest, 1)}
	client, err := New(Config{Connection: testConnection(t, server), Rules: rules})
	require.NoError(t, err)
	var delivered []*binlog.VEvent
	err = client.Watch(t.Context(), func(event *binlog.VEvent) error {
		delivered = append(delivered, event)
		return nil
	})
	require.ErrorIs(t, err, io.EOF)
	require.Len(t, delivered, len(events))
	for i := range events {
		require.True(t, proto.Equal(events[i], delivered[i]))
	}
	request := <-server.requests
	require.Len(t, request.Filter.Rules, 2)
	require.Equal(t, "records", request.Filter.Rules[0].Match)
	require.Equal(t, "select id, value from records", request.Filter.Rules[0].Filter)
	require.Equal(t, "settings", request.Filter.Rules[1].Match)
	require.Equal(t, "select name from settings", request.Filter.Rules[1].Filter)
}

func TestWatch_RemembersCommittedProgressWithoutDeliveringCheckpoints(t *testing.T) {
	server := &scriptedServer{
		requests: make(chan *vtgate.VStreamRequest, 2),
		responses: []*vtgate.VStreamResponse{{Events: []*binlog.VEvent{
			{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{RowChanges: []*binlog.RowChange{{After: &query.Row{Lengths: []int64{1}, Values: []byte("a")}}}}},
			{Type: binlog.VEventType_VGTID, Vgtid: &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: "unkey", Shard: "0", Gtid: "committed-7"}}}},
			{Type: binlog.VEventType_COMMIT},
			{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{RowChanges: []*binlog.RowChange{{After: &query.Row{Lengths: []int64{1}, Values: []byte("b")}}}}},
		}}},
	}
	client, err := New(Config{
		Connection: testConnection(t, server),
		Rules:      []Rule{{Table: "records", Query: "select id from records"}},
	})
	require.NoError(t, err)
	failure := errors.New("apply failed")
	var delivered []string
	apply := func(event *binlog.VEvent) error {
		require.Equal(t, binlog.VEventType_ROW, event.Type)
		id := string(event.RowEvent.RowChanges[0].After.Values)
		if id == "b" {
			return failure
		}
		delivered = append(delivered, id)
		return nil
	}
	require.ErrorIs(t, client.Watch(t.Context(), apply), failure)
	require.Equal(t, []string{"a"}, delivered)
	require.Empty(t, (<-server.requests).Vgtid.ShardGtids[0].Gtid)
	require.ErrorIs(t, client.Watch(t.Context(), apply), failure)
	require.Equal(t, "committed-7", (<-server.requests).Vgtid.ShardGtids[0].Gtid)
}

func TestForward_FailedCheckpointRetainsPreviousPosition(t *testing.T) {
	server := &scriptedServer{
		requests: make(chan *vtgate.VStreamRequest, 2),
		responses: []*vtgate.VStreamResponse{{Events: []*binlog.VEvent{
			{Type: binlog.VEventType_VGTID, Vgtid: &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: "unkey", Shard: "0", Gtid: "next-8"}}}},
			{Type: binlog.VEventType_COMMIT},
		}}},
	}
	initial := []byte(`{"rules":[{"table":"records","query":"select id from records"}],"position":{"shardGtids":[{"keyspace":"unkey","shard":"0","gtid":"previous-7"}]}}`)
	client, err := New(Config{
		Connection:  testConnection(t, server),
		Rules:       []Rule{{Table: "records", Query: "select id from records"}},
		ResumeToken: initial,
	})
	require.NoError(t, err)
	failure := errors.New("send failed")
	for range 2 {
		err := client.Forward(t.Context(), func(event Event) error {
			require.Nil(t, event.Change)
			require.NotEmpty(t, event.ResumeToken)
			return failure
		})
		require.ErrorIs(t, err, failure)
		require.Equal(t, initial, client.ResumeToken())
		require.Equal(t, "previous-7", (<-server.requests).Vgtid.ShardGtids[0].Gtid)
	}
}

func TestNew_ClientsOwnTheirRulesAndTokens(t *testing.T) {
	server := &scriptedServer{requests: make(chan *vtgate.VStreamRequest, 2)}
	connection := testConnection(t, server)
	rules := []Rule{{Table: "records", Query: "select id from records"}}
	token := []byte(`{"rules":[{"table":"records","query":"select id from records"}],"position":{"shardGtids":[{"keyspace":"unkey","shard":"0","gtid":"position-7"}]}}`)
	resuming, err := New(Config{Connection: connection, Rules: rules, ResumeToken: token})
	require.NoError(t, err)
	fresh, err := New(Config{Connection: connection, Rules: []Rule{{Table: "settings", Query: "select name from settings"}}})
	require.NoError(t, err)
	rules[0].Query = "select value from records"
	token[0] = '!'
	exported := resuming.ResumeToken()
	exported[0] = '!'
	for _, test := range []struct {
		client *Client
		query  string
		gtid   string
	}{
		{client: resuming, query: "select id from records", gtid: "position-7"},
		{client: fresh, query: "select name from settings", gtid: ""},
	} {
		t.Run(test.query, func(t *testing.T) {
			err := test.client.Watch(t.Context(), func(*binlog.VEvent) error { return errors.New("unexpected change") })
			require.ErrorIs(t, err, io.EOF)
			request := <-server.requests
			require.Equal(t, test.query, request.Filter.Rules[0].Filter)
			require.Equal(t, test.gtid, request.Vgtid.ShardGtids[0].Gtid)
		})
	}
}

func TestForward_DeliveryPrecedesCheckpointAcrossResponses(t *testing.T) {
	rules := []Rule{{Table: "records", Query: "select id from records"}}
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
						{After: &query.Row{Lengths: []int64{8}, Values: []byte("record_a")}},
						{After: &query.Row{Lengths: []int64{8}, Values: []byte("record_b")}},
					}}},
					{Type: binlog.VEventType_VGTID, Vgtid: position},
				}},
				{Events: []*binlog.VEvent{{Type: binlog.VEventType_COMMIT}}},
			}}
			client, err := New(Config{Connection: testConnection(t, server), Rules: rules})
			require.NoError(t, err)
			var delivered []string
			var token []byte
			failure := errors.New("apply failed")
			err = client.Forward(t.Context(), func(event Event) error {
				if event.Change == nil {
					require.Equal(t, []string{"record_a", "record_b"}, delivered)
					token = event.ResumeToken
					return nil
				}
				require.Empty(t, event.ResumeToken)
				for _, row := range event.Change.GetRowEvent().GetRowChanges() {
					id := string(row.After.Values)
					if fail && id == "record_b" {
						return failure
					}
					delivered = append(delivered, id)
				}
				return nil
			})
			if fail {
				require.ErrorIs(t, err, failure)
				require.Equal(t, []string{"record_a"}, delivered)
				require.Empty(t, token)
			} else {
				require.ErrorIs(t, err, io.EOF)
				require.NotEmpty(t, token)
			}
		})
	}
}

func TestForward_CoalescesOnlyCommittedCheckpoints(t *testing.T) {
	rules := []Rule{{Table: "records", Query: "select id from records"}}
	position := func(gtid string) *binlog.VGtid {
		return &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: "unkey", Shard: "0", Gtid: gtid}}}
	}
	server := &scriptedServer{responses: []*vtgate.VStreamResponse{{Events: []*binlog.VEvent{
		{Type: binlog.VEventType_VGTID, Vgtid: position("first")},
		{Type: binlog.VEventType_COMMIT},
		{Type: binlog.VEventType_VGTID, Vgtid: position("idle")},
		{Type: binlog.VEventType_COMMIT},
		{Type: binlog.VEventType_ROW, RowEvent: &binlog.RowEvent{RowChanges: []*binlog.RowChange{{After: &query.Row{Lengths: []int64{8}, Values: []byte("record_a")}}}}},
		{Type: binlog.VEventType_VGTID, Vgtid: position("pending")},
		{Type: binlog.VEventType_HEARTBEAT},
		{Type: binlog.VEventType_COMMIT},
	}}}}
	client, err := New(Config{Connection: testConnection(t, server), Rules: rules})
	require.NoError(t, err)
	controlled := clock.NewTestClock()
	client.clock = controlled
	var checkpoints [][]byte
	err = client.Forward(t.Context(), func(event Event) error {
		if event.Change == nil {
			checkpoints = append(checkpoints, event.ResumeToken)
			return nil
		}
		require.Empty(t, event.ResumeToken)
		require.Len(t, checkpoints, 1, "unrelated commits must be coalesced")
		controlled.Tick(31 * time.Second)
		return nil
	})
	require.ErrorIs(t, err, io.EOF)
	require.Len(t, checkpoints, 3)
	for i, want := range []string{"first", "idle", "pending"} {
		resumed := &scriptedServer{requests: make(chan *vtgate.VStreamRequest, 1)}
		client, err := New(Config{Connection: testConnection(t, resumed), Rules: rules, ResumeToken: checkpoints[i]})
		require.NoError(t, err)
		err = client.Watch(t.Context(), func(*binlog.VEvent) error { return errors.New("unexpected event") })
		require.ErrorIs(t, err, io.EOF)
		request := <-resumed.requests
		require.Equal(t, want, request.Vgtid.ShardGtids[0].Gtid, "resume must use only committed progress")
	}
}

func TestForward_HeartbeatDoesNotCommitPendingPosition(t *testing.T) {
	rules := []Rule{{Table: "records", Query: "select id from records"}}
	connection := testConnection(t, &scriptedServer{responses: []*vtgate.VStreamResponse{
		{Events: []*binlog.VEvent{{Type: binlog.VEventType_VGTID, Vgtid: &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: "unkey", Shard: "0", Gtid: "uncommitted"}}}}}},
		{Events: []*binlog.VEvent{{Type: binlog.VEventType_HEARTBEAT}}},
	}})
	client, err := New(Config{Connection: connection, Rules: rules})
	require.NoError(t, err)
	err = client.Forward(t.Context(), func(Event) error { return errors.New("event before commit") })
	require.ErrorIs(t, err, io.EOF)
}

func TestWatch_StopsStalledUpstream(t *testing.T) {
	rules := []Rule{{Table: "records", Query: "select id from records"}}
	client, err := New(Config{Connection: testConnection(t, &scriptedServer{wait: true}), Rules: rules})
	require.NoError(t, err)
	controlled := &observedClock{TestClock: clock.NewTestClock(), started: make(chan struct{}, 1)}
	client.clock = controlled
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() {
		done <- client.Watch(ctx, func(*binlog.VEvent) error { return errors.New("unexpected event") })
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
	requests  chan *vtgate.VStreamRequest
}

func (s *scriptedServer) VStream(request *vtgate.VStreamRequest, stream vtgateservice.Vitess_VStreamServer) error {
	if s.requests != nil {
		s.requests <- request
	}
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

func testConnection(t *testing.T, implementation vtgateservice.VitessServer) *Connection {
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
	return &Connection{connection: connection, client: vtgateservice.NewVitessClient(connection), keyspace: "unkey"}
}

func TestWatch_OnlyExpiredBinlogsRequireSnapshot(t *testing.T) {
	rules := []Rule{{Table: "records", Query: "select id from records"}}
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
			client, err := New(Config{
				Connection:  testConnection(t, &scriptedServer{err: status.Error(codes.Unknown, test.message)}),
				Rules:       rules,
				ResumeToken: []byte(`{"rules":[{"table":"records","query":"select id from records"}],"position":{"shardGtids":[{"keyspace":"unkey","shard":"0","gtid":"position-7"}]}}`),
			})
			require.NoError(t, err)
			err = client.Watch(t.Context(), func(*binlog.VEvent) error { return errors.New("unexpected event") })
			require.Error(t, err)
			require.Equal(t, test.expired, errors.Is(err, ErrExpired))
		})
	}
}

func TestNew_RejectsForeignSnapshotTable(t *testing.T) {
	rules := []Rule{{Table: "records", Query: "select id from records"}}
	client, err := New(Config{
		Connection:  testConnection(t, &scriptedServer{}),
		Rules:       rules,
		ResumeToken: []byte(`{"rules":[{"table":"records","query":"select id from records"}],"position":{"shardGtids":[{"keyspace":"unkey","shard":"0","gtid":"position-7","tablePKs":[{"tableName":"workspaces"}]}]}}`),
	})
	require.ErrorIs(t, err, ErrInvalidToken)
	require.Nil(t, client)
}

func TestNew_TokensBindToAllRules(t *testing.T) {
	rules := []Rule{{Table: "records", Query: "select id from records"}, {Table: "settings", Query: "select name from settings where enabled = 1"}}
	connection := testConnection(t, &scriptedServer{responses: []*vtgate.VStreamResponse{{Events: []*binlog.VEvent{
		{Type: binlog.VEventType_VGTID, Vgtid: &binlog.VGtid{ShardGtids: []*binlog.ShardGtid{{Keyspace: "unkey", Shard: "0", Gtid: "position-7"}}}},
		{Type: binlog.VEventType_COMMIT},
	}}}})
	client, err := New(Config{Connection: connection, Rules: rules})
	require.NoError(t, err)
	err = client.Watch(t.Context(), func(*binlog.VEvent) error { return errors.New("unexpected event") })
	require.ErrorIs(t, err, io.EOF)
	token := client.ResumeToken()
	require.NotEmpty(t, token)
	for _, test := range []struct {
		name  string
		rules []Rule
		want  error
	}{
		{name: "unchanged", rules: rules},
		{name: "changed predicate", rules: []Rule{rules[0], {Table: "settings", Query: "select name from settings where enabled = 0"}}, want: ErrInvalidToken},
		{name: "removed table", rules: rules[:1], want: ErrInvalidToken},
		{name: "changed table", rules: []Rule{rules[0], {Table: "other_settings", Query: rules[1].Query}}, want: ErrInvalidToken},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := New(Config{Connection: connection, Rules: test.rules, ResumeToken: token})
			require.ErrorIs(t, err, test.want)
		})
	}
}
