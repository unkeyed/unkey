package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ClickHouse/ch-go/proto"
	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/circuitbreaker"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestFlushFrontlineBodyBlocksPreservesRows(t *testing.T) {
	cfg := containers.ClickHouse(t)
	for name, dsn := range map[string]string{"native": cfg.DSN, "http": cfg.HTTPDSN} {
		t.Run(name, func(t *testing.T) {
			client, err := New(Config{URL: dsn})
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, client.Close()) })
			workspaceID := uid.New(uid.WorkspacePrefix)
			rows := make([]schema.FrontlineRequest, 5)
			for i := range rows {
				rows[i] = schema.FrontlineRequest{
					RequestID:       fmt.Sprintf("request_%d", i),
					WorkspaceID:     workspaceID,
					Time:            time.Now().UnixMilli(),
					RequestBody:     strings.Repeat(fmt.Sprint(i), 512<<10),
					ResponseBody:    strings.Repeat(fmt.Sprint(i+1), 128<<10),
					RequestHeaders:  []string{"content-type: application/json"},
					ResponseHeaders: []string{"x-test: value"},
					QueryParams:     map[string][]string{"key": {"one", "two"}},
					ResponseStatus:  200,
				}
			}
			require.NoError(t, flush(client, t.Context(), rows))
			var got []schema.FrontlineRequest
			query := "SELECT " + rows[0].InsertColumns() + " FROM " + rows[0].Table() + " WHERE workspace_id = ? ORDER BY request_id"
			require.NoError(t, client.conn.Select(t.Context(), &got, query, workspaceID))
			require.Equal(t, rows, got)
		})
	}
}

func TestAppendRowsStopsOnBlockFailure(t *testing.T) {
	wantErr := errors.New("block send failed")
	batch := &bodyColumnBatch{flushErr: wantErr}
	body := strings.Repeat("x", 1<<20)
	err := appendRows(batch, []schema.FrontlineRequest{{RequestBody: body}, {ResponseBody: body}})
	require.ErrorIs(t, err, wantErr)
	require.Equal(t, 1, batch.request.Rows())
	require.Equal(t, 1, batch.response.Rows())
}

func TestFlushRetriesAfterIntermediateEOFWithOneConnection(t *testing.T) {
	cfg := containers.ClickHouse(t)
	for _, test := range []struct {
		name      string
		rows      int
		failAfter int64
	}{
		{name: "async", rows: 5, failAfter: 1 << 20},
		{name: "synchronous fallback", rows: 24, failAfter: 12 << 20},
	} {
		t.Run(test.name, func(t *testing.T) {
			opts, err := ch.ParseDSN(cfg.DSN)
			require.NoError(t, err)
			opts.MaxOpenConns = 1
			opts.MaxIdleConns = 1
			opts.Compression = &ch.Compression{Method: ch.CompressionNone}
			var written atomic.Int64
			var failed atomic.Bool
			opts.DialContext = func(ctx context.Context, addr string) (net.Conn, error) {
				conn, dialErr := (&net.Dialer{}).DialContext(ctx, "tcp", addr)
				if dialErr != nil {
					return nil, dialErr
				}
				return &blockFailureConn{Conn: conn, written: &written, failed: &failed, failAfter: test.failAfter}, nil
			}
			conn, err := ch.Open(opts)
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, conn.Close()) })
			client := &Client{
				conn:           conn,
				circuitBreaker: circuitbreaker.New[struct{}](t.Name()),
				retry:          retry.New(retry.Attempts(2)),
			}
			workspaceID := uid.New(uid.WorkspacePrefix)
			rows := make([]schema.FrontlineRequest, test.rows)
			for i := range rows {
				rows[i] = schema.FrontlineRequest{
					RequestID:    fmt.Sprintf("request_%d", i),
					WorkspaceID:  workspaceID,
					Time:         time.Now().UnixMilli(),
					RequestBody:  strings.Repeat("a", 512<<10),
					ResponseBody: strings.Repeat("b", 128<<10),
				}
			}
			ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
			t.Cleanup(cancel)
			require.NoError(t, flush(client, ctx, rows))
			require.True(t, failed.Load(), "must exercise an intermediate write failure")
			var got []schema.FrontlineRequest
			require.NoError(t, conn.Select(ctx, &got,
				"SELECT request_id, request_body, response_body FROM "+rows[0].Table()+" WHERE workspace_id = ?", workspaceID))
			seen := make(map[string]bool)
			for _, row := range got {
				require.Equal(t, rows[0].RequestBody, row.RequestBody)
				require.Equal(t, rows[0].ResponseBody, row.ResponseBody)
				seen[row.RequestID] = true
			}
			require.Len(t, seen, len(rows))
			for _, row := range rows {
				require.True(t, seen[row.RequestID])
			}
			require.NoError(t, conn.Ping(ctx))
		})
	}
}

type blockFailureConn struct {
	net.Conn
	written   *atomic.Int64
	failed    *atomic.Bool
	failAfter int64
}

func (c *blockFailureConn) Write(p []byte) (int, error) {
	if c.written.Add(int64(len(p))) > c.failAfter && c.failed.CompareAndSwap(false, true) {
		return 0, io.EOF
	}
	return c.Conn.Write(p)
}

func BenchmarkFrontlineBodyBlocks(b *testing.B) {
	rows := make([]schema.FrontlineRequest, 64)
	for i := range rows {
		rows[i].RequestBody = strings.Repeat("a", 64<<10)
		rows[i].ResponseBody = strings.Repeat("b", 128<<10)
	}
	for _, bounded := range []bool{false, true} {
		name := "single-block"
		if bounded {
			name = "bounded-blocks"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				batch := &bodyColumnBatch{}
				if bounded {
					require.NoError(b, appendRows(batch, rows))
				} else {
					for i := range rows {
						require.NoError(b, batch.AppendStruct(&rows[i]))
					}
				}
			}
		})
	}
}

type bodyColumnBatch struct {
	driver.Batch
	request  proto.ColStr
	response proto.ColStr
	flushErr error
}

func (b *bodyColumnBatch) AppendStruct(value any) error {
	row, ok := value.(*schema.FrontlineRequest)
	if !ok {
		return fmt.Errorf("unexpected row type %T", value)
	}
	b.request.Append(row.RequestBody)
	b.response.Append(row.ResponseBody)
	return nil
}

func (b *bodyColumnBatch) Flush() error {
	if b.flushErr != nil {
		return b.flushErr
	}
	b.request.Reset()
	b.response.Reset()
	return nil
}
