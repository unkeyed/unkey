package analytics

import (
	"context"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	queryparser "github.com/unkeyed/unkey/pkg/clickhouse/query-parser"
	"github.com/unkeyed/unkey/pkg/logger"
)

// QueryPolicy authorizes a parsed query and may return additional row filters.
// The executor reparses the original query with those filters before execution.
type QueryPolicy func(*queryparser.Parser) ([]queryparser.SecurityFilter, error)

// FilterBuilder returns row filters after the workspace connection is resolved
// and before the query is parsed.
type FilterBuilder func() ([]queryparser.SecurityFilter, error)

// ExecuteRequest describes one constrained analytics query execution.
type ExecuteRequest struct {
	Query         string
	ParserConfig  queryparser.Config
	FilterBuilder FilterBuilder
	Policy        QueryPolicy
}

// Execute resolves the connection first, then parses, applies route policy and
// executes. Returned policy filters are applied by reparsing before execution.
func Execute(ctx context.Context, manager ConnectionManager, req ExecuteRequest) ([]map[string]any, error) {
	if err := assert.NotEmpty(req.ParserConfig.WorkspaceID, "analytics parser workspace ID is required"); err != nil {
		return nil, err
	}

	conn, settings, err := manager.GetConnection(ctx, req.ParserConfig.WorkspaceID)
	if err != nil {
		return nil, err
	}
	req.ParserConfig.Limit = int(settings.ClickhouseWorkspaceSetting.MaxQueryResultRows)
	req.ParserConfig.QueryRangeDaysMax = settings.Quotas.LogsRetentionDays
	if req.FilterBuilder != nil {
		filters, filterErr := req.FilterBuilder()
		if filterErr != nil {
			return nil, filterErr
		}
		req.ParserConfig.SecurityFilters = append(req.ParserConfig.SecurityFilters, filters...)
	}

	parser := queryparser.NewParser(req.ParserConfig)
	parsedQuery, err := parser.Parse(ctx, req.Query)
	if err != nil {
		return nil, err
	}
	if req.Policy != nil {
		filters, policyErr := req.Policy(parser)
		if policyErr != nil {
			return nil, policyErr
		}
		if len(filters) > 0 {
			req.ParserConfig.SecurityFilters = append(req.ParserConfig.SecurityFilters, filters...)
			parsedQuery, err = queryparser.NewParser(req.ParserConfig).Parse(ctx, req.Query)
			if err != nil {
				return nil, err
			}
		}
	}

	logger.Debug("executing query", "original", req.Query, "parsed", parsedQuery)
	rows, err := conn.QueryToMaps(ctx, parsedQuery)
	if err != nil {
		return nil, clickhouse.WrapClickHouseError(err)
	}
	return rows, nil
}
