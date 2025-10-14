package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// SecurityFilter represents a row-level security constraint
type SecurityFilter struct {
	Column        string   // Column name
	AllowedValues []string // Values user is allowed to access
}

// Config for the parser
type Config struct {
	WorkspaceID     string
	TableAliases    map[string]string
	AllowedTables   []string
	SecurityFilters []SecurityFilter // Row-level security filters (auto-injected)
	Limit           int
}

// Parser rewrites ClickHouse queries
type Parser struct {
	config   Config
	stmt     *clickhouse.SelectQuery
	cteNames map[string]bool // Tracks CTE names defined in WITH clause
}
