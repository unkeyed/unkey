package clickhouse

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/logger/loggertest"
	"github.com/unkeyed/unkey/pkg/retry"
)

type acquireFailureConn struct {
	ch.Conn
}

func (*acquireFailureConn) Stats() driver.Stats {
	return driver.Stats{Open: 100, Idle: 0, MaxOpenConns: 100}
}

func (*acquireFailureConn) PrepareBatch(context.Context, string, ...driver.PrepareBatchOption) (driver.Batch, error) {
	time.Sleep(40 * time.Millisecond)
	return nil, ch.ErrAcquireConnTimeout
}

func TestFlushLogsAcquireFailure(t *testing.T) {
	capture := loggertest.Install(t)
	client := &Client{
		conn:             &acquireFailureConn{},
		logInsertTimings: true,
		circuitBreaker:   circuitbreaker.New[struct{}]("test"),
		retry:            retry.New(retry.Attempts(2), retry.Backoff(func(int) time.Duration { return 0 })),
	}
	err := flush(client, context.Background(), []staleRow{{ID: "not-logged"}})
	require.True(t, errors.Is(err, ch.ErrAcquireConnTimeout))
	failures := 0
	for _, record := range capture.Records() {
		if record.Message != "clickhouse insert attempt failed" {
			continue
		}
		failures++
		attrs := map[string]slog.Value{}
		record.Attrs(func(a slog.Attr) bool { attrs[a.Key] = a.Value; return true })
		require.Equal(t, "prepare", attrs["stage"].String())
		require.GreaterOrEqual(t, attrs["prepare_ms"].Int64(), int64(40))
		require.Equal(t, int64(failures), attrs["attempt"].Int64())
		require.Equal(t, int64(100), attrs["pool_open"].Int64())
		require.Equal(t, int64(100), attrs["pool_max"].Int64())
		require.Contains(t, attrs["error"].String(), "acquire conn timeout")
	}
	require.Equal(t, 2, failures)
}
