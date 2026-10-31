package analytics

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

type fakeConnection struct {
	clickhouse.ClickHouse
	query    string
	deadline time.Time
	rows     []map[string]any
	queryFn  func(context.Context, string) ([]map[string]any, error)
}

func (f *fakeConnection) QueryToMaps(ctx context.Context, query string, _ ...any) ([]map[string]any, error) {
	f.query = query
	f.deadline, _ = ctx.Deadline()
	if f.queryFn != nil {
		return f.queryFn(ctx, query)
	}
	if f.rows != nil {
		return f.rows, nil
	}

	return []map[string]any{{"ok": true}}, nil
}

type fakeManager struct {
	connection clickhouse.ClickHouse
	workspace  string
}

var _ ConnectionManager = (*fakeManager)(nil)

func (f *fakeManager) GetConnection(_ context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	f.workspace = workspaceID
	return f.connection, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
		ClickhouseMaxQueryResultRows: 100,
		QuotaLogsRetentionDays:       30,
	}, nil
}

// TestExecuteAppliesSecurityFilters guarantees route and workspace filters
// reach ClickHouse.
func TestExecuteAppliesSecurityFilters(t *testing.T) {
	connection := &fakeConnection{}
	manager := &fakeManager{connection: connection}
	startedAt := time.Now()
	rows, err := Execute(context.Background(), manager, ExecuteRequest{
		Query:           "SELECT * FROM events WHERE namespace_id = 'requested' OR 1 = 1",
		WorkspaceID:     "ws_test",
		TableAliases:    map[string]string{"events": "default.events"},
		AllowedTables:   []string{"default.events"},
		SecurityFilters: []queryparser.SecurityFilter{{Column: "namespace_id", AllowedValues: []string{"allowed"}}},
	})
	require.NoError(t, err)
	require.Equal(t, "ws_test", manager.workspace)
	require.Equal(t, []map[string]any{{"ok": true}}, rows)
	require.Contains(t, connection.query, "events.namespace_id IN ('allowed')")
	require.Contains(t, connection.query, "events.workspace_id = 'ws_test'")
	require.WithinDuration(t, startedAt.Add(clickhouse.AnalyticsQueryTimeout), connection.deadline, time.Second)
}

func TestExecuteEmptySecurityFilterFailsClosed(t *testing.T) {
	connection := &fakeConnection{}
	_, err := Execute(context.Background(), &fakeManager{connection: connection}, ExecuteRequest{
		Query:           "SELECT * FROM events",
		WorkspaceID:     "ws_test",
		TableAliases:    map[string]string{"events": "default.events"},
		AllowedTables:   []string{"default.events"},
		SecurityFilters: []queryparser.SecurityFilter{{Column: "namespace_id"}},
	})
	require.NoError(t, err)
	require.Contains(t, connection.query, "AND (0)")
}

func TestExecutePreservesNilAndEmptySecurityScopes(t *testing.T) {
	for name, test := range map[string]struct {
		scopes   []queryparser.SecurityScope
		contains string
	}{
		"nil is unrestricted": {scopes: nil, contains: "events.workspace_id = 'ws_test' LIMIT 100"},
		"empty denies all":    {scopes: []queryparser.SecurityScope{}, contains: "AND ((0))"},
	} {
		t.Run(name, func(t *testing.T) {
			connection := &fakeConnection{}
			_, err := Execute(context.Background(), &fakeManager{connection: connection}, ExecuteRequest{
				Query:          "SELECT * FROM events",
				WorkspaceID:    "ws_test",
				TableAliases:   map[string]string{"events": "default.events"},
				AllowedTables:  []string{"default.events"},
				SecurityScopes: test.scopes,
			})
			require.NoError(t, err)
			require.Contains(t, connection.query, test.contains)
		})
	}
}

