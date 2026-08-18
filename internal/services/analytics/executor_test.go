package analytics

import (
	"context"
	"encoding/json"
	"math"
	"testing"
	"time"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/db"
)

type fakeConnection struct {
	clickhouse.ClickHouse
	query    string
	deadline time.Time
	rows     []map[string]any
}

func (f *fakeConnection) QueryToMaps(ctx context.Context, query string, _ ...any) ([]map[string]any, error) {
	f.query = query
	f.deadline, _ = ctx.Deadline()
	if f.rows != nil {
		return f.rows, nil
	}

	return []map[string]any{{"ok": true}}, nil
}

type fakeManager struct {
	connection *fakeConnection
	workspace  string
}

var _ ConnectionManager = (*fakeManager)(nil)

func (f *fakeManager) GetConnection(_ context.Context, workspaceID string) (clickhouse.ClickHouse, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow, error) {
	f.workspace = workspaceID
	return f.connection, db.FindClickhouseWorkspaceSettingsByWorkspaceIDRow{
		ClickhouseWorkspaceSetting: db.ClickhouseWorkspaceSetting{MaxQueryResultRows: 100},
		Limit:                      db.Limit{LogsRetentionDaysMax: 30},
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
