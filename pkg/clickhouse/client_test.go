package clickhouse

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestMaxOpenConns(t *testing.T) {
	chCfg := containers.ClickHouse(t)
	for _, tc := range []struct {
		name       string
		configured int
		want       int
	}{
		{name: "default", want: 50},
		{name: "worker override", configured: 100, want: 100},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var client *Client
			var err error
			if tc.configured == 0 {
				client, err = New(Config{URL: chCfg.DSN})
			} else {
				client, err = NewWithDiagnostics(Config{URL: chCfg.DSN}, tc.configured)
			}
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			require.Equal(t, tc.want, client.conn.Stats().MaxOpenConns)
		})
	}
}

func TestCancelledAcquiresDoNotLeakPoolSlots(t *testing.T) {
	dialErr := errors.New("test dial reached")
	conn, err := ch.Open(&ch.Options{
		Addr:         []string{"unused:9000"},
		MaxOpenConns: 10,
		DialTimeout:  50 * time.Millisecond,
		DialContext:  func(context.Context, string) (net.Conn, error) { return nil, dialErr },
	})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, conn.Close()) })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for range 200 {
		require.ErrorIs(t, conn.Ping(ctx), context.Canceled)
	}
	require.Zero(t, conn.Stats().Open, "cancelled acquires must return every pool slot")
	require.ErrorIs(t, conn.Ping(context.Background()), dialErr, "healthy acquire must reach dial, not time out waiting for a leaked slot")
}

func TestDiagnosticAuditInsertStoresNestedTargets(t *testing.T) {
	chCfg := containers.ClickHouse(t)
	client, err := NewWithDiagnostics(Config{URL: chCfg.DSN}, 100)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	event := auditlog.Event{
		EventID: uid.New("evt"), Time: time.Now().UnixMilli(), WorkspaceID: uid.New("ws"),
		Bucket: "audit", Event: "test.export", Source: auditlog.EventSourcePlatform,
		Actor:   auditlog.EventActor{Type: "system", ID: "test"},
		Targets: []auditlog.EventTarget{{Type: "key", ID: "first"}, {Type: "api", ID: "second"}},
	}
	rows, err := EncodeAuditLogEvents([]auditlog.Event{event})
	require.NoError(t, err)
	require.NoError(t, client.InsertAuditLogs(context.Background(), rows))
	var targetIDs []string
	require.NoError(t, client.conn.QueryRow(context.Background(),
		"SELECT `targets.id` FROM default.audit_logs_raw_v1 WHERE event_id = ?", event.EventID).Scan(&targetIDs))
	require.Equal(t, []string{"first", "second"}, targetIDs)
}