// TestNonFiniteValueBreaksJSON records the reason for the nullifyNonFinite pass.
// A ClickHouse NaN reaches the response encoder inside a Dynamic wrapper.
func TestNonFiniteValueBreaksJSON(t *testing.T) {
	for name, value := range map[string]any{
		"nan":          math.NaN(),
		"positive inf": math.Inf(1),
		"negative inf": math.Inf(-1),
		"float32 nan":  float32(math.NaN()),
	} {
		t.Run(name, func(t *testing.T) {
			_, err := json.Marshal(map[string]any{"value": ch.NewDynamic(value)})
			require.ErrorContains(t, err, "unsupported value")
		})
	}
}

// TestExecuteNullifiesNonFiniteValues guarantees a row with a non-finite value
// stays JSON encodable. The driver puts each column in a Dynamic wrapper. Thus
// the pass must read through that wrapper.
func TestExecuteNullifiesNonFiniteValues(t *testing.T) {
	for name, column := range map[string]any{
		"dynamic nan":          ch.NewDynamic(math.NaN()),
		"dynamic positive inf": ch.NewDynamic(math.Inf(1)),
		"dynamic negative inf": ch.NewDynamic(math.Inf(-1)),
		"typed dynamic nan":    ch.NewDynamicWithType(math.NaN(), "Float64"),
		"float32 nan":          float32(math.NaN()),
		"float32 inf":          float32(math.Inf(1)),
		"dynamic null":         ch.NewDynamic(nil),
		"bare nan":             math.NaN(),
	} {
		t.Run(name, func(t *testing.T) {
			connection := &fakeConnection{rows: []map[string]any{{"value": column}}}
			rows, err := Execute(context.Background(), &fakeManager{connection: connection}, ExecuteRequest{
				Query:           "SELECT quantile(0.95)(latency) AS value FROM events",
				WorkspaceID:     "ws_test",
				TableAliases:    map[string]string{"events": "default.events"},
				AllowedTables:   []string{"default.events"},
				SecurityFilters: nil,
			})
			require.NoError(t, err)
			require.Len(t, rows, 1)
			require.Nil(t, rows[0]["value"])

			encoded, err := json.Marshal(rows)
			require.NoError(t, err)
			require.Contains(t, string(encoded), `"value":null`)
		})
	}
}

// TestExecuteKeepsFiniteValues guarantees the nullifyNonFinite pass changes no
// legal value. A zero and a negative number must survive.
func TestExecuteKeepsFiniteValues(t *testing.T) {
	connection := &fakeConnection{rows: []map[string]any{{
		"total": ch.NewDynamic(uint64(0)),
		"p95":   ch.NewDynamic(float64(0)),
		"drift": ch.NewDynamic(-12.5),
		"path":  ch.NewDynamic("/kebap"),
		"tags":  ch.NewDynamic([]string{"kebap"}),
	}}}

	rows, err := Execute(context.Background(), &fakeManager{connection: connection}, ExecuteRequest{
		Query:           "SELECT quantile(0.95)(latency) AS p95 FROM events",
		WorkspaceID:     "ws_test",
		TableAliases:    map[string]string{"events": "default.events"},
		AllowedTables:   []string{"default.events"},
		SecurityFilters: nil,
	})
	require.NoError(t, err)
	require.Len(t, rows, 1)

	encoded, err := json.Marshal(rows[0])
	require.NoError(t, err)

	var decoded map[string]any
	require.NoError(t, json.Unmarshal(encoded, &decoded))
	require.Equal(t, map[string]any{
		"total": float64(0),
		"p95":   float64(0),
		"drift": -12.5,
		"path":  "/kebap",
		"tags":  []any{"kebap"},
	}, decoded)
}

// TestExecuteRequiresParserWorkspaceID guarantees callers cannot open an
// unscoped analytics connection or inject an empty workspace predicate.
func TestExecuteRequiresParserWorkspaceID(t *testing.T) {
	_, err := Execute(context.Background(), &fakeManager{}, ExecuteRequest{
		Query:           "SELECT count(*) FROM events",
		WorkspaceID:     "",
		TableAliases:    nil,
		AllowedTables:   nil,
		SecurityFilters: nil,
	})
	require.ErrorContains(t, err, "analytics parser workspace ID is required")
}

