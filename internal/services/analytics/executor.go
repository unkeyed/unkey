package analytics

import (
	"context"
	"math"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/logger"
)

// ExecuteRequest describes one constrained analytics query execution.
type ExecuteRequest struct {
	Query           string
	WorkspaceID     string
	TableAliases    map[string]string
	AllowedTables   []string
	SecurityFilters []queryparser.SecurityFilter
}

// Execute resolves the connection, applies mandatory workspace and route-level
// security filters, and executes the parsed query.
func Execute(ctx context.Context, manager ConnectionManager, req ExecuteRequest) ([]map[string]any, error) {
	if err := assert.NotEmpty(req.WorkspaceID, "analytics parser workspace ID is required"); err != nil {
		return nil, err
	}

	conn, settings, err := manager.GetConnection(ctx, req.WorkspaceID)
	if err != nil {
		return nil, err
	}
	parser := queryparser.NewParser(queryparser.Config{
		WorkspaceID:       req.WorkspaceID,
		TableAliases:      req.TableAliases,
		AllowedTables:     req.AllowedTables,
		SecurityFilters:   append([]queryparser.SecurityFilter(nil), req.SecurityFilters...),
		Limit:             int(settings.ClickhouseWorkspaceSetting.MaxQueryResultRows),
		QueryRangeDaysMax: int32(settings.Limit.LogsRetentionDaysMax),
	})

	parsedQuery, err := parser.Parse(ctx, req.Query)
	if err != nil {
		return nil, err
	}

	logger.Debug("executing query", "original", req.Query, "parsed", parsedQuery)
	queryCtx, cancel := context.WithTimeout(ctx, clickhouse.AnalyticsQueryTimeout)
	defer cancel()
	rows, err := conn.QueryToMaps(queryCtx, parsedQuery)
	if err != nil {
		return nil, clickhouse.WrapClickHouseError(err)
	}

	for _, row := range rows {
		for column, value := range row {
			row[column] = nullifyNonFinite(value)
		}
	}

	return rows, nil
}

// nullifyNonFinite replaces a NaN or Inf value with nil.
//
// ClickHouse gives NaN for an aggregate such as quantile or avg when no row
// matches the query. It gives Inf for a division by zero. JSON has no encoding
// for a non-finite float. Thus json.Marshal fails the full response.
//
// A percentile from a rollup table needs the Float32 case. quantileTDigestMerge
// gives Float32. The aggregate state holds Float64.
//
// An array column can also hold such a value. One example is
// groupArray(latency / 0). That query still answers 500. Each array depth is a
// different Go type, and no caller writes such a query.
func nullifyNonFinite(value any) any {
	switch v := value.(type) {
	case ch.Dynamic: // QueryToMaps scans each column into a Dynamic
		if nullifyNonFinite(v.Any()) == nil {
			return nil
		}
	case float64: // quantile, avg, or a division
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil
		}
	case float32: // quantileTDigestMerge
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return nil
		}
	}

	return value
}
