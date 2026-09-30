package clickhouse

import (
	"context"
	"errors"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/pkg/retry"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
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

func TestFlushTracesAcquireFailure(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	previous := tracing.GetGlobalTraceProvider()
	tracing.SetGlobalTraceProvider(provider)
	t.Cleanup(func() {
		tracing.SetGlobalTraceProvider(previous)
		require.NoError(t, provider.Shutdown(context.Background()))
	})
	client := &Client{
		conn:           &acquireFailureConn{},
		circuitBreaker: circuitbreaker.New[struct{}]("test"),
		retry:          retry.New(retry.Attempts(2), retry.Backoff(func(int) time.Duration { return 0 })),
	}
	err := flush(client, context.Background(), []staleRow{{ID: "not-logged"}})
	require.True(t, errors.Is(err, ch.ErrAcquireConnTimeout))
	spans := recorder.Ended()
	require.Len(t, spans, 3)
	parent := spans[2]
	require.Equal(t, "clickhouse.flush", parent.Name())
	require.Equal(t, codes.Error, parent.Status().Code)
	for i, span := range spans[:2] {
		require.Equal(t, "clickhouse.insert", span.Name())
		require.Equal(t, parent.SpanContext().SpanID(), span.Parent().SpanID())
		require.Equal(t, codes.Error, span.Status().Code)
		require.Contains(t, span.Status().Description, "acquire conn timeout")
		require.Len(t, span.Events(), 1)
		require.Equal(t, "prepare", span.Events()[0].Name)
		require.GreaterOrEqual(t, span.EndTime().Sub(span.Events()[0].Time), 40*time.Millisecond)
		attrs := map[string]int64{}
		for _, attr := range span.Attributes() {
			attrs[string(attr.Key)] = attr.Value.AsInt64()
		}
		require.Equal(t, int64(i+1), attrs["attempt"])
		require.Equal(t, int64(100), attrs["pool.open"])
		require.Equal(t, int64(100), attrs["pool.max"])
	}
}