func TestExecuteRetriesOnlyInterruptedConnections(t *testing.T) {
	for _, test := range []struct {
		name  string
		err   error
		retry bool
	}{
		{name: "HTTP EOF", err: &url.Error{Op: "Post", URL: "https://clickhouse.example.com", Err: io.EOF}, retry: true},
		{name: "truncated response", err: io.ErrUnexpectedEOF, retry: true},
		{name: "connection reset", err: &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}, retry: true},
		{name: "broken pipe", err: &net.OpError{Op: "write", Net: "tcp", Err: syscall.EPIPE}, retry: true},
		{name: "SQL error", err: &ch.Exception{Code: 62, Message: "Syntax error"}},
		{name: "authentication", err: &ch.Exception{Code: 516, Message: "Authentication failed"}},
		{name: "query limit", err: &ch.Exception{Code: 241, Message: "Memory limit exceeded"}},
		{name: "connection pool exhausted", err: ch.ErrAcquireConnTimeout},
		{name: "canceled", err: context.Canceled},
		{name: "deadline exceeded", err: context.DeadlineExceeded},
		{name: "untyped EOF", err: errors.New("invalid value: EOF")},
	} {
		t.Run(test.name, func(t *testing.T) {
			failure := test.err
			connection := &fakeConnection{queryFn: func(context.Context, string) ([]map[string]any, error) {
				if failure != nil {
					err := failure
					failure = nil
					return []map[string]any{{"answer": -1}}, clickhouse.WrapClickHouseError(err)
				}
				return []map[string]any{{"answer": 42}}, nil
			}}
			rows, err := Execute(t.Context(), &fakeManager{connection: connection}, ExecuteRequest{
				Query:         "SELECT 42 AS answer FROM events",
				WorkspaceID:   uid.New(uid.WorkspacePrefix),
				TableAliases:  map[string]string{"events": "default.events"},
				AllowedTables: []string{"default.events"},
			})
			if test.retry {
				require.NoError(t, err)
				require.Equal(t, []map[string]any{{"answer": 42}}, rows)
			} else {
				require.ErrorIs(t, err, test.err)
				require.Nil(t, rows)
			}
		})
	}
}

func TestExecuteStopsAfterOneRetry(t *testing.T) {
	failures := []error{io.EOF, &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}}
	connection := &fakeConnection{queryFn: func(context.Context, string) ([]map[string]any, error) {
		if len(failures) > 0 {
			err := failures[0]
			failures = failures[1:]
			return nil, clickhouse.WrapClickHouseError(err)
		}
		return []map[string]any{{"answer": 42}}, nil
	}}
	rows, err := Execute(t.Context(), &fakeManager{connection: connection}, ExecuteRequest{
		Query:         "SELECT 42 AS answer FROM events",
		WorkspaceID:   uid.New(uid.WorkspacePrefix),
		TableAliases:  map[string]string{"events": "default.events"},
		AllowedTables: []string{"default.events"},
	})
	require.ErrorIs(t, err, syscall.ECONNRESET)
	require.Nil(t, rows)
	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.Data.Analytics.ConnectionFailed.URN(), code)
}

func TestExecuteRetryPreservesDeadlineAndSecurityFilters(t *testing.T) {
	var deadline time.Time
	workspaceID := uid.New(uid.WorkspacePrefix)
	connection := &fakeConnection{queryFn: func(ctx context.Context, query string) ([]map[string]any, error) {
		require.Contains(t, query, "events.workspace_id = '"+workspaceID+"'")
		require.Contains(t, query, "events.namespace_id IN ('allowed')")
		deadlineNext, ok := ctx.Deadline()
		require.True(t, ok)
		if deadline.IsZero() {
			deadline = deadlineNext
			return nil, clickhouse.WrapClickHouseError(io.EOF)
		}
		require.Equal(t, deadline, deadlineNext)
		return []map[string]any{{"answer": 42}}, nil
	}}
	rows, err := Execute(t.Context(), &fakeManager{connection: connection}, ExecuteRequest{
		Query:           "SELECT * FROM events",
		WorkspaceID:     workspaceID,
		TableAliases:    map[string]string{"events": "default.events"},
		AllowedTables:   []string{"default.events"},
		SecurityFilters: []queryparser.SecurityFilter{{Column: "namespace_id", AllowedValues: []string{"allowed"}}},
	})
	require.NoError(t, err)
	require.Equal(t, []map[string]any{{"answer": 42}}, rows)
}

