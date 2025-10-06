package queryparser

import (
	"context"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	resulttransformer "github.com/unkeyed/unkey/go/pkg/clickhouse/result-transformer"
)

// VirtualColumnResolver resolves virtual IDs to actual IDs
type VirtualColumnResolver func(ctx context.Context, virtualIDs []string) (map[string]string, error)

// VirtualColumn configuration
type VirtualColumn struct {
	ActualColumn string
	Aliases      []string // Additional names that map to this column (e.g. ["api_id"] for "apiId")
	Resolver     VirtualColumnResolver
}

// SecurityFilter represents a row-level security constraint
type SecurityFilter struct {
	Column        string   // Column name - can be virtual (e.g., "api_id") or actual (e.g., "status")
	AllowedValues []string // Values user is allowed to access
}

// Config for the parser
type Config struct {
	WorkspaceID     string
	TableAliases    map[string]string
	AllowedTables   []string
	VirtualColumns  map[string]VirtualColumn
	SecurityFilters []SecurityFilter // Row-level security filters (auto-injected)
	Limit           int
}

// ParseResult contains the rewritten query and column mappings for transformation
type ParseResult struct {
	Query          string                            // The rewritten SQL query
	ColumnMappings []resulttransformer.ColumnMapping // Maps result columns to actual columns for transformation
}

// Parser rewrites ClickHouse queries
type Parser struct {
	config         Config
	stmt           *clickhouse.SelectQuery
	aliasMap       map[string]string // Maps aliases to canonical virtual column names
	columnMappings map[string]string // Maps result column names to actual ClickHouse columns
}
