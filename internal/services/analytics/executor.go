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

// QueryConfig describes the stable route policy for one analytics query.
type QueryConfig struct {
	WorkspaceID   string
	TableAliases  map[string]string
	AllowedTables []string
}

// ExecuteRequest describes one constrained analytics query execution.
type ExecuteRequest struct {
	Query                  string
	Config                 QueryConfig
	InitialSecurityFilters []queryparser.SecurityFilter
	Policy                 QueryPolicy
}

// Execute resolves the connection first, then parses, applies route policy and
// executes. Returned policy filters are applied by reparsing before execution.
func Execute(ctx context.Context, manager ConnectionManager, req ExecuteRequest) ([]map[string]any, error) {
	if err := assert.NotEmpty(req.Config.WorkspaceID, "analytics parser workspace ID is required"); err != nil {
		return nil, err
	}

	conn, settings, err := manager.GetConnection(ctx, req.Config.WorkspaceID)
	if err != nil {
		return nil, err
	}
	parserConfig := queryparser.Config{
		WorkspaceID:       req.Config.WorkspaceID,
		TableAliases:      req.Config.TableAliases,
		AllowedTables:     req.Config.AllowedTables,
		SecurityFilters:   append([]queryparser.SecurityFilter(nil), req.InitialSecurityFilters...),
		Limit:             int(settings.ClickhouseWorkspaceSetting.MaxQueryResultRows),
		QueryRangeDaysMax: settings.Quotas.LogsRetentionDays,
	}

	parser := queryparser.NewParser(parserConfig)
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
			parserConfig.SecurityFilters = append(parserConfig.SecurityFilters, filters...)
			parsedQuery, err = queryparser.NewParser(parserConfig).Parse(ctx, req.Query)
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