func TestExecuteDoesNotRetryAfterContextEnds(t *testing.T) {
	for _, canceled := range []bool{true, false} {
		name := "deadline"
		if canceled {
			name = "canceled"
		}
		t.Run(name, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if canceled {
				ctx, cancel = context.WithCancel(t.Context())
			} else {
				ctx, cancel = context.WithTimeout(t.Context(), 20*time.Millisecond)
			}
			t.Cleanup(cancel)
			interrupted := false
			connection := &fakeConnection{queryFn: func(context.Context, string) ([]map[string]any, error) {
				if !interrupted {
					interrupted = true
					if canceled {
						cancel()
					} else {
						<-ctx.Done()
					}
					return nil, clickhouse.WrapClickHouseError(io.EOF)
				}
				return []map[string]any{{"answer": 42}}, nil
			}}
			rows, err := Execute(ctx, &fakeManager{connection: connection}, ExecuteRequest{
				Query:         "SELECT 42 AS answer FROM events",
				WorkspaceID:   uid.New(uid.WorkspacePrefix),
				TableAliases:  map[string]string{"events": "default.events"},
				AllowedTables: []string{"default.events"},
			})
			if canceled {
				require.ErrorIs(t, err, context.Canceled)
			} else {
				require.ErrorIs(t, err, context.DeadlineExceeded)
			}
			require.Nil(t, rows)
		})
	}
}

func TestExecuteRecoversAfterHTTPConnectionCloses(t *testing.T) {
	config := containers.ClickHouse(t)
	target, err := url.Parse(config.HTTPDSN)
	require.NoError(t, err)
	target.Path = ""
	target.RawQuery = ""
	proxy := httputil.NewSingleHostReverseProxy(target)
	transport := &http.Transport{}
	t.Cleanup(transport.CloseIdleConnections)
	proxy.Transport = transport

	var dropNext atomic.Bool
	var connections atomic.Int64
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !dropNext.Swap(false) {
			proxy.ServeHTTP(w, r)
			return
		}
		conn, _, err := http.NewResponseController(w).Hijack()
		if err != nil {
			t.Errorf("hijack analytics connection: %v", err)
			return
		}
		if err := conn.Close(); err != nil {
			t.Errorf("close analytics connection: %v", err)
		}
	}))
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	t.Cleanup(server.Close)
	clientURL, err := url.Parse(server.URL)
	require.NoError(t, err)
	clientURL.User = target.User
	client, err := clickhouse.New(clickhouse.Config{URL: clientURL.String()})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })

	connectionsBefore := connections.Load()
	dropNext.Store(true)
	rows, err := Execute(t.Context(), &fakeManager{connection: client}, ExecuteRequest{
		Query:         "SELECT count(*) + 42 AS answer FROM events",
		WorkspaceID:   uid.New(uid.WorkspacePrefix),
		TableAliases:  map[string]string{"events": "default.key_verifications_raw_v2"},
		AllowedTables: []string{"default.key_verifications_raw_v2"},
	})
	require.NoError(t, err)
	require.Greater(t, connections.Load(), connectionsBefore)
	encoded, err := json.Marshal(rows)
	require.NoError(t, err)
	var result []struct {
		Answer uint64 `json:"answer"`
	}
	require.NoError(t, json.Unmarshal(encoded, &result))
	require.Len(t, result, 1)
	require.Equal(t, uint64(42), result[0].Answer)
}
