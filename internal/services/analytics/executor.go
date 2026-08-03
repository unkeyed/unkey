package analytics

import (
	"context"

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
		Limit:             int(settings.ClickhouseMaxQueryResultRows),
		QueryRangeDaysMax: settings.QuotaLogsRetentionDays,
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
	return rows, nil
}
